package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/atvirokodosprendimai/knitu/internal/agent/docker"
	"github.com/atvirokodosprendimai/knitu/internal/messaging"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "knit-agent",
		Usage: "The agent that runs on each node to execute tasks.",
		Commands: []*cli.Command{
			{
				Name:  "start",
				Usage: "Start the Knit agent",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "nats-url",
						Value: nats.DefaultURL,
						Usage: "URL of the NATS server to connect to (e.g., nats://10.0.0.1:4222)",
					},
					&cli.StringFlag{
						Name:  "node-id-file",
						Value: "/var/lib/knit-agent/node-id",
						Usage: "Path to persistent node id file",
					},
					&cli.StringFlag{
						Name:  "labels",
						Value: "",
						Usage: "Node labels as comma-separated key=value pairs (e.g., region=eu,role=api)",
					},
				},
				Action: runAgent,
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func runAgent(ctx context.Context, cmd *cli.Command) error {
	log.Println("Starting Knit Agent...")

	nodeIDFile := cmd.String("node-id-file")
	nodeID, err := loadOrCreateNodeID(nodeIDFile)
	if err != nil {
		return fmt.Errorf("could not load/create node id: %w", err)
	}
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	log.Printf("Agent initialized with Node ID: %s on Host: %s", nodeID, hostname)
	labels := parseLabels(cmd.String("labels"))

	// 1. Connect to NATS via the provided URL
	natsURL := cmd.Value("nats-url").(string)
	log.Printf("Attempting to connect to NATS at %s...", natsURL)
	nc, err := messaging.Connect(natsURL)
	if err != nil {
		return err
	}
	defer nc.Close()

	// 2. Create Docker Client
	dockerClient, err := docker.NewClient()
	if err != nil {
		return err
	}

	// 3. Subscribe to deployment tasks
	_, err = nc.Subscribe(messaging.SubjectTaskDeployBroadcast, deploymentTaskHandler(ctx, nodeID, dockerClient, nc))
	if err != nil {
		return fmt.Errorf("could not subscribe to deployment tasks: %w", err)
	}
	_, err = nc.Subscribe(messaging.SubjectTaskDeployNode(nodeID), deploymentTaskHandler(ctx, nodeID, dockerClient, nc))
	if err != nil {
		return fmt.Errorf("could not subscribe to node deployment tasks: %w", err)
	}
	_, err = nc.Subscribe(messaging.SubjectTaskUndeployBroadcast, undeployTaskHandler(ctx, nodeID, dockerClient, nc))
	if err != nil {
		return fmt.Errorf("could not subscribe to undeploy tasks: %w", err)
	}
	log.Println("Subscribed to deployment tasks.")

	// 4. Start heartbeat ticker
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			publishHeartbeat(nc, nodeID, hostname, labels)
		case <-ctx.Done():
			log.Println("Shutting down agent...")
			return nil
		}
	}
}

func publishHeartbeat(nc *nats.Conn, nodeID, hostname string, labels map[string]string) {
	hb := messaging.Heartbeat{
		NodeID:    nodeID,
		Hostname:  hostname,
		Labels:    labels,
		Timestamp: time.Now(),
	}
	hbBytes, err := json.Marshal(hb)
	if err != nil {
		log.Printf("[ERROR] Marshalling heartbeat: %v", err)
		return
	}
	if err := nc.Publish(messaging.SubjectAgentHeartbeat, hbBytes); err != nil {
		log.Printf("[ERROR] Publishing heartbeat: %v", err)
	}
}

func parseLabels(raw string) map[string]string {
	labels := map[string]string{}
	if strings.TrimSpace(raw) == "" {
		return labels
	}
	for _, kv := range strings.Split(raw, ",") {
		parts := strings.SplitN(strings.TrimSpace(kv), "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		if k == "" {
			continue
		}
		labels[k] = v
	}
	return labels
}

func loadOrCreateNodeID(path string) (string, error) {
	if b, err := os.ReadFile(path); err == nil {
		id := strings.TrimSpace(string(b))
		if id != "" {
			return id, nil
		}
	}

	id := uuid.New().String()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(id+"\n"), 0o600); err != nil {
		return "", err
	}
	return id, nil
}

func deploymentTaskHandler(ctx context.Context, nodeID string, dc *docker.Client, nc *nats.Conn) nats.MsgHandler {
	return func(m *nats.Msg) {
		var task messaging.DeployTask
		if err := json.Unmarshal(m.Data, &task); err != nil {
			log.Printf("[ERROR] Unmarshalling deploy task: %v", err)
			return
		}

		log.Printf("[INFO] Received deploy task for '%s' (ID: %d)", task.Name, task.DeploymentID)

		status := messaging.TaskStatus{
			TaskType:     "deploy",
			DeploymentID: task.DeploymentID,
			NodeID:       nodeID,
			Success:      false,
		}

		containerID, err := dc.DeployContainer(ctx, &task)
		if err != nil {
			log.Printf("[ERROR] Failed to deploy container for '%s': %v", task.Name, err)
			status.Message = err.Error()
		} else {
			log.Printf("[INFO] Container for '%s' started successfully: %s", task.Name, containerID)
			status.Success = true
			status.ContainerID = containerID
		}

		statusBytes, err := json.Marshal(status)
		if err != nil {
			log.Printf("[ERROR] Marshalling status response: %v", err)
			return
		}

		if err := nc.Publish(messaging.SubjectTaskStatus, statusBytes); err != nil {
			log.Printf("[ERROR] Publishing task status: %v", err)
		}
	}
}

func undeployTaskHandler(ctx context.Context, nodeID string, dc *docker.Client, nc *nats.Conn) nats.MsgHandler {
	return func(m *nats.Msg) {
		var task messaging.UndeployTask
		if err := json.Unmarshal(m.Data, &task); err != nil {
			log.Printf("[ERROR] Unmarshalling undeploy task: %v", err)
			return
		}

		status := messaging.TaskStatus{
			TaskType:     "undeploy",
			DeploymentID: task.DeploymentID,
			NodeID:       nodeID,
			Success:      false,
		}

		if err := dc.UndeployContainer(ctx, task.Name); err != nil {
			status.Message = err.Error()
			log.Printf("[ERROR] Failed to undeploy '%s': %v", task.Name, err)
		} else {
			status.Success = true
			log.Printf("[INFO] Undeployed '%s'", task.Name)
		}

		b, err := json.Marshal(status)
		if err != nil {
			log.Printf("[ERROR] Marshalling undeploy status: %v", err)
			return
		}
		if err := nc.Publish(messaging.SubjectTaskStatus, b); err != nil {
			log.Printf("[ERROR] Publishing undeploy status: %v", err)
		}
	}
}

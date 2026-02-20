package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
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
						Usage: "URL of the NATS server to connect to",
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

	nodeID := uuid.New().String()
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	log.Printf("Agent initialized with Node ID: %s on Host: %s", nodeID, hostname)

	// Connect to NATS
	natsURL := cmd.Value("nats-url").(string)
	nc, err := messaging.Connect(natsURL)
	if err != nil {
		return err
	}
	defer nc.Close()

	// Create Docker Client
	dockerClient, err := docker.NewClient()
	if err != nil {
		return err
	}

	// Subscribe to deployment tasks
	_, err = nc.Subscribe(messaging.SubjectTaskDeployBroadcast, deploymentTaskHandler(ctx, nodeID, dockerClient, nc))
	if err != nil {
		return fmt.Errorf("could not subscribe to deployment tasks: %w", err)
	}
	log.Println("Subscribed to deployment tasks.")

	// Start heartbeat ticker
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hb := messaging.Heartbeat{
				NodeID:    nodeID,
				Hostname:  hostname,
				Timestamp: time.Now(),
			}
			hbBytes, err := json.Marshal(hb)
			if err != nil {
				log.Printf("[ERROR] Marshalling heartbeat: %v", err)
				continue
			}
			if err := nc.Publish(messaging.SubjectAgentHeartbeat, hbBytes); err != nil {
				log.Printf("[ERROR] Publishing heartbeat: %v", err)
			}
		case <-ctx.Done():
			log.Println("Shutting down agent...")
			return nil
		}
	}
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

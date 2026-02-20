package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/atvirokodosprendimai/knitu/internal/db"
	"github.com/atvirokodosprendimai/knitu/internal/messaging"
	"github.com/atvirokodosprendimai/knitu/internal/spec"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/urfave/cli/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func main() {
	cmd := &cli.Command{
		Name:  "knit-server",
		Usage: "The central control plane for the Knit container orchestrator.",
		Commands: []*cli.Command{
			{
				Name:  "start",
				Usage: "Start the Knit server and embedded NATS",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "http-addr", Value: ":8080", Usage: "HTTP server address"},
					&cli.StringFlag{Name: "db-path", Value: "knit.db", Usage: "Path to the SQLite database file"},
					&cli.IntFlag{Name: "nats-port", Value: 4222, Usage: "Port for the embedded NATS server"},
				},
				Action: runServer,
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func runServer(ctx context.Context, cmd *cli.Command) error {
	log.Println("Starting Knit Server...")

	// 1. Start Embedded NATS Server
	natsPort := cmd.Value("nats-port").(int)
	ns, err := server.NewServer(&server.Options{Port: natsPort})
	if err != nil {
		return fmt.Errorf("could not start embedded NATS server: %w", err)
	}
	go ns.Start()
	if !ns.ReadyForConnections(4 * time.Second) {
		return fmt.Errorf("embedded NATS server did not become ready")
	}
	log.Printf("Embedded NATS server started on port %d", natsPort)
	natsURL := ns.ClientURL()

	// 2. Initialize Database
	dbPath := cmd.Value("db-path").(string)
	gormDB, err := db.NewDatabase(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// 3. Connect to our own embedded NATS
	nc, err := messaging.Connect(natsURL)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	defer nc.Close()

	// 4. Subscribe to Subjects
	_, err = nc.Subscribe(messaging.SubjectAgentHeartbeat, heartbeatHandler(gormDB))
	if err != nil {
		return fmt.Errorf("failed to subscribe to heartbeats: %w", err)
	}
	_, err = nc.Subscribe(messaging.SubjectTaskStatus, taskStatusHandler(gormDB))
	if err != nil {
		return fmt.Errorf("failed to subscribe to task status: %w", err)
	}
	log.Println("Subscribed to agent heartbeats and task statuses.")

	// 5. Start Chi HTTP Server
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
	})
	r.Post("/deployments", deploymentCreateHandler(gormDB, nc))

	httpAddr := cmd.Value("http-addr").(string)
	log.Printf("HTTP server listening on %s", httpAddr)
	return http.ListenAndServe(httpAddr, r)
}

func deploymentCreateHandler(gormDB *gorm.DB, nc *nats.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var spec spec.DeploymentSpec
		if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		deployment := db.Deployment{
			Name:  spec.Name,
			Image: spec.Image,
		}

		if err := gormDB.Create(&deployment).Error; err != nil {
			http.Error(w, fmt.Sprintf("Failed to save deployment: %v", err), http.StatusInternalServerError)
			return
		}

		task := messaging.DeployTask{
			DeploymentID:   deployment.ID,
			DeploymentSpec: spec,
		}

		taskBytes, err := json.Marshal(task)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create task: %v", err), http.StatusInternalServerError)
			return
		}

		if err := nc.Publish(messaging.SubjectTaskDeployBroadcast, taskBytes); err != nil {
			http.Error(w, fmt.Sprintf("Failed to publish task: %v", err), http.StatusInternalServerError)
			return
		}

		log.Printf("[INFO] Published deployment task for '%s' (ID: %d)", deployment.Name, deployment.ID)
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(deployment)
	}
}

func taskStatusHandler(gormDB *gorm.DB) nats.MsgHandler {
	return func(m *nats.Msg) {
		var status messaging.TaskStatus
		if err := json.Unmarshal(m.Data, &status); err != nil {
			log.Printf("[ERROR] Unmarshalling task status: %v", err)
			return
		}

		log.Printf("[INFO] Received task status: DeploymentID=%d, Success=%v from NodeID=%s", status.DeploymentID, status.Success, status.NodeID)

		// Find the internal Node PK from the agent's string ID
		var node db.Node
		if err := gormDB.First(&node, "node_id = ?", status.NodeID).Error; err != nil {
			log.Printf("[ERROR] Could not find node with NodeID %s: %v", status.NodeID, err)
			return
		}

		instance := db.ContainerInstance{
			DeploymentID: status.DeploymentID,
			NodeID:       node.ID, // Use the uint PK of the found node
			ContainerID:  status.ContainerID,
			Status:       "failed",
		}
		if status.Success {
			instance.Status = "running"
		}

		if err := gormDB.Create(&instance).Error; err != nil {
			log.Printf("[ERROR] Creating container instance record: %v", err)
		}
	}
}

func heartbeatHandler(gormDB *gorm.DB) nats.MsgHandler {
	return func(m *nats.Msg) {
		var hb messaging.Heartbeat
		if err := json.Unmarshal(m.Data, &hb); err != nil {
			log.Printf("[ERROR] Unmarshalling heartbeat: %v", err)
			return
		}

		log.Printf("[INFO] Heartbeat received: NodeID=%s, Hostname=%s", hb.NodeID, hb.Hostname)

		node := db.Node{
			NodeID:        hb.NodeID,
			Hostname:      hb.Hostname,
			LastHeartbeat: hb.Timestamp,
			Status:        "healthy",
		}

		// Upsert the node record based on the unique NodeID
		result := gormDB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "node_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"hostname", "last_heartbeat", "status"}),
		}).Create(&node)

		if result.Error != nil {
			log.Printf("[ERROR] Upserting node: %v", result.Error)
		}
	}
}

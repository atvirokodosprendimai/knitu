package messaging

import (
	"github.com/atvirokodosprendimai/knitu/internal/spec"
	"log"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	// SubjectAgentHeartbeat is the subject for agent heartbeats.
	SubjectAgentHeartbeat = "knit.agent.heartbeat"
	// SubjectTaskDeployBroadcast is the subject for broadcasting new deployment tasks.
	SubjectTaskDeployBroadcast = "knit.tasks.deploy.broadcast"
	// SubjectTaskStatus is the subject for agents to report the status of a task.
	SubjectTaskStatus = "knit.task.status"
	// SubjectTaskUndeployBroadcast is the subject for undeploy tasks.
	SubjectTaskUndeployBroadcast = "knit.tasks.undeploy.broadcast"
)

// Heartbeat is the message sent by an agent.
type Heartbeat struct {
	NodeID    string            `json:"node_id"`
	Hostname  string            `json:"hostname"`
	Labels    map[string]string `json:"labels,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// SubjectTaskDeployNode returns the node-specific subject for deployments.
func SubjectTaskDeployNode(nodeID string) string {
	return "knit.tasks.deploy.node." + strings.ReplaceAll(nodeID, " ", "")
}

// DeployTask is the message sent from the server to an agent to start a deployment.
type DeployTask struct {
	DeploymentID uint `json:"deployment_id"`
	spec.DeploymentSpec
}

// UndeployTask asks agents to remove a deployed container.
type UndeployTask struct {
	DeploymentID uint   `json:"deployment_id"`
	Name         string `json:"name"`
}

// TaskStatus is the message sent from an agent to the server to report task status.
type TaskStatus struct {
	TaskType     string `json:"task_type"` // e.g., "deploy"
	DeploymentID uint   `json:"deployment_id"`
	NodeID       string `json:"node_id"`
	Success      bool   `json:"success"`
	Message      string `json:"message"` // Error message on failure
	ContainerID  string `json:"container_id,omitempty"`
}

// Connect establishes a connection to a NATS server.
func Connect(natsURL string) (*nats.Conn, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}
	log.Println("Connected to NATS server at", natsURL)
	return nc, nil
}

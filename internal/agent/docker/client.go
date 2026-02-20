package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/atvirokodosprendimai/knitu/internal/messaging"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// Client is a wrapper around the official Docker client.
type Client struct {
	cli *client.Client
}

// NewClient creates a new Docker client.
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("could not create docker client: %w", err)
	}
	return &Client{cli: cli}, nil
}

// DeployContainer pulls an image, creates a container, and starts it.
func (c *Client) DeployContainer(ctx context.Context, task *messaging.DeployTask) (string, error) {
	// 1. Pull Image
	authStr, err := getAuthString(task.Registry.Username, task.Registry.Password)
	if err != nil {
		return "", fmt.Errorf("could not get auth string: %w", err)
	}

	reader, err := c.cli.ImagePull(ctx, task.Image, types.ImagePullOptions{RegistryAuth: authStr})
	if err != nil {
		return "", fmt.Errorf("could not pull image '%s': %w", task.Image, err)
	}
	io.Copy(os.Stdout, reader) // Show pull progress

	// 2. Configure Container
	containerConfig := &container.Config{
		Image: task.Image,
	}
	hostConfig := &container.HostConfig{}

	// TODO: Add support for Env, Ports, Templates, Network

	// 3. Create Container
	resp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, task.Name)
	if err != nil {
		return "", fmt.Errorf("could not create container: %w", err)
	}

	// 4. Start Container
	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("could not start container: %w", err)
	}

	return resp.ID, nil
}

func getAuthString(username, password string) (string, error) {
	if username == "" && password == "" {
		return "", nil
	}
	authConfig := types.AuthConfig{
		Username: username,
		Password: password,
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(encodedJSON), nil
}

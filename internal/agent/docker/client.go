package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/atvirokodosprendimai/knitu/internal/messaging"
	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/registry"
	"github.com/moby/moby/client"
)

// Client is a wrapper around the official Docker client.
type Client struct {
	cli *client.Client
}

// NewClient creates a new Docker client.
func NewClient() (*Client, error) {
	cli, err := client.New(client.FromEnv, client.WithAPIVersionNegotiation())
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
	pullOpts := client.ImagePullOptions{RegistryAuth: authStr}
	reader, err := c.cli.ImagePull(ctx, task.Image, pullOpts)
	if err != nil {
		return "", fmt.Errorf("could not pull image '%s': %w", task.Image, err)
	}
	io.Copy(os.Stdout, reader)
	defer reader.Close()

	// 2. Prepare Host and Container Configuration
	hostConfig := &container.HostConfig{}
	containerConfig := &container.Config{
		Image: task.Image,
	}

	// Handle Templates
	if len(task.Templates) > 0 {
		mounts, err := c.prepareTemplates(task, nil)
		if err != nil {
			return "", fmt.Errorf("could not prepare templates: %w", err)
		}
		hostConfig.Mounts = mounts
	}

	// Handle Port Mappings
	if len(task.Ports) > 0 {
		exposedPorts := make(network.PortSet)
		portBindings := make(network.PortMap)

		for _, p := range task.Ports {
			containerPort, err := network.ParsePort(fmt.Sprintf("%d/tcp", p.ContainerPort))
			if err != nil {
				return "", fmt.Errorf("invalid container port %d: %w", p.ContainerPort, err)
			}
			exposedPorts[containerPort] = struct{}{}

			hostIP := netip.Addr{}
			if p.HostIP != "" {
				hostIP, err = netip.ParseAddr(p.HostIP)
				if err != nil {
					return "", fmt.Errorf("invalid host IP '%s': %w", p.HostIP, err)
				}
			}

			portBindings[containerPort] = []network.PortBinding{
				{
					HostIP:   hostIP,
					HostPort: strconv.Itoa(p.HostPort),
				},
			}
		}
		containerConfig.ExposedPorts = exposedPorts
		hostConfig.PortBindings = portBindings
	}

	// 3. Create Container
	if err := c.removeContainerIfExists(ctx, task.Name); err != nil {
		return "", fmt.Errorf("could not prepare container name '%s': %w", task.Name, err)
	}

	createOptions := client.ContainerCreateOptions{
		Config:     containerConfig,
		HostConfig: hostConfig,
		Name:       task.Name,
	}
	resp, err := c.cli.ContainerCreate(ctx, createOptions)
	if err != nil {
		return "", fmt.Errorf("could not create container: %w", err)
	}

	// 4. Start Container
	startOpts := client.ContainerStartOptions{}
	if _, err := c.cli.ContainerStart(ctx, resp.ID, startOpts); err != nil {
		return "", fmt.Errorf("could not start container: %w", err)
	}

	return resp.ID, nil
}

func (c *Client) prepareTemplates(task *messaging.DeployTask, data interface{}) ([]mount.Mount, error) {
	mounts := []mount.Mount{}
	tempDir, err := os.MkdirTemp("", "knit-templates-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir for templates: %w", err)
	}
	log.Printf("[INFO] Created temp dir for templates: %s. Note: This directory is not automatically cleaned up.", tempDir)

	for i, t := range task.Templates {
		tempFilePath := filepath.Join(tempDir, fmt.Sprintf("template-%d", i))
		tempFile, err := os.Create(tempFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create temp file for template %d: %w", i, err)
		}
		defer tempFile.Close()

		tmpl, err := template.New(fmt.Sprintf("template-%d", i)).Parse(t.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %d: %w", i, err)
		}
		if err := tmpl.Execute(tempFile, data); err != nil {
			return nil, fmt.Errorf("failed to execute template %d: %w", i, err)
		}

		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: tempFilePath,
			Target: t.Destination,
		})
		log.Printf("[INFO] Prepared template to be mounted from %s to %s", tempFilePath, t.Destination)
	}
	return mounts, nil
}

func getAuthString(username, password string) (string, error) {
	if username == "" && password == "" {
		return "", nil
	}
	authConfig := registry.AuthConfig{
		Username: username,
		Password: password,
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(encodedJSON), nil
}

// UndeployContainer removes container by name if it exists.
func (c *Client) UndeployContainer(ctx context.Context, name string) error {
	return c.removeContainerIfExists(ctx, name)
}

func (c *Client) removeContainerIfExists(ctx context.Context, containerName string) error {
	if containerName == "" {
		return nil
	}

	_, err := c.cli.ContainerInspect(ctx, containerName, client.ContainerInspectOptions{})
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return nil
		}
		return err
	}

	log.Printf("[INFO] Container name '%s' already exists, removing for redeploy", containerName)
	_, err = c.cli.ContainerRemove(ctx, containerName, client.ContainerRemoveOptions{Force: true, RemoveVolumes: false})
	if err != nil {
		return err
	}
	return nil
}

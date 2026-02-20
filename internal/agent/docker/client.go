package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"text/template"

	"github.com/atvirokodosprendimai/knitu/internal/messaging"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
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

	// 2. Prepare Host Configuration (including templates)
	hostConfig := &container.HostConfig{}
	if len(task.Templates) > 0 {
		mounts, err := c.prepareTemplates(task)
		if err != nil {
			return "", fmt.Errorf("could not prepare templates: %w", err)
		}
		hostConfig.Mounts = mounts
	}
	// TODO: Add support for Env, Ports, Network

	// 3. Configure Container
	containerConfig := &container.Config{
		Image: task.Image,
	}

	// 4. Create Container
	createOptions := client.ContainerCreateOptions{
		Config:     containerConfig,
		HostConfig: hostConfig,
		Name:       task.Name,
	}
	resp, err := c.cli.ContainerCreate(ctx, createOptions)
	if err != nil {
		return "", fmt.Errorf("could not create container: %w", err)
	}

	// 5. Start Container
	startOpts := client.ContainerStartOptions{}
	if _, err := c.cli.ContainerStart(ctx, resp.ID, startOpts); err != nil {
		return "", fmt.Errorf("could not start container: %w", err)
	}

	return resp.ID, nil
}

func (c *Client) prepareTemplates(task *messaging.DeployTask) ([]mount.Mount, error) {
	mounts := []mount.Mount{}
	tempDir, err := os.MkdirTemp("", "knit-templates-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir for templates: %w", err)
	}
	log.Printf("[INFO] Created temp dir for templates: %s. Note: This directory is not automatically cleaned up.", tempDir)

	for i, t := range task.Templates {
		// Create a file within the temp dir
		tempFilePath := filepath.Join(tempDir, fmt.Sprintf("template-%d", i))
		tempFile, err := os.Create(tempFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create temp file for template %d: %w", i, err)
		}
		defer tempFile.Close()

		// Parse and execute the template
		tmpl, err := template.New(fmt.Sprintf("template-%d", i)).Parse(t.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %d: %w", i, err)
		}
		// Passing nil for data for now. This can be extended.
		if err := tmpl.Execute(tempFile, nil); err != nil {
			return nil, fmt.Errorf("failed to execute template %d: %w", i, err)
		}

		// Add to mounts
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

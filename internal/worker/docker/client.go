package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// Client wraps Docker SDK functionality for container management.
type Client struct {
	cli *client.Client
}

// NewClient creates a new Docker client.
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return &Client{cli: cli}, nil
}

// Close closes the Docker client connection.
func (c *Client) Close() error {
	if c.cli != nil {
		return c.cli.Close()
	}
	return nil
}

// PullImage pulls a Docker image from a registry.
func (c *Client) PullImage(ctx context.Context, imageName string) error {
	reader, err := c.cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer func() { _ = reader.Close() }()

	// Read all output to ensure pull completes
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to read image pull output: %w", err)
	}

	return nil
}

// CreateContainer creates a new container with the given configuration.
func (c *Client) CreateContainer(ctx context.Context, imageName string, env []string) (string, error) {
	config := &container.Config{
		Image: imageName,
		Env:   env,
	}

	resp, err := c.cli.ContainerCreate(ctx, config, nil, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}

// StartContainer starts a container by ID.
func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container %s: %w", containerID, err)
	}
	return nil
}

// StopContainer stops a running container.
func (c *Client) StopContainer(ctx context.Context, containerID string) error {
	timeout := 10 // seconds
	err := c.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
	if err != nil {
		return fmt.Errorf("failed to stop container %s: %w", containerID, err)
	}
	return nil
}

// RemoveContainer removes a container by ID.
func (c *Client) RemoveContainer(ctx context.Context, containerID string) error {
	err := c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
	if err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerID, err)
	}
	return nil
}

// GetContainerStatus returns the current status of a container.
func (c *Client) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	inspect, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}
	return inspect.State.Status, nil
}

// WaitContainer waits for a container to finish and returns the exit code.
func (c *Client) WaitContainer(ctx context.Context, containerID string) (int64, error) {
	statusCh, errCh := c.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return -1, fmt.Errorf("error waiting for container %s: %w", containerID, err)
		}
	case status := <-statusCh:
		return status.StatusCode, nil
	}
	return 0, nil
}

// GetContainerLogs retrieves logs from a container.
func (c *Client) GetContainerLogs(ctx context.Context, containerID string, tail int) (string, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", tail),
	}

	reader, err := c.cli.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return "", fmt.Errorf("failed to get logs for container %s: %w", containerID, err)
	}
	defer func() { _ = reader.Close() }()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	return buf.String(), nil
}

// ExecInContainer executes a command in a running container
func (c *Client) ExecInContainer(ctx context.Context, containerID string, cmd []string) (int, string, error) {
	execConfig := container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
	}

	execID, err := c.cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return -1, "", fmt.Errorf("failed to create exec: %w", err)
	}

	resp, err := c.cli.ContainerExecAttach(ctx, execID.ID, container.ExecStartOptions{})
	if err != nil {
		return -1, "", fmt.Errorf("failed to attach to exec: %w", err)
	}
	defer resp.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Reader)
	if err != nil {
		return -1, "", fmt.Errorf("failed to read exec output: %w", err)
	}

	inspectResp, err := c.cli.ContainerExecInspect(ctx, execID.ID)
	if err != nil {
		return -1, buf.String(), fmt.Errorf("failed to inspect exec: %w", err)
	}

	return inspectResp.ExitCode, buf.String(), nil
}

// GetContainerIP returns the IP address of a container
func (c *Client) GetContainerIP(ctx context.Context, containerID string) (string, error) {
	inspect, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}

	// Get IP from networks
	if inspect.NetworkSettings != nil && inspect.NetworkSettings.Networks != nil {
		// Try default bridge network first
		if bridge, ok := inspect.NetworkSettings.Networks["bridge"]; ok && bridge.IPAddress != "" {
			return bridge.IPAddress, nil
		}

		// Fall back to any available network
		for _, network := range inspect.NetworkSettings.Networks {
			if network.IPAddress != "" {
				return network.IPAddress, nil
			}
		}
	}

	return "", fmt.Errorf("no IP address found for container %s", containerID)
}

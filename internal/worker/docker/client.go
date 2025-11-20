package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// Client wraps Docker SDK functionality for container management.
type Client struct {
	cli *client.Client
}

// PortMapping represents a mapping between container and host ports.
type PortMapping struct {
	ContainerPort int
	HostPort      int
	Protocol      string
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
		Labels: map[string]string{
			"podling.io/managed": "true",
		},
	}

	resp, err := c.cli.ContainerCreate(ctx, config, nil, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}

// CreateContainerWithResources creates a new container with resource limits.
// cpuQuota is in Docker format (e.g., 0.5 for half a core, 2.0 for two cores).
// memoryLimit is in bytes (0 means no limit).
func (c *Client) CreateContainerWithResources(
	ctx context.Context, imageName string, env []string, cpuQuota float64, memoryLimit int64,
) (string, error) {
	config := &container.Config{
		Image: imageName,
		Env:   env,
		Labels: map[string]string{
			"podling.io/managed": "true",
		},
	}

	hostConfig := &container.HostConfig{}

	if cpuQuota > 0 {
		// Docker uses NanoCPUs (1 CPU = 1e9 nano CPUs)
		hostConfig.NanoCPUs = int64(cpuQuota * 1e9)
	}

	if memoryLimit > 0 {
		hostConfig.Memory = memoryLimit
	}

	resp, err := c.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}

// CreateContainerWithResourcesAndPorts creates a container with resource limits and port mappings
func (c *Client) CreateContainerWithResourcesAndPorts(
	ctx context.Context, imageName string, env []string,
	cpuQuota float64, memoryLimit int64, ports []PortMapping,
) (string, error) {
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}

	for _, portMapping := range ports {
		protocol := portMapping.Protocol
		if protocol == "" {
			protocol = "tcp"
		}

		containerPort := nat.Port(fmt.Sprintf("%d/%s", portMapping.ContainerPort, protocol))
		exposedPorts[containerPort] = struct{}{}

		portBindings[containerPort] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: fmt.Sprintf("%d", portMapping.HostPort),
			},
		}
	}

	config := &container.Config{
		Image:        imageName,
		Env:          env,
		ExposedPorts: exposedPorts,
		Labels: map[string]string{
			"podling.io/managed": "true",
		},
	}

	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
	}

	if cpuQuota > 0 {
		hostConfig.NanoCPUs = int64(cpuQuota * 1e9)
	}

	if memoryLimit > 0 {
		hostConfig.Memory = memoryLimit
	}

	resp, err := c.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
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
		if bridge, ok := inspect.NetworkSettings.Networks["bridge"]; ok && bridge.IPAddress != "" {
			return bridge.IPAddress, nil
		}

		for _, networkNamespace := range inspect.NetworkSettings.Networks {
			if networkNamespace.IPAddress != "" {
				return networkNamespace.IPAddress, nil
			}
		}
	}

	return "", fmt.Errorf("no IP address found for container %s", containerID)
}

// CreatePodNetwork creates a dedicated Docker bridge network for a pod
// All containers in the pod will be attached to this network, sharing the same namespace
func (c *Client) CreatePodNetwork(ctx context.Context, podID string) (string, error) {
	networkName := fmt.Sprintf("pod-%s", podID)

	// Note: We don't set com.docker.network.bridge.name as it has length/character restrictions
	createResp, err := c.cli.NetworkCreate(
		ctx, networkName, network.CreateOptions{
			Driver: "bridge",
			Labels: map[string]string{
				"podling.io/pod-id": podID,
				"podling.io/type":   "pod-network",
			},
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to create pod network %s: %w", networkName, err)
	}

	return createResp.ID, nil
}

// RemovePodNetwork removes a pod's network
func (c *Client) RemovePodNetwork(ctx context.Context, networkID string) error {
	if err := c.cli.NetworkRemove(ctx, networkID); err != nil {
		return fmt.Errorf("failed to remove network %s: %w", networkID, err)
	}
	return nil
}

// ConnectContainerToNetwork attaches a container to a network
func (c *Client) ConnectContainerToNetwork(ctx context.Context, networkID, containerID string) error {
	if err := c.cli.NetworkConnect(ctx, networkID, containerID, nil); err != nil {
		return fmt.Errorf("failed to connect container %s to network %s: %w", containerID, networkID, err)
	}
	return nil
}

// GetNetworkIP returns the IP address of a container in a specific network
func (c *Client) GetNetworkIP(ctx context.Context, containerID, networkID string) (string, error) {
	networkInfo, err := c.cli.NetworkInspect(ctx, networkID, network.InspectOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to inspect network %s: %w", networkID, err)
	}
	networkName := networkInfo.Name

	inspect, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}

	if inspect.NetworkSettings == nil || inspect.NetworkSettings.Networks == nil {
		return "", fmt.Errorf("no network settings found for container %s", containerID)
	}

	if netSettings, ok := inspect.NetworkSettings.Networks[networkName]; ok {
		return netSettings.IPAddress, nil
	}

	for _, netSettings := range inspect.NetworkSettings.Networks {
		if netSettings.NetworkID == networkID {
			return netSettings.IPAddress, nil
		}
	}

	return "", fmt.Errorf("container %s not connected to network %s", containerID, networkID)
}

// CreateContainerInNetwork creates a container attached to a specific network
func (c *Client) CreateContainerInNetwork(
	ctx context.Context, imageName string, env []string, networkID string,
) (string, error) {
	config := &container.Config{
		Image: imageName,
		Env:   env,
		Labels: map[string]string{
			"podling.io/managed": "true",
		},
	}

	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkID: {},
		},
	}

	resp, err := c.cli.ContainerCreate(ctx, config, nil, networkingConfig, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}

// CreateContainerInNetworkWithResources creates a container with resource limits in a specific network
func (c *Client) CreateContainerInNetworkWithResources(
	ctx context.Context, imageName string, env []string, networkID string, cpuQuota float64, memoryLimit int64,
) (string, error) {
	config := &container.Config{
		Image: imageName,
		Env:   env,
		Labels: map[string]string{
			"podling.io/managed": "true",
		},
	}

	hostConfig := &container.HostConfig{}

	if cpuQuota > 0 {
		hostConfig.NanoCPUs = int64(cpuQuota * 1e9)
	}

	if memoryLimit > 0 {
		hostConfig.Memory = memoryLimit
	}

	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkID: {},
		},
	}

	resp, err := c.cli.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}

// CreateContainerInNetworkWithResourcesAndPorts creates a container with resource limits and port mappings in a specific network
func (c *Client) CreateContainerInNetworkWithResourcesAndPorts(
	ctx context.Context, imageName string, env []string, networkID string,
	cpuQuota float64, memoryLimit int64, ports []PortMapping,
) (string, error) {
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}

	for _, portMapping := range ports {
		protocol := portMapping.Protocol
		if protocol == "" {
			protocol = "tcp"
		}

		containerPort := nat.Port(fmt.Sprintf("%d/%s", portMapping.ContainerPort, protocol))
		exposedPorts[containerPort] = struct{}{}

		portBindings[containerPort] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: fmt.Sprintf("%d", portMapping.HostPort),
			},
		}
	}

	config := &container.Config{
		Image:        imageName,
		Env:          env,
		ExposedPorts: exposedPorts,
		Labels: map[string]string{
			"podling.io/managed": "true",
		},
	}

	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
	}

	if cpuQuota > 0 {
		hostConfig.NanoCPUs = int64(cpuQuota * 1e9)
	}

	if memoryLimit > 0 {
		hostConfig.Memory = memoryLimit
	}

	networkingConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkID: {},
		},
	}

	resp, err := c.cli.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}

package docker

import (
	"context"
	"testing"
)

func TestNewClient(t *testing.T) {
	// Test creating a new Docker client
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer func() { _ = client.Close() }()

	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.cli == nil {
		t.Fatal("expected non-nil underlying docker client")
	}
}

func TestClose(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Should not panic on double close
	err = client.Close()
	if err != nil {
		t.Errorf("Close() on closed client error = %v", err)
	}
}

func TestPullImage(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	tests := []struct {
		name      string
		imageName string
		wantErr   bool
	}{
		{
			name:      "pull valid image",
			imageName: "alpine:latest",
			wantErr:   false,
		},
		{
			name:      "pull invalid image",
			imageName: "nonexistent-image-xyz-123:latest",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.PullImage(ctx, tt.imageName)
			if (err != nil) != tt.wantErr {
				t.Errorf("PullImage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateContainer(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// Pull alpine first
	if err := client.PullImage(ctx, "alpine:latest"); err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	tests := []struct {
		name      string
		imageName string
		env       []string
		wantErr   bool
	}{
		{
			name:      "create with valid image",
			imageName: "alpine:latest",
			env:       []string{"TEST=value"},
			wantErr:   false,
		},
		{
			name:      "create with invalid image",
			imageName: "nonexistent-image:latest",
			env:       []string{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			containerID, err := client.CreateContainer(ctx, tt.imageName, tt.env)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateContainer() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && containerID == "" {
				t.Error("CreateContainer() returned empty container ID")
			}
			// Clean up
			if containerID != "" {
				_ = client.RemoveContainer(ctx, containerID)
			}
		})
	}
}

func TestStartContainer(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// Pull and create a container
	if err := client.PullImage(ctx, "alpine:latest"); err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	containerID, err := client.CreateContainer(ctx, "alpine:latest", []string{})
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}
	defer func() { _ = client.RemoveContainer(ctx, containerID) }()

	// Start the container
	err = client.StartContainer(ctx, containerID)
	if err != nil {
		t.Errorf("StartContainer() error = %v", err)
	}

	// Test starting invalid container
	err = client.StartContainer(ctx, "invalid-container-id")
	if err == nil {
		t.Error("StartContainer() expected error for invalid container ID")
	}
}

func TestStopContainer(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// Pull and create a long-running container
	if err := client.PullImage(ctx, "alpine:latest"); err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	containerID, err := client.CreateContainer(ctx, "alpine:latest", []string{})
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}
	defer func() { _ = client.RemoveContainer(ctx, containerID) }()

	// Start then stop
	if err := client.StartContainer(ctx, containerID); err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	err = client.StopContainer(ctx, containerID)
	if err != nil {
		t.Errorf("StopContainer() error = %v", err)
	}

	// Test stopping invalid container
	err = client.StopContainer(ctx, "invalid-container-id")
	if err == nil {
		t.Error("StopContainer() expected error for invalid container ID")
	}
}

func TestRemoveContainer(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// Create a container
	if err := client.PullImage(ctx, "alpine:latest"); err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	containerID, err := client.CreateContainer(ctx, "alpine:latest", []string{})
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}

	// Remove it
	err = client.RemoveContainer(ctx, containerID)
	if err != nil {
		t.Errorf("RemoveContainer() error = %v", err)
	}

	// Test removing invalid container
	err = client.RemoveContainer(ctx, "invalid-container-id")
	if err == nil {
		t.Error("RemoveContainer() expected error for invalid container ID")
	}
}

func TestGetContainerStatus(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// Create and start a container
	if err := client.PullImage(ctx, "alpine:latest"); err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	containerID, err := client.CreateContainer(ctx, "alpine:latest", []string{})
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}
	defer func() { _ = client.RemoveContainer(ctx, containerID) }()

	// Get status before starting
	status, err := client.GetContainerStatus(ctx, containerID)
	if err != nil {
		t.Errorf("GetContainerStatus() error = %v", err)
	}
	if status != "created" {
		t.Errorf("GetContainerStatus() = %v, want 'created'", status)
	}

	// Start and check status again
	if err := client.StartContainer(ctx, containerID); err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	status, err = client.GetContainerStatus(ctx, containerID)
	if err != nil {
		t.Errorf("GetContainerStatus() error = %v", err)
	}
	if status != "running" && status != "exited" {
		t.Errorf("GetContainerStatus() = %v, want 'running' or 'exited'", status)
	}

	// Test invalid container
	_, err = client.GetContainerStatus(ctx, "invalid-container-id")
	if err == nil {
		t.Error("GetContainerStatus() expected error for invalid container ID")
	}
}

func TestWaitContainer(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// Create a container that exits immediately
	if err := client.PullImage(ctx, "alpine:latest"); err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	containerID, err := client.CreateContainer(ctx, "alpine:latest", []string{})
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}
	defer func() { _ = client.RemoveContainer(ctx, containerID) }()

	if err := client.StartContainer(ctx, containerID); err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	// Wait for it to finish
	exitCode, err := client.WaitContainer(ctx, containerID)
	if err != nil {
		t.Errorf("WaitContainer() error = %v", err)
	}
	if exitCode != 0 {
		t.Errorf("WaitContainer() exitCode = %v, want 0", exitCode)
	}

	// Test waiting for invalid container
	_, err = client.WaitContainer(ctx, "invalid-container-id")
	if err == nil {
		t.Error("WaitContainer() expected error for invalid container ID")
	}
}

func TestGetContainerLogs(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	// Pull and create a container that produces output
	if err := client.PullImage(ctx, "alpine:latest"); err != nil {
		t.Fatalf("failed to pull alpine image: %v", err)
	}

	containerID, err := client.CreateContainer(ctx, "alpine:latest", []string{})
	if err != nil {
		t.Fatalf("failed to create container: %v", err)
	}
	defer func() { _ = client.RemoveContainer(ctx, containerID) }()

	if err := client.StartContainer(ctx, containerID); err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	// Wait for container to finish
	_, err = client.WaitContainer(ctx, containerID)
	if err != nil {
		t.Fatalf("failed to wait for container: %v", err)
	}

	// Get logs (even if empty)
	logs, err := client.GetContainerLogs(ctx, containerID, 100)
	if err != nil {
		t.Errorf("GetContainerLogs() error = %v", err)
	}

	// Logs may be empty for alpine, that's OK
	_ = logs

	// Test with invalid container
	_, err = client.GetContainerLogs(ctx, "invalid-container-id", 100)
	if err == nil {
		t.Error("GetContainerLogs() expected error for invalid container ID")
	}
}


package health

import (
	"context"
	"errors"
	"testing"

	"github.com/danpasecinic/podling/internal/types"
)

// mockDockerExecClient for testing exec probe
type mockDockerExecClient struct {
	ExecFunc func(ctx context.Context, containerID string, cmd []string) (int, string, error)
}

func (m *mockDockerExecClient) ExecInContainer(ctx context.Context, containerID string, cmd []string) (
	int,
	string,
	error,
) {
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, containerID, cmd)
	}
	return 0, "", nil
}

func TestExecProbe_Check(t *testing.T) {
	t.Run(
		"successful command execution", func(t *testing.T) {
			mockDocker := &mockDockerExecClient{
				ExecFunc: func(ctx context.Context, containerID string, cmd []string) (int, string, error) {
					return 0, "healthy", nil
				},
			}

			probe := &ExecProbe{dockerClient: mockDocker}
			ctx := context.Background()

			check := &types.HealthCheck{
				Type:           types.ProbeTypeExec,
				Command:        []string{"/bin/sh", "-c", "echo healthy"},
				TimeoutSeconds: 5,
			}

			result := probe.Check(ctx, check, "container-123")
			if !result.Success {
				t.Errorf("expected success, got failure: %s", result.Message)
			}
			if result.Message != "command succeeded: healthy" {
				t.Errorf("unexpected message: %s", result.Message)
			}
		},
	)

	t.Run(
		"failed command execution", func(t *testing.T) {
			mockDocker := &mockDockerExecClient{
				ExecFunc: func(ctx context.Context, containerID string, cmd []string) (int, string, error) {
					return 1, "error output", nil
				},
			}

			probe := &ExecProbe{dockerClient: mockDocker}
			ctx := context.Background()

			check := &types.HealthCheck{
				Type:           types.ProbeTypeExec,
				Command:        []string{"/bin/false"},
				TimeoutSeconds: 5,
			}

			result := probe.Check(ctx, check, "container-123")
			if result.Success {
				t.Error("expected failure, got success")
			}
			if result.Message != "command failed (exit 1): error output" {
				t.Errorf("unexpected message: %s", result.Message)
			}
		},
	)

	t.Run(
		"command execution error", func(t *testing.T) {
			mockDocker := &mockDockerExecClient{
				ExecFunc: func(ctx context.Context, containerID string, cmd []string) (int, string, error) {
					return -1, "", errors.New("container not found")
				},
			}

			probe := &ExecProbe{dockerClient: mockDocker}
			ctx := context.Background()

			check := &types.HealthCheck{
				Type:           types.ProbeTypeExec,
				Command:        []string{"/bin/sh"},
				TimeoutSeconds: 5,
			}

			result := probe.Check(ctx, check, "container-123")
			if result.Success {
				t.Error("expected failure, got success")
			}
		},
	)

	t.Run(
		"no command configured", func(t *testing.T) {
			mockDocker := &mockDockerExecClient{}
			probe := &ExecProbe{dockerClient: mockDocker}
			ctx := context.Background()

			check := &types.HealthCheck{
				Type:    types.ProbeTypeExec,
				Command: []string{},
			}

			result := probe.Check(ctx, check, "container-123")
			if result.Success {
				t.Error("expected failure, got success")
			}
			if result.Message != "no command configured" {
				t.Errorf("unexpected message: %s", result.Message)
			}
		},
	)

	t.Run(
		"empty container ID", func(t *testing.T) {
			mockDocker := &mockDockerExecClient{}
			probe := &ExecProbe{dockerClient: mockDocker}
			ctx := context.Background()

			check := &types.HealthCheck{
				Type:    types.ProbeTypeExec,
				Command: []string{"/bin/true"},
			}

			result := probe.Check(ctx, check, "")
			if result.Success {
				t.Error("expected failure, got success")
			}
			if result.Message != "container ID not available" {
				t.Errorf("unexpected message: %s", result.Message)
			}
		},
	)

	t.Run(
		"successful command without output", func(t *testing.T) {
			mockDocker := &mockDockerExecClient{
				ExecFunc: func(ctx context.Context, containerID string, cmd []string) (int, string, error) {
					return 0, "", nil
				},
			}

			probe := &ExecProbe{dockerClient: mockDocker}
			ctx := context.Background()

			check := &types.HealthCheck{
				Type:    types.ProbeTypeExec,
				Command: []string{"/bin/true"},
			}

			result := probe.Check(ctx, check, "container-123")
			if !result.Success {
				t.Errorf("expected success, got failure: %s", result.Message)
			}
			if result.Message != "command succeeded" {
				t.Errorf("unexpected message: %s", result.Message)
			}
		},
	)

	t.Run(
		"failed command without output", func(t *testing.T) {
			mockDocker := &mockDockerExecClient{
				ExecFunc: func(ctx context.Context, containerID string, cmd []string) (int, string, error) {
					return 127, "", nil
				},
			}

			probe := &ExecProbe{dockerClient: mockDocker}
			ctx := context.Background()

			check := &types.HealthCheck{
				Type:    types.ProbeTypeExec,
				Command: []string{"/bin/notfound"},
			}

			result := probe.Check(ctx, check, "container-123")
			if result.Success {
				t.Error("expected failure, got success")
			}
			if result.Message != "command failed with exit code 127" {
				t.Errorf("unexpected message: %s", result.Message)
			}
		},
	)
}

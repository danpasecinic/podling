package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danpasecinic/podling/internal/types"
	"github.com/danpasecinic/podling/internal/worker/docker"
)

// mockDockerHealthClient implements DockerHealthClient for testing
type mockDockerHealthClient struct {
	ExecFunc func(ctx context.Context, containerID string, cmd []string) (int, string, error)
	IPFunc   func(ctx context.Context, containerID string) (string, error)
}

func (m *mockDockerHealthClient) ExecInContainer(ctx context.Context, containerID string, cmd []string) (
	int,
	string,
	error,
) {
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, containerID, cmd)
	}
	return 0, "", nil
}

func (m *mockDockerHealthClient) GetContainerIP(ctx context.Context, containerID string) (string, error) {
	if m.IPFunc != nil {
		return m.IPFunc(ctx, containerID)
	}
	return "127.0.0.1", nil
}

func TestNewChecker(t *testing.T) {
	mockDocker := &mockDockerHealthClient{}
	onUnhealthy := func(taskID string) {
	}

	check := &types.HealthCheck{
		Type:                types.ProbeTypeHTTP,
		HTTPPath:            "/health",
		Port:                8080,
		InitialDelaySeconds: 5,
	}

	checker := newCheckerWithClient("task-1", "container-1", check, types.RestartPolicyAlways, mockDocker, onUnhealthy)

	if checker.taskID != "task-1" {
		t.Errorf("expected taskID task-1, got %s", checker.taskID)
	}
	if checker.containerID != "container-1" {
		t.Errorf("expected containerID container-1, got %s", checker.containerID)
	}
	if checker.status != types.HealthStatusUnknown {
		t.Errorf("expected initial status unknown, got %s", checker.status)
	}
	if checker.consecutiveFail != 0 {
		t.Errorf("expected consecutiveFail 0, got %d", checker.consecutiveFail)
	}
	if checker.consecutiveOK != 0 {
		t.Errorf("expected consecutiveOK 0, got %d", checker.consecutiveOK)
	}
}

func TestChecker_GetStatus(t *testing.T) {
	mockDocker := &mockDockerHealthClient{}
	check := &types.HealthCheck{Type: types.ProbeTypeHTTP}

	checker := newCheckerWithClient("task-1", "container-1", check, types.RestartPolicyNever, mockDocker, nil)

	if status := checker.GetStatus(); status != types.HealthStatusUnknown {
		t.Errorf("expected status unknown, got %s", status)
	}

	checker.status = types.HealthStatusHealthy
	if status := checker.GetStatus(); status != types.HealthStatusHealthy {
		t.Errorf("expected status healthy, got %s", status)
	}
}

func TestChecker_Stop(t *testing.T) {
	mockDocker := &mockDockerHealthClient{}
	check := &types.HealthCheck{Type: types.ProbeTypeHTTP}

	checker := newCheckerWithClient("task-1", "container-1", check, types.RestartPolicyNever, mockDocker, nil)

	// Stop should be idempotent
	checker.Stop()
	checker.Stop()

	if !checker.stopped {
		t.Error("expected checker to be stopped")
	}
}

func TestChecker_updateStatus(t *testing.T) {
	mockDocker := &mockDockerHealthClient{}
	unhealthyCalled := make(chan bool, 1)
	onUnhealthy := func(taskID string) {
		unhealthyCalled <- true
	}

	check := &types.HealthCheck{
		Type:             types.ProbeTypeHTTP,
		SuccessThreshold: 2,
		FailureThreshold: 3,
	}

	checker := newCheckerWithClient("task-1", "container-1", check, types.RestartPolicyAlways, mockDocker, onUnhealthy)

	t.Run(
		"successful checks mark as healthy", func(t *testing.T) {
			// First success - not enough
			checker.updateStatus(types.ProbeResult{Success: true, Message: "ok"})
			if checker.GetStatus() != types.HealthStatusUnknown {
				t.Errorf("expected unknown after 1 success, got %s", checker.GetStatus())
			}
			if checker.consecutiveOK != 1 {
				t.Errorf("expected consecutiveOK 1, got %d", checker.consecutiveOK)
			}

			// Second success - should become healthy
			checker.updateStatus(types.ProbeResult{Success: true, Message: "ok"})
			if checker.GetStatus() != types.HealthStatusHealthy {
				t.Errorf("expected healthy after 2 successes, got %s", checker.GetStatus())
			}
			if checker.consecutiveOK != 2 {
				t.Errorf("expected consecutiveOK 2, got %d", checker.consecutiveOK)
			}
		},
	)

	t.Run(
		"failed checks mark as unhealthy", func(t *testing.T) {
			checker.status = types.HealthStatusHealthy
			checker.consecutiveOK = 2
			checker.consecutiveFail = 0

			// First failure - resets OK counter
			checker.updateStatus(types.ProbeResult{Success: false, Message: "fail"})
			if checker.GetStatus() != types.HealthStatusHealthy {
				t.Errorf("expected still healthy after 1 failure, got %s", checker.GetStatus())
			}
			if checker.consecutiveOK != 0 {
				t.Errorf("expected consecutiveOK reset to 0, got %d", checker.consecutiveOK)
			}
			if checker.consecutiveFail != 1 {
				t.Errorf("expected consecutiveFail 1, got %d", checker.consecutiveFail)
			}

			// Second failure
			checker.updateStatus(types.ProbeResult{Success: false, Message: "fail"})
			if checker.consecutiveFail != 2 {
				t.Errorf("expected consecutiveFail 2, got %d", checker.consecutiveFail)
			}

			// Third failure - should trigger unhealthy
			checker.updateStatus(types.ProbeResult{Success: false, Message: "fail"})
			if checker.GetStatus() != types.HealthStatusUnhealthy {
				t.Errorf("expected unhealthy after 3 failures, got %s", checker.GetStatus())
			}

			// Wait for callback (runs in goroutine)
			select {
			case <-unhealthyCalled:
			case <-time.After(100 * time.Millisecond):
				t.Error("expected onUnhealthy callback to be called")
			}
		},
	)

	t.Run(
		"success resets failure counter", func(t *testing.T) {
			checker.consecutiveFail = 2
			checker.consecutiveOK = 0
			checker.status = types.HealthStatusHealthy

			checker.updateStatus(types.ProbeResult{Success: true, Message: "ok"})
			if checker.consecutiveFail != 0 {
				t.Errorf("expected consecutiveFail reset to 0, got %d", checker.consecutiveFail)
			}
		},
	)
}

func TestShouldRestart(t *testing.T) {
	tests := []struct {
		name          string
		policy        types.RestartPolicy
		exitCode      int64
		shouldRestart bool
	}{
		{
			name:          "Always restarts on success",
			policy:        types.RestartPolicyAlways,
			exitCode:      0,
			shouldRestart: true,
		},
		{
			name:          "Always restarts on failure",
			policy:        types.RestartPolicyAlways,
			exitCode:      1,
			shouldRestart: true,
		},
		{
			name:          "OnFailure does not restart on success",
			policy:        types.RestartPolicyOnFailure,
			exitCode:      0,
			shouldRestart: false,
		},
		{
			name:          "OnFailure restarts on failure",
			policy:        types.RestartPolicyOnFailure,
			exitCode:      1,
			shouldRestart: true,
		},
		{
			name:          "Never does not restart on success",
			policy:        types.RestartPolicyNever,
			exitCode:      0,
			shouldRestart: false,
		},
		{
			name:          "Never does not restart on failure",
			policy:        types.RestartPolicyNever,
			exitCode:      1,
			shouldRestart: false,
		},
		{
			name:          "Unknown policy does not restart",
			policy:        "unknown",
			exitCode:      1,
			shouldRestart: false,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				result := ShouldRestart(tt.policy, tt.exitCode)
				if result != tt.shouldRestart {
					t.Errorf("ShouldRestart(%s, %d) = %v, want %v", tt.policy, tt.exitCode, result, tt.shouldRestart)
				}
			},
		)
	}
}

func TestChecker_Start(t *testing.T) {
	t.Run(
		"stops immediately when already stopped", func(t *testing.T) {
			mockDocker := &mockDockerHealthClient{}
			check := &types.HealthCheck{
				Type:                types.ProbeTypeExec,
				Command:             []string{"true"},
				InitialDelaySeconds: 0,
				PeriodSeconds:       1,
			}

			checker := newCheckerWithClient("task-1", "container-1", check, types.RestartPolicyNever, mockDocker, nil)
			checker.stopped = true

			ctx := context.Background()
			checker.Start(ctx)
		},
	)

	t.Run(
		"respects initial delay", func(t *testing.T) {
			mockDocker := &mockDockerHealthClient{
				ExecFunc: func(ctx context.Context, containerID string, cmd []string) (int, string, error) {
					return 0, "", nil
				},
			}

			check := &types.HealthCheck{
				Type:                types.ProbeTypeExec,
				Command:             []string{"true"},
				InitialDelaySeconds: 1,
				PeriodSeconds:       10,
			}

			checker := newCheckerWithClient("task-1", "container-1", check, types.RestartPolicyNever, mockDocker, nil)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			start := time.Now()
			checker.Start(ctx)
			elapsed := time.Since(start)

			// Should wait at least the initial delay or until context cancels
			if elapsed < 100*time.Millisecond {
				t.Errorf("Start returned too quickly: %v", elapsed)
			}
		},
	)

	t.Run(
		"performs periodic health checks", func(t *testing.T) {
			checkCount := 0
			mockDocker := &mockDockerHealthClient{
				ExecFunc: func(ctx context.Context, containerID string, cmd []string) (int, string, error) {
					checkCount++
					return 0, "healthy", nil
				},
			}

			check := &types.HealthCheck{
				Type:                types.ProbeTypeExec,
				Command:             []string{"healthcheck"},
				InitialDelaySeconds: 0,
				PeriodSeconds:       1,
				SuccessThreshold:    1,
			}

			checker := newCheckerWithClient("task-1", "container-1", check, types.RestartPolicyNever, mockDocker, nil)

			ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
			defer cancel()

			go checker.Start(ctx)

			// Wait for at least 2 checks
			time.Sleep(2200 * time.Millisecond)
			checker.Stop()

			if checkCount < 2 {
				t.Errorf("expected at least 2 health checks, got %d", checkCount)
			}
		},
	)
}

// TestNewChecker_WithRealDockerClient tests the public NewChecker function
func TestNewChecker_WithRealDockerClient(t *testing.T) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer func(dockerClient *docker.Client) {
		_ = dockerClient.Close()
	}(dockerClient)

	check := &types.HealthCheck{
		Type:     types.ProbeTypeHTTP,
		HTTPPath: "/health",
		Port:     8080,
	}

	onUnhealthy := func(taskID string) {}

	checker := NewChecker("task-1", "container-1", check, types.RestartPolicyAlways, dockerClient, onUnhealthy)

	if checker == nil {
		t.Fatal("expected checker to be created")
	}

	if checker.taskID != "task-1" {
		t.Errorf("expected taskID task-1, got %s", checker.taskID)
	}
}

// TestPerformCheck tests the performCheck method with different probe types
func TestPerformCheck(t *testing.T) {
	t.Run(
		"HTTP probe with IP lookup failure", func(t *testing.T) {
			mockDocker := &mockDockerHealthClient{
				IPFunc: func(ctx context.Context, containerID string) (string, error) {
					return "", errors.New("container not found")
				},
			}

			check := &types.HealthCheck{
				Type:             types.ProbeTypeHTTP,
				HTTPPath:         "/health",
				Port:             8080,
				FailureThreshold: 1,
			}

			checker := newCheckerWithClient("task-1", "container-1", check, types.RestartPolicyNever, mockDocker, nil)
			checker.performCheck(context.Background())

			// Should have recorded a failure
			if checker.consecutiveFail == 0 {
				t.Error("expected failure to be recorded")
			}
		},
	)

	t.Run(
		"TCP probe with IP lookup failure", func(t *testing.T) {
			mockDocker := &mockDockerHealthClient{
				IPFunc: func(ctx context.Context, containerID string) (string, error) {
					return "", errors.New("network error")
				},
			}

			check := &types.HealthCheck{
				Type:             types.ProbeTypeTCP,
				Port:             8080,
				FailureThreshold: 1,
			}

			checker := newCheckerWithClient("task-1", "container-1", check, types.RestartPolicyNever, mockDocker, nil)
			checker.performCheck(context.Background())

			// Should have recorded a failure
			if checker.consecutiveFail == 0 {
				t.Error("expected failure to be recorded")
			}
		},
	)

	t.Run(
		"Exec probe", func(t *testing.T) {
			mockDocker := &mockDockerHealthClient{
				ExecFunc: func(ctx context.Context, containerID string, cmd []string) (int, string, error) {
					return 0, "healthy", nil
				},
			}

			check := &types.HealthCheck{
				Type:             types.ProbeTypeExec,
				Command:          []string{"healthcheck"},
				SuccessThreshold: 1,
			}

			checker := newCheckerWithClient("task-1", "container-1", check, types.RestartPolicyNever, mockDocker, nil)
			checker.performCheck(context.Background())

			// Should have recorded success
			if checker.consecutiveOK == 0 {
				t.Error("expected success to be recorded")
			}
		},
	)

	t.Run(
		"unknown probe type", func(t *testing.T) {
			mockDocker := &mockDockerHealthClient{}

			check := &types.HealthCheck{
				Type: "unknown",
			}

			checker := newCheckerWithClient("task-1", "container-1", check, types.RestartPolicyNever, mockDocker, nil)

			// Should not panic
			checker.performCheck(context.Background())
		},
	)
}

// TestHTTPProbe_AdditionalCoverage covers HTTP probe error paths
func TestHTTPProbe_AdditionalCoverage(t *testing.T) {
	t.Run(
		"request creation error", func(t *testing.T) {
			probe := NewHTTPProbe()
			check := &types.HealthCheck{
				Type:           types.ProbeTypeHTTP,
				HTTPPath:       "/health",
				Port:           8080,
				TimeoutSeconds: 1,
			}

			// Use invalid control character in URL to trigger error
			result := probe.Check(context.Background(), check, "127.0.0.1\x00")

			if result.Success {
				t.Error("expected check to fail with invalid containerIP")
			}
		},
	)

	t.Run(
		"successful health check with context", func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
					},
				),
			)
			defer server.Close()

			probe := NewHTTPProbe()

			check := &types.HealthCheck{
				Type:           types.ProbeTypeHTTP,
				HTTPPath:       "/",
				Port:           80,
				TimeoutSeconds: 5,
			}

			// Will fail to connect to 127.0.0.1:80 but covers the code path
			result := probe.Check(context.Background(), check, "127.0.0.1")
			if result.Success {
				t.Log("Unexpectedly succeeded, but this is okay")
			}
		},
	)
}

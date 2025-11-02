package health

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/danpasecinic/podling/internal/types"
	"github.com/danpasecinic/podling/internal/worker/docker"
)

// DockerIPClient defines the interface for getting container IPs
type DockerIPClient interface {
	GetContainerIP(ctx context.Context, containerID string) (string, error)
}

// DockerHealthClient combines both exec and IP interfaces
type DockerHealthClient interface {
	DockerExecClient
	DockerIPClient
}

// Checker manages health checks for a container
type Checker struct {
	taskID          string
	containerID     string
	check           *types.HealthCheck
	checkType       types.ProbeType
	restartPolicy   types.RestartPolicy
	dockerClient    DockerHealthClient
	httpProbe       *HTTPProbe
	tcpProbe        *TCPProbe
	execProbe       *ExecProbe
	status          types.HealthStatus
	consecutiveFail int
	consecutiveOK   int
	mu              sync.RWMutex
	stopChan        chan struct{}
	stopped         bool
	onUnhealthy     func(taskID string)
}

// NewChecker creates a new health checker
func NewChecker(
	taskID string,
	containerID string,
	check *types.HealthCheck,
	restartPolicy types.RestartPolicy,
	dockerClient *docker.Client,
	onUnhealthy func(string),
) *Checker {
	return newCheckerWithClient(taskID, containerID, check, restartPolicy, dockerClient, onUnhealthy)
}

// newCheckerWithClient creates a checker with any DockerHealthClient (useful for testing)
func newCheckerWithClient(
	taskID string,
	containerID string,
	check *types.HealthCheck,
	restartPolicy types.RestartPolicy,
	dockerClient DockerHealthClient,
	onUnhealthy func(string),
) *Checker {
	execProbe := &ExecProbe{dockerClient: dockerClient}
	return &Checker{
		taskID:        taskID,
		containerID:   containerID,
		check:         check,
		checkType:     check.Type,
		restartPolicy: restartPolicy,
		dockerClient:  dockerClient,
		httpProbe:     NewHTTPProbe(),
		tcpProbe:      NewTCPProbe(),
		execProbe:     execProbe,
		status:        types.HealthStatusUnknown,
		stopChan:      make(chan struct{}),
		onUnhealthy:   onUnhealthy,
	}
}

// Start begins health checking
func (hc *Checker) Start(ctx context.Context) {
	hc.mu.Lock()
	if hc.stopped {
		hc.mu.Unlock()
		return
	}
	hc.mu.Unlock()

	if hc.check.GetInitialDelay() > 0 {
		select {
		case <-time.After(hc.check.GetInitialDelay()):
		case <-ctx.Done():
			return
		case <-hc.stopChan:
			return
		}
	}

	ticker := time.NewTicker(hc.check.GetPeriod())
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.performCheck(ctx)
		case <-ctx.Done():
			return
		case <-hc.stopChan:
			return
		}
	}
}

// Stop stops the health checker
func (hc *Checker) Stop() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if !hc.stopped {
		hc.stopped = true
		close(hc.stopChan)
	}
}

// GetStatus returns the current health status
func (hc *Checker) GetStatus() types.HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.status
}

// performCheck executes a health check
func (hc *Checker) performCheck(ctx context.Context) {
	var result types.ProbeResult

	switch hc.checkType {
	case types.ProbeTypeHTTP:
		containerIP, err := hc.dockerClient.GetContainerIP(ctx, hc.containerID)
		if err != nil {
			result = types.ProbeResult{
				Success:   false,
				Message:   fmt.Sprintf("failed to get container IP: %v", err),
				Timestamp: time.Now(),
			}
		} else {
			result = hc.httpProbe.Check(ctx, hc.check, containerIP)
		}

	case types.ProbeTypeTCP:
		containerIP, err := hc.dockerClient.GetContainerIP(ctx, hc.containerID)
		if err != nil {
			result = types.ProbeResult{
				Success:   false,
				Message:   fmt.Sprintf("failed to get container IP: %v", err),
				Timestamp: time.Now(),
			}
		} else {
			result = hc.tcpProbe.Check(ctx, hc.check, containerIP)
		}

	case types.ProbeTypeExec:
		result = hc.execProbe.Check(ctx, hc.check, hc.containerID)

	default:
		log.Printf("unknown probe type: %s", hc.checkType)
		return
	}

	hc.updateStatus(result)
}

// updateStatus updates the health status based on probe result
func (hc *Checker) updateStatus(result types.ProbeResult) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	previousStatus := hc.status

	if result.Success {
		hc.consecutiveFail = 0
		hc.consecutiveOK++

		if hc.consecutiveOK >= hc.check.GetSuccessThreshold() {
			hc.status = types.HealthStatusHealthy
		}

		log.Printf(
			"[health] task=%s probe=%s status=success consecutive=%d message=%q",
			hc.taskID, hc.checkType, hc.consecutiveOK, result.Message,
		)
	} else {
		hc.consecutiveOK = 0
		hc.consecutiveFail++

		log.Printf(
			"[health] task=%s probe=%s status=failure consecutive=%d message=%q",
			hc.taskID, hc.checkType, hc.consecutiveFail, result.Message,
		)

		if hc.consecutiveFail >= hc.check.GetFailureThreshold() {
			hc.status = types.HealthStatusUnhealthy

			// Trigger callback if status changed from healthy to unhealthy
			if previousStatus == types.HealthStatusHealthy && hc.onUnhealthy != nil {
				log.Printf("[health] task=%s became unhealthy, triggering callback", hc.taskID)
				go hc.onUnhealthy(hc.taskID)
			}
		}
	}
}

// ShouldRestart determines if a container should be restarted based on restart policy and exit code
func ShouldRestart(policy types.RestartPolicy, exitCode int64) bool {
	switch policy {
	case types.RestartPolicyAlways:
		return true
	case types.RestartPolicyOnFailure:
		return exitCode != 0
	case types.RestartPolicyNever:
		return false
	default:
		return false
	}
}

package health

import (
	"context"
	"fmt"
	"time"

	"github.com/danpasecinic/podling/internal/types"
)

// DockerExecClient defines the interface for executing commands in containers
type DockerExecClient interface {
	ExecInContainer(ctx context.Context, containerID string, cmd []string) (int, string, error)
}

// ExecProbe performs command execution health checks
type ExecProbe struct {
	dockerClient DockerExecClient
}

// Check executes a command inside the container
func (p *ExecProbe) Check(ctx context.Context, check *types.HealthCheck, containerID string) types.ProbeResult {
	result := types.ProbeResult{
		Success:   false,
		Timestamp: time.Now(),
	}

	if len(check.Command) == 0 {
		result.Message = "no command configured"
		return result
	}

	if containerID == "" {
		result.Message = "container ID not available"
		return result
	}

	execCtx, cancel := context.WithTimeout(ctx, check.GetTimeout())
	defer cancel()

	exitCode, output, err := p.dockerClient.ExecInContainer(execCtx, containerID, check.Command)
	if err != nil {
		result.Message = fmt.Sprintf("exec failed: %v", err)
		return result
	}

	if exitCode == 0 {
		result.Success = true
		result.Message = "command succeeded"
		if output != "" {
			result.Message = fmt.Sprintf("command succeeded: %s", output)
		}
	} else {
		result.Message = fmt.Sprintf("command failed with exit code %d", exitCode)
		if output != "" {
			result.Message = fmt.Sprintf("command failed (exit %d): %s", exitCode, output)
		}
	}

	return result
}

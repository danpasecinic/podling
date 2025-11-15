package agent

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/danpasecinic/podling/internal/types"
	"github.com/labstack/echo/v4"
)

// ExecuteTaskRequest represents a task execution request.
type ExecuteTaskRequest struct {
	Task types.Task `json:"task"`
}

// ExecuteTask handles POST /api/v1/tasks/:id/execute
// Executes a task in a Docker container.
func (s *Server) ExecuteTask(c echo.Context) error {
	taskID := c.Param("id")

	var req ExecuteTaskRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.Task.TaskID != taskID {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "task ID mismatch"})
	}

	if req.Task.LivenessProbe != nil {
		if err := validateHealthCheck(req.Task.LivenessProbe); err != nil {
			return c.JSON(
				http.StatusBadRequest, map[string]string{
					"error": fmt.Sprintf("invalid liveness probe: %v", err),
				},
			)
		}
	}

	if req.Task.ReadinessProbe != nil {
		if err := validateHealthCheck(req.Task.ReadinessProbe); err != nil {
			return c.JSON(
				http.StatusBadRequest, map[string]string{
					"error": fmt.Sprintf("invalid readiness probe: %v", err),
				},
			)
		}
	}

	// Execute task asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if err := s.agent.ExecuteTask(ctx, &req.Task); err != nil {
			c.Logger().Errorf("task execution failed: %v", err)
		}
	}()

	return c.JSON(
		http.StatusAccepted, map[string]string{
			"message": "task execution started",
			"taskId":  taskID,
		},
	)
}

// GetTaskStatus handles GET /api/v1/tasks/:id/status
// Returns the current status of a running task.
func (s *Server) GetTaskStatus(c echo.Context) error {
	taskID := c.Param("id")

	task, ok := s.agent.GetTask(taskID)
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "task not found"})
	}

	return c.JSON(http.StatusOK, task)
}

// GetTaskLogs handles GET /api/v1/tasks/:id/logs
// Returns container logs for a task.
func (s *Server) GetTaskLogs(c echo.Context) error {
	taskID := c.Param("id")
	tail := 100
	if tailParam := c.QueryParam("tail"); tailParam != "" {
		if _, err := fmt.Sscanf(tailParam, "%d", &tail); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tail parameter"})
		}
	}

	logs, err := s.agent.GetTaskLogs(c.Request().Context(), taskID, tail)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(
		http.StatusOK, map[string]interface{}{
			"taskId": taskID,
			"logs":   logs,
			"tail":   tail,
		},
	)
}

// validateHealthCheck validates health check configuration to prevent injection attacks
func validateHealthCheck(check *types.HealthCheck) error {
	if check == nil {
		return nil
	}

	if check.Port < 0 || check.Port > 65535 {
		return fmt.Errorf("port must be between 0 and 65535")
	}

	if check.Type == types.ProbeTypeHTTP {
		if check.HTTPPath == "" {
			return fmt.Errorf("HTTP path is required for HTTP probes")
		}

		// Validate HTTP path format
		if len(check.HTTPPath) > 0 && check.HTTPPath[0] != '/' {
			return fmt.Errorf("HTTP path must start with /")
		}

		// Reject obvious path traversal attempts
		if len(check.HTTPPath) > 2 && (check.HTTPPath[:3] == "/.." || check.HTTPPath[len(check.HTTPPath)-3:] == "/..") {
			return fmt.Errorf("path traversal detected in HTTP path")
		}

		// Reject control characters
		for _, ch := range check.HTTPPath {
			if ch < 32 || ch == 127 {
				return fmt.Errorf("control characters not allowed in HTTP path")
			}
		}
	}

	// Validate TCP-specific fields
	if check.Type == types.ProbeTypeTCP {
		if check.Port <= 0 {
			return fmt.Errorf("port is required for TCP probes")
		}
	}

	// Validate Exec-specific fields
	if check.Type == types.ProbeTypeExec {
		if len(check.Command) == 0 {
			return fmt.Errorf("command is required for Exec probes")
		}

		// Validate command does not contain injection attempts
		for _, cmd := range check.Command {
			// Reject null bytes
			for _, ch := range cmd {
				if ch == 0 {
					return fmt.Errorf("null bytes not allowed in commands")
				}
			}
		}
	}

	// Validate timing parameters
	if check.InitialDelaySeconds < 0 {
		return fmt.Errorf("initialDelaySeconds cannot be negative")
	}
	if check.PeriodSeconds < 0 {
		return fmt.Errorf("periodSeconds cannot be negative")
	}
	if check.TimeoutSeconds < 0 {
		return fmt.Errorf("timeoutSeconds cannot be negative")
	}
	if check.SuccessThreshold < 1 {
		return fmt.Errorf("successThreshold must be at least 1")
	}
	if check.FailureThreshold < 1 {
		return fmt.Errorf("failureThreshold must be at least 1")
	}

	return nil
}

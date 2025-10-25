package agent

import (
	"context"
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

	// Validate task ID matches
	if req.Task.TaskID != taskID {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "task ID mismatch"})
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

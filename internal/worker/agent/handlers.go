package agent

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danpasecinic/podling/internal/types"
	"github.com/labstack/echo/v4"
)

type ExecuteTaskRequest struct {
	Task types.Task `json:"task"`
}

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

func (s *Server) GetTaskStatus(c echo.Context) error {
	taskID := c.Param("id")

	task, ok := s.agent.GetTask(taskID)
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "task not found"})
	}

	return c.JSON(http.StatusOK, task)
}

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

type ExecutePodRequest struct {
	Pod types.Pod `json:"pod"`
}

func (s *Server) ExecutePod(c echo.Context) error {
	podID := c.Param("id")

	var req ExecutePodRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.Pod.PodID != podID {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "pod ID mismatch"})
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if err := s.agent.ExecutePod(ctx, &req.Pod); err != nil {
			c.Logger().Errorf("pod execution failed: %v", err)
		}
	}()

	return c.JSON(
		http.StatusAccepted, map[string]string{
			"message": "pod execution started",
			"podId":   podID,
		},
	)
}

func (s *Server) GetPodStatus(c echo.Context) error {
	podID := c.Param("id")

	pod, ok := s.agent.GetPod(podID)
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "pod not found"})
	}

	return c.JSON(http.StatusOK, pod)
}

func (s *Server) GetPodLogs(c echo.Context) error {
	podID := c.Param("id")
	containerName := c.QueryParam("container")
	tail := 100

	if tailParam := c.QueryParam("tail"); tailParam != "" {
		if n, err := fmt.Sscanf(tailParam, "%d", &tail); err != nil || n != 1 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tail parameter"})
		}
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	logs, err := s.agent.GetPodLogs(ctx, podID, containerName, tail)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	return c.JSON(
		http.StatusOK, map[string]interface{}{
			"podId": podID,
			"logs":  logs,
			"tail":  tail,
		},
	)
}

func (s *Server) DeletePod(c echo.Context) error {
	podID := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	if err := s.agent.CleanupPod(ctx, podID); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "pod cleaned up successfully"})
}

func (s *Server) StreamTaskLogs(c echo.Context) error {
	taskID := c.Param("id")
	tail := 100
	if tailParam := c.QueryParam("tail"); tailParam != "" {
		if _, err := fmt.Sscanf(tailParam, "%d", &tail); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tail parameter"})
		}
	}

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("Access-Control-Allow-Origin", "*")
	c.Response().WriteHeader(http.StatusOK)

	stream, err := s.agent.StreamTaskLogs(c.Request().Context(), taskID, tail)
	if err != nil {
		_, _ = fmt.Fprintf(c.Response(), "event: error\ndata: %s\n\n", err.Error())
		c.Response().Flush()
		return nil
	}
	defer func() { _ = stream.Reader.Close() }()

	scanner := bufio.NewScanner(stream.Reader)
	for scanner.Scan() {
		select {
		case <-c.Request().Context().Done():
			return nil
		default:
			line := scanner.Text()
			line = stripDockerLogHeader(line)
			_, _ = fmt.Fprintf(c.Response(), "data: %s\n\n", line)
			c.Response().Flush()
		}
	}

	if err := scanner.Err(); err != nil {
		_, _ = fmt.Fprintf(c.Response(), "event: error\ndata: %s\n\n", err.Error())
		c.Response().Flush()
	}

	return nil
}

func (s *Server) StreamPodLogs(c echo.Context) error {
	podID := c.Param("id")
	containerName := c.QueryParam("container")
	tail := 100

	if tailParam := c.QueryParam("tail"); tailParam != "" {
		if n, err := fmt.Sscanf(tailParam, "%d", &tail); err != nil || n != 1 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tail parameter"})
		}
	}

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("Access-Control-Allow-Origin", "*")
	c.Response().WriteHeader(http.StatusOK)

	stream, err := s.agent.StreamPodLogs(c.Request().Context(), podID, containerName, tail)
	if err != nil {
		_, _ = fmt.Fprintf(c.Response(), "event: error\ndata: %s\n\n", err.Error())
		c.Response().Flush()
		return nil
	}
	defer func() { _ = stream.Reader.Close() }()

	scanner := bufio.NewScanner(stream.Reader)
	for scanner.Scan() {
		select {
		case <-c.Request().Context().Done():
			return nil
		default:
			line := scanner.Text()
			line = stripDockerLogHeader(line)
			_, _ = fmt.Fprintf(c.Response(), "data: %s\n\n", line)
			c.Response().Flush()
		}
	}

	if err := scanner.Err(); err != nil {
		_, _ = fmt.Fprintf(c.Response(), "event: error\ndata: %s\n\n", err.Error())
		c.Response().Flush()
	}

	return nil
}

func stripDockerLogHeader(line string) string {
	if len(line) < 8 {
		return line
	}
	header := line[:8]
	if (header[0] == 1 || header[0] == 2) && header[1] == 0 && header[2] == 0 && header[3] == 0 {
		return strings.TrimPrefix(line[8:], " ")
	}
	return line
}

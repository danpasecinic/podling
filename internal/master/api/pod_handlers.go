package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/danpasecinic/podling/internal/master/state"
	"github.com/danpasecinic/podling/internal/types"
	"github.com/labstack/echo/v4"
)

// CreatePodRequest represents a request to create a new pod
type CreatePodRequest struct {
	Name          string              `json:"name" validate:"required"`
	Namespace     string              `json:"namespace,omitempty"`
	Labels        map[string]string   `json:"labels,omitempty"`
	Annotations   map[string]string   `json:"annotations,omitempty"`
	Containers    []types.Container   `json:"containers" validate:"required,min=1"`
	RestartPolicy types.RestartPolicy `json:"restartPolicy,omitempty"`
}

// UpdatePodStatusRequest represents a request to update a pod's status
type UpdatePodStatusRequest struct {
	Status      types.PodStatus   `json:"status" validate:"required"`
	Containers  []types.Container `json:"containers,omitempty"`
	Message     string            `json:"message,omitempty"`
	Reason      string            `json:"reason,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// CreatePod handles POST /api/v1/pods
func (s *Server) CreatePod(c echo.Context) error {
	var req CreatePodRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}

	if len(req.Containers) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "at least one container is required"})
	}

	containerNames := make(map[string]bool)
	for _, container := range req.Containers {
		if container.Name == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "all containers must have a name"})
		}
		if container.Image == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "all containers must have an image"})
		}
		if containerNames[container.Name] {
			return c.JSON(
				http.StatusBadRequest, map[string]string{"error": "container names must be unique within a pod"},
			)
		}
		containerNames[container.Name] = true
	}

	namespace := req.Namespace
	if namespace == "" {
		namespace = "default"
	}

	containers := req.Containers
	for i := range containers {
		containers[i].Status = types.ContainerWaiting
		containers[i].HealthStatus = types.HealthStatusUnknown
	}

	pod := types.Pod{
		PodID:         generateID(),
		Name:          req.Name,
		Namespace:     namespace,
		Labels:        req.Labels,
		Annotations:   req.Annotations,
		Containers:    containers,
		Status:        types.PodPending,
		RestartPolicy: req.RestartPolicy,
		CreatedAt:     time.Now(),
	}

	if err := s.store.AddPod(pod); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if err := s.schedulePod(pod.PodID); err != nil {
		updatedPod, _ := s.store.GetPod(pod.PodID)
		response := map[string]interface{}{
			"podId":           updatedPod.PodID,
			"name":            updatedPod.Name,
			"namespace":       updatedPod.Namespace,
			"labels":          updatedPod.Labels,
			"annotations":     updatedPod.Annotations,
			"containers":      updatedPod.Containers,
			"status":          updatedPod.Status,
			"restartPolicy":   updatedPod.RestartPolicy,
			"createdAt":       updatedPod.CreatedAt,
			"schedulingError": err.Error(),
		}
		return c.JSON(http.StatusCreated, response)
	}

	updatedPod, _ := s.store.GetPod(pod.PodID)
	return c.JSON(http.StatusCreated, updatedPod)
}

// ListPods handles GET /api/v1/pods
// Returns all pods in the system
func (s *Server) ListPods(c echo.Context) error {
	pods, err := s.store.ListPods()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, pods)
}

// GetPod handles GET /api/v1/pods/:id
func (s *Server) GetPod(c echo.Context) error {
	podID := c.Param("id")

	pod, err := s.store.GetPod(podID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "pod not found"})
	}

	return c.JSON(http.StatusOK, pod)
}

// UpdatePodStatus handles PUT /api/v1/pods/:id/status
func (s *Server) UpdatePodStatus(c echo.Context) error {
	podID := c.Param("id")

	var req UpdatePodStatusRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	now := time.Now()
	update := state.PodUpdate{
		Status:  &req.Status,
		Message: &req.Message,
		Reason:  &req.Reason,
	}

	// Update containers if provided
	if req.Containers != nil {
		update.Containers = req.Containers
	}

	// Update annotations if provided
	if req.Annotations != nil {
		update.Annotations = &req.Annotations
	}

	switch req.Status {
	case types.PodScheduled:
		update.ScheduledAt = &now
	case types.PodRunning:
		update.StartedAt = &now
	case types.PodSucceeded, types.PodFailed:
		update.FinishedAt = &now
	}

	if err := s.store.UpdatePod(podID, update); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "pod not found"})
	}

	pod, _ := s.store.GetPod(podID)
	return c.JSON(http.StatusOK, pod)
}

// DeletePod handles DELETE /api/v1/pods/:id
func (s *Server) DeletePod(c echo.Context) error {
	podID := c.Param("id")

	if err := s.store.DeletePod(podID); err != nil {
		if errors.Is(err, state.ErrPodNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "pod not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "pod deleted successfully"})
}

// schedulePod schedules a pod to an available node
func (s *Server) schedulePod(podID string) error {
	pod, err := s.store.GetPod(podID)
	if err != nil {
		return err
	}

	nodes, err := s.store.GetAvailableNodes()
	if err != nil {
		return err
	}

	selectedNode, err := s.scheduler.SelectNodeForPod(pod, nodes)
	if err != nil {
		return err
	}

	now := time.Now()
	update := state.PodUpdate{
		Status:      ptrTo(types.PodScheduled),
		NodeID:      &selectedNode.NodeID,
		ScheduledAt: &now,
	}

	if err := s.store.UpdatePod(podID, update); err != nil {
		return err
	}

	go s.triggerPodExecution(podID, *selectedNode)

	return nil
}

func (s *Server) triggerPodExecution(podID string, node types.Node) {
	pod, err := s.store.GetPod(podID)
	if err != nil {
		return
	}

	url := fmt.Sprintf("http://%s:%d/api/v1/pods/%s/execute", node.Hostname, node.Port, podID)

	payload := map[string]interface{}{
		"pod": pod,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

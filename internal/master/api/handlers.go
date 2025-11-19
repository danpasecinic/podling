package api

import (
	cryptoRand "crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/danpasecinic/podling/internal/master/state"
	"github.com/danpasecinic/podling/internal/types"
	"github.com/labstack/echo/v4"
)

// CreateTaskRequest represents a request to create a new task.
type CreateTaskRequest struct {
	Name           string              `json:"name" validate:"required"`
	Image          string              `json:"image" validate:"required"`
	Env            map[string]string   `json:"env"`
	LivenessProbe  *types.HealthCheck  `json:"livenessProbe,omitempty"`
	ReadinessProbe *types.HealthCheck  `json:"readinessProbe,omitempty"`
	RestartPolicy  types.RestartPolicy `json:"restartPolicy,omitempty"`
}

// UpdateTaskStatusRequest represents a request to update a task's status.
type UpdateTaskStatusRequest struct {
	Status       types.TaskStatus   `json:"status" validate:"required"`
	ContainerID  string             `json:"containerId"`
	Error        string             `json:"error"`
	HealthStatus types.HealthStatus `json:"healthStatus,omitempty"`
}

// RegisterNodeRequest represents a request to register a new worker node.
type RegisterNodeRequest struct {
	Hostname string `json:"hostname" validate:"required"`
	Port     int    `json:"port" validate:"required"`
	CPU      string `json:"cpu" validate:"required"`    // e.g., "2", "500m", "2.5"
	Memory   string `json:"memory" validate:"required"` // e.g., "1Gi", "512Mi", "1073741824"
}

// CreateTask handles POST /api/v1/tasks.
// Creates a new task and automatically schedules it to an available node.
func (s *Server) CreateTask(c echo.Context) error {
	var req CreateTaskRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.Name == "" || req.Image == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name and image are required"})
	}

	task := types.Task{
		TaskID:         generateID(),
		Name:           req.Name,
		Image:          req.Image,
		Env:            req.Env,
		Status:         types.TaskPending,
		CreatedAt:      time.Now(),
		LivenessProbe:  req.LivenessProbe,
		ReadinessProbe: req.ReadinessProbe,
		RestartPolicy:  req.RestartPolicy,
		HealthStatus:   types.HealthStatusUnknown,
	}

	if err := s.store.AddTask(task); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	if err := s.scheduleTask(task.TaskID); err != nil {
		updatedTask, _ := s.store.GetTask(task.TaskID)
		response := map[string]interface{}{
			"taskId":          updatedTask.TaskID,
			"name":            updatedTask.Name,
			"image":           updatedTask.Image,
			"env":             updatedTask.Env,
			"status":          updatedTask.Status,
			"createdAt":       updatedTask.CreatedAt,
			"schedulingError": err.Error(),
		}
		return c.JSON(http.StatusCreated, response)
	}

	updatedTask, _ := s.store.GetTask(task.TaskID)
	return c.JSON(http.StatusCreated, updatedTask)
}

// ListTasks handles GET /api/v1/tasks.
// Returns all tasks in the system.
func (s *Server) ListTasks(c echo.Context) error {
	tasks, err := s.store.ListTasks()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, tasks)
}

// GetTask handles GET /api/v1/tasks/:id.
// Returns details for a specific task.
func (s *Server) GetTask(c echo.Context) error {
	taskID := c.Param("id")

	task, err := s.store.GetTask(taskID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "task not found"})
	}

	return c.JSON(http.StatusOK, task)
}

// UpdateTaskStatus handles PUT /api/v1/tasks/:id/status.
// Updates the status of a task, typically called by worker nodes.
func (s *Server) UpdateTaskStatus(c echo.Context) error {
	taskID := c.Param("id")

	var req UpdateTaskStatusRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	now := time.Now()
	update := state.TaskUpdate{
		Status:      &req.Status,
		ContainerID: &req.ContainerID,
	}

	if req.HealthStatus != "" {
		update.HealthStatus = &req.HealthStatus
	}

	switch req.Status {
	case types.TaskRunning:
		update.StartedAt = &now
	case types.TaskCompleted, types.TaskFailed:
		update.FinishedAt = &now
		if req.Error != "" {
			update.Error = &req.Error
		}
	}

	if err := s.store.UpdateTask(taskID, update); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "task not found"})
	}

	task, _ := s.store.GetTask(taskID)
	return c.JSON(http.StatusOK, task)
}

// RegisterNode handles POST /api/v1/nodes/register.
// Registers a new worker node with the master.
func (s *Server) RegisterNode(c echo.Context) error {
	var req RegisterNodeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.Hostname == "" || req.Port == 0 || req.CPU == "" || req.Memory == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "hostname, port, cpu, and memory are required"})
	}

	cpuMillicores, err := types.ParseCPU(req.CPU)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid cpu format: %v", err)})
	}

	memoryBytes, err := types.ParseMemory(req.Memory)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid memory format: %v", err)})
	}

	node := types.Node{
		NodeID:        generateID(),
		Hostname:      req.Hostname,
		Port:          req.Port,
		Status:        types.NodeOnline,
		RunningTasks:  0,
		LastHeartbeat: time.Now(),
		Resources: &types.NodeResources{
			Capacity: types.ResourceList{
				CPU:    cpuMillicores,
				Memory: memoryBytes,
			},
			Allocatable: types.ResourceList{
				CPU:    cpuMillicores,
				Memory: memoryBytes,
			},
			Used: types.ResourceList{
				CPU:    0,
				Memory: 0,
			},
		},
	}

	if err := s.store.AddNode(node); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, node)
}

// NodeHeartbeat handles POST /api/v1/nodes/:id/heartbeat.
// Updates the last heartbeat time for a worker node.
func (s *Server) NodeHeartbeat(c echo.Context) error {
	nodeID := c.Param("id")

	now := time.Now()
	update := state.NodeUpdate{
		Status:        ptrTo(types.NodeOnline),
		LastHeartbeat: &now,
	}

	if err := s.store.UpdateNode(nodeID, update); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "node not found"})
	}

	node, _ := s.store.GetNode(nodeID)
	return c.JSON(http.StatusOK, node)
}

// ListNodes handles GET /api/v1/nodes.
// Returns all registered worker nodes with updated status based on heartbeat.
func (s *Server) ListNodes(c echo.Context) error {
	nodes, err := s.store.ListNodes()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	now := time.Now()
	heartbeatTimeout := 90 * time.Second

	for i := range nodes {
		if nodes[i].LastHeartbeat.Add(heartbeatTimeout).Before(now) {
			nodes[i].Status = types.NodeOffline
		}
	}

	return c.JSON(http.StatusOK, nodes)
}

func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randString(8)
}

func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		idx, err := cryptoRand.Int(cryptoRand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			panic(err)
		}
		b[i] = letters[idx.Int64()]
	}
	return string(b)
}

func ptrTo[T any](v T) *T {
	return &v
}

func (s *Server) scheduleTask(taskID string) error {
	task, err := s.store.GetTask(taskID)
	if err != nil {
		return err
	}

	nodes, err := s.store.GetAvailableNodes()
	if err != nil {
		return err
	}

	selectedNode, err := s.scheduler.SelectNode(task, nodes)
	if err != nil {
		return err
	}

	update := state.TaskUpdate{
		Status: ptrTo(types.TaskScheduled),
		NodeID: &selectedNode.NodeID,
	}

	return s.store.UpdateTask(taskID, update)
}

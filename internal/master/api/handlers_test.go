package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danpasecinic/podling/internal/master/scheduler"
	"github.com/danpasecinic/podling/internal/master/services"
	"github.com/danpasecinic/podling/internal/master/state"
	"github.com/danpasecinic/podling/internal/types"
	"github.com/labstack/echo/v4"
)

func setupTestServer() (*Server, *echo.Echo) {
	store := state.NewInMemoryStore()
	sched := scheduler.NewRoundRobin()
	endpointController := services.NewEndpointController(store)
	server := NewServer(store, sched, endpointController)
	e := echo.New()
	server.RegisterRoutes(e)
	return server, e
}

func TestCreateTask(t *testing.T) {
	tests := []struct {
		name       string
		reqBody    string
		wantStatus int
		wantFields map[string]interface{}
	}{
		{
			name:       "valid task creation",
			reqBody:    `{"name":"test-task","image":"nginx:latest"}`,
			wantStatus: http.StatusCreated,
			wantFields: map[string]interface{}{
				"name":   "test-task",
				"image":  "nginx:latest",
				"status": string(types.TaskPending),
			},
		},
		{
			name:       "task with environment variables",
			reqBody:    `{"name":"env-task","image":"redis:latest","env":{"KEY":"value"}}`,
			wantStatus: http.StatusCreated,
			wantFields: map[string]interface{}{
				"name":  "env-task",
				"image": "redis:latest",
			},
		},
		{
			name:       "invalid JSON",
			reqBody:    `{"name":"test"`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty body",
			reqBody:    `{}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				_, e := setupTestServer()

				req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader(tt.reqBody))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()

				e.ServeHTTP(rec, req)

				if rec.Code != tt.wantStatus {
					t.Errorf("CreateTask() status = %v, want %v", rec.Code, tt.wantStatus)
				}

				if tt.wantStatus == http.StatusCreated {
					var resp map[string]interface{}
					if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
						t.Fatalf("Failed to unmarshal response: %v", err)
					}

					for key, want := range tt.wantFields {
						if got, ok := resp[key]; !ok || got != want {
							t.Errorf("CreateTask() %s = %v, want %v", key, got, want)
						}
					}

					if _, ok := resp["taskId"]; !ok {
						t.Error("CreateTask() response missing taskId")
					}
				}
			},
		)
	}
}

func TestListTasks(t *testing.T) {
	server, e := setupTestServer()

	task1 := types.Task{TaskID: "task1", Name: "test1", Image: "nginx", Status: types.TaskPending}
	task2 := types.Task{TaskID: "task2", Name: "test2", Image: "redis", Status: types.TaskRunning}
	_ = server.store.AddTask(task1)
	_ = server.store.AddTask(task2)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("ListTasks() status = %v, want %v", rec.Code, http.StatusOK)
	}

	var tasks []types.Task
	if err := json.Unmarshal(rec.Body.Bytes(), &tasks); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("ListTasks() returned %d tasks, want 2", len(tasks))
	}
}

func TestGetTask(t *testing.T) {
	server, e := setupTestServer()

	task := types.Task{TaskID: "task123", Name: "test-task", Image: "nginx", Status: types.TaskPending}
	_ = server.store.AddTask(task)

	tests := []struct {
		name       string
		taskID     string
		wantStatus int
	}{
		{
			name:       "existing task",
			taskID:     "task123",
			wantStatus: http.StatusOK,
		},
		{
			name:       "non-existent task",
			taskID:     "nonexistent",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+tt.taskID, nil)
				rec := httptest.NewRecorder()

				e.ServeHTTP(rec, req)

				if rec.Code != tt.wantStatus {
					t.Errorf("GetTask() status = %v, want %v", rec.Code, tt.wantStatus)
				}

				if tt.wantStatus == http.StatusOK {
					var resp types.Task
					if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
						t.Fatalf("Failed to unmarshal response: %v", err)
					}

					if resp.TaskID != tt.taskID {
						t.Errorf("GetTask() taskID = %v, want %v", resp.TaskID, tt.taskID)
					}
				}
			},
		)
	}
}

func TestUpdateTaskStatus(t *testing.T) {
	server, e := setupTestServer()

	task := types.Task{TaskID: "task123", Name: "test-task", Image: "nginx", Status: types.TaskPending}
	_ = server.store.AddTask(task)

	tests := []struct {
		name       string
		taskID     string
		reqBody    string
		wantStatus int
		checkField func(*testing.T, types.Task)
	}{
		{
			name:       "update to running",
			taskID:     "task123",
			reqBody:    `{"status":"running","containerId":"abc123"}`,
			wantStatus: http.StatusOK,
			checkField: func(t *testing.T, task types.Task) {
				if task.Status != types.TaskRunning {
					t.Errorf("status = %v, want running", task.Status)
				}
				if task.ContainerID != "abc123" {
					t.Errorf("containerID = %v, want abc123", task.ContainerID)
				}
				if task.StartedAt == nil {
					t.Error("startedAt should be set")
				}
			},
		},
		{
			name:       "update to completed",
			taskID:     "task123",
			reqBody:    `{"status":"completed"}`,
			wantStatus: http.StatusOK,
			checkField: func(t *testing.T, task types.Task) {
				if task.Status != types.TaskCompleted {
					t.Errorf("status = %v, want completed", task.Status)
				}
				if task.FinishedAt == nil {
					t.Error("finishedAt should be set")
				}
			},
		},
		{
			name:       "update to failed with error",
			taskID:     "task123",
			reqBody:    `{"status":"failed","error":"container crashed"}`,
			wantStatus: http.StatusOK,
			checkField: func(t *testing.T, task types.Task) {
				if task.Status != types.TaskFailed {
					t.Errorf("status = %v, want failed", task.Status)
				}
				if task.Error != "container crashed" {
					t.Errorf("error = %v, want 'container crashed'", task.Error)
				}
			},
		},
		{
			name:       "non-existent task",
			taskID:     "nonexistent",
			reqBody:    `{"status":"running"}`,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				req := httptest.NewRequest(
					http.MethodPut, "/api/v1/tasks/"+tt.taskID+"/status", strings.NewReader(tt.reqBody),
				)
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()

				e.ServeHTTP(rec, req)

				if rec.Code != tt.wantStatus {
					t.Errorf("UpdateTaskStatus() status = %v, want %v", rec.Code, tt.wantStatus)
				}

				if tt.wantStatus == http.StatusOK && tt.checkField != nil {
					var resp types.Task
					if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
						t.Fatalf("Failed to unmarshal response: %v", err)
					}
					tt.checkField(t, resp)
				}
			},
		)
	}
}

func TestRegisterNode(t *testing.T) {
	tests := []struct {
		name       string
		reqBody    string
		wantStatus int
		wantFields map[string]interface{}
	}{
		{
			name:       "valid node registration",
			reqBody:    `{"hostname":"worker1","port":8081,"cpu":"10","memory":"10Gi"}`,
			wantStatus: http.StatusCreated,
			wantFields: map[string]interface{}{
				"hostname": "worker1",
				"port":     float64(8081),
				"status":   string(types.NodeOnline),
			},
		},
		{
			name:       "invalid JSON",
			reqBody:    `{"hostname":"worker1"`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing required fields",
			reqBody:    `{"hostname":"worker1"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid CPU format",
			reqBody:    `{"hostname":"worker1","port":8081,"cpu":"invalid","memory":"10Gi"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid memory format",
			reqBody:    `{"hostname":"worker1","port":8081,"cpu":"10","memory":"invalid"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				_, e := setupTestServer()

				req := httptest.NewRequest(http.MethodPost, "/api/v1/nodes/register", strings.NewReader(tt.reqBody))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()

				e.ServeHTTP(rec, req)

				if rec.Code != tt.wantStatus {
					t.Errorf("RegisterNode() status = %v, want %v", rec.Code, tt.wantStatus)
				}

				if tt.wantStatus == http.StatusCreated {
					var resp map[string]interface{}
					if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
						t.Fatalf("Failed to unmarshal response: %v", err)
					}

					for key, want := range tt.wantFields {
						if got, ok := resp[key]; !ok || got != want {
							t.Errorf("RegisterNode() %s = %v, want %v", key, got, want)
						}
					}

					if _, ok := resp["nodeId"]; !ok {
						t.Error("RegisterNode() response missing nodeId")
					}
				}
			},
		)
	}
}

func TestNodeHeartbeat(t *testing.T) {
	server, e := setupTestServer()

	node := types.Node{NodeID: "node123", Hostname: "worker1", Status: types.NodeOnline}
	_ = server.store.AddNode(node)

	tests := []struct {
		name       string
		nodeID     string
		wantStatus int
	}{
		{
			name:       "existing node",
			nodeID:     "node123",
			wantStatus: http.StatusOK,
		},
		{
			name:       "non-existent node",
			nodeID:     "nonexistent",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/nodes/"+tt.nodeID+"/heartbeat", nil)
				rec := httptest.NewRecorder()

				e.ServeHTTP(rec, req)

				if rec.Code != tt.wantStatus {
					t.Errorf("NodeHeartbeat() status = %v, want %v", rec.Code, tt.wantStatus)
				}

				if tt.wantStatus == http.StatusOK {
					var resp types.Node
					if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
						t.Fatalf("Failed to unmarshal response: %v", err)
					}

					if resp.Status != types.NodeOnline {
						t.Errorf("NodeHeartbeat() status = %v, want online", resp.Status)
					}
				}
			},
		)
	}
}

func TestListNodes(t *testing.T) {
	server, e := setupTestServer()

	node1 := types.Node{NodeID: "node1", Hostname: "worker1", Status: types.NodeOnline}
	node2 := types.Node{NodeID: "node2", Hostname: "worker2", Status: types.NodeOnline}
	_ = server.store.AddNode(node1)
	_ = server.store.AddNode(node2)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nodes", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("ListNodes() status = %v, want %v", rec.Code, http.StatusOK)
	}

	var nodes []types.Node
	if err := json.Unmarshal(rec.Body.Bytes(), &nodes); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(nodes) != 2 {
		t.Errorf("ListNodes() returned %d nodes, want 2", len(nodes))
	}
}

func TestTaskScheduling(t *testing.T) {
	server, e := setupTestServer()

	node := types.Node{
		NodeID:   "node1",
		Hostname: "worker1",
		Status:   types.NodeOnline,
		Resources: &types.NodeResources{
			Capacity:    types.ResourceList{CPU: 10000, Memory: 10 * 1024 * 1024 * 1024},
			Allocatable: types.ResourceList{CPU: 10000, Memory: 10 * 1024 * 1024 * 1024},
			Used:        types.ResourceList{CPU: 0, Memory: 0},
		},
	}
	_ = server.store.AddNode(node)

	reqBody := `{"name":"scheduled-task","image":"nginx:latest"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("CreateTask() status = %v, want %v", rec.Code, http.StatusCreated)
	}

	var resp types.Task
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Status != types.TaskScheduled {
		t.Errorf("Task status = %v, want scheduled", resp.Status)
	}

	if resp.NodeID != "node1" {
		t.Errorf("Task nodeID = %v, want node1", resp.NodeID)
	}
}

func TestCreateTaskWithoutAvailableNodes(t *testing.T) {
	_, e := setupTestServer()

	reqBody := `{"name":"orphan-task","image":"nginx:latest"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("CreateTask() status = %v, want %v", rec.Code, http.StatusCreated)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if _, hasError := resp["schedulingError"]; !hasError {
		t.Error("Expected schedulingError field when no nodes available")
	}
}

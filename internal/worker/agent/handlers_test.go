package agent

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danpasecinic/podling/internal/types"
	"github.com/labstack/echo/v4"
)

func TestExecuteTask(t *testing.T) {
	agent, _ := NewAgent("test-node", "http://localhost:8080")
	defer agent.Stop()

	server := NewServer("test-node", "localhost", 8081, agent)
	e := echo.New()
	server.RegisterRoutes(e)

	task := types.Task{
		TaskID: "task-1",
		Name:   "test",
		Image:  "alpine",
		Status: types.TaskPending,
	}

	reqBody := ExecuteTaskRequest{Task: task}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/task-1/execute", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("task-1")

	if err := server.ExecuteTask(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", rec.Code)
	}
}

func TestGetTaskStatus(t *testing.T) {
	agent, _ := NewAgent("test-node", "http://localhost:8080")
	defer agent.Stop()

	task := &types.Task{
		TaskID: "task-1",
		Name:   "test",
		Image:  "nginx",
		Status: types.TaskRunning,
	}
	agent.mu.Lock()
	agent.runningTasks[task.TaskID] = task
	agent.mu.Unlock()

	server := NewServer("test-node", "localhost", 8081, agent)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/task-1/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("task-1")

	if err := server.GetTaskStatus(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/tasks/nonexistent/status", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("nonexistent")

	if err := server.GetTaskStatus(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestGetTaskLogsHandler(t *testing.T) {
	agent, _ := NewAgent("test-node", "http://localhost:8080")
	defer agent.Stop()

	server := NewServer("test-node", "localhost", 8081, agent)
	e := echo.New()

	// Test: task not found
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/nonexistent/logs?tail=100", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("nonexistent")

	if err := server.GetTaskLogs(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	// Test: task without container
	task := &types.Task{
		TaskID:      "task-1",
		Name:        "test",
		Image:       "nginx",
		Status:      types.TaskRunning,
		ContainerID: "",
	}
	agent.mu.Lock()
	agent.runningTasks[task.TaskID] = task
	agent.mu.Unlock()

	req = httptest.NewRequest(http.MethodGet, "/api/v1/tasks/task-1/logs", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("task-1")

	if err := server.GetTaskLogs(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	// Test: invalid tail parameter
	req = httptest.NewRequest(http.MethodGet, "/api/v1/tasks/task-1/logs?tail=invalid", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("task-1")

	if err := server.GetTaskLogs(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestExecuteTaskMismatchID(t *testing.T) {
	agent, _ := NewAgent("test-node", "http://localhost:8080")
	defer agent.Stop()

	server := NewServer("test-node", "localhost", 8081, agent)
	e := echo.New()
	server.RegisterRoutes(e)

	task := types.Task{
		TaskID: "task-2",
		Name:   "test",
		Image:  "alpine",
		Status: types.TaskPending,
	}

	reqBody := ExecuteTaskRequest{Task: task}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/task-1/execute", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("task-1")

	if err := server.ExecuteTask(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for ID mismatch, got %d", rec.Code)
	}
}

func TestExecuteTaskInvalidJSON(t *testing.T) {
	agent, _ := NewAgent("test-node", "http://localhost:8080")
	defer agent.Stop()

	server := NewServer("test-node", "localhost", 8081, agent)
	e := echo.New()
	server.RegisterRoutes(e)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/task-1/execute", bytes.NewReader([]byte("invalid json")))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("task-1")

	if err := server.ExecuteTask(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid JSON, got %d", rec.Code)
	}
}

func TestValidateHealthCheck(t *testing.T) {
	tests := []struct {
		name    string
		check   *types.HealthCheck
		wantErr bool
	}{
		{
			name:    "nil check",
			check:   nil,
			wantErr: false,
		},
		{
			name: "valid HTTP probe",
			check: &types.HealthCheck{
				Type:             types.ProbeTypeHTTP,
				HTTPPath:         "/health",
				Port:             8080,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			},
			wantErr: false,
		},
		{
			name: "valid TCP probe",
			check: &types.HealthCheck{
				Type:             types.ProbeTypeTCP,
				Port:             3306,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			},
			wantErr: false,
		},
		{
			name: "valid Exec probe",
			check: &types.HealthCheck{
				Type:             types.ProbeTypeExec,
				Command:          []string{"cat", "/tmp/healthy"},
				SuccessThreshold: 1,
				FailureThreshold: 3,
			},
			wantErr: false,
		},
		{
			name: "invalid port - negative",
			check: &types.HealthCheck{
				Type: types.ProbeTypeTCP,
				Port: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid port - too high",
			check: &types.HealthCheck{
				Type: types.ProbeTypeTCP,
				Port: 65536,
			},
			wantErr: true,
		},
		{
			name: "HTTP probe without path",
			check: &types.HealthCheck{
				Type: types.ProbeTypeHTTP,
				Port: 8080,
			},
			wantErr: true,
		},
		{
			name: "HTTP probe with invalid path - no leading slash",
			check: &types.HealthCheck{
				Type:     types.ProbeTypeHTTP,
				HTTPPath: "health",
				Port:     8080,
			},
			wantErr: true,
		},
		{
			name: "HTTP probe with path traversal - beginning",
			check: &types.HealthCheck{
				Type:     types.ProbeTypeHTTP,
				HTTPPath: "/../etc/passwd",
				Port:     8080,
			},
			wantErr: true,
		},
		{
			name: "HTTP probe with path traversal - end",
			check: &types.HealthCheck{
				Type:     types.ProbeTypeHTTP,
				HTTPPath: "/health/..",
				Port:     8080,
			},
			wantErr: true,
		},
		{
			name: "HTTP probe with control characters",
			check: &types.HealthCheck{
				Type:     types.ProbeTypeHTTP,
				HTTPPath: "/health\x00",
				Port:     8080,
			},
			wantErr: true,
		},
		{
			name: "TCP probe without port",
			check: &types.HealthCheck{
				Type: types.ProbeTypeTCP,
				Port: 0,
			},
			wantErr: true,
		},
		{
			name: "Exec probe without command",
			check: &types.HealthCheck{
				Type:    types.ProbeTypeExec,
				Command: []string{},
			},
			wantErr: true,
		},
		{
			name: "Exec probe with null byte in command",
			check: &types.HealthCheck{
				Type:    types.ProbeTypeExec,
				Command: []string{"cat\x00/etc/passwd"},
			},
			wantErr: true,
		},
		{
			name: "negative initialDelaySeconds",
			check: &types.HealthCheck{
				Type:                types.ProbeTypeTCP,
				Port:                8080,
				InitialDelaySeconds: -5,
			},
			wantErr: true,
		},
		{
			name: "negative periodSeconds",
			check: &types.HealthCheck{
				Type:          types.ProbeTypeTCP,
				Port:          8080,
				PeriodSeconds: -10,
			},
			wantErr: true,
		},
		{
			name: "negative timeoutSeconds",
			check: &types.HealthCheck{
				Type:           types.ProbeTypeTCP,
				Port:           8080,
				TimeoutSeconds: -3,
			},
			wantErr: true,
		},
		{
			name: "successThreshold less than 1",
			check: &types.HealthCheck{
				Type:             types.ProbeTypeTCP,
				Port:             8080,
				SuccessThreshold: 0,
			},
			wantErr: true,
		},
		{
			name: "failureThreshold less than 1",
			check: &types.HealthCheck{
				Type:             types.ProbeTypeTCP,
				Port:             8080,
				FailureThreshold: 0,
			},
			wantErr: true,
		},
		{
			name: "valid health check with all timing parameters",
			check: &types.HealthCheck{
				Type:                types.ProbeTypeTCP,
				Port:                8080,
				InitialDelaySeconds: 10,
				PeriodSeconds:       5,
				TimeoutSeconds:      3,
				SuccessThreshold:    1,
				FailureThreshold:    3,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				err := validateHealthCheck(tt.check)
				if (err != nil) != tt.wantErr {
					t.Errorf("validateHealthCheck() error = %v, wantErr %v", err, tt.wantErr)
				}
			},
		)
	}
}

func TestExecuteTask_WithHealthChecks(t *testing.T) {
	agent, _ := NewAgent("test-node", "http://localhost:8080")
	defer agent.Stop()

	server := NewServer("test-node", "localhost", 8081, agent)
	e := echo.New()
	server.RegisterRoutes(e)

	tests := []struct {
		name           string
		livenessProbe  *types.HealthCheck
		readinessProbe *types.HealthCheck
		expectedStatus int
	}{
		{
			name: "valid liveness probe",
			livenessProbe: &types.HealthCheck{
				Type:             types.ProbeTypeHTTP,
				HTTPPath:         "/health",
				Port:             8080,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name: "invalid liveness probe - path traversal",
			livenessProbe: &types.HealthCheck{
				Type:     types.ProbeTypeHTTP,
				HTTPPath: "/../etc/passwd",
				Port:     8080,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "valid readiness probe",
			readinessProbe: &types.HealthCheck{
				Type:             types.ProbeTypeHTTP,
				HTTPPath:         "/ready",
				Port:             8080,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name: "invalid readiness probe - control characters",
			readinessProbe: &types.HealthCheck{
				Type:     types.ProbeTypeHTTP,
				HTTPPath: "/health\x00",
				Port:     8080,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "both probes valid",
			livenessProbe: &types.HealthCheck{
				Type:             types.ProbeTypeHTTP,
				HTTPPath:         "/health",
				Port:             8080,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			},
			readinessProbe: &types.HealthCheck{
				Type:             types.ProbeTypeTCP,
				Port:             3306,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			},
			expectedStatus: http.StatusAccepted,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				task := types.Task{
					TaskID:         "task-1",
					Name:           "test",
					Image:          "alpine",
					Status:         types.TaskPending,
					LivenessProbe:  tt.livenessProbe,
					ReadinessProbe: tt.readinessProbe,
				}

				reqBody := ExecuteTaskRequest{Task: task}
				body, _ := json.Marshal(reqBody)

				req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/task-1/execute", bytes.NewReader(body))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("id")
				c.SetParamValues("task-1")

				if err := server.ExecuteTask(c); err != nil {
					t.Fatalf("handler returned error: %v", err)
				}

				if rec.Code != tt.expectedStatus {
					t.Errorf("expected status %d, got %d. Body: %s", tt.expectedStatus, rec.Code, rec.Body.String())
				}
			},
		)
	}
}

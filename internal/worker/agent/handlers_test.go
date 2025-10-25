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

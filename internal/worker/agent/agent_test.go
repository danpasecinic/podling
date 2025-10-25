package agent

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/danpasecinic/podling/internal/types"
)

func TestNewAgent(t *testing.T) {
	agent, err := NewAgent("test-node", "http://localhost:8080")
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}
	defer agent.Stop()

	if agent.nodeID != "test-node" {
		t.Errorf("expected nodeID test-node, got %s", agent.nodeID)
	}
	if agent.masterURL != "http://localhost:8080" {
		t.Errorf("expected masterURL http://localhost:8080, got %s", agent.masterURL)
	}
}

func TestGetTask(t *testing.T) {
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

	retrieved, ok := agent.GetTask("task-1")
	if !ok {
		t.Fatal("expected task to be found")
	}
	if retrieved.TaskID != "task-1" {
		t.Errorf("expected task ID task-1, got %s", retrieved.TaskID)
	}

	_, ok = agent.GetTask("nonexistent")
	if ok {
		t.Error("expected task not to be found")
	}
}

func TestHeartbeat(t *testing.T) {
	var callCount int
	var mu sync.Mutex
	
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v1/nodes/test-node/heartbeat" && r.Method == http.MethodPost {
					mu.Lock()
					callCount++
					mu.Unlock()
					w.WriteHeader(http.StatusOK)
				}
			},
		),
	)
	defer server.Close()

	agent, _ := NewAgent("test-node", server.URL)
	defer agent.Stop()

	agent.Start(100 * time.Millisecond)
	time.Sleep(350 * time.Millisecond)

	mu.Lock()
	count := callCount
	mu.Unlock()

	if count < 2 {
		t.Errorf("expected at least 2 heartbeats, got %d", count)
	}
}

package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danpasecinic/podling/internal/master/scheduler"
	"github.com/danpasecinic/podling/internal/master/state"
	"github.com/danpasecinic/podling/internal/types"
	"github.com/labstack/echo/v4"
)

func TestCreatePod(t *testing.T) {
	e := echo.New()
	store := state.NewInMemoryStore()
	sched := scheduler.NewRoundRobin()
	server := NewServer(store, sched)

	node := types.Node{
		NodeID:        "node-1",
		Hostname:      "localhost",
		Port:          8081,
		Status:        types.NodeOnline,
		RunningTasks:  0,
		LastHeartbeat: time.Now(),
		Resources: &types.NodeResources{
			Capacity:    types.ResourceList{CPU: 10000, Memory: 10 * 1024 * 1024 * 1024},
			Allocatable: types.ResourceList{CPU: 10000, Memory: 10 * 1024 * 1024 * 1024},
			Used:        types.ResourceList{CPU: 0, Memory: 0},
		},
	}
	_ = store.AddNode(node)

	t.Run(
		"Create pod with single container", func(t *testing.T) {
			payload := map[string]interface{}{
				"name":      "test-pod",
				"namespace": "default",
				"containers": []map[string]interface{}{
					{
						"name":  "nginx",
						"image": "nginx:latest",
					},
				},
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/pods", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := server.CreatePod(c)
			if err != nil {
				t.Fatalf("CreatePod failed: %v", err)
			}

			if rec.Code != http.StatusCreated {
				t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
			}

			var pod types.Pod
			if err := json.Unmarshal(rec.Body.Bytes(), &pod); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if pod.Name != "test-pod" {
				t.Errorf("expected name test-pod, got %s", pod.Name)
			}
			if pod.Namespace != "default" {
				t.Errorf("expected namespace default, got %s", pod.Namespace)
			}
			if len(pod.Containers) != 1 {
				t.Errorf("expected 1 container, got %d", len(pod.Containers))
			}
			if pod.Status != types.PodScheduled {
				t.Errorf("expected status scheduled, got %s", pod.Status)
			}
		},
	)

	t.Run(
		"Create pod with multiple containers", func(t *testing.T) {
			payload := map[string]interface{}{
				"name":      "multi-container-pod",
				"namespace": "production",
				"labels": map[string]string{
					"app":     "myapp",
					"version": "1.0",
				},
				"containers": []map[string]interface{}{
					{
						"name":  "app",
						"image": "myapp:1.0",
						"env": map[string]string{
							"PORT": "8080",
						},
					},
					{
						"name":  "sidecar",
						"image": "nginx:latest",
					},
				},
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/pods", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := server.CreatePod(c)
			if err != nil {
				t.Fatalf("CreatePod failed: %v", err)
			}

			if rec.Code != http.StatusCreated {
				t.Errorf("expected status %d, got %d", http.StatusCreated, rec.Code)
			}

			var pod types.Pod
			_ = json.Unmarshal(rec.Body.Bytes(), &pod)

			if len(pod.Containers) != 2 {
				t.Errorf("expected 2 containers, got %d", len(pod.Containers))
			}
			if pod.Labels["app"] != "myapp" {
				t.Errorf("expected label app=myapp, got %s", pod.Labels["app"])
			}
			if pod.Containers[0].Env["PORT"] != "8080" {
				t.Error("expected PORT env var in first container")
			}
		},
	)

	t.Run(
		"Create pod without name", func(t *testing.T) {
			payload := map[string]interface{}{
				"containers": []map[string]interface{}{
					{"name": "nginx", "image": "nginx:latest"},
				},
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/pods", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			_ = server.CreatePod(c)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		},
	)

	t.Run(
		"Create pod without containers", func(t *testing.T) {
			payload := map[string]interface{}{
				"name":       "empty-pod",
				"containers": []map[string]interface{}{},
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/pods", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			_ = server.CreatePod(c)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		},
	)

	t.Run(
		"Create pod with duplicate container names", func(t *testing.T) {
			payload := map[string]interface{}{
				"name": "duplicate-containers",
				"containers": []map[string]interface{}{
					{"name": "nginx", "image": "nginx:latest"},
					{"name": "nginx", "image": "nginx:alpine"},
				},
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/pods", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			_ = server.CreatePod(c)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		},
	)
}

func TestListPods(t *testing.T) {
	e := echo.New()
	store := state.NewInMemoryStore()
	sched := scheduler.NewRoundRobin()
	server := NewServer(store, sched)

	// Add test pods
	for i := 1; i <= 3; i++ {
		pod := types.Pod{
			PodID:     generateTestPodID(i),
			Name:      generateTestPodName(i),
			Namespace: "default",
			Containers: []types.Container{
				{Name: "container", Image: "nginx:latest"},
			},
			Status:    types.PodPending,
			CreatedAt: time.Now(),
		}
		_ = store.AddPod(pod)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pods", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := server.ListPods(c)
	if err != nil {
		t.Fatalf("ListPods failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var pods []types.Pod
	if err := json.Unmarshal(rec.Body.Bytes(), &pods); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(pods) != 3 {
		t.Errorf("expected 3 pods, got %d", len(pods))
	}
}

func TestGetPod(t *testing.T) {
	e := echo.New()
	store := state.NewInMemoryStore()
	sched := scheduler.NewRoundRobin()
	server := NewServer(store, sched)

	pod := types.Pod{
		PodID:     "pod-123",
		Name:      "test-pod",
		Namespace: "default",
		Labels:    map[string]string{"app": "web"},
		Containers: []types.Container{
			{Name: "nginx", Image: "nginx:latest"},
		},
		Status:    types.PodRunning,
		CreatedAt: time.Now(),
	}
	_ = store.AddPod(pod)

	t.Run(
		"Get existing pod", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/pods/pod-123", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues("pod-123")

			err := server.GetPod(c)
			if err != nil {
				t.Fatalf("GetPod failed: %v", err)
			}

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			var retrieved types.Pod
			_ = json.Unmarshal(rec.Body.Bytes(), &retrieved)

			if retrieved.PodID != "pod-123" {
				t.Errorf("expected PodID pod-123, got %s", retrieved.PodID)
			}
			if retrieved.Labels["app"] != "web" {
				t.Error("expected label app=web")
			}
		},
	)

	t.Run(
		"Get non-existent pod", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/pods/non-existent", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues("non-existent")

			_ = server.GetPod(c)

			if rec.Code != http.StatusNotFound {
				t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
			}
		},
	)
}

func TestUpdatePodStatus(t *testing.T) {
	e := echo.New()
	store := state.NewInMemoryStore()
	sched := scheduler.NewRoundRobin()
	server := NewServer(store, sched)

	pod := types.Pod{
		PodID:     "pod-123",
		Name:      "test-pod",
		Namespace: "default",
		Containers: []types.Container{
			{Name: "nginx", Image: "nginx:latest", Status: types.ContainerWaiting},
		},
		Status:    types.PodScheduled,
		CreatedAt: time.Now(),
	}
	_ = store.AddPod(pod)

	t.Run(
		"Update pod status", func(t *testing.T) {
			payload := map[string]interface{}{
				"status":  "running",
				"message": "All containers started",
				"reason":  "Started",
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/pods/pod-123/status", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues("pod-123")

			err := server.UpdatePodStatus(c)
			if err != nil {
				t.Fatalf("UpdatePodStatus failed: %v", err)
			}

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			updated, _ := store.GetPod("pod-123")
			if updated.Status != types.PodRunning {
				t.Errorf("expected status running, got %s", updated.Status)
			}
			if updated.StartedAt == nil {
				t.Error("expected StartedAt to be set")
			}
		},
	)

	t.Run(
		"Update pod with container statuses", func(t *testing.T) {
			payload := map[string]interface{}{
				"status": "running",
				"containers": []map[string]interface{}{
					{
						"name":        "nginx",
						"image":       "nginx:latest",
						"status":      "running",
						"containerId": "container-abc123",
					},
				},
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/pods/pod-123/status", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues("pod-123")

			_ = server.UpdatePodStatus(c)

			updated, _ := store.GetPod("pod-123")
			if updated.Containers[0].ContainerID != "container-abc123" {
				t.Error("expected ContainerID to be updated")
			}
		},
	)
}

func TestDeletePod(t *testing.T) {
	e := echo.New()
	store := state.NewInMemoryStore()
	sched := scheduler.NewRoundRobin()
	server := NewServer(store, sched)

	pod := types.Pod{
		PodID:     "pod-123",
		Name:      "test-pod",
		Namespace: "default",
		Containers: []types.Container{
			{Name: "nginx", Image: "nginx:latest"},
		},
		Status:    types.PodPending,
		CreatedAt: time.Now(),
	}
	_ = store.AddPod(pod)

	t.Run(
		"Delete existing pod", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/pods/pod-123", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues("pod-123")

			err := server.DeletePod(c)
			if err != nil {
				t.Fatalf("DeletePod failed: %v", err)
			}

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			_, err = store.GetPod("pod-123")
			if !errors.Is(err, state.ErrPodNotFound) {
				t.Error("expected pod to be deleted")
			}
		},
	)

	t.Run(
		"Delete non-existent pod", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/pods/non-existent", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues("non-existent")

			_ = server.DeletePod(c)

			if rec.Code != http.StatusNotFound {
				t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
			}
		},
	)
}

// Helper functions
func generateTestPodID(n int) string {
	return time.Now().Format("20060102150405") + "-pod-" + string(rune('0'+n))
}

func generateTestPodName(n int) string {
	return "test-pod-" + string(rune('0'+n))
}

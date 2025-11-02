package state

import (
	"errors"
	"testing"
	"time"

	"github.com/danpasecinic/podling/internal/types"
)

func TestInMemoryStore_AddPod(t *testing.T) {
	store := NewInMemoryStore()

	pod := types.Pod{
		PodID:     "pod-1",
		Name:      "test-pod",
		Namespace: "default",
		Containers: []types.Container{
			{Name: "nginx", Image: "nginx:latest"},
		},
		Status:    types.PodPending,
		CreatedAt: time.Now(),
	}

	err := store.AddPod(pod)
	if err != nil {
		t.Fatalf("AddPod failed: %v", err)
	}

	err = store.AddPod(pod)
	if !errors.Is(err, ErrPodAlreadyExists) {
		t.Errorf("expected ErrPodAlreadyExists, got %v", err)
	}
}

func TestInMemoryStore_GetPod(t *testing.T) {
	store := NewInMemoryStore()

	pod := types.Pod{
		PodID:     "pod-1",
		Name:      "test-pod",
		Namespace: "default",
		Labels:    map[string]string{"app": "web"},
		Containers: []types.Container{
			{Name: "nginx", Image: "nginx:latest"},
			{Name: "sidecar", Image: "busybox:latest"},
		},
		Status:    types.PodPending,
		CreatedAt: time.Now(),
	}

	if err := store.AddPod(pod); err != nil {
		t.Fatalf("AddPod failed: %v", err)
	}

	retrieved, err := store.GetPod("pod-1")
	if err != nil {
		t.Fatalf("GetPod failed: %v", err)
	}

	if retrieved.PodID != pod.PodID {
		t.Errorf("expected PodID %s, got %s", pod.PodID, retrieved.PodID)
	}
	if retrieved.Name != pod.Name {
		t.Errorf("expected Name %s, got %s", pod.Name, retrieved.Name)
	}
	if len(retrieved.Containers) != 2 {
		t.Errorf("expected 2 containers, got %d", len(retrieved.Containers))
	}
	if retrieved.Labels["app"] != "web" {
		t.Errorf("expected label app=web, got %s", retrieved.Labels["app"])
	}

	_, err = store.GetPod("non-existent")
	if !errors.Is(err, ErrPodNotFound) {
		t.Errorf("expected ErrPodNotFound, got %v", err)
	}
}

func TestInMemoryStore_UpdatePod(t *testing.T) {
	store := NewInMemoryStore()

	pod := types.Pod{
		PodID:     "pod-1",
		Name:      "test-pod",
		Namespace: "default",
		Containers: []types.Container{
			{Name: "nginx", Image: "nginx:latest", Status: types.ContainerWaiting},
		},
		Status:    types.PodPending,
		CreatedAt: time.Now(),
	}

	if err := store.AddPod(pod); err != nil {
		t.Fatalf("AddPod failed: %v", err)
	}

	newStatus := types.PodRunning
	now := time.Now()
	update := PodUpdate{
		Status:    &newStatus,
		StartedAt: &now,
	}

	err := store.UpdatePod("pod-1", update)
	if err != nil {
		t.Fatalf("UpdatePod failed: %v", err)
	}

	updated, _ := store.GetPod("pod-1")
	if updated.Status != types.PodRunning {
		t.Errorf("expected status PodRunning, got %s", updated.Status)
	}
	if updated.StartedAt == nil {
		t.Error("expected StartedAt to be set")
	}

	updatedContainers := []types.Container{
		{
			Name:        "nginx",
			Image:       "nginx:latest",
			Status:      types.ContainerRunning,
			ContainerID: "container-123",
		},
	}
	update = PodUpdate{
		Containers: updatedContainers,
	}

	err = store.UpdatePod("pod-1", update)
	if err != nil {
		t.Fatalf("UpdatePod containers failed: %v", err)
	}

	updated, _ = store.GetPod("pod-1")
	if updated.Containers[0].Status != types.ContainerRunning {
		t.Errorf("expected container status Running, got %s", updated.Containers[0].Status)
	}
	if updated.Containers[0].ContainerID != "container-123" {
		t.Errorf("expected ContainerID container-123, got %s", updated.Containers[0].ContainerID)
	}

	err = store.UpdatePod("non-existent", PodUpdate{Status: &newStatus})
	if !errors.Is(err, ErrPodNotFound) {
		t.Errorf("expected ErrPodNotFound, got %v", err)
	}
}

func TestInMemoryStore_ListPods(t *testing.T) {
	store := NewInMemoryStore()

	for i := 1; i <= 3; i++ {
		pod := types.Pod{
			PodID:     generatePodID(i),
			Name:      generatePodName(i),
			Namespace: "default",
			Containers: []types.Container{
				{Name: "container", Image: "nginx:latest"},
			},
			Status:    types.PodPending,
			CreatedAt: time.Now(),
		}
		if err := store.AddPod(pod); err != nil {
			t.Fatalf("AddPod failed: %v", err)
		}
	}

	pods, err := store.ListPods()
	if err != nil {
		t.Fatalf("ListPods failed: %v", err)
	}

	if len(pods) != 3 {
		t.Errorf("expected 3 pods, got %d", len(pods))
	}
}

func TestInMemoryStore_DeletePod(t *testing.T) {
	store := NewInMemoryStore()

	pod := types.Pod{
		PodID:     "pod-1",
		Name:      "test-pod",
		Namespace: "default",
		Containers: []types.Container{
			{Name: "nginx", Image: "nginx:latest"},
		},
		Status:    types.PodPending,
		CreatedAt: time.Now(),
	}

	if err := store.AddPod(pod); err != nil {
		t.Fatalf("AddPod failed: %v", err)
	}

	err := store.DeletePod("pod-1")
	if err != nil {
		t.Fatalf("DeletePod failed: %v", err)
	}

	_, err = store.GetPod("pod-1")
	if !errors.Is(err, ErrPodNotFound) {
		t.Errorf("expected ErrPodNotFound after delete, got %v", err)
	}

	err = store.DeletePod("non-existent")
	if !errors.Is(err, ErrPodNotFound) {
		t.Errorf("expected ErrPodNotFound, got %v", err)
	}
}

func TestInMemoryStore_ConcurrentPodOperations(t *testing.T) {
	store := NewInMemoryStore()

	pod := types.Pod{
		PodID:     "pod-1",
		Name:      "test-pod",
		Namespace: "default",
		Containers: []types.Container{
			{Name: "nginx", Image: "nginx:latest"},
		},
		Status:    types.PodPending,
		CreatedAt: time.Now(),
	}

	if err := store.AddPod(pod); err != nil {
		t.Fatalf("AddPod failed: %v", err)
	}

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			_, err := store.GetPod("pod-1")
			if err != nil {
				t.Errorf("GetPod failed: %v", err)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		go func(n int) {
			status := types.PodRunning
			update := PodUpdate{
				Status: &status,
			}
			err := store.UpdatePod("pod-1", update)
			if err != nil {
				t.Errorf("UpdatePod failed: %v", err)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 20; i++ {
		<-done
	}

	final, err := store.GetPod("pod-1")
	if err != nil {
		t.Fatalf("GetPod failed: %v", err)
	}
	if final.Status != types.PodRunning {
		t.Errorf("expected status PodRunning, got %s", final.Status)
	}
}

func TestPodHelperMethods(t *testing.T) {
	t.Run(
		"IsPodTerminal", func(t *testing.T) {
			pod := types.Pod{Status: types.PodSucceeded}
			if !pod.IsPodTerminal() {
				t.Error("expected PodSucceeded to be terminal")
			}

			pod.Status = types.PodFailed
			if !pod.IsPodTerminal() {
				t.Error("expected PodFailed to be terminal")
			}

			pod.Status = types.PodRunning
			if pod.IsPodTerminal() {
				t.Error("expected PodRunning to not be terminal")
			}
		},
	)

	t.Run(
		"IsAllContainersRunning", func(t *testing.T) {
			pod := types.Pod{
				Containers: []types.Container{
					{Status: types.ContainerRunning},
					{Status: types.ContainerRunning},
				},
			}
			if !pod.IsAllContainersRunning() {
				t.Error("expected all containers running")
			}

			pod.Containers[1].Status = types.ContainerWaiting
			if pod.IsAllContainersRunning() {
				t.Error("expected not all containers running")
			}

			pod.Containers = []types.Container{}
			if pod.IsAllContainersRunning() {
				t.Error("expected empty pod to not have all containers running")
			}
		},
	)

	t.Run(
		"IsAnyContainerFailed", func(t *testing.T) {
			exitCode := 1
			pod := types.Pod{
				Containers: []types.Container{
					{Status: types.ContainerTerminated, ExitCode: &exitCode},
				},
			}
			if !pod.IsAnyContainerFailed() {
				t.Error("expected container to be failed")
			}

			exitCode = 0
			pod.Containers[0].ExitCode = &exitCode
			if pod.IsAnyContainerFailed() {
				t.Error("expected no containers failed")
			}
		},
	)

	t.Run(
		"GetContainerByName", func(t *testing.T) {
			pod := types.Pod{
				Containers: []types.Container{
					{Name: "nginx", Image: "nginx:latest"},
					{Name: "sidecar", Image: "busybox:latest"},
				},
			}

			container := pod.GetContainerByName("nginx")
			if container == nil {
				t.Fatal("expected to find nginx container")
			}
			if container.Image != "nginx:latest" {
				t.Errorf("expected image nginx:latest, got %s", container.Image)
			}

			container = pod.GetContainerByName("non-existent")
			if container != nil {
				t.Error("expected nil for non-existent container")
			}
		},
	)
}

// Helper functions
func generatePodID(n int) string {
	return time.Now().Format("20060102150405") + "-pod-" + string(rune('0'+n))
}

func generatePodName(n int) string {
	return "test-pod-" + string(rune('0'+n))
}

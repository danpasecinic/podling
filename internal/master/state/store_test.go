package state

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/danpasecinic/podling/internal/types"
)

func TestAddAndGetTask(t *testing.T) {
	store := NewInMemoryStore()

	task := types.Task{
		TaskID:    "task-1",
		Name:      "test-task",
		Image:     "nginx:alpine",
		Status:    types.TaskPending,
		CreatedAt: time.Now(),
	}

	// Test adding task
	err := store.AddTask(task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	// Test retrieving task
	retrieved, err := store.GetTask("task-1")
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	if retrieved.TaskID != task.TaskID {
		t.Errorf("Expected task ID %s, got %s", task.TaskID, retrieved.TaskID)
	}
	if retrieved.Name != task.Name {
		t.Errorf("Expected name %s, got %s", task.Name, retrieved.Name)
	}
	if retrieved.Status != task.Status {
		t.Errorf("Expected status %s, got %s", task.Status, retrieved.Status)
	}
}

func TestAddDuplicateTask(t *testing.T) {
	store := NewInMemoryStore()

	task := types.Task{
		TaskID:    "task-1",
		Name:      "test-task",
		Image:     "nginx:alpine",
		Status:    types.TaskPending,
		CreatedAt: time.Now(),
	}

	err := store.AddTask(task)
	if err != nil {
		t.Fatalf("Failed to add task first time: %v", err)
	}

	// Try to add the same task again
	err = store.AddTask(task)
	if err != ErrTaskAlreadyExists {
		t.Errorf("Expected ErrTaskAlreadyExists, got %v", err)
	}
}

func TestGetNonexistentTask(t *testing.T) {
	store := NewInMemoryStore()

	_, err := store.GetTask("nonexistent")
	if err != ErrTaskNotFound {
		t.Errorf("Expected ErrTaskNotFound, got %v", err)
	}
}

func TestUpdateTask(t *testing.T) {
	store := NewInMemoryStore()

	task := types.Task{
		TaskID:    "task-1",
		Name:      "test-task",
		Image:     "nginx:alpine",
		Status:    types.TaskPending,
		CreatedAt: time.Now(),
	}

	err := store.AddTask(task)
	if err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}

	// Update task status
	newStatus := types.TaskRunning
	nodeID := "worker-1"
	startTime := time.Now()

	updates := TaskUpdate{
		Status:    &newStatus,
		NodeID:    &nodeID,
		StartedAt: &startTime,
	}

	err = store.UpdateTask("task-1", updates)
	if err != nil {
		t.Fatalf("Failed to update task: %v", err)
	}

	// Verify updates
	updated, err := store.GetTask("task-1")
	if err != nil {
		t.Fatalf("Failed to get updated task: %v", err)
	}

	if updated.Status != types.TaskRunning {
		t.Errorf("Expected status %s, got %s", types.TaskRunning, updated.Status)
	}
	if updated.NodeID != nodeID {
		t.Errorf("Expected node ID %s, got %s", nodeID, updated.NodeID)
	}
	if updated.StartedAt == nil {
		t.Error("Expected StartedAt to be set")
	}
}

func TestUpdateNonexistentTask(t *testing.T) {
	store := NewInMemoryStore()

	newStatus := types.TaskRunning
	updates := TaskUpdate{
		Status: &newStatus,
	}

	err := store.UpdateTask("nonexistent", updates)
	if err != ErrTaskNotFound {
		t.Errorf("Expected ErrTaskNotFound, got %v", err)
	}
}

func TestListTasks(t *testing.T) {
	store := NewInMemoryStore()

	tasks := []types.Task{
		{
			TaskID:    "task-1",
			Name:      "task-1",
			Image:     "nginx:alpine",
			Status:    types.TaskPending,
			CreatedAt: time.Now(),
		},
		{
			TaskID:    "task-2",
			Name:      "task-2",
			Image:     "redis:latest",
			Status:    types.TaskRunning,
			CreatedAt: time.Now(),
		},
	}

	for _, task := range tasks {
		err := store.AddTask(task)
		if err != nil {
			t.Fatalf("Failed to add task %s: %v", task.TaskID, err)
		}
	}

	retrieved, err := store.ListTasks()
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}

	if len(retrieved) != len(tasks) {
		t.Errorf("Expected %d tasks, got %d", len(tasks), len(retrieved))
	}
}

func TestAddAndGetNode(t *testing.T) {
	store := NewInMemoryStore()

	node := types.Node{
		NodeID:        "worker-1",
		Hostname:      "192.168.1.100",
		Port:          8081,
		Status:        types.NodeOnline,
		Capacity:      10,
		RunningTasks:  0,
		LastHeartbeat: time.Now(),
	}

	err := store.AddNode(node)
	if err != nil {
		t.Fatalf("Failed to add node: %v", err)
	}

	retrieved, err := store.GetNode("worker-1")
	if err != nil {
		t.Fatalf("Failed to get node: %v", err)
	}

	if retrieved.NodeID != node.NodeID {
		t.Errorf("Expected node ID %s, got %s", node.NodeID, retrieved.NodeID)
	}
	if retrieved.Hostname != node.Hostname {
		t.Errorf("Expected hostname %s, got %s", node.Hostname, retrieved.Hostname)
	}
	if retrieved.Status != node.Status {
		t.Errorf("Expected status %s, got %s", node.Status, retrieved.Status)
	}
}

func TestAddDuplicateNode(t *testing.T) {
	store := NewInMemoryStore()

	node := types.Node{
		NodeID:        "worker-1",
		Hostname:      "192.168.1.100",
		Port:          8081,
		Status:        types.NodeOnline,
		Capacity:      10,
		LastHeartbeat: time.Now(),
	}

	err := store.AddNode(node)
	if err != nil {
		t.Fatalf("Failed to add node first time: %v", err)
	}

	err = store.AddNode(node)
	if err != ErrNodeAlreadyExists {
		t.Errorf("Expected ErrNodeAlreadyExists, got %v", err)
	}
}

func TestGetNonexistentNode(t *testing.T) {
	store := NewInMemoryStore()

	_, err := store.GetNode("nonexistent")
	if err != ErrNodeNotFound {
		t.Errorf("Expected ErrNodeNotFound, got %v", err)
	}
}

func TestUpdateNode(t *testing.T) {
	store := NewInMemoryStore()

	node := types.Node{
		NodeID:        "worker-1",
		Hostname:      "192.168.1.100",
		Port:          8081,
		Status:        types.NodeOnline,
		Capacity:      10,
		RunningTasks:  0,
		LastHeartbeat: time.Now(),
	}

	err := store.AddNode(node)
	if err != nil {
		t.Fatalf("Failed to add node: %v", err)
	}

	// Update node
	runningTasks := 5
	heartbeat := time.Now()
	updates := NodeUpdate{
		RunningTasks:  &runningTasks,
		LastHeartbeat: &heartbeat,
	}

	err = store.UpdateNode("worker-1", updates)
	if err != nil {
		t.Fatalf("Failed to update node: %v", err)
	}

	// Verify updates
	updated, err := store.GetNode("worker-1")
	if err != nil {
		t.Fatalf("Failed to get updated node: %v", err)
	}

	if updated.RunningTasks != runningTasks {
		t.Errorf("Expected running tasks %d, got %d", runningTasks, updated.RunningTasks)
	}
}

func TestUpdateNonexistentNode(t *testing.T) {
	store := NewInMemoryStore()

	runningTasks := 5
	updates := NodeUpdate{
		RunningTasks: &runningTasks,
	}

	err := store.UpdateNode("nonexistent", updates)
	if err != ErrNodeNotFound {
		t.Errorf("Expected ErrNodeNotFound, got %v", err)
	}
}

func TestListNodes(t *testing.T) {
	store := NewInMemoryStore()

	nodes := []types.Node{
		{
			NodeID:        "worker-1",
			Hostname:      "192.168.1.100",
			Port:          8081,
			Status:        types.NodeOnline,
			Capacity:      10,
			LastHeartbeat: time.Now(),
		},
		{
			NodeID:        "worker-2",
			Hostname:      "192.168.1.101",
			Port:          8082,
			Status:        types.NodeOnline,
			Capacity:      10,
			LastHeartbeat: time.Now(),
		},
	}

	for _, node := range nodes {
		err := store.AddNode(node)
		if err != nil {
			t.Fatalf("Failed to add node %s: %v", node.NodeID, err)
		}
	}

	retrieved, err := store.ListNodes()
	if err != nil {
		t.Fatalf("Failed to list nodes: %v", err)
	}

	if len(retrieved) != len(nodes) {
		t.Errorf("Expected %d nodes, got %d", len(nodes), len(retrieved))
	}
}

func TestGetAvailableNodes(t *testing.T) {
	store := NewInMemoryStore()

	nodes := []types.Node{
		{
			NodeID:        "worker-1",
			Hostname:      "192.168.1.100",
			Port:          8081,
			Status:        types.NodeOnline,
			Capacity:      10,
			LastHeartbeat: time.Now(),
		},
		{
			NodeID:        "worker-2",
			Hostname:      "192.168.1.101",
			Port:          8082,
			Status:        types.NodeOffline,
			Capacity:      10,
			LastHeartbeat: time.Now().Add(-2 * time.Minute),
		},
		{
			NodeID:        "worker-3",
			Hostname:      "192.168.1.102",
			Port:          8083,
			Status:        types.NodeOnline,
			Capacity:      10,
			LastHeartbeat: time.Now(),
		},
	}

	for _, node := range nodes {
		err := store.AddNode(node)
		if err != nil {
			t.Fatalf("Failed to add node %s: %v", node.NodeID, err)
		}
	}

	available, err := store.GetAvailableNodes()
	if err != nil {
		t.Fatalf("Failed to get available nodes: %v", err)
	}

	// Should only have 2 online nodes (worker-1 and worker-3)
	if len(available) != 2 {
		t.Errorf("Expected 2 available nodes, got %d", len(available))
	}

	// Verify that only online nodes are returned
	for _, node := range available {
		if node.Status != types.NodeOnline {
			t.Errorf("Expected only online nodes, got node %s with status %s", node.NodeID, node.Status)
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	store := NewInMemoryStore()

	// Test concurrent task additions
	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			task := types.Task{
				TaskID:    fmt.Sprintf("task-%d", id),
				Name:      "concurrent-task",
				Image:     "nginx:alpine",
				Status:    types.TaskPending,
				CreatedAt: time.Now(),
			}

			err := store.AddTask(task)
			if err != nil && err != ErrTaskAlreadyExists {
				t.Errorf("Unexpected error adding task: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Test concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, err := store.ListTasks()
			if err != nil {
				t.Errorf("Error listing tasks: %v", err)
			}
		}()
	}

	wg.Wait()
}

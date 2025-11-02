package state

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/danpasecinic/podling/internal/types"
)

// getTestPostgresStore creates a test PostgreSQL store
// Skips the test if TEST_DATABASE_URL is not set
func getTestPostgresStore(t *testing.T) *PostgresStore {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping PostgreSQL tests")
	}

	store, err := NewPostgresStore(dbURL)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}

	_, _ = store.db.Exec("DELETE FROM tasks")
	_, _ = store.db.Exec("DELETE FROM nodes")

	t.Cleanup(
		func() {
			_, _ = store.db.Exec("DELETE FROM tasks")
			_, _ = store.db.Exec("DELETE FROM nodes")
			_ = store.Close()
		},
	)

	return store
}

func TestPostgresStore_AddTask(t *testing.T) {
	store := getTestPostgresStore(t)

	task := types.Task{
		TaskID:    "test-task-1",
		Name:      "test-task",
		Image:     "nginx:latest",
		Env:       map[string]string{"PORT": "8080"},
		Status:    types.TaskPending,
		CreatedAt: time.Now(),
	}

	err := store.AddTask(task)
	if err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	// Try to add duplicate
	err = store.AddTask(task)
	if !errors.Is(err, ErrTaskAlreadyExists) {
		t.Errorf("expected ErrTaskAlreadyExists, got: %v", err)
	}
}

func TestPostgresStore_GetTask(t *testing.T) {
	store := getTestPostgresStore(t)

	task := types.Task{
		TaskID:    "test-task-2",
		Name:      "test-task",
		Image:     "redis:alpine",
		Env:       map[string]string{"MODE": "standalone"},
		Status:    types.TaskRunning,
		NodeID:    "node-1",
		CreatedAt: time.Now(),
	}

	err := store.AddTask(task)
	if err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	retrieved, err := store.GetTask("test-task-2")
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}

	if retrieved.TaskID != task.TaskID {
		t.Errorf("expected TaskID %s, got %s", task.TaskID, retrieved.TaskID)
	}
	if retrieved.Name != task.Name {
		t.Errorf("expected Name %s, got %s", task.Name, retrieved.Name)
	}
	if retrieved.Image != task.Image {
		t.Errorf("expected Image %s, got %s", task.Image, retrieved.Image)
	}
	if retrieved.Status != task.Status {
		t.Errorf("expected Status %s, got %s", task.Status, retrieved.Status)
	}
	if retrieved.Env["MODE"] != "standalone" {
		t.Errorf("expected Env MODE=standalone, got %s", retrieved.Env["MODE"])
	}

	// Test non-existent task
	_, err = store.GetTask("non-existent")
	if err != ErrTaskNotFound {
		t.Errorf("expected ErrTaskNotFound, got: %v", err)
	}
}

func TestPostgresStore_UpdateTask(t *testing.T) {
	store := getTestPostgresStore(t)

	task := types.Task{
		TaskID:    "test-task-3",
		Name:      "test-task",
		Image:     "nginx:latest",
		Status:    types.TaskPending,
		CreatedAt: time.Now(),
	}

	err := store.AddTask(task)
	if err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	newStatus := types.TaskRunning
	containerID := "container-123"
	err = store.UpdateTask(
		"test-task-3", TaskUpdate{
			Status:      &newStatus,
			ContainerID: &containerID,
		},
	)
	if err != nil {
		t.Fatalf("failed to update task: %v", err)
	}

	updated, err := store.GetTask("test-task-3")
	if err != nil {
		t.Fatalf("failed to get updated task: %v", err)
	}

	if updated.Status != types.TaskRunning {
		t.Errorf("expected Status running, got %s", updated.Status)
	}
	if updated.ContainerID != "container-123" {
		t.Errorf("expected ContainerID container-123, got %s", updated.ContainerID)
	}
}

func TestPostgresStore_ListTasks(t *testing.T) {
	store := getTestPostgresStore(t)

	for i := 1; i <= 3; i++ {
		task := types.Task{
			TaskID:    "task-" + string(rune('0'+i)),
			Name:      "test-task",
			Image:     "nginx:latest",
			Status:    types.TaskPending,
			CreatedAt: time.Now(),
		}
		err := store.AddTask(task)
		if err != nil {
			t.Fatalf("failed to add task %d: %v", i, err)
		}
	}

	tasks, err := store.ListTasks()
	if err != nil {
		t.Fatalf("failed to list tasks: %v", err)
	}

	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tasks))
	}
}

func TestPostgresStore_AddNode(t *testing.T) {
	store := getTestPostgresStore(t)

	node := types.Node{
		NodeID:        "node-1",
		Hostname:      "worker-1",
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

	err := store.AddNode(node)
	if err != nil {
		t.Fatalf("failed to add node: %v", err)
	}

	// Try to add duplicate
	err = store.AddNode(node)
	if !errors.Is(err, ErrNodeAlreadyExists) {
		t.Errorf("expected ErrNodeAlreadyExists, got: %v", err)
	}
}

func TestPostgresStore_GetNode(t *testing.T) {
	store := getTestPostgresStore(t)

	node := types.Node{
		NodeID:        "node-2",
		Hostname:      "worker-2",
		Port:          8082,
		Status:        types.NodeOnline,
		RunningTasks:  2,
		LastHeartbeat: time.Now(),
		Resources: &types.NodeResources{
			Capacity:    types.ResourceList{CPU: 5000, Memory: 5 * 1024 * 1024 * 1024},
			Allocatable: types.ResourceList{CPU: 5000, Memory: 5 * 1024 * 1024 * 1024},
			Used:        types.ResourceList{CPU: 0, Memory: 0},
		},
	}

	err := store.AddNode(node)
	if err != nil {
		t.Fatalf("failed to add node: %v", err)
	}

	retrieved, err := store.GetNode("node-2")
	if err != nil {
		t.Fatalf("failed to get node: %v", err)
	}

	if retrieved.NodeID != node.NodeID {
		t.Errorf("expected NodeID %s, got %s", node.NodeID, retrieved.NodeID)
	}
	if retrieved.Hostname != node.Hostname {
		t.Errorf("expected Hostname %s, got %s", node.Hostname, retrieved.Hostname)
	}
	if retrieved.Port != node.Port {
		t.Errorf("expected Port %d, got %d", node.Port, retrieved.Port)
	}
	if retrieved.Resources == nil {
		t.Error("expected Resources to be set")
	} else {
		if retrieved.Resources.Capacity.CPU != node.Resources.Capacity.CPU {
			t.Errorf("expected CPU %d, got %d", node.Resources.Capacity.CPU, retrieved.Resources.Capacity.CPU)
		}
		if retrieved.Resources.Capacity.Memory != node.Resources.Capacity.Memory {
			t.Errorf("expected Memory %d, got %d", node.Resources.Capacity.Memory, retrieved.Resources.Capacity.Memory)
		}
	}

	// Test non-existent node
	_, err = store.GetNode("non-existent")
	if !errors.Is(err, ErrNodeNotFound) {
		t.Errorf("expected ErrNodeNotFound, got: %v", err)
	}
}

func TestPostgresStore_UpdateNode(t *testing.T) {
	store := getTestPostgresStore(t)

	node := types.Node{
		NodeID:        "node-3",
		Hostname:      "worker-3",
		Port:          8083,
		Status:        types.NodeOnline,
		RunningTasks:  0,
		LastHeartbeat: time.Now(),
		Resources: &types.NodeResources{
			Capacity:    types.ResourceList{CPU: 10000, Memory: 10 * 1024 * 1024 * 1024},
			Allocatable: types.ResourceList{CPU: 10000, Memory: 10 * 1024 * 1024 * 1024},
			Used:        types.ResourceList{CPU: 0, Memory: 0},
		},
	}

	err := store.AddNode(node)
	if err != nil {
		t.Fatalf("failed to add node: %v", err)
	}

	newTasks := 5
	newHeartbeat := time.Now()
	err = store.UpdateNode(
		"node-3", NodeUpdate{
			RunningTasks:  &newTasks,
			LastHeartbeat: &newHeartbeat,
		},
	)
	if err != nil {
		t.Fatalf("failed to update node: %v", err)
	}

	updated, err := store.GetNode("node-3")
	if err != nil {
		t.Fatalf("failed to get updated node: %v", err)
	}

	if updated.RunningTasks != 5 {
		t.Errorf("expected RunningTasks 5, got %d", updated.RunningTasks)
	}
}

func TestPostgresStore_GetAvailableNodes(t *testing.T) {
	store := getTestPostgresStore(t)

	node1 := types.Node{
		NodeID:        "node-avail-1",
		Hostname:      "worker-1",
		Port:          8081,
		Status:        types.NodeOnline,
		RunningTasks:  5,
		LastHeartbeat: time.Now(),
		Resources: &types.NodeResources{
			Capacity:    types.ResourceList{CPU: 10000, Memory: 10 * 1024 * 1024 * 1024},
			Allocatable: types.ResourceList{CPU: 10000, Memory: 10 * 1024 * 1024 * 1024},
			Used:        types.ResourceList{CPU: 0, Memory: 0},
		},
	}
	err := store.AddNode(node1)
	if err != nil {
		t.Fatalf("failed to add node 1: %v", err)
	}

	node2 := types.Node{
		NodeID:        "node-avail-2",
		Hostname:      "worker-2",
		Port:          8082,
		Status:        types.NodeOffline,
		RunningTasks:  0,
		LastHeartbeat: time.Now(),
		Resources: &types.NodeResources{
			Capacity:    types.ResourceList{CPU: 10000, Memory: 10 * 1024 * 1024 * 1024},
			Allocatable: types.ResourceList{CPU: 10000, Memory: 10 * 1024 * 1024 * 1024},
			Used:        types.ResourceList{CPU: 0, Memory: 0},
		},
	}
	err = store.AddNode(node2)
	if err != nil {
		t.Fatalf("failed to add node 2: %v", err)
	}

	node3 := types.Node{
		NodeID:        "node-avail-3",
		Hostname:      "worker-3",
		Port:          8083,
		Status:        types.NodeOnline,
		RunningTasks:  5,
		LastHeartbeat: time.Now(),
		Resources: &types.NodeResources{
			Capacity:    types.ResourceList{CPU: 5000, Memory: 5 * 1024 * 1024 * 1024},
			Allocatable: types.ResourceList{CPU: 5000, Memory: 5 * 1024 * 1024 * 1024},
			Used:        types.ResourceList{CPU: 0, Memory: 0},
		},
	}
	err = store.AddNode(node3)
	if err != nil {
		t.Fatalf("failed to add node 3: %v", err)
	}

	available, err := store.GetAvailableNodes()
	if err != nil {
		t.Fatalf("failed to get available nodes: %v", err)
	}

	if len(available) != 1 {
		t.Errorf("expected 1 available node, got %d", len(available))
	}
	if len(available) > 0 && available[0].NodeID != "node-avail-1" {
		t.Errorf("expected available node to be node-avail-1, got %s", available[0].NodeID)
	}
}

func TestPostgresStore_ListNodes(t *testing.T) {
	store := getTestPostgresStore(t)

	nodes := []types.Node{
		{
			NodeID:        "node-list-1",
			Hostname:      "worker-1",
			Port:          8081,
			Status:        types.NodeOnline,
			RunningTasks:  2,
			LastHeartbeat: time.Now(),
			Resources: &types.NodeResources{
				Capacity:    types.ResourceList{CPU: 10000, Memory: 10 * 1024 * 1024 * 1024},
				Allocatable: types.ResourceList{CPU: 10000, Memory: 10 * 1024 * 1024 * 1024},
				Used:        types.ResourceList{CPU: 0, Memory: 0},
			},
		},
		{
			NodeID:        "node-list-2",
			Hostname:      "worker-2",
			Port:          8082,
			Status:        types.NodeOffline,
			RunningTasks:  0,
			LastHeartbeat: time.Now().Add(-1 * time.Hour),
			Resources: &types.NodeResources{
				Capacity:    types.ResourceList{CPU: 5000, Memory: 5 * 1024 * 1024 * 1024},
				Allocatable: types.ResourceList{CPU: 5000, Memory: 5 * 1024 * 1024 * 1024},
				Used:        types.ResourceList{CPU: 0, Memory: 0},
			},
		},
		{
			NodeID:        "node-list-3",
			Hostname:      "worker-3",
			Port:          8083,
			Status:        types.NodeOnline,
			RunningTasks:  8,
			LastHeartbeat: time.Now(),
			Resources: &types.NodeResources{
				Capacity:    types.ResourceList{CPU: 15000, Memory: 15 * 1024 * 1024 * 1024},
				Allocatable: types.ResourceList{CPU: 15000, Memory: 15 * 1024 * 1024 * 1024},
				Used:        types.ResourceList{CPU: 0, Memory: 0},
			},
		},
	}

	for _, node := range nodes {
		err := store.AddNode(node)
		if err != nil {
			t.Fatalf("failed to add node %s: %v", node.NodeID, err)
		}
	}

	allNodes, err := store.ListNodes()
	if err != nil {
		t.Fatalf("failed to list nodes: %v", err)
	}

	if len(allNodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(allNodes))
	}

	// Verify nodes are ordered by last_heartbeat DESC (most recent first)
	if len(allNodes) >= 2 {
		for i := 0; i < 2; i++ {
			if allNodes[i].NodeID != "node-list-1" && allNodes[i].NodeID != "node-list-3" {
				t.Errorf("expected recent nodes first, got %s at position %d", allNodes[i].NodeID, i)
			}
		}
	}
}

func TestPostgresStore_UpdateTask_AllFields(t *testing.T) {
	store := getTestPostgresStore(t)

	task := types.Task{
		TaskID:    "test-task-update-all",
		Name:      "test-task",
		Image:     "nginx:latest",
		Status:    types.TaskPending,
		CreatedAt: time.Now(),
	}

	err := store.AddTask(task)
	if err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	newStatus := types.TaskRunning
	newNodeID := "node-123"
	newContainerID := "container-456"
	startedAt := time.Now()
	errorMsg := "test error"

	err = store.UpdateTask(
		"test-task-update-all", TaskUpdate{
			Status:      &newStatus,
			NodeID:      &newNodeID,
			ContainerID: &newContainerID,
			StartedAt:   &startedAt,
			Error:       &errorMsg,
		},
	)
	if err != nil {
		t.Fatalf("failed to update task: %v", err)
	}

	updated, err := store.GetTask("test-task-update-all")
	if err != nil {
		t.Fatalf("failed to get updated task: %v", err)
	}

	if updated.Status != newStatus {
		t.Errorf("expected status %s, got %s", newStatus, updated.Status)
	}
	if updated.NodeID != newNodeID {
		t.Errorf("expected node ID %s, got %s", newNodeID, updated.NodeID)
	}
	if updated.ContainerID != newContainerID {
		t.Errorf("expected container ID %s, got %s", newContainerID, updated.ContainerID)
	}
	if updated.Error != errorMsg {
		t.Errorf("expected error %s, got %s", errorMsg, updated.Error)
	}
	if updated.StartedAt == nil {
		t.Error("expected StartedAt to be set")
	}
}

func TestPostgresStore_UpdateNode_AllFields(t *testing.T) {
	store := getTestPostgresStore(t)

	node := types.Node{
		NodeID:        "node-update-all",
		Hostname:      "worker-1",
		Port:          8081,
		Status:        types.NodeOnline,
		RunningTasks:  0,
		LastHeartbeat: time.Now().Add(-1 * time.Hour),
		Resources: &types.NodeResources{
			Capacity:    types.ResourceList{CPU: 10000, Memory: 10 * 1024 * 1024 * 1024},
			Allocatable: types.ResourceList{CPU: 10000, Memory: 10 * 1024 * 1024 * 1024},
			Used:        types.ResourceList{CPU: 0, Memory: 0},
		},
	}

	err := store.AddNode(node)
	if err != nil {
		t.Fatalf("failed to add node: %v", err)
	}

	newStatus := types.NodeOffline
	newRunningTasks := 7
	newHeartbeat := time.Now()

	err = store.UpdateNode(
		"node-update-all", NodeUpdate{
			Status:        &newStatus,
			RunningTasks:  &newRunningTasks,
			LastHeartbeat: &newHeartbeat,
		},
	)
	if err != nil {
		t.Fatalf("failed to update node: %v", err)
	}

	updated, err := store.GetNode("node-update-all")
	if err != nil {
		t.Fatalf("failed to get updated node: %v", err)
	}

	if updated.Status != newStatus {
		t.Errorf("expected status %s, got %s", newStatus, updated.Status)
	}
	if updated.RunningTasks != newRunningTasks {
		t.Errorf("expected running tasks %d, got %d", newRunningTasks, updated.RunningTasks)
	}
	// Heartbeat should be updated (within 1 second tolerance)
	if updated.LastHeartbeat.Before(time.Now().Add(-1 * time.Second)) {
		t.Error("expected LastHeartbeat to be updated")
	}
}

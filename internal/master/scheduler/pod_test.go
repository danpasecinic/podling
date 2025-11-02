package scheduler

import (
	"errors"
	"testing"
	"time"

	"github.com/danpasecinic/podling/internal/types"
)

func TestRoundRobin_SelectNodeForPod(t *testing.T) {
	scheduler := NewRoundRobin()

	nodes := []types.Node{
		newTestNode("node-1", types.NodeOnline, 2),
		newTestNode("node-2", types.NodeOnline, 5),
		newTestNode("node-3", types.NodeOnline, 1),
	}

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

	t.Run(
		"Round-robin distribution", func(t *testing.T) {
			selections := make(map[string]int)

			for i := 0; i < 10; i++ {
				node, err := scheduler.SelectNodeForPod(pod, nodes)
				if err != nil {
					t.Fatalf("SelectNodeForPod failed: %v", err)
				}
				selections[node.NodeID]++
			}

			// Each node should be selected at least once in round-robin
			if len(selections) != 3 {
				t.Errorf("expected all 3 nodes to be selected, got %d", len(selections))
			}

			// Distribution should be relatively even (3-4 selections each for 10 total)
			for nodeID, count := range selections {
				if count < 2 || count > 5 {
					t.Errorf("node %s selected %d times, expected 2-5", nodeID, count)
				}
			}
		},
	)

	t.Run(
		"Multi-container pod", func(t *testing.T) {
			multiPod := types.Pod{
				PodID:     "multi-pod",
				Name:      "multi-container-pod",
				Namespace: "default",
				Containers: []types.Container{
					{Name: "app", Image: "myapp:1.0"},
					{Name: "sidecar", Image: "nginx:latest"},
				},
				Status:    types.PodPending,
				CreatedAt: time.Now(),
			}

			node, err := scheduler.SelectNodeForPod(multiPod, nodes)
			if err != nil {
				t.Fatalf("SelectNodeForPod failed for multi-container pod: %v", err)
			}

			if node == nil {
				t.Error("expected node to be selected")
			}
		},
	)

	t.Run(
		"No available nodes", func(t *testing.T) {
			_, err := scheduler.SelectNodeForPod(pod, []types.Node{})
			if !errors.Is(err, ErrNoAvailableNodes) {
				t.Errorf("expected ErrNoAvailableNodes, got %v", err)
			}
		},
	)

	t.Run(
		"Only offline nodes", func(t *testing.T) {
			offlineNodes := []types.Node{
				newTestNode("node-1", types.NodeOffline, 0),
			}

			_, err := scheduler.SelectNodeForPod(pod, offlineNodes)
			if !errors.Is(err, ErrNoAvailableNodes) {
				t.Errorf("expected ErrNoAvailableNodes for offline nodes, got %v", err)
			}
		},
	)

	t.Run(
		"Node at full capacity", func(t *testing.T) {
			fullNodes := []types.Node{
				newTestNode("node-1", types.NodeOnline, 10),
			}

			_, err := scheduler.SelectNodeForPod(pod, fullNodes)
			if !errors.Is(err, ErrNoAvailableNodes) {
				t.Errorf("expected ErrNoAvailableNodes for full node, got %v", err)
			}
		},
	)

	t.Run(
		"Mixed available and unavailable nodes", func(t *testing.T) {
			mixedNodes := []types.Node{
				newTestNode("node-1", types.NodeOffline, 0),
				newTestNode("node-2", types.NodeOnline, 5),
				newTestNode("node-3", types.NodeOnline, 2),
			}

			node, err := scheduler.SelectNodeForPod(pod, mixedNodes)
			if err != nil {
				t.Fatalf("SelectNodeForPod failed: %v", err)
			}

			// Should select one of the available nodes (node-2 or node-3)
			if node.NodeID != "node-2" && node.NodeID != "node-3" {
				t.Errorf("expected node-2 or node-3 to be selected, got %s", node.NodeID)
			}
		},
	)
}

func TestRoundRobin_PodAndTaskScheduling(t *testing.T) {
	scheduler := NewRoundRobin()

	nodes := []types.Node{
		newTestNode("node-1", types.NodeOnline, 0),
		newTestNode("node-2", types.NodeOnline, 0),
	}

	task := types.Task{
		TaskID:    "task-1",
		Name:      "test-task",
		Image:     "nginx:latest",
		Status:    types.TaskPending,
		CreatedAt: time.Now(),
	}

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

	taskNode, err := scheduler.SelectNode(task, nodes)
	if err != nil {
		t.Fatalf("SelectNode failed: %v", err)
	}

	podNode, err := scheduler.SelectNodeForPod(pod, nodes)
	if err != nil {
		t.Fatalf("SelectNodeForPod failed: %v", err)
	}

	// Round-robin should alternate between nodes
	if taskNode.NodeID == podNode.NodeID {
		t.Error("expected round-robin to select different nodes")
	}
}

func TestRoundRobin_Concurrent(t *testing.T) {
	scheduler := NewRoundRobin()

	nodes := []types.Node{
		newTestNode("node-1", types.NodeOnline, 0),
		newTestNode("node-2", types.NodeOnline, 0),
	}

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

	done := make(chan string, 20)

	for i := 0; i < 20; i++ {
		go func() {
			node, err := scheduler.SelectNodeForPod(pod, nodes)
			if err != nil {
				t.Errorf("SelectNodeForPod failed: %v", err)
				done <- ""
				return
			}
			done <- node.NodeID
		}()
	}

	selections := make(map[string]int)
	for i := 0; i < 20; i++ {
		nodeID := <-done
		if nodeID != "" {
			selections[nodeID]++
		}
	}

	if len(selections) != 2 {
		t.Errorf("expected both nodes to be selected, got %d", len(selections))
	}

	// Distribution should be exactly even (10 each for 20 total)
	for nodeID, count := range selections {
		if count != 10 {
			t.Errorf("node %s selected %d times, expected 10", nodeID, count)
		}
	}
}

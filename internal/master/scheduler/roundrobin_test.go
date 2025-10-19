package scheduler

import (
	"errors"
	"testing"
	"time"

	"github.com/danpasecinic/podling/internal/types"
)

func TestRoundRobin_SelectNode(t *testing.T) {
	tests := []struct {
		name      string
		nodes     []types.Node
		task      types.Task
		wantErr   error
		wantNode  string
		callCount int
	}{
		{
			name:    "no nodes",
			nodes:   []types.Node{},
			task:    types.Task{TaskID: "task1"},
			wantErr: ErrNoAvailableNodes,
		},
		{
			name: "no available nodes - all offline",
			nodes: []types.Node{
				{NodeID: "node1", Status: types.NodeOffline, Capacity: 10},
				{NodeID: "node2", Status: types.NodeOffline, Capacity: 10},
			},
			task:    types.Task{TaskID: "task1"},
			wantErr: ErrNoAvailableNodes,
		},
		{
			name: "no available nodes - all at capacity",
			nodes: []types.Node{
				{NodeID: "node1", Status: types.NodeOnline, Capacity: 2, RunningTasks: 2},
				{NodeID: "node2", Status: types.NodeOnline, Capacity: 1, RunningTasks: 1},
			},
			task:    types.Task{TaskID: "task1"},
			wantErr: ErrNoAvailableNodes,
		},
		{
			name: "single available node",
			nodes: []types.Node{
				{NodeID: "node1", Status: types.NodeOnline, Capacity: 10, RunningTasks: 0},
			},
			task:      types.Task{TaskID: "task1"},
			wantErr:   nil,
			wantNode:  "node1",
			callCount: 3,
		},
		{
			name: "multiple available nodes - round robin",
			nodes: []types.Node{
				{NodeID: "node1", Status: types.NodeOnline, Capacity: 10, RunningTasks: 0},
				{NodeID: "node2", Status: types.NodeOnline, Capacity: 10, RunningTasks: 0},
				{NodeID: "node3", Status: types.NodeOnline, Capacity: 10, RunningTasks: 0},
			},
			task:      types.Task{TaskID: "task1"},
			wantErr:   nil,
			callCount: 1,
		},
		{
			name: "mixed online and offline nodes",
			nodes: []types.Node{
				{NodeID: "node1", Status: types.NodeOffline, Capacity: 10, RunningTasks: 0},
				{NodeID: "node2", Status: types.NodeOnline, Capacity: 10, RunningTasks: 5},
				{NodeID: "node3", Status: types.NodeOnline, Capacity: 10, RunningTasks: 0},
			},
			task:      types.Task{TaskID: "task1"},
			wantErr:   nil,
			wantNode:  "node2",
			callCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				rr := NewRoundRobin()

				callsToMake := 1
				if tt.callCount > 0 {
					callsToMake = tt.callCount
				}

				for i := 0; i < callsToMake; i++ {
					node, err := rr.SelectNode(tt.task, tt.nodes)

					if !errors.Is(err, tt.wantErr) {
						t.Errorf("SelectNode() error = %v, wantErr %v", err, tt.wantErr)
						return
					}

					if tt.wantErr != nil {
						return
					}

					if node == nil {
						t.Error("SelectNode() returned nil node when error was nil")
						return
					}

					if tt.wantNode != "" && node.NodeID != tt.wantNode {
						t.Errorf("SelectNode() nodeID = %v, want %v", node.NodeID, tt.wantNode)
					}
				}
			},
		)
	}
}

func TestRoundRobin_RoundRobinOrder(t *testing.T) {
	nodes := []types.Node{
		{NodeID: "node1", Status: types.NodeOnline, Capacity: 10, RunningTasks: 0},
		{NodeID: "node2", Status: types.NodeOnline, Capacity: 10, RunningTasks: 0},
		{NodeID: "node3", Status: types.NodeOnline, Capacity: 10, RunningTasks: 0},
	}

	rr := NewRoundRobin()
	task := types.Task{TaskID: "task1"}

	expectedOrder := []string{"node1", "node2", "node3", "node1", "node2", "node3"}

	for i, expected := range expectedOrder {
		node, err := rr.SelectNode(task, nodes)
		if err != nil {
			t.Fatalf("SelectNode() unexpected error at iteration %d: %v", i, err)
		}

		if node.NodeID != expected {
			t.Errorf("iteration %d: got nodeID = %v, want %v", i, node.NodeID, expected)
		}
	}
}

func TestRoundRobin_ConcurrentSelection(t *testing.T) {
	nodes := []types.Node{
		{NodeID: "node1", Status: types.NodeOnline, Capacity: 10, RunningTasks: 0},
		{NodeID: "node2", Status: types.NodeOnline, Capacity: 10, RunningTasks: 0},
	}

	rr := NewRoundRobin()
	task := types.Task{TaskID: "task1"}

	results := make(chan string, 100)
	done := make(chan bool)

	for i := 0; i < 100; i++ {
		go func() {
			node, err := rr.SelectNode(task, nodes)
			if err == nil {
				results <- node.NodeID
			}
			done <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}
	close(results)

	selectedNodes := make(map[string]int)
	for nodeID := range results {
		selectedNodes[nodeID]++
	}

	if len(selectedNodes) != 2 {
		t.Errorf("Expected both nodes to be selected, got %d unique nodes", len(selectedNodes))
	}

	for nodeID, count := range selectedNodes {
		if count < 40 || count > 60 {
			t.Errorf("Node %s was selected %d times, expected roughly 50", nodeID, count)
		}
	}
}

func TestFilterAvailable(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name  string
		nodes []types.Node
		want  int
	}{
		{
			name:  "empty nodes",
			nodes: []types.Node{},
			want:  0,
		},
		{
			name: "all available",
			nodes: []types.Node{
				{NodeID: "node1", Status: types.NodeOnline, Capacity: 10, RunningTasks: 5, LastHeartbeat: now},
				{NodeID: "node2", Status: types.NodeOnline, Capacity: 5, RunningTasks: 0, LastHeartbeat: now},
			},
			want: 2,
		},
		{
			name: "mixed availability",
			nodes: []types.Node{
				{NodeID: "node1", Status: types.NodeOnline, Capacity: 10, RunningTasks: 5},
				{NodeID: "node2", Status: types.NodeOffline, Capacity: 10, RunningTasks: 0},
				{NodeID: "node3", Status: types.NodeOnline, Capacity: 2, RunningTasks: 2},
				{NodeID: "node4", Status: types.NodeOnline, Capacity: 10, RunningTasks: 0},
			},
			want: 2,
		},
		{
			name: "none available",
			nodes: []types.Node{
				{NodeID: "node1", Status: types.NodeOffline, Capacity: 10, RunningTasks: 0},
				{NodeID: "node2", Status: types.NodeOnline, Capacity: 5, RunningTasks: 5},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got := filterAvailable(tt.nodes)
				if len(got) != tt.want {
					t.Errorf("filterAvailable() returned %d nodes, want %d", len(got), tt.want)
				}

				for _, node := range got {
					if node.Status != types.NodeOnline {
						t.Errorf("filterAvailable() returned offline node: %s", node.NodeID)
					}
					if node.RunningTasks >= node.Capacity {
						t.Errorf("filterAvailable() returned node at capacity: %s", node.NodeID)
					}
				}
			},
		)
	}
}

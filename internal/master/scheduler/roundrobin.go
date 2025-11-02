package scheduler

import (
	"errors"
	"sync"

	"github.com/danpasecinic/podling/internal/types"
)

// ErrNoAvailableNodes is returned when no nodes are available to run tasks.
var ErrNoAvailableNodes = errors.New("no available nodes")

// RoundRobin implements a round-robin scheduling algorithm.
// It distributes tasks evenly across available nodes in a circular manner.
// RoundRobin is safe for concurrent use.
type RoundRobin struct {
	mu       sync.Mutex
	lastUsed int
}

// NewRoundRobin creates a new round-robin scheduler.
func NewRoundRobin() *RoundRobin {
	return &RoundRobin{
		lastUsed: -1,
	}
}

// SelectNode selects the next available node in round-robin order.
// Nodes are filtered to only include those that are online and have capacity.
// Returns ErrNoAvailableNodes if no suitable nodes are found.
func (rr *RoundRobin) SelectNode(_ types.Task, nodes []types.Node) (*types.Node, error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	availableNodes := filterAvailable(nodes)
	if len(availableNodes) == 0 {
		return nil, ErrNoAvailableNodes
	}

	rr.lastUsed = (rr.lastUsed + 1) % len(availableNodes)
	return &availableNodes[rr.lastUsed], nil
}

// SelectNodeForPod selects the next available node for a pod in round-robin order.
// Pods can contain multiple containers, but for now they follow the same scheduling
// logic as tasks. Future improvements could consider pod resource requirements.
// Returns ErrNoAvailableNodes if no suitable nodes are found.
func (rr *RoundRobin) SelectNodeForPod(_ types.Pod, nodes []types.Node) (*types.Node, error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	availableNodes := filterAvailable(nodes)
	if len(availableNodes) == 0 {
		return nil, ErrNoAvailableNodes
	}

	rr.lastUsed = (rr.lastUsed + 1) % len(availableNodes)
	return &availableNodes[rr.lastUsed], nil
}

func filterAvailable(nodes []types.Node) []types.Node {
	available := make([]types.Node, 0)
	for _, node := range nodes {
		if node.Status == types.NodeOnline && node.RunningTasks < node.Capacity {
			available = append(available, node)
		}
	}
	return available
}

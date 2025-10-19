package scheduler

import (
	"errors"
	"sync"

	"github.com/danpasecinic/podling/internal/types"
)

var ErrNoAvailableNodes = errors.New("no available nodes")

type RoundRobin struct {
	mu       sync.Mutex
	lastUsed int
}

func NewRoundRobin() *RoundRobin {
	return &RoundRobin{
		lastUsed: -1,
	}
}

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

func filterAvailable(nodes []types.Node) []types.Node {
	available := make([]types.Node, 0)
	for _, node := range nodes {
		if node.Status == types.NodeOnline && node.RunningTasks < node.Capacity {
			available = append(available, node)
		}
	}
	return available
}

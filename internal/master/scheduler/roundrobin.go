package scheduler

import (
	"errors"
	"sync"

	"github.com/danpasecinic/podling/internal/types"
)

// ErrNoAvailableNodes is returned when no nodes are available to run tasks.
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

func (rr *RoundRobin) SelectNode(task types.Task, nodes []types.Node) (*types.Node, error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	availableNodes := filterAvailableForTask(task, nodes)
	if len(availableNodes) == 0 {
		return nil, ErrNoAvailableNodes
	}

	rr.lastUsed = (rr.lastUsed + 1) % len(availableNodes)
	return &availableNodes[rr.lastUsed], nil
}

func (rr *RoundRobin) SelectNodeForPod(pod types.Pod, nodes []types.Node) (*types.Node, error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	availableNodes := filterAvailableForPod(pod, nodes)
	if len(availableNodes) == 0 {
		return nil, ErrNoAvailableNodes
	}

	rr.lastUsed = (rr.lastUsed + 1) % len(availableNodes)
	return &availableNodes[rr.lastUsed], nil
}

func filterAvailableForTask(task types.Task, nodes []types.Node) []types.Node {
	available := make([]types.Node, 0)
	for _, node := range nodes {
		if node.Status != types.NodeOnline {
			continue
		}

		if node.Resources == nil {
			continue
		}

		maxSlots := node.GetMaxTaskSlots()
		if node.RunningTasks >= maxSlots {
			continue
		}

		if !task.Resources.Requests.IsZero() {
			if !node.Resources.CanFit(task.Resources) {
				continue
			}
		}

		available = append(available, node)
	}
	return available
}

func filterAvailableForPod(pod types.Pod, nodes []types.Node) []types.Node {
	available := make([]types.Node, 0)
	totalResources := pod.GetTotalResourceRequests()

	for _, node := range nodes {
		if node.Status != types.NodeOnline {
			continue
		}

		if node.Resources == nil {
			continue
		}

		maxSlots := node.GetMaxTaskSlots()
		if node.RunningTasks >= maxSlots {
			continue
		}

		if !totalResources.Requests.IsZero() {
			if !node.Resources.CanFit(totalResources) {
				continue
			}
		}

		available = append(available, node)
	}
	return available
}

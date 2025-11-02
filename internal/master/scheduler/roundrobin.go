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
// If the task specifies resource requirements, only nodes with sufficient resources are considered.
// Returns ErrNoAvailableNodes if no suitable nodes are found.
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

// SelectNodeForPod selects the next available node for a pod in round-robin order.
// Pods can contain multiple containers. The scheduler considers the total resource
// requirements across all containers when selecting a node.
// Returns ErrNoAvailableNodes if no suitable nodes are found.
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

// filterAvailableForTask filters nodes to those that are online, have task capacity,
// and have sufficient resources for the task (if resource requirements are specified)
func filterAvailableForTask(task types.Task, nodes []types.Node) []types.Node {
	available := make([]types.Node, 0)
	for _, node := range nodes {
		if node.Status != types.NodeOnline {
			continue
		}

		if node.RunningTasks >= node.Capacity {
			continue
		}

		if node.Resources != nil && !task.Resources.Requests.IsZero() {
			if !node.Resources.CanFit(task.Resources) {
				continue
			}
		}

		available = append(available, node)
	}
	return available
}

// filterAvailableForPod filters nodes to those that are online, have task capacity,
// and have sufficient resources for the pod's total resource requirements
func filterAvailableForPod(pod types.Pod, nodes []types.Node) []types.Node {
	available := make([]types.Node, 0)
	totalResources := pod.GetTotalResourceRequests()

	for _, node := range nodes {
		if node.Status != types.NodeOnline {
			continue
		}

		if node.RunningTasks >= node.Capacity {
			continue
		}

		if node.Resources != nil && !totalResources.Requests.IsZero() {
			if !node.Resources.CanFit(totalResources) {
				continue
			}
		}

		available = append(available, node)
	}
	return available
}

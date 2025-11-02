package scheduler

import "github.com/danpasecinic/podling/internal/types"

// Scheduler selects a node to run a task or pod on.
// Implementations must be safe for concurrent use.
type Scheduler interface {
	// SelectNode chooses a node from the available nodes to run the given task.
	// Returns ErrNoAvailableNodes if no suitable node is found.
	SelectNode(task types.Task, nodes []types.Node) (*types.Node, error)

	// SelectNodeForPod chooses a node from the available nodes to run the given pod.
	// Returns ErrNoAvailableNodes if no suitable node is found.
	SelectNodeForPod(pod types.Pod, nodes []types.Node) (*types.Node, error)
}

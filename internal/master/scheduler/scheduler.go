package scheduler

import "github.com/danpasecinic/podling/internal/types"

type Scheduler interface {
	SelectNode(task types.Task, nodes []types.Node) (*types.Node, error)
}

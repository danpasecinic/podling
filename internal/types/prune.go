package types

// PruneResult represents the result of a prune operation.
type PruneResult struct {
	PodsRemoved     int `json:"podsRemoved"`
	NodesRemoved    int `json:"nodesRemoved"`
	ServicesRemoved int `json:"servicesRemoved"`
	TasksRemoved    int `json:"tasksRemoved"`
}

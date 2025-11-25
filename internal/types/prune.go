package types

type PruneResult struct {
	PodsRemoved     int `json:"podsRemoved"`
	NodesRemoved    int `json:"nodesRemoved"`
	ServicesRemoved int `json:"servicesRemoved"`
	TasksRemoved    int `json:"tasksRemoved"`
}

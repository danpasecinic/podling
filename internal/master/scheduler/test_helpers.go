package scheduler

import (
	"time"

	"github.com/danpasecinic/podling/internal/types"
)

// newTestNode creates a node with default resource capacity for testing
func newTestNode(nodeID string, status types.NodeStatus, runningTasks int) types.Node {
	return newTestNodeWithResources(nodeID, status, runningTasks, 4000, 8*1024*1024*1024)
}

// newTestNodeWithResources creates a node with custom resource capacity for testing
func newTestNodeWithResources(nodeID string, status types.NodeStatus, runningTasks int, cpuMillicores int64, memoryBytes int64) types.Node {
	return types.Node{
		NodeID:        nodeID,
		Hostname:      "localhost",
		Port:          8081,
		Status:        status,
		RunningTasks:  runningTasks,
		LastHeartbeat: time.Now(),
		Resources: &types.NodeResources{
			Capacity: types.ResourceList{
				CPU:    cpuMillicores,
				Memory: memoryBytes,
			},
			Allocatable: types.ResourceList{
				CPU:    cpuMillicores,
				Memory: memoryBytes,
			},
			Used: types.ResourceList{
				CPU:    0,
				Memory: 0,
			},
		},
	}
}

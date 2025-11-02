package types

import "time"

// NodeStatus represents the current state of a worker node
type NodeStatus string

const (
	NodeOnline  NodeStatus = "online"
	NodeOffline NodeStatus = "offline"
)

// Node represents a worker node in the system
type Node struct {
	NodeID        string         `json:"nodeId"`
	Hostname      string         `json:"hostname"`
	Port          int            `json:"port"`
	Status        NodeStatus     `json:"status"`
	RunningTasks  int            `json:"runningTasks"`
	LastHeartbeat time.Time      `json:"lastHeartbeat"`
	Resources     *NodeResources `json:"resources"`
}

// GetMaxTaskSlots returns the maximum number of tasks that can run on the node
// This is calculated based on resource capacity using a simple heuristic
func (n *Node) GetMaxTaskSlots() int {
	if n.Resources == nil {
		return 10
	}

	// Each task slot represents ~1 CPU core and ~1GB of memory
	cpuSlots := n.Resources.Capacity.CPU / 1000
	memorySlots := n.Resources.Capacity.Memory / (1024 * 1024 * 1024)

	// Return the minimum to ensure we don't exceed either limit
	if cpuSlots < memorySlots {
		return int(cpuSlots)
	}
	return int(memorySlots)
}

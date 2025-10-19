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
	NodeID        string     `json:"nodeId"`
	Hostname      string     `json:"hostname"`
	Port          int        `json:"port"`
	Status        NodeStatus `json:"status"`
	Capacity      int        `json:"capacity"`
	RunningTasks  int        `json:"runningTasks"`
	LastHeartbeat time.Time  `json:"lastHeartbeat"`
}

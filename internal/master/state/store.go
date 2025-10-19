package state

import (
	"errors"
	"sync"
	"time"

	"github.com/danpasecinic/podling/internal/types"
)

var (
	// ErrTaskNotFound is returned when a task is not found in the store
	ErrTaskNotFound = errors.New("task not found")
	// ErrNodeNotFound is returned when a node is not found in the store
	ErrNodeNotFound = errors.New("node not found")
	// ErrTaskAlreadyExists is returned when attempting to add a duplicate task
	ErrTaskAlreadyExists = errors.New("task already exists")
	// ErrNodeAlreadyExists is returned when attempting to add a duplicate node
	ErrNodeAlreadyExists = errors.New("node already exists")
)

// TaskUpdate contains fields that can be updated for a task
type TaskUpdate struct {
	Status      *types.TaskStatus
	NodeID      *string
	ContainerID *string
	StartedAt   *time.Time
	FinishedAt  *time.Time
	Error       *string
}

// NodeUpdate contains fields that can be updated for a node
type NodeUpdate struct {
	Status        *types.NodeStatus
	RunningTasks  *int
	LastHeartbeat *time.Time
}

// StateStore defines the interface for managing task and node state
type StateStore interface {
	// Task operations
	AddTask(task types.Task) error
	GetTask(taskID string) (types.Task, error)
	UpdateTask(taskID string, updates TaskUpdate) error
	ListTasks() ([]types.Task, error)

	// Node operations
	AddNode(node types.Node) error
	GetNode(nodeID string) (types.Node, error)
	UpdateNode(nodeID string, updates NodeUpdate) error
	ListNodes() ([]types.Node, error)

	// Utility
	GetAvailableNodes() ([]types.Node, error)
}

// InMemoryStore is a thread-safe in-memory implementation of StateStore
type InMemoryStore struct {
	mu    sync.RWMutex
	tasks map[string]types.Task
	nodes map[string]types.Node
}

// NewInMemoryStore creates a new in-memory state store
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		tasks: make(map[string]types.Task),
		nodes: make(map[string]types.Node),
	}
}

// AddTask adds a new task to the store
func (s *InMemoryStore) AddTask(task types.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[task.TaskID]; exists {
		return ErrTaskAlreadyExists
	}

	s.tasks[task.TaskID] = task
	return nil
}

// GetTask retrieves a task by ID
func (s *InMemoryStore) GetTask(taskID string) (types.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return types.Task{}, ErrTaskNotFound
	}

	return task, nil
}

// UpdateTask updates specific fields of a task
func (s *InMemoryStore) UpdateTask(taskID string, updates TaskUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return ErrTaskNotFound
	}

	// Apply updates if provided
	if updates.Status != nil {
		task.Status = *updates.Status
	}
	if updates.NodeID != nil {
		task.NodeID = *updates.NodeID
	}
	if updates.ContainerID != nil {
		task.ContainerID = *updates.ContainerID
	}
	if updates.StartedAt != nil {
		task.StartedAt = updates.StartedAt
	}
	if updates.FinishedAt != nil {
		task.FinishedAt = updates.FinishedAt
	}
	if updates.Error != nil {
		task.Error = *updates.Error
	}

	s.tasks[taskID] = task
	return nil
}

// ListTasks returns all tasks in the store
func (s *InMemoryStore) ListTasks() ([]types.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]types.Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// AddNode adds a new node to the store
func (s *InMemoryStore) AddNode(node types.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.nodes[node.NodeID]; exists {
		return ErrNodeAlreadyExists
	}

	s.nodes[node.NodeID] = node
	return nil
}

// GetNode retrieves a node by ID
func (s *InMemoryStore) GetNode(nodeID string) (types.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	node, exists := s.nodes[nodeID]
	if !exists {
		return types.Node{}, ErrNodeNotFound
	}

	return node, nil
}

// UpdateNode updates specific fields of a node
func (s *InMemoryStore) UpdateNode(nodeID string, updates NodeUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	node, exists := s.nodes[nodeID]
	if !exists {
		return ErrNodeNotFound
	}

	// Apply updates if provided
	if updates.Status != nil {
		node.Status = *updates.Status
	}
	if updates.RunningTasks != nil {
		node.RunningTasks = *updates.RunningTasks
	}
	if updates.LastHeartbeat != nil {
		node.LastHeartbeat = *updates.LastHeartbeat
	}

	s.nodes[nodeID] = node
	return nil
}

// ListNodes returns all nodes in the store
func (s *InMemoryStore) ListNodes() ([]types.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes := make([]types.Node, 0, len(s.nodes))
	for _, node := range s.nodes {
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// GetAvailableNodes returns all online nodes
func (s *InMemoryStore) GetAvailableNodes() ([]types.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes := make([]types.Node, 0)
	for _, node := range s.nodes {
		if node.Status == types.NodeOnline {
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

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
	// ErrPodNotFound is returned when a pod is not found in the store
	ErrPodNotFound = errors.New("pod not found")
	// ErrPodAlreadyExists is returned when attempting to add a duplicate pod
	ErrPodAlreadyExists = errors.New("pod already exists")
)

// TaskUpdate contains fields that can be updated for a task
type TaskUpdate struct {
	Status       *types.TaskStatus
	NodeID       *string
	ContainerID  *string
	StartedAt    *time.Time
	FinishedAt   *time.Time
	Error        *string
	HealthStatus *types.HealthStatus
}

// NodeUpdate contains fields that can be updated for a node
type NodeUpdate struct {
	Status        *types.NodeStatus
	RunningTasks  *int
	LastHeartbeat *time.Time
}

// PodUpdate contains fields that can be updated for a pod
type PodUpdate struct {
	Status      *types.PodStatus
	NodeID      *string
	Containers  []types.Container
	ScheduledAt *time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
	Message     *string
	Reason      *string
}

// StateStore defines the interface for managing task and node state
type StateStore interface {
	// Task operations
	AddTask(task types.Task) error
	GetTask(taskID string) (types.Task, error)
	UpdateTask(taskID string, updates TaskUpdate) error
	ListTasks() ([]types.Task, error)

	// Pod operations
	AddPod(pod types.Pod) error
	GetPod(podID string) (types.Pod, error)
	UpdatePod(podID string, updates PodUpdate) error
	ListPods() ([]types.Pod, error)
	DeletePod(podID string) error

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
	pods  map[string]types.Pod
	nodes map[string]types.Node
}

// NewInMemoryStore creates a new in-memory state store
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		tasks: make(map[string]types.Task),
		pods:  make(map[string]types.Pod),
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
	if updates.HealthStatus != nil {
		task.HealthStatus = *updates.HealthStatus
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

// AddPod adds a new pod to the store
func (s *InMemoryStore) AddPod(pod types.Pod) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.pods[pod.PodID]; exists {
		return ErrPodAlreadyExists
	}

	s.pods[pod.PodID] = pod
	return nil
}

// GetPod retrieves a pod by ID
func (s *InMemoryStore) GetPod(podID string) (types.Pod, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pod, exists := s.pods[podID]
	if !exists {
		return types.Pod{}, ErrPodNotFound
	}

	return pod, nil
}

// UpdatePod updates specific fields of a pod
func (s *InMemoryStore) UpdatePod(podID string, updates PodUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pod, exists := s.pods[podID]
	if !exists {
		return ErrPodNotFound
	}

	// Apply updates if provided
	if updates.Status != nil {
		pod.Status = *updates.Status
	}
	if updates.NodeID != nil {
		pod.NodeID = *updates.NodeID
	}
	if updates.Containers != nil {
		pod.Containers = updates.Containers
	}
	if updates.ScheduledAt != nil {
		pod.ScheduledAt = updates.ScheduledAt
	}
	if updates.StartedAt != nil {
		pod.StartedAt = updates.StartedAt
	}
	if updates.FinishedAt != nil {
		pod.FinishedAt = updates.FinishedAt
	}
	if updates.Message != nil {
		pod.Message = *updates.Message
	}
	if updates.Reason != nil {
		pod.Reason = *updates.Reason
	}

	s.pods[podID] = pod
	return nil
}

// ListPods returns all pods in the store
func (s *InMemoryStore) ListPods() ([]types.Pod, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pods := make([]types.Pod, 0, len(s.pods))
	for _, pod := range s.pods {
		pods = append(pods, pod)
	}

	return pods, nil
}

// DeletePod removes a pod from the store
func (s *InMemoryStore) DeletePod(podID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.pods[podID]; !exists {
		return ErrPodNotFound
	}

	delete(s.pods, podID)
	return nil
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

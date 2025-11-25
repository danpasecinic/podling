package state

import (
	"sync"
	"time"

	"github.com/danpasecinic/podling/internal/types"
)

type MockStateStore struct {
	mu        sync.RWMutex
	tasks     map[string]types.Task
	pods      map[string]types.Pod
	nodes     map[string]types.Node
	services  map[string]types.Service
	endpoints map[string]types.Endpoints
}

func NewMockStateStore() *MockStateStore {
	return &MockStateStore{
		tasks:     make(map[string]types.Task),
		pods:      make(map[string]types.Pod),
		nodes:     make(map[string]types.Node),
		services:  make(map[string]types.Service),
		endpoints: make(map[string]types.Endpoints),
	}
}

func (s *MockStateStore) AddTask(task types.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tasks[task.TaskID]; exists {
		return ErrTaskAlreadyExists
	}
	s.tasks[task.TaskID] = task
	return nil
}

func (s *MockStateStore) GetTask(taskID string) (types.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, exists := s.tasks[taskID]
	if !exists {
		return types.Task{}, ErrTaskNotFound
	}
	return task, nil
}

func (s *MockStateStore) UpdateTask(taskID string, updates TaskUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, exists := s.tasks[taskID]
	if !exists {
		return ErrTaskNotFound
	}
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

func (s *MockStateStore) ListTasks() ([]types.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tasks := make([]types.Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (s *MockStateStore) DeleteTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tasks[taskID]; !exists {
		return ErrTaskNotFound
	}
	delete(s.tasks, taskID)
	return nil
}

func (s *MockStateStore) AddNode(node types.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.nodes[node.NodeID]; exists {
		return ErrNodeAlreadyExists
	}
	s.nodes[node.NodeID] = node
	return nil
}

func (s *MockStateStore) GetNode(nodeID string) (types.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	node, exists := s.nodes[nodeID]
	if !exists {
		return types.Node{}, ErrNodeNotFound
	}
	return node, nil
}

func (s *MockStateStore) UpdateNode(nodeID string, updates NodeUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	node, exists := s.nodes[nodeID]
	if !exists {
		return ErrNodeNotFound
	}
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

func (s *MockStateStore) ListNodes() ([]types.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nodes := make([]types.Node, 0, len(s.nodes))
	for _, node := range s.nodes {
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (s *MockStateStore) DeleteNode(nodeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.nodes[nodeID]; !exists {
		return ErrNodeNotFound
	}
	delete(s.nodes, nodeID)
	return nil
}

func (s *MockStateStore) GetAvailableNodes() ([]types.Node, error) {
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

func (s *MockStateStore) AddPod(pod types.Pod) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.pods[pod.PodID]; exists {
		return ErrPodAlreadyExists
	}
	s.pods[pod.PodID] = pod
	return nil
}

func (s *MockStateStore) GetPod(podID string) (types.Pod, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pod, exists := s.pods[podID]
	if !exists {
		return types.Pod{}, ErrPodNotFound
	}
	return pod, nil
}

func (s *MockStateStore) UpdatePod(podID string, updates PodUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	pod, exists := s.pods[podID]
	if !exists {
		return ErrPodNotFound
	}
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
	if updates.Annotations != nil {
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}
		for k, v := range *updates.Annotations {
			pod.Annotations[k] = v
		}
	}
	s.pods[podID] = pod
	return nil
}

func (s *MockStateStore) ListPods() ([]types.Pod, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pods := make([]types.Pod, 0, len(s.pods))
	for _, pod := range s.pods {
		pods = append(pods, pod)
	}
	return pods, nil
}

func (s *MockStateStore) DeletePod(podID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.pods[podID]; !exists {
		return ErrPodNotFound
	}
	delete(s.pods, podID)
	return nil
}

func (s *MockStateStore) ListPodsByLabels(namespace string, labels map[string]string) ([]types.Pod, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if namespace == "" {
		namespace = "default"
	}
	pods := make([]types.Pod, 0)
	for _, pod := range s.pods {
		podNamespace := pod.Namespace
		if podNamespace == "" {
			podNamespace = "default"
		}
		if podNamespace != namespace {
			continue
		}
		matches := true
		for key, value := range labels {
			if podValue, ok := pod.Labels[key]; !ok || podValue != value {
				matches = false
				break
			}
		}
		if matches {
			pods = append(pods, pod)
		}
	}
	return pods, nil
}

func (s *MockStateStore) AddService(service types.Service) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.services[service.ServiceID]; exists {
		return ErrServiceAlreadyExists
	}
	s.services[service.ServiceID] = service
	return nil
}

func (s *MockStateStore) GetService(serviceID string) (types.Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	service, exists := s.services[serviceID]
	if !exists {
		return types.Service{}, ErrServiceNotFound
	}
	return service, nil
}

func (s *MockStateStore) GetServiceByName(namespace, name string) (types.Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if namespace == "" {
		namespace = "default"
	}
	for _, service := range s.services {
		svcNamespace := service.Namespace
		if svcNamespace == "" {
			svcNamespace = "default"
		}
		if svcNamespace == namespace && service.Name == name {
			return service, nil
		}
	}
	return types.Service{}, ErrServiceNotFound
}

func (s *MockStateStore) UpdateService(serviceID string, updates types.ServiceUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	service, exists := s.services[serviceID]
	if !exists {
		return ErrServiceNotFound
	}
	if updates.Selector != nil {
		service.Selector = *updates.Selector
	}
	if updates.Ports != nil {
		service.Ports = *updates.Ports
	}
	if updates.Labels != nil {
		service.Labels = *updates.Labels
	}
	if updates.Annotations != nil {
		service.Annotations = *updates.Annotations
	}
	if updates.SessionAffinity != nil {
		service.SessionAffinity = *updates.SessionAffinity
	}
	service.UpdatedAt = time.Now()
	s.services[serviceID] = service
	return nil
}

func (s *MockStateStore) ListServices(namespace string) ([]types.Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	services := make([]types.Service, 0)
	for _, service := range s.services {
		svcNamespace := service.Namespace
		if svcNamespace == "" {
			svcNamespace = "default"
		}
		if namespace == "" || svcNamespace == namespace {
			services = append(services, service)
		}
	}
	return services, nil
}

func (s *MockStateStore) DeleteService(serviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.services[serviceID]; !exists {
		return ErrServiceNotFound
	}
	delete(s.services, serviceID)
	return nil
}

func (s *MockStateStore) SetEndpoints(endpoints types.Endpoints) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	endpoints.UpdatedAt = time.Now()
	s.endpoints[endpoints.ServiceID] = endpoints
	return nil
}

func (s *MockStateStore) GetEndpoints(serviceID string) (types.Endpoints, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	endpoints, exists := s.endpoints[serviceID]
	if !exists {
		return types.Endpoints{}, ErrEndpointsNotFound
	}
	return endpoints, nil
}

func (s *MockStateStore) GetEndpointsByServiceName(namespace, serviceName string) (types.Endpoints, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if namespace == "" {
		namespace = "default"
	}
	for _, endpoints := range s.endpoints {
		epNamespace := endpoints.Namespace
		if epNamespace == "" {
			epNamespace = "default"
		}
		if epNamespace == namespace && endpoints.ServiceName == serviceName {
			return endpoints, nil
		}
	}
	return types.Endpoints{}, ErrEndpointsNotFound
}

func (s *MockStateStore) DeleteEndpoints(serviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.endpoints[serviceID]; !exists {
		return ErrEndpointsNotFound
	}
	delete(s.endpoints, serviceID)
	return nil
}

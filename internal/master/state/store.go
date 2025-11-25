package state

import (
	"errors"
	"time"

	"github.com/danpasecinic/podling/internal/types"
)

var (
	ErrTaskNotFound         = errors.New("task not found")
	ErrNodeNotFound         = errors.New("node not found")
	ErrTaskAlreadyExists    = errors.New("task already exists")
	ErrNodeAlreadyExists    = errors.New("node already exists")
	ErrPodNotFound          = errors.New("pod not found")
	ErrPodAlreadyExists     = errors.New("pod already exists")
	ErrServiceNotFound      = errors.New("service not found")
	ErrServiceAlreadyExists = errors.New("service already exists")
	ErrEndpointsNotFound    = errors.New("endpoints not found")
)

type TaskUpdate struct {
	Status       *types.TaskStatus
	NodeID       *string
	ContainerID  *string
	StartedAt    *time.Time
	FinishedAt   *time.Time
	Error        *string
	HealthStatus *types.HealthStatus
}

type NodeUpdate struct {
	Status        *types.NodeStatus
	RunningTasks  *int
	LastHeartbeat *time.Time
}

type PodUpdate struct {
	Status      *types.PodStatus
	NodeID      *string
	Containers  []types.Container
	ScheduledAt *time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
	Message     *string
	Reason      *string
	Annotations *map[string]string
}

type StateStore interface {
	AddTask(task types.Task) error
	GetTask(taskID string) (types.Task, error)
	UpdateTask(taskID string, updates TaskUpdate) error
	ListTasks() ([]types.Task, error)
	DeleteTask(taskID string) error

	AddPod(pod types.Pod) error
	GetPod(podID string) (types.Pod, error)
	UpdatePod(podID string, updates PodUpdate) error
	ListPods() ([]types.Pod, error)
	DeletePod(podID string) error

	AddNode(node types.Node) error
	GetNode(nodeID string) (types.Node, error)
	UpdateNode(nodeID string, updates NodeUpdate) error
	ListNodes() ([]types.Node, error)
	DeleteNode(nodeID string) error

	AddService(service types.Service) error
	GetService(serviceID string) (types.Service, error)
	GetServiceByName(namespace, name string) (types.Service, error)
	UpdateService(serviceID string, updates types.ServiceUpdate) error
	ListServices(namespace string) ([]types.Service, error)
	DeleteService(serviceID string) error

	SetEndpoints(endpoints types.Endpoints) error
	GetEndpoints(serviceID string) (types.Endpoints, error)
	GetEndpointsByServiceName(namespace, serviceName string) (types.Endpoints, error)
	DeleteEndpoints(serviceID string) error

	GetAvailableNodes() ([]types.Node, error)
	ListPodsByLabels(namespace string, labels map[string]string) ([]types.Pod, error)
}

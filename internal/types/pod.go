package types

import "time"

// PodStatus represents the current state of a pod
type PodStatus string

const (
	PodPending   PodStatus = "pending"
	PodScheduled PodStatus = "scheduled"
	PodRunning   PodStatus = "running"
	PodSucceeded PodStatus = "succeeded"
	PodFailed    PodStatus = "failed"
	PodUnknown   PodStatus = "unknown"
)

// Pod represents a group of one or more containers that share network and storage
// Similar to Kubernetes Pods, all containers in a pod are co-located and co-scheduled
type Pod struct {
	// PodID is the unique identifier for the pod
	PodID string `json:"podId"`

	// Name is a human-readable name for the pod
	Name string `json:"name"`

	// Namespace is the logical grouping for the pod (future: multi-tenancy)
	Namespace string `json:"namespace,omitempty"`

	// Labels are key-value pairs for organizing and selecting pods
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are key-value pairs for storing arbitrary metadata
	Annotations map[string]string `json:"annotations,omitempty"`

	// Containers is the list of containers that belong to this pod
	Containers []Container `json:"containers"`

	// Status is the current state of the pod
	Status PodStatus `json:"status"`

	// NodeID is the ID of the node where the pod is scheduled
	NodeID string `json:"nodeId,omitempty"`

	// RestartPolicy defines how containers should be restarted
	RestartPolicy RestartPolicy `json:"restartPolicy,omitempty"`

	// CreatedAt is when the pod was created
	CreatedAt time.Time `json:"createdAt"`

	// ScheduledAt is when the pod was assigned to a node
	ScheduledAt *time.Time `json:"scheduledAt,omitempty"`

	// StartedAt is when the pod started running
	StartedAt *time.Time `json:"startedAt,omitempty"`

	// FinishedAt is when the pod finished (succeeded or failed)
	FinishedAt *time.Time `json:"finishedAt,omitempty"`

	// Message provides human-readable information about the pod
	Message string `json:"message,omitempty"`

	// Reason is a brief CamelCase message indicating why the pod is in its current state
	Reason string `json:"reason,omitempty"`
}

// Container represents a single container within a pod
type Container struct {
	// Name is a unique name for the container within the pod
	Name string `json:"name"`

	// Image is the Docker image to run
	Image string `json:"image"`

	// Command overrides the default entrypoint
	Command []string `json:"command,omitempty"`

	// Args are arguments to the command
	Args []string `json:"args,omitempty"`

	// Env is a map of environment variables
	Env map[string]string `json:"env,omitempty"`

	// Ports are the ports exposed by the container
	Ports []ContainerPort `json:"ports,omitempty"`

	// LivenessProbe checks if the container is alive
	LivenessProbe *HealthCheck `json:"livenessProbe,omitempty"`

	// ReadinessProbe checks if the container is ready to serve traffic
	ReadinessProbe *HealthCheck `json:"readinessProbe,omitempty"`

	// WorkingDir is the working directory for the container
	WorkingDir string `json:"workingDir,omitempty"`

	// Resources specifies the compute resources required by this container
	Resources ResourceRequirements `json:"resources,omitempty"`

	// ---- Runtime fields (populated by worker) ----

	// ContainerID is the Docker container ID (set by worker)
	ContainerID string `json:"containerId,omitempty"`

	// Status is the current state of the container
	Status ContainerStatus `json:"status,omitempty"`

	// HealthStatus is the current health of the container
	HealthStatus HealthStatus `json:"healthStatus,omitempty"`

	// StartedAt is when the container started
	StartedAt *time.Time `json:"startedAt,omitempty"`

	// FinishedAt is when the container finished
	FinishedAt *time.Time `json:"finishedAt,omitempty"`

	// ExitCode is the exit code of the container (if finished)
	ExitCode *int `json:"exitCode,omitempty"`

	// Error contains error information if the container failed
	Error string `json:"error,omitempty"`

	// RestartCount is the number of times the container has been restarted
	RestartCount int `json:"restartCount,omitempty"`
}

// ContainerPort represents a network port in a single container
type ContainerPort struct {
	// Name is an optional name for the port (e.g., "http", "metrics")
	Name string `json:"name,omitempty"`

	// ContainerPort is the port number exposed by the container
	ContainerPort int `json:"containerPort"`

	// Protocol is the network protocol (TCP or UDP)
	Protocol string `json:"protocol,omitempty"` // Default: TCP

	// HostPort is the port number on the host (optional, for NodePort services)
	HostPort int `json:"hostPort,omitempty"`
}

// ContainerStatus represents the state of a container
type ContainerStatus string

const (
	ContainerWaiting    ContainerStatus = "waiting"
	ContainerRunning    ContainerStatus = "running"
	ContainerTerminated ContainerStatus = "terminated"
)

// IsPodTerminal returns true if the pod is in a terminal state
func (p *Pod) IsPodTerminal() bool {
	return p.Status == PodSucceeded || p.Status == PodFailed
}

// IsAllContainersRunning returns true if all containers are running
func (p *Pod) IsAllContainersRunning() bool {
	if len(p.Containers) == 0 {
		return false
	}
	for _, container := range p.Containers {
		if container.Status != ContainerRunning {
			return false
		}
	}
	return true
}

// IsAnyContainerFailed returns true if any container has failed
func (p *Pod) IsAnyContainerFailed() bool {
	for _, container := range p.Containers {
		if container.Status == ContainerTerminated && container.ExitCode != nil && *container.ExitCode != 0 {
			return true
		}
	}
	return false
}

// GetContainerByName returns a container by name
func (p *Pod) GetContainerByName(name string) *Container {
	for i := range p.Containers {
		if p.Containers[i].Name == name {
			return &p.Containers[i]
		}
	}
	return nil
}

// GetTotalResourceRequests returns the sum of all container resource requests
// This is used for scheduling - the pod needs at least this much resources
func (p *Pod) GetTotalResourceRequests() ResourceRequirements {
	var totalCPU, totalMemory int64

	for _, container := range p.Containers {
		totalCPU += container.Resources.Requests.CPU
		totalMemory += container.Resources.Requests.Memory
	}

	return ResourceRequirements{
		Requests: ResourceList{
			CPU:    totalCPU,
			Memory: totalMemory,
		},
	}
}

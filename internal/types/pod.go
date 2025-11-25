package types

import "time"

type PodStatus string

const (
	PodPending   PodStatus = "pending"
	PodScheduled PodStatus = "scheduled"
	PodRunning   PodStatus = "running"
	PodSucceeded PodStatus = "succeeded"
	PodFailed    PodStatus = "failed"
	PodUnknown   PodStatus = "unknown"
)

type Pod struct {
	PodID         string            `json:"podId"`
	Name          string            `json:"name"`
	Namespace     string            `json:"namespace,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty"`
	Containers    []Container       `json:"containers"`
	Status        PodStatus         `json:"status"`
	NodeID        string            `json:"nodeId,omitempty"`
	RestartPolicy RestartPolicy     `json:"restartPolicy,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
	ScheduledAt   *time.Time        `json:"scheduledAt,omitempty"`
	StartedAt     *time.Time        `json:"startedAt,omitempty"`
	FinishedAt    *time.Time        `json:"finishedAt,omitempty"`
	Message       string            `json:"message,omitempty"`
	Reason        string            `json:"reason,omitempty"`
}

type Container struct {
	Name           string               `json:"name"`
	Image          string               `json:"image"`
	Command        []string             `json:"command,omitempty"`
	Args           []string             `json:"args,omitempty"`
	Env            map[string]string    `json:"env,omitempty"`
	Ports          []ContainerPort      `json:"ports,omitempty"`
	LivenessProbe  *HealthCheck         `json:"livenessProbe,omitempty"`
	ReadinessProbe *HealthCheck         `json:"readinessProbe,omitempty"`
	WorkingDir     string               `json:"workingDir,omitempty"`
	Resources      ResourceRequirements `json:"resources,omitempty"`
	ContainerID    string               `json:"containerId,omitempty"`
	Status         ContainerStatus      `json:"status,omitempty"`
	HealthStatus   HealthStatus         `json:"healthStatus,omitempty"`
	StartedAt      *time.Time           `json:"startedAt,omitempty"`
	FinishedAt     *time.Time           `json:"finishedAt,omitempty"`
	ExitCode       *int                 `json:"exitCode,omitempty"`
	Error          string               `json:"error,omitempty"`
	RestartCount   int                  `json:"restartCount,omitempty"`
}

type ContainerPort struct {
	Name          string `json:"name,omitempty"`
	ContainerPort int    `json:"containerPort"`
	Protocol      string `json:"protocol,omitempty"`
	HostPort      int    `json:"hostPort,omitempty"`
}

type ContainerStatus string

const (
	ContainerWaiting    ContainerStatus = "waiting"
	ContainerRunning    ContainerStatus = "running"
	ContainerTerminated ContainerStatus = "terminated"
)

func (p *Pod) IsPodTerminal() bool {
	return p.Status == PodSucceeded || p.Status == PodFailed
}

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

func (p *Pod) IsAnyContainerFailed() bool {
	for _, container := range p.Containers {
		if container.Status == ContainerTerminated && container.ExitCode != nil && *container.ExitCode != 0 {
			return true
		}
	}
	return false
}

func (p *Pod) GetContainerByName(name string) *Container {
	for i := range p.Containers {
		if p.Containers[i].Name == name {
			return &p.Containers[i]
		}
	}
	return nil
}

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

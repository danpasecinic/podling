package types

import "time"

// TaskStatus represents the current state of a task
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskScheduled TaskStatus = "scheduled"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
)

// Task represents a container execution task in the system
type Task struct {
	TaskID         string            `json:"taskId"`
	Name           string            `json:"name"`
	Image          string            `json:"image"`
	Env            map[string]string `json:"env,omitempty"`
	Status         TaskStatus        `json:"status"`
	NodeID         string            `json:"nodeId,omitempty"`
	ContainerID    string            `json:"containerId,omitempty"`
	CreatedAt      time.Time         `json:"createdAt"`
	StartedAt      *time.Time        `json:"startedAt,omitempty"`
	FinishedAt     *time.Time        `json:"finishedAt,omitempty"`
	Error          string            `json:"error,omitempty"`
	LivenessProbe  *HealthCheck      `json:"livenessProbe,omitempty"`
	ReadinessProbe *HealthCheck      `json:"readinessProbe,omitempty"`
	RestartPolicy  RestartPolicy     `json:"restartPolicy,omitempty"`
	HealthStatus   HealthStatus      `json:"healthStatus,omitempty"`
}

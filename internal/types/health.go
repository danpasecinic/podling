package types

import "time"

type ProbeType string

const (
	ProbeTypeHTTP ProbeType = "http"
	ProbeTypeTCP  ProbeType = "tcp"
	ProbeTypeExec ProbeType = "exec"
)

type RestartPolicy string

const (
	RestartPolicyAlways    RestartPolicy = "Always"
	RestartPolicyOnFailure RestartPolicy = "OnFailure"
	RestartPolicyNever     RestartPolicy = "Never"
)

type HealthCheck struct {
	Type                ProbeType `json:"type"`
	HTTPPath            string    `json:"httpPath,omitempty"`
	Port                int       `json:"port,omitempty"`
	Command             []string  `json:"command,omitempty"`
	InitialDelaySeconds int       `json:"initialDelaySeconds,omitempty"`
	PeriodSeconds       int       `json:"periodSeconds,omitempty"`
	TimeoutSeconds      int       `json:"timeoutSeconds,omitempty"`
	SuccessThreshold    int       `json:"successThreshold,omitempty"`
	FailureThreshold    int       `json:"failureThreshold,omitempty"`
}

func (h *HealthCheck) GetInitialDelay() time.Duration {
	if h.InitialDelaySeconds <= 0 {
		return 0
	}
	return time.Duration(h.InitialDelaySeconds) * time.Second
}

func (h *HealthCheck) GetPeriod() time.Duration {
	if h.PeriodSeconds <= 0 {
		return 10 * time.Second // default
	}
	return time.Duration(h.PeriodSeconds) * time.Second
}

func (h *HealthCheck) GetTimeout() time.Duration {
	if h.TimeoutSeconds <= 0 {
		return 1 * time.Second // default
	}
	return time.Duration(h.TimeoutSeconds) * time.Second
}

func (h *HealthCheck) GetSuccessThreshold() int {
	if h.SuccessThreshold <= 0 {
		return 1 // default
	}
	return h.SuccessThreshold
}

func (h *HealthCheck) GetFailureThreshold() int {
	if h.FailureThreshold <= 0 {
		return 3 // default
	}
	return h.FailureThreshold
}

type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

type ProbeResult struct {
	Success   bool
	Message   string
	Timestamp time.Time
}

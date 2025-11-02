package types

import "time"

// ProbeType defines the type of health check probe
type ProbeType string

const (
	// ProbeTypeHTTP performs HTTP GET requests
	ProbeTypeHTTP ProbeType = "http"
	// ProbeTypeTCP performs TCP socket connections
	ProbeTypeTCP ProbeType = "tcp"
	// ProbeTypeExec executes a command inside the container
	ProbeTypeExec ProbeType = "exec"
)

// RestartPolicy defines when a container should be restarted
type RestartPolicy string

const (
	// RestartPolicyAlways always restart the container when it exits
	RestartPolicyAlways RestartPolicy = "Always"
	// RestartPolicyOnFailure restart only when container exits with non-zero code
	RestartPolicyOnFailure RestartPolicy = "OnFailure"
	// RestartPolicyNever never restart the container
	RestartPolicyNever RestartPolicy = "Never"
)

// HealthCheck defines a probe to check container health
type HealthCheck struct {
	// Type of probe: http, tcp, or exec
	Type ProbeType `json:"type"`

	// HTTPPath is the path for HTTP probes (e.g., "/health")
	HTTPPath string `json:"httpPath,omitempty"`

	// Port to check (for HTTP and TCP probes)
	Port int `json:"port,omitempty"`

	// Command to execute (for exec probes)
	Command []string `json:"command,omitempty"`

	// InitialDelaySeconds before first probe
	InitialDelaySeconds int `json:"initialDelaySeconds,omitempty"`

	// PeriodSeconds between probes
	PeriodSeconds int `json:"periodSeconds,omitempty"`

	// TimeoutSeconds for each probe
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`

	// SuccessThreshold minimum consecutive successes to be considered healthy
	SuccessThreshold int `json:"successThreshold,omitempty"`

	// FailureThreshold consecutive failures before considered unhealthy
	FailureThreshold int `json:"failureThreshold,omitempty"`
}

// GetInitialDelay returns the initial delay as a duration
func (h *HealthCheck) GetInitialDelay() time.Duration {
	if h.InitialDelaySeconds <= 0 {
		return 0
	}
	return time.Duration(h.InitialDelaySeconds) * time.Second
}

// GetPeriod returns the period as a duration
func (h *HealthCheck) GetPeriod() time.Duration {
	if h.PeriodSeconds <= 0 {
		return 10 * time.Second // default
	}
	return time.Duration(h.PeriodSeconds) * time.Second
}

// GetTimeout returns the timeout as a duration
func (h *HealthCheck) GetTimeout() time.Duration {
	if h.TimeoutSeconds <= 0 {
		return 1 * time.Second // default
	}
	return time.Duration(h.TimeoutSeconds) * time.Second
}

// GetSuccessThreshold returns the success threshold with default
func (h *HealthCheck) GetSuccessThreshold() int {
	if h.SuccessThreshold <= 0 {
		return 1 // default
	}
	return h.SuccessThreshold
}

// GetFailureThreshold returns the failure threshold with default
func (h *HealthCheck) GetFailureThreshold() int {
	if h.FailureThreshold <= 0 {
		return 3 // default
	}
	return h.FailureThreshold
}

// HealthStatus represents the current health status
type HealthStatus string

const (
	// HealthStatusHealthy indicates the container is healthy
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusUnhealthy indicates the container is unhealthy
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	// HealthStatusUnknown indicates health status is unknown
	HealthStatusUnknown HealthStatus = "unknown"
)

// ProbeResult represents the result of a health probe
type ProbeResult struct {
	// Success indicates if the probe succeeded
	Success bool
	// Message contains additional information about the probe
	Message string
	// Timestamp when the probe was executed
	Timestamp time.Time
}

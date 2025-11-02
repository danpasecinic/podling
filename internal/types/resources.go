package types

import (
	"fmt"
	"strconv"
	"strings"
)

// ResourceRequirements specifies the compute resources required by a container
// Following Kubernetes resource model with requests and limits
type ResourceRequirements struct {
	// Requests describes the minimum amount of resources required
	// Used for scheduling decisions - a pod will only be scheduled on a node
	// that has enough available resources to satisfy the request
	Requests ResourceList `json:"requests,omitempty"`

	// Limits describes the maximum amount of resources allowed
	// Enforced by the container runtime (Docker) - if a container exceeds
	// its memory limit, it will be OOM killed. CPU limits throttle the container.
	Limits ResourceList `json:"limits,omitempty"`
}

// ResourceList is a set of (resource name, quantity) pairs
type ResourceList struct {
	// CPU in millicores (1000m = 1 CPU core)
	// Examples: "500m" (half a core), "2000m" or "2" (2 cores)
	CPU int64 `json:"cpu,omitempty"`

	// Memory in bytes
	// Can be parsed from strings like "256Mi", "1Gi", "512000000" (bytes)
	Memory int64 `json:"memory,omitempty"`
}

// NodeResources tracks the total capacity and current usage of a node
type NodeResources struct {
	// Capacity is the total amount of resources available on the node
	Capacity ResourceList `json:"capacity"`

	// Allocatable is the amount available for scheduling (capacity - system reserved)
	// For now, we'll use Allocatable = Capacity, but in production systems
	// some resources are reserved for system daemons
	Allocatable ResourceList `json:"allocatable"`

	// Used is the current amount of resources allocated to scheduled workloads
	// This is based on resource requests, not actual usage
	Used ResourceList `json:"used"`
}

// Available returns the amount of resources still available for scheduling
func (nr *NodeResources) Available() ResourceList {
	return ResourceList{
		CPU:    nr.Allocatable.CPU - nr.Used.CPU,
		Memory: nr.Allocatable.Memory - nr.Used.Memory,
	}
}

// CanFit checks if the given resource requirements can fit on the node
func (nr *NodeResources) CanFit(req ResourceRequirements) bool {
	available := nr.Available()

	// Check if requests (minimum required) fit in available resources
	if req.Requests.CPU > available.CPU {
		return false
	}
	if req.Requests.Memory > available.Memory {
		return false
	}

	return true
}

// Allocate adds the given resource requirements to the used resources
func (nr *NodeResources) Allocate(req ResourceRequirements) {
	nr.Used.CPU += req.Requests.CPU
	nr.Used.Memory += req.Requests.Memory
}

// Release subtracts the given resource requirements from the used resources
func (nr *NodeResources) Release(req ResourceRequirements) {
	nr.Used.CPU -= req.Requests.CPU
	nr.Used.Memory -= req.Requests.Memory

	if nr.Used.CPU < 0 {
		nr.Used.CPU = 0
	}
	if nr.Used.Memory < 0 {
		nr.Used.Memory = 0
	}
}

// ParseCPU parses a CPU quantity string into millicores
// Supports formats: "500m" (millicores), "1" or "1000m" (1 core), "2.5" (2.5 cores)
func ParseCPU(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}

	// Handle millicores format: "500m"
	if strings.HasSuffix(s, "m") {
		millis := strings.TrimSuffix(s, "m")
		return strconv.ParseInt(millis, 10, 64)
	}

	// Handle decimal format: "1", "2.5"
	cores, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid CPU format: %s", s)
	}

	// Convert to millicores
	return int64(cores * 1000), nil
}

// ParseMemory parses a memory quantity string into bytes
// Supports formats: "256Mi", "1Gi", "512000000" (bytes), "512M", "1G"
func ParseMemory(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}

	// Handle binary units (IEC standard): Ki, Mi, Gi, Ti
	for _, unit := range []struct {
		suffix     string
		multiplier int64
	}{
		{"Ti", 1024 * 1024 * 1024 * 1024},
		{"Gi", 1024 * 1024 * 1024},
		{"Mi", 1024 * 1024},
		{"Ki", 1024},
		// Decimal units (SI standard): K, M, G, T
		{"T", 1000 * 1000 * 1000 * 1000},
		{"G", 1000 * 1000 * 1000},
		{"M", 1000 * 1000},
		{"K", 1000},
	} {
		if strings.HasSuffix(s, unit.suffix) {
			numStr := strings.TrimSuffix(s, unit.suffix)
			num, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid memory format: %s", s)
			}
			return int64(num * float64(unit.multiplier)), nil
		}
	}

	bytes, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid memory format: %s", s)
	}

	return bytes, nil
}

// FormatCPU formats CPU millicores as a human-readable string
func FormatCPU(millicores int64) string {
	if millicores == 0 {
		return "0"
	}
	if millicores < 1000 {
		return fmt.Sprintf("%dm", millicores)
	}
	cores := float64(millicores) / 1000.0
	// Remove trailing zeros
	if cores == float64(int64(cores)) {
		return fmt.Sprintf("%d", int64(cores))
	}
	return fmt.Sprintf("%.1f", cores)
}

// FormatMemory formats bytes as a human-readable string
func FormatMemory(bytes int64) string {
	if bytes == 0 {
		return "0"
	}

	const (
		Ki = 1024
		Mi = 1024 * Ki
		Gi = 1024 * Mi
		Ti = 1024 * Gi
	)

	switch {
	case bytes >= Ti:
		return fmt.Sprintf("%.1fTi", float64(bytes)/float64(Ti))
	case bytes >= Gi:
		return fmt.Sprintf("%.1fGi", float64(bytes)/float64(Gi))
	case bytes >= Mi:
		return fmt.Sprintf("%.0fMi", float64(bytes)/float64(Mi))
	case bytes >= Ki:
		return fmt.Sprintf("%.0fKi", float64(bytes)/float64(Ki))
	default:
		return fmt.Sprintf("%d", bytes)
	}
}

// GetCPULimitForDocker returns CPU limit in the format Docker expects
// Docker uses --cpus flag which takes a decimal value (e.g., 0.5 for half a core)
func (rl *ResourceList) GetCPULimitForDocker() float64 {
	if rl.CPU == 0 {
		return 0 // No limit
	}
	return float64(rl.CPU) / 1000.0
}

// GetMemoryLimitForDocker returns memory limit in bytes for Docker
func (rl *ResourceList) GetMemoryLimitForDocker() int64 {
	return rl.Memory
}

// IsZero returns true if the resource list has no resources specified
func (rl *ResourceList) IsZero() bool {
	return rl.CPU == 0 && rl.Memory == 0
}

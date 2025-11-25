package types

import (
	"fmt"
	"strconv"
	"strings"
)

type ResourceRequirements struct {
	Requests ResourceList `json:"requests,omitempty"`
	Limits   ResourceList `json:"limits,omitempty"`
}

type ResourceList struct {
	CPU    int64 `json:"cpu,omitempty"`
	Memory int64 `json:"memory,omitempty"`
}

type NodeResources struct {
	Capacity    ResourceList `json:"capacity"`
	Allocatable ResourceList `json:"allocatable"`
	Used        ResourceList `json:"used"`
}

func (nr *NodeResources) Available() ResourceList {
	return ResourceList{
		CPU:    nr.Allocatable.CPU - nr.Used.CPU,
		Memory: nr.Allocatable.Memory - nr.Used.Memory,
	}
}

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

func (nr *NodeResources) Allocate(req ResourceRequirements) {
	nr.Used.CPU += req.Requests.CPU
	nr.Used.Memory += req.Requests.Memory
}

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

func (rl *ResourceList) GetCPULimitForDocker() float64 {
	if rl.CPU == 0 {
		return 0 // No limit
	}
	return float64(rl.CPU) / 1000.0
}

func (rl *ResourceList) GetMemoryLimitForDocker() int64 {
	return rl.Memory
}

func (rl *ResourceList) IsZero() bool {
	return rl.CPU == 0 && rl.Memory == 0
}

package types

import (
	"testing"
)

func TestParseCPU(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{"empty string", "", 0, false},
		{"millicores", "500m", 500, false},
		{"one core", "1", 1000, false},
		{"one core with m", "1000m", 1000, false},
		{"two cores", "2", 2000, false},
		{"half core decimal", "0.5", 500, false},
		{"two and half cores", "2.5", 2500, false},
		{"invalid format", "abc", 0, true},
		{"invalid millicores", "abcm", 0, true},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got, err := ParseCPU(tt.input)
				if (err != nil) != tt.wantErr {
					t.Errorf("ParseCPU() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if got != tt.want {
					t.Errorf("ParseCPU() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func TestParseMemory(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{"empty string", "", 0, false},
		{"bytes", "1024", 1024, false},
		{"Ki", "1Ki", 1024, false},
		{"Mi", "256Mi", 256 * 1024 * 1024, false},
		{"Gi", "1Gi", 1024 * 1024 * 1024, false},
		{"Ti", "1Ti", 1024 * 1024 * 1024 * 1024, false},
		{"K decimal", "1K", 1000, false},
		{"M decimal", "512M", 512 * 1000 * 1000, false},
		{"G decimal", "2G", 2 * 1000 * 1000 * 1000, false},
		{"T decimal", "1T", 1000 * 1000 * 1000 * 1000, false},
		{"decimal Mi", "1.5Mi", int64(1.5 * 1024 * 1024), false},
		{"invalid format", "abc", 0, true},
		{"invalid unit", "123xyz", 0, true},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got, err := ParseMemory(tt.input)
				if (err != nil) != tt.wantErr {
					t.Errorf("ParseMemory() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if got != tt.want {
					t.Errorf("ParseMemory() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func TestFormatCPU(t *testing.T) {
	tests := []struct {
		name       string
		millicores int64
		want       string
	}{
		{"zero", 0, "0"},
		{"millicores", 500, "500m"},
		{"one core", 1000, "1"},
		{"two cores", 2000, "2"},
		{"one and half", 1500, "1.5"},
		{"two and half", 2500, "2.5"},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got := FormatCPU(tt.millicores)
				if got != tt.want {
					t.Errorf("FormatCPU() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func TestFormatMemory(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"zero", 0, "0"},
		{"bytes", 512, "512"},
		{"Ki", 2048, "2Ki"},
		{"Mi", 256 * 1024 * 1024, "256Mi"},
		{"Gi", 2 * 1024 * 1024 * 1024, "2.0Gi"},
		{"Ti", 1024 * 1024 * 1024 * 1024, "1.0Ti"},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got := FormatMemory(tt.bytes)
				if got != tt.want {
					t.Errorf("FormatMemory() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func TestNodeResources_Available(t *testing.T) {
	nr := &NodeResources{
		Capacity:    ResourceList{CPU: 4000, Memory: 8 * 1024 * 1024 * 1024},
		Allocatable: ResourceList{CPU: 4000, Memory: 8 * 1024 * 1024 * 1024},
		Used:        ResourceList{CPU: 1000, Memory: 2 * 1024 * 1024 * 1024},
	}

	available := nr.Available()

	if available.CPU != 3000 {
		t.Errorf("Available CPU = %v, want 3000", available.CPU)
	}
	if available.Memory != 6*1024*1024*1024 {
		t.Errorf("Available Memory = %v, want %v", available.Memory, 6*1024*1024*1024)
	}
}

func TestNodeResources_CanFit(t *testing.T) {
	nr := &NodeResources{
		Capacity:    ResourceList{CPU: 4000, Memory: 8 * 1024 * 1024 * 1024},
		Allocatable: ResourceList{CPU: 4000, Memory: 8 * 1024 * 1024 * 1024},
		Used:        ResourceList{CPU: 1000, Memory: 2 * 1024 * 1024 * 1024},
	}

	tests := []struct {
		name string
		req  ResourceRequirements
		want bool
	}{
		{
			name: "fits easily",
			req: ResourceRequirements{
				Requests: ResourceList{CPU: 500, Memory: 1024 * 1024 * 1024},
			},
			want: true,
		},
		{
			name: "exactly fits",
			req: ResourceRequirements{
				Requests: ResourceList{CPU: 3000, Memory: 6 * 1024 * 1024 * 1024},
			},
			want: true,
		},
		{
			name: "CPU too large",
			req: ResourceRequirements{
				Requests: ResourceList{CPU: 4000, Memory: 1024 * 1024 * 1024},
			},
			want: false,
		},
		{
			name: "Memory too large",
			req: ResourceRequirements{
				Requests: ResourceList{CPU: 500, Memory: 8 * 1024 * 1024 * 1024},
			},
			want: false,
		},
		{
			name: "both too large",
			req: ResourceRequirements{
				Requests: ResourceList{CPU: 5000, Memory: 10 * 1024 * 1024 * 1024},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got := nr.CanFit(tt.req)
				if got != tt.want {
					t.Errorf("CanFit() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func TestNodeResources_Allocate(t *testing.T) {
	nr := &NodeResources{
		Capacity:    ResourceList{CPU: 4000, Memory: 8 * 1024 * 1024 * 1024},
		Allocatable: ResourceList{CPU: 4000, Memory: 8 * 1024 * 1024 * 1024},
		Used:        ResourceList{CPU: 1000, Memory: 2 * 1024 * 1024 * 1024},
	}

	req := ResourceRequirements{
		Requests: ResourceList{CPU: 500, Memory: 1024 * 1024 * 1024},
	}

	nr.Allocate(req)

	if nr.Used.CPU != 1500 {
		t.Errorf("Used CPU after allocate = %v, want 1500", nr.Used.CPU)
	}
	if nr.Used.Memory != 3*1024*1024*1024 {
		t.Errorf("Used Memory after allocate = %v, want %v", nr.Used.Memory, 3*1024*1024*1024)
	}
}

func TestNodeResources_Release(t *testing.T) {
	nr := &NodeResources{
		Capacity:    ResourceList{CPU: 4000, Memory: 8 * 1024 * 1024 * 1024},
		Allocatable: ResourceList{CPU: 4000, Memory: 8 * 1024 * 1024 * 1024},
		Used:        ResourceList{CPU: 1500, Memory: 3 * 1024 * 1024 * 1024},
	}

	req := ResourceRequirements{
		Requests: ResourceList{CPU: 500, Memory: 1024 * 1024 * 1024},
	}

	nr.Release(req)

	if nr.Used.CPU != 1000 {
		t.Errorf("Used CPU after release = %v, want 1000", nr.Used.CPU)
	}
	if nr.Used.Memory != 2*1024*1024*1024 {
		t.Errorf("Used Memory after release = %v, want %v", nr.Used.Memory, 2*1024*1024*1024)
	}
}

func TestNodeResources_Release_NoNegative(t *testing.T) {
	nr := &NodeResources{
		Capacity:    ResourceList{CPU: 4000, Memory: 8 * 1024 * 1024 * 1024},
		Allocatable: ResourceList{CPU: 4000, Memory: 8 * 1024 * 1024 * 1024},
		Used:        ResourceList{CPU: 500, Memory: 512 * 1024 * 1024},
	}

	// Try to release more than allocated
	req := ResourceRequirements{
		Requests: ResourceList{CPU: 1000, Memory: 1024 * 1024 * 1024},
	}

	nr.Release(req)

	// Should not go negative
	if nr.Used.CPU != 0 {
		t.Errorf("Used CPU after over-release = %v, want 0", nr.Used.CPU)
	}
	if nr.Used.Memory != 0 {
		t.Errorf("Used Memory after over-release = %v, want 0", nr.Used.Memory)
	}
}

func TestResourceList_GetCPULimitForDocker(t *testing.T) {
	tests := []struct {
		name string
		cpu  int64
		want float64
	}{
		{"zero (no limit)", 0, 0},
		{"half core", 500, 0.5},
		{"one core", 1000, 1.0},
		{"two cores", 2000, 2.0},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				rl := &ResourceList{CPU: tt.cpu}
				got := rl.GetCPULimitForDocker()
				if got != tt.want {
					t.Errorf("GetCPULimitForDocker() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func TestResourceList_GetMemoryLimitForDocker(t *testing.T) {
	rl := &ResourceList{Memory: 256 * 1024 * 1024}
	got := rl.GetMemoryLimitForDocker()
	want := int64(256 * 1024 * 1024)

	if got != want {
		t.Errorf("GetMemoryLimitForDocker() = %v, want %v", got, want)
	}
}

func TestResourceList_IsZero(t *testing.T) {
	tests := []struct {
		name string
		rl   ResourceList
		want bool
	}{
		{"zero", ResourceList{}, true},
		{"has CPU", ResourceList{CPU: 100}, false},
		{"has Memory", ResourceList{Memory: 100}, false},
		{"has both", ResourceList{CPU: 100, Memory: 100}, false},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got := tt.rl.IsZero()
				if got != tt.want {
					t.Errorf("IsZero() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func TestPod_GetTotalResourceRequests(t *testing.T) {
	pod := &Pod{
		Containers: []Container{
			{
				Name:  "app",
				Image: "nginx",
				Resources: ResourceRequirements{
					Requests: ResourceList{CPU: 500, Memory: 256 * 1024 * 1024},
				},
			},
			{
				Name:  "sidecar",
				Image: "busybox",
				Resources: ResourceRequirements{
					Requests: ResourceList{CPU: 250, Memory: 128 * 1024 * 1024},
				},
			},
		},
	}

	total := pod.GetTotalResourceRequests()

	if total.Requests.CPU != 750 {
		t.Errorf("Total CPU requests = %v, want 750", total.Requests.CPU)
	}
	wantMemory := int64(384 * 1024 * 1024)
	if total.Requests.Memory != wantMemory {
		t.Errorf("Total Memory requests = %v, want %v", total.Requests.Memory, wantMemory)
	}
}

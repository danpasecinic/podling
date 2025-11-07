package services

import (
	"testing"
	"time"

	"github.com/danpasecinic/podling/internal/master/state"
	"github.com/danpasecinic/podling/internal/types"
)

func TestClusterIPAllocator(t *testing.T) {
	allocator := NewClusterIPAllocator("10.96.0.0/16")

	ip1, err := allocator.Allocate()
	if err != nil {
		t.Fatalf("Failed to allocate first IP: %v", err)
	}
	if ip1 == "" {
		t.Error("Expected non-empty IP")
	}

	ip2, err := allocator.Allocate()
	if err != nil {
		t.Fatalf("Failed to allocate second IP: %v", err)
	}
	if ip2 == ip1 {
		t.Errorf("Expected different IP, got same: %s", ip2)
	}

	err = allocator.Release(ip1)
	if err != nil {
		t.Fatalf("Failed to release IP: %v", err)
	}

	ip3, err := allocator.Allocate()
	if err != nil {
		t.Fatalf("Failed to allocate after release: %v", err)
	}
	if ip3 == ip2 {
		t.Error("Should not allocate already-allocated IP")
	}

	err = allocator.Release(ip1)
	if err == nil {
		t.Error("Expected error when releasing already-released IP")
	}

	err = allocator.Release("10.96.1.254")
	if err == nil {
		t.Error("Expected error when releasing never-allocated IP")
	}
}

func TestClusterIPAllocatorExhaustion(t *testing.T) {
	allocator := NewClusterIPAllocator("10.96.0.0/30")

	allocatedIPs := make([]string, 0)
	for i := 0; i < 10; i++ {
		ip, err := allocator.Allocate()
		if err != nil {
			if len(allocatedIPs) == 0 {
				t.Fatalf("Failed to allocate any IPs: %v", err)
			}
			break
		}
		allocatedIPs = append(allocatedIPs, ip)
	}

	if len(allocatedIPs) < 2 {
		t.Errorf("Expected to allocate at least 2 IPs, got %d", len(allocatedIPs))
	}

	err := allocator.Release(allocatedIPs[0])
	if err != nil {
		t.Fatalf("Failed to release IP: %v", err)
	}

	ip, err := allocator.Allocate()
	if err != nil {
		t.Fatalf("Failed to allocate after release: %v", err)
	}
	if ip == "" {
		t.Error("Expected non-empty IP after release")
	}
}

func TestNextIP(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"10.96.0.1", "10.96.0.2"},
		{"10.96.0.255", "10.96.1.0"},
		{"10.96.255.255", "10.97.0.0"},
	}

	for _, tt := range tests {
		t.Run(
			tt.input, func(t *testing.T) {
				var ip []byte
				switch tt.input {
				case "10.96.0.1":
					ip = []byte{10, 96, 0, 1}
				case "10.96.0.255":
					ip = []byte{10, 96, 0, 255}
				case "10.96.255.255":
					ip = []byte{10, 96, 255, 255}
				}

				next := nextIP(ip)
				expected := tt.expected
				got := next.String()

				if got != expected {
					t.Errorf("Expected %s, got %s", expected, got)
				}
			},
		)
	}
}

func TestEndpointControllerGetPodIP(t *testing.T) {
	store := state.NewInMemoryStore()
	ec := NewEndpointController(store)

	pod := types.Pod{
		PodID:     "pod-1",
		Name:      "test-pod",
		Namespace: "default",
	}

	ip := ec.getPodIP(pod)
	if ip != "" {
		t.Errorf("Expected empty IP for pod without annotations, got %s", ip)
	}

	pod.Annotations = map[string]string{
		"podling.io/pod-ip": "172.17.0.2",
	}

	ip = ec.getPodIP(pod)
	if ip != "172.17.0.2" {
		t.Errorf("Expected IP 172.17.0.2, got %s", ip)
	}

	pod.Annotations = map[string]string{
		"other": "annotation",
	}

	ip = ec.getPodIP(pod)
	if ip != "" {
		t.Errorf("Expected empty IP for pod without IP annotation, got %s", ip)
	}
}

func TestEndpointControllerIsPodReady(t *testing.T) {
	store := state.NewInMemoryStore()
	ec := NewEndpointController(store)

	pod := types.Pod{
		Status: types.PodRunning,
		Containers: []types.Container{
			{
				Name:   "nginx",
				Status: types.ContainerRunning,
			},
			{
				Name:   "sidecar",
				Status: types.ContainerRunning,
			},
		},
	}

	if !ec.isPodReady(pod) {
		t.Error("Expected pod to be ready with all containers running")
	}

	pod.Containers[1].Status = types.ContainerWaiting

	if ec.isPodReady(pod) {
		t.Error("Expected pod to not be ready with waiting container")
	}

	pod.Status = types.PodPending
	pod.Containers[1].Status = types.ContainerRunning

	if ec.isPodReady(pod) {
		t.Error("Expected pod to not be ready when not in running state")
	}

	pod.Status = types.PodRunning
	pod.Containers[0].ReadinessProbe = &types.HealthCheck{
		Type: "http",
	}
	pod.Containers[0].HealthStatus = types.HealthStatusUnhealthy

	if ec.isPodReady(pod) {
		t.Error("Expected pod to not be ready with unhealthy readiness probe")
	}

	pod.Containers[0].HealthStatus = types.HealthStatusHealthy

	if !ec.isPodReady(pod) {
		t.Error("Expected pod to be ready with healthy readiness probe")
	}
}

func TestEndpointControllerBuildEndpoints(t *testing.T) {
	store := state.NewInMemoryStore()
	ec := NewEndpointController(store)

	service := types.Service{
		ServiceID: "svc-1",
		Name:      "web-service",
		Namespace: "default",
		Selector:  map[string]string{"app": "nginx"},
		Ports: []types.ServicePort{
			{
				Name:       "http",
				Port:       80,
				TargetPort: 8080,
				Protocol:   "TCP",
			},
		},
	}

	pods := []types.Pod{
		{
			PodID:     "pod-1",
			Name:      "web-1",
			Namespace: "default",
			Status:    types.PodRunning,
			NodeID:    "node-1",
			Annotations: map[string]string{
				"podling.io/pod-ip": "172.17.0.2",
			},
			Containers: []types.Container{
				{
					Name:   "nginx",
					Status: types.ContainerRunning,
				},
			},
		},
		{
			PodID:     "pod-2",
			Name:      "web-2",
			Namespace: "default",
			Status:    types.PodRunning,
			NodeID:    "node-1",
			Annotations: map[string]string{
				"podling.io/pod-ip": "172.17.0.3",
			},
			Containers: []types.Container{
				{
					Name:   "nginx",
					Status: types.ContainerRunning,
				},
			},
		},
		{
			PodID:     "pod-3",
			Name:      "web-3",
			Namespace: "default",
			Status:    types.PodRunning,
			NodeID:    "node-1",
			Annotations: map[string]string{
				"podling.io/pod-ip": "172.17.0.4",
			},
			Containers: []types.Container{
				{
					Name:           "nginx",
					Status:         types.ContainerRunning,
					ReadinessProbe: &types.HealthCheck{Type: "http"},
					HealthStatus:   types.HealthStatusUnhealthy, // Not ready
				},
			},
		},
	}

	endpoints := ec.buildEndpoints(service, pods)

	if endpoints.ServiceID != service.ServiceID {
		t.Errorf("Expected service ID %s, got %s", service.ServiceID, endpoints.ServiceID)
	}

	if len(endpoints.Subsets) != 1 {
		t.Fatalf("Expected 1 subset, got %d", len(endpoints.Subsets))
	}

	subset := endpoints.Subsets[0]

	if len(subset.Addresses) != 2 {
		t.Errorf("Expected 2 ready addresses, got %d", len(subset.Addresses))
	}

	if len(subset.NotReadyAddresses) != 1 {
		t.Errorf("Expected 1 not-ready address, got %d", len(subset.NotReadyAddresses))
	}

	if len(subset.Ports) != 1 {
		t.Errorf("Expected 1 port, got %d", len(subset.Ports))
	}

	if subset.Ports[0].Port != 8080 {
		t.Errorf("Expected target port 8080, got %d", subset.Ports[0].Port)
	}
}

func TestEndpointControllerSyncServiceEndpoints(t *testing.T) {
	store := state.NewInMemoryStore()
	ec := NewEndpointController(store)

	service := types.Service{
		ServiceID: "svc-1",
		Name:      "web-service",
		Namespace: "default",
		Type:      types.ServiceTypeClusterIP,
		ClusterIP: "10.96.0.1",
		Selector: map[string]string{
			"app": "nginx",
		},
		Ports: []types.ServicePort{
			{Port: 80, TargetPort: 8080, Protocol: "TCP"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := store.AddService(service)
	if err != nil {
		t.Fatalf("Failed to add service: %v", err)
	}

	pod := types.Pod{
		PodID:     "pod-1",
		Name:      "web-1",
		Namespace: "default",
		Labels: map[string]string{
			"app": "nginx",
		},
		Status: types.PodRunning,
		NodeID: "node-1",
		Annotations: map[string]string{
			"podling.io/pod-ip": "172.17.0.2",
		},
		Containers: []types.Container{
			{
				Name:   "nginx",
				Status: types.ContainerRunning,
			},
		},
		CreatedAt: time.Now(),
	}

	err = store.AddPod(pod)
	if err != nil {
		t.Fatalf("Failed to add pod: %v", err)
	}

	err = ec.syncServiceEndpoints(service)
	if err != nil {
		t.Fatalf("Failed to sync service endpoints: %v", err)
	}

	endpoints, err := store.GetEndpoints("svc-1")
	if err != nil {
		t.Fatalf("Failed to get endpoints: %v", err)
	}

	if !endpoints.HasEndpoints() {
		t.Error("Expected endpoints to have addresses")
	}

	if len(endpoints.Subsets) == 0 {
		t.Fatal("Expected at least one subset")
	}

	if len(endpoints.Subsets[0].Addresses) != 1 {
		t.Errorf("Expected 1 ready address, got %d", len(endpoints.Subsets[0].Addresses))
	}

	if endpoints.Subsets[0].Addresses[0].IP != "172.17.0.2" {
		t.Errorf("Expected IP 172.17.0.2, got %s", endpoints.Subsets[0].Addresses[0].IP)
	}
}

func TestEndpointControllerAllocateReleaseClusterIP(t *testing.T) {
	store := state.NewInMemoryStore()
	ec := NewEndpointController(store)

	ip, err := ec.AllocateClusterIP()
	if err != nil {
		t.Fatalf("Failed to allocate ClusterIP: %v", err)
	}

	if ip == "" {
		t.Error("Expected non-empty ClusterIP")
	}

	err = ec.ReleaseClusterIP(ip)
	if err != nil {
		t.Fatalf("Failed to release ClusterIP: %v", err)
	}

	err = ec.ReleaseClusterIP(ip)
	if err == nil {
		t.Error("Expected error when releasing already-released IP")
	}
}

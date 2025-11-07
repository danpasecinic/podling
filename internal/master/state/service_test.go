package state

import (
	"testing"
	"time"

	"github.com/danpasecinic/podling/internal/types"
)

func TestAddAndGetService(t *testing.T) {
	store := NewInMemoryStore()

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
			{
				Port:       80,
				TargetPort: 8080,
				Protocol:   "TCP",
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Test adding service
	err := store.AddService(service)
	if err != nil {
		t.Fatalf("Failed to add service: %v", err)
	}

	// Test retrieving service
	retrieved, err := store.GetService("svc-1")
	if err != nil {
		t.Fatalf("Failed to get service: %v", err)
	}

	if retrieved.ServiceID != service.ServiceID {
		t.Errorf("Expected service ID %s, got %s", service.ServiceID, retrieved.ServiceID)
	}
	if retrieved.Name != service.Name {
		t.Errorf("Expected name %s, got %s", service.Name, retrieved.Name)
	}
	if retrieved.ClusterIP != service.ClusterIP {
		t.Errorf("Expected ClusterIP %s, got %s", service.ClusterIP, retrieved.ClusterIP)
	}
	if len(retrieved.Ports) != 1 {
		t.Errorf("Expected 1 port, got %d", len(retrieved.Ports))
	}
}

func TestAddDuplicateService(t *testing.T) {
	store := NewInMemoryStore()

	service := types.Service{
		ServiceID: "svc-1",
		Name:      "web-service",
		Namespace: "default",
		Type:      types.ServiceTypeClusterIP,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := store.AddService(service)
	if err != nil {
		t.Fatalf("Failed to add service first time: %v", err)
	}

	// Try to add the same service again
	err = store.AddService(service)
	if err != ErrServiceAlreadyExists {
		t.Errorf("Expected ErrServiceAlreadyExists, got %v", err)
	}
}

func TestGetNonexistentService(t *testing.T) {
	store := NewInMemoryStore()

	_, err := store.GetService("nonexistent")
	if err != ErrServiceNotFound {
		t.Errorf("Expected ErrServiceNotFound, got %v", err)
	}
}

func TestGetServiceByName(t *testing.T) {
	store := NewInMemoryStore()

	service := types.Service{
		ServiceID: "svc-1",
		Name:      "web-service",
		Namespace: "production",
		Type:      types.ServiceTypeClusterIP,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := store.AddService(service)
	if err != nil {
		t.Fatalf("Failed to add service: %v", err)
	}

	// Test retrieving by name and namespace
	retrieved, err := store.GetServiceByName("production", "web-service")
	if err != nil {
		t.Fatalf("Failed to get service by name: %v", err)
	}

	if retrieved.ServiceID != service.ServiceID {
		t.Errorf("Expected service ID %s, got %s", service.ServiceID, retrieved.ServiceID)
	}

	// Test default namespace normalization
	service2 := types.Service{
		ServiceID: "svc-2",
		Name:      "api-service",
		Namespace: "", // Should default to "default"
		Type:      types.ServiceTypeClusterIP,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = store.AddService(service2)
	if err != nil {
		t.Fatalf("Failed to add service2: %v", err)
	}

	retrieved, err = store.GetServiceByName("default", "api-service")
	if err != nil {
		t.Fatalf("Failed to get service by default namespace: %v", err)
	}

	if retrieved.ServiceID != service2.ServiceID {
		t.Errorf("Expected service ID %s, got %s", service2.ServiceID, retrieved.ServiceID)
	}
}

func TestUpdateService(t *testing.T) {
	store := NewInMemoryStore()

	service := types.Service{
		ServiceID: "svc-1",
		Name:      "web-service",
		Namespace: "default",
		Type:      types.ServiceTypeClusterIP,
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

	// Update selector
	newSelector := map[string]string{
		"app":  "nginx",
		"tier": "frontend",
	}
	newPorts := []types.ServicePort{
		{Port: 80, TargetPort: 8080, Protocol: "TCP"},
		{Port: 443, TargetPort: 8443, Protocol: "TCP"},
	}

	updates := types.ServiceUpdate{
		Selector: &newSelector,
		Ports:    &newPorts,
	}

	err = store.UpdateService("svc-1", updates)
	if err != nil {
		t.Fatalf("Failed to update service: %v", err)
	}

	// Verify updates
	retrieved, err := store.GetService("svc-1")
	if err != nil {
		t.Fatalf("Failed to get updated service: %v", err)
	}

	if len(retrieved.Selector) != 2 {
		t.Errorf("Expected 2 selector labels, got %d", len(retrieved.Selector))
	}
	if retrieved.Selector["tier"] != "frontend" {
		t.Errorf("Expected tier=frontend, got %v", retrieved.Selector["tier"])
	}
	if len(retrieved.Ports) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(retrieved.Ports))
	}
}

func TestListServices(t *testing.T) {
	store := NewInMemoryStore()

	// Add services in different namespaces
	services := []types.Service{
		{
			ServiceID: "svc-1",
			Name:      "web-1",
			Namespace: "production",
			Type:      types.ServiceTypeClusterIP,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ServiceID: "svc-2",
			Name:      "web-2",
			Namespace: "production",
			Type:      types.ServiceTypeClusterIP,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ServiceID: "svc-3",
			Name:      "api",
			Namespace: "staging",
			Type:      types.ServiceTypeClusterIP,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, svc := range services {
		err := store.AddService(svc)
		if err != nil {
			t.Fatalf("Failed to add service %s: %v", svc.ServiceID, err)
		}
	}

	// List all services (empty namespace)
	allServices, err := store.ListServices("")
	if err != nil {
		t.Fatalf("Failed to list all services: %v", err)
	}
	if len(allServices) != 3 {
		t.Errorf("Expected 3 services, got %d", len(allServices))
	}

	// List services in production namespace
	prodServices, err := store.ListServices("production")
	if err != nil {
		t.Fatalf("Failed to list production services: %v", err)
	}
	if len(prodServices) != 2 {
		t.Errorf("Expected 2 production services, got %d", len(prodServices))
	}

	// List services in staging namespace
	stagingServices, err := store.ListServices("staging")
	if err != nil {
		t.Fatalf("Failed to list staging services: %v", err)
	}
	if len(stagingServices) != 1 {
		t.Errorf("Expected 1 staging service, got %d", len(stagingServices))
	}
}

func TestDeleteService(t *testing.T) {
	store := NewInMemoryStore()

	service := types.Service{
		ServiceID: "svc-1",
		Name:      "web-service",
		Namespace: "default",
		Type:      types.ServiceTypeClusterIP,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := store.AddService(service)
	if err != nil {
		t.Fatalf("Failed to add service: %v", err)
	}

	// Delete service
	err = store.DeleteService("svc-1")
	if err != nil {
		t.Fatalf("Failed to delete service: %v", err)
	}

	// Verify deletion
	_, err = store.GetService("svc-1")
	if err != ErrServiceNotFound {
		t.Errorf("Expected ErrServiceNotFound after deletion, got %v", err)
	}

	// Try to delete non-existent service
	err = store.DeleteService("nonexistent")
	if err != ErrServiceNotFound {
		t.Errorf("Expected ErrServiceNotFound for nonexistent service, got %v", err)
	}
}

func TestSetAndGetEndpoints(t *testing.T) {
	store := NewInMemoryStore()

	endpoints := types.Endpoints{
		ServiceID:   "svc-1",
		ServiceName: "web-service",
		Namespace:   "default",
		Subsets: []types.EndpointSubset{
			{
				Addresses: []types.EndpointAddress{
					{IP: "172.17.0.2", PodID: "pod-1", NodeID: "node-1"},
					{IP: "172.17.0.3", PodID: "pod-2", NodeID: "node-1"},
				},
				Ports: []types.EndpointPort{
					{Port: 8080, Protocol: "TCP"},
				},
			},
		},
	}

	// Set endpoints
	err := store.SetEndpoints(endpoints)
	if err != nil {
		t.Fatalf("Failed to set endpoints: %v", err)
	}

	// Get endpoints
	retrieved, err := store.GetEndpoints("svc-1")
	if err != nil {
		t.Fatalf("Failed to get endpoints: %v", err)
	}

	if retrieved.ServiceID != endpoints.ServiceID {
		t.Errorf("Expected service ID %s, got %s", endpoints.ServiceID, retrieved.ServiceID)
	}
	if len(retrieved.Subsets) != 1 {
		t.Errorf("Expected 1 subset, got %d", len(retrieved.Subsets))
	}
	if len(retrieved.Subsets[0].Addresses) != 2 {
		t.Errorf("Expected 2 addresses, got %d", len(retrieved.Subsets[0].Addresses))
	}
}

func TestGetEndpointsByServiceName(t *testing.T) {
	store := NewInMemoryStore()

	endpoints := types.Endpoints{
		ServiceID:   "svc-1",
		ServiceName: "web-service",
		Namespace:   "production",
		Subsets: []types.EndpointSubset{
			{
				Addresses: []types.EndpointAddress{
					{IP: "172.17.0.2", PodID: "pod-1"},
				},
				Ports: []types.EndpointPort{
					{Port: 8080, Protocol: "TCP"},
				},
			},
		},
	}

	err := store.SetEndpoints(endpoints)
	if err != nil {
		t.Fatalf("Failed to set endpoints: %v", err)
	}

	// Get by name and namespace
	retrieved, err := store.GetEndpointsByServiceName("production", "web-service")
	if err != nil {
		t.Fatalf("Failed to get endpoints by name: %v", err)
	}

	if retrieved.ServiceID != endpoints.ServiceID {
		t.Errorf("Expected service ID %s, got %s", endpoints.ServiceID, retrieved.ServiceID)
	}
}

func TestDeleteEndpoints(t *testing.T) {
	store := NewInMemoryStore()

	endpoints := types.Endpoints{
		ServiceID:   "svc-1",
		ServiceName: "web-service",
		Namespace:   "default",
		Subsets:     []types.EndpointSubset{},
	}

	err := store.SetEndpoints(endpoints)
	if err != nil {
		t.Fatalf("Failed to set endpoints: %v", err)
	}

	// Delete endpoints
	err = store.DeleteEndpoints("svc-1")
	if err != nil {
		t.Fatalf("Failed to delete endpoints: %v", err)
	}

	// Verify deletion
	_, err = store.GetEndpoints("svc-1")
	if err != ErrEndpointsNotFound {
		t.Errorf("Expected ErrEndpointsNotFound after deletion, got %v", err)
	}
}

func TestListPodsByLabels(t *testing.T) {
	store := NewInMemoryStore()

	// Add pods with different labels
	pods := []types.Pod{
		{
			PodID:     "pod-1",
			Name:      "web-1",
			Namespace: "default",
			Labels: map[string]string{
				"app":  "nginx",
				"tier": "frontend",
			},
			Status:    types.PodRunning,
			CreatedAt: time.Now(),
		},
		{
			PodID:     "pod-2",
			Name:      "web-2",
			Namespace: "default",
			Labels: map[string]string{
				"app":  "nginx",
				"tier": "frontend",
			},
			Status:    types.PodRunning,
			CreatedAt: time.Now(),
		},
		{
			PodID:     "pod-3",
			Name:      "api-1",
			Namespace: "default",
			Labels: map[string]string{
				"app":  "api",
				"tier": "backend",
			},
			Status:    types.PodRunning,
			CreatedAt: time.Now(),
		},
		{
			PodID:     "pod-4",
			Name:      "web-3",
			Namespace: "production",
			Labels: map[string]string{
				"app":  "nginx",
				"tier": "frontend",
			},
			Status:    types.PodRunning,
			CreatedAt: time.Now(),
		},
	}

	for _, pod := range pods {
		err := store.AddPod(pod)
		if err != nil {
			t.Fatalf("Failed to add pod %s: %v", pod.PodID, err)
		}
	}

	// Test label matching in default namespace
	selector := map[string]string{"app": "nginx"}
	matchingPods, err := store.ListPodsByLabels("default", selector)
	if err != nil {
		t.Fatalf("Failed to list pods by labels: %v", err)
	}
	if len(matchingPods) != 2 {
		t.Errorf("Expected 2 pods matching app=nginx in default, got %d", len(matchingPods))
	}

	// Test multiple label matching
	selector = map[string]string{"app": "nginx", "tier": "frontend"}
	matchingPods, err = store.ListPodsByLabels("default", selector)
	if err != nil {
		t.Fatalf("Failed to list pods by labels: %v", err)
	}
	if len(matchingPods) != 2 {
		t.Errorf("Expected 2 pods matching app=nginx,tier=frontend in default, got %d", len(matchingPods))
	}

	// Test namespace filtering
	matchingPods, err = store.ListPodsByLabels("production", selector)
	if err != nil {
		t.Fatalf("Failed to list pods by labels: %v", err)
	}
	if len(matchingPods) != 1 {
		t.Errorf("Expected 1 pod matching in production, got %d", len(matchingPods))
	}

	// Test non-matching selector
	selector = map[string]string{"app": "nonexistent"}
	matchingPods, err = store.ListPodsByLabels("default", selector)
	if err != nil {
		t.Fatalf("Failed to list pods by labels: %v", err)
	}
	if len(matchingPods) != 0 {
		t.Errorf("Expected 0 pods matching nonexistent app, got %d", len(matchingPods))
	}
}

func TestEndpointsHasEndpoints(t *testing.T) {
	// Test with ready endpoints
	endpoints := types.Endpoints{
		Subsets: []types.EndpointSubset{
			{
				Addresses: []types.EndpointAddress{
					{IP: "172.17.0.2"},
				},
			},
		},
	}
	if !endpoints.HasEndpoints() {
		t.Error("Expected HasEndpoints() to return true")
	}

	// Test with no endpoints
	endpoints = types.Endpoints{
		Subsets: []types.EndpointSubset{},
	}
	if endpoints.HasEndpoints() {
		t.Error("Expected HasEndpoints() to return false")
	}

	// Test with not-ready endpoints only
	endpoints = types.Endpoints{
		Subsets: []types.EndpointSubset{
			{
				NotReadyAddresses: []types.EndpointAddress{
					{IP: "172.17.0.2"},
				},
			},
		},
	}
	if endpoints.HasEndpoints() {
		t.Error("Expected HasEndpoints() to return false for not-ready only")
	}
}

func TestServiceGetDNSName(t *testing.T) {
	tests := []struct {
		name     string
		service  types.Service
		expected string
	}{
		{
			name: "service with namespace",
			service: types.Service{
				Name:      "web",
				Namespace: "production",
			},
			expected: "web.production.svc.cluster.local",
		},
		{
			name: "service without namespace (defaults to default)",
			service: types.Service{
				Name:      "api",
				Namespace: "",
			},
			expected: "api.default.svc.cluster.local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.service.GetDNSName()
			if got != tt.expected {
				t.Errorf("Expected DNS name %s, got %s", tt.expected, got)
			}
		})
	}
}

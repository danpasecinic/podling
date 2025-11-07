package types

import (
	"testing"
)

func TestService_GetPortByName(t *testing.T) {
	service := Service{
		Ports: []ServicePort{
			{Name: "http", Port: 80, TargetPort: 8080, Protocol: "TCP"},
			{Name: "https", Port: 443, TargetPort: 8443, Protocol: "TCP"},
			{Name: "metrics", Port: 9090, TargetPort: 9090, Protocol: "TCP"},
		},
	}

	port := service.GetPortByName("http")
	if port == nil {
		t.Fatal("Expected to find port 'http'")
	}
	if port.Port != 80 {
		t.Errorf("Expected port 80, got %d", port.Port)
	}
	if port.TargetPort != 8080 {
		t.Errorf("Expected target port 8080, got %d", port.TargetPort)
	}

	port = service.GetPortByName("https")
	if port == nil {
		t.Fatal("Expected to find port 'https'")
	}
	if port.Port != 443 {
		t.Errorf("Expected port 443, got %d", port.Port)
	}

	port = service.GetPortByName("nonexistent")
	if port != nil {
		t.Error("Expected not to find nonexistent port")
	}
}

func TestService_GetDNSName(t *testing.T) {
	tests := []struct {
		name     string
		service  Service
		expected string
	}{
		{
			name: "service with namespace",
			service: Service{
				Name:      "web-service",
				Namespace: "production",
			},
			expected: "web-service.production.svc.cluster.local",
		},
		{
			name: "service with default namespace",
			service: Service{
				Name:      "api-service",
				Namespace: "default",
			},
			expected: "api-service.default.svc.cluster.local",
		},
		{
			name: "service with empty namespace",
			service: Service{
				Name:      "db-service",
				Namespace: "",
			},
			expected: "db-service.default.svc.cluster.local",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				dns := tt.service.GetDNSName()
				if dns != tt.expected {
					t.Errorf("GetDNSName() = %v, want %v", dns, tt.expected)
				}
			},
		)
	}
}

func TestEndpoints_HasEndpoints(t *testing.T) {
	tests := []struct {
		name      string
		endpoints Endpoints
		want      bool
	}{
		{
			name: "endpoints with addresses",
			endpoints: Endpoints{
				Subsets: []EndpointSubset{
					{
						Addresses: []EndpointAddress{
							{IP: "192.168.1.1", PodID: "pod-1"},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "endpoints with only not ready addresses",
			endpoints: Endpoints{
				Subsets: []EndpointSubset{
					{
						NotReadyAddresses: []EndpointAddress{
							{IP: "192.168.1.2", PodID: "pod-2"},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "endpoints with both ready and not ready",
			endpoints: Endpoints{
				Subsets: []EndpointSubset{
					{
						Addresses: []EndpointAddress{
							{IP: "192.168.1.1", PodID: "pod-1"},
						},
						NotReadyAddresses: []EndpointAddress{
							{IP: "192.168.1.2", PodID: "pod-2"},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "endpoints with empty subsets",
			endpoints: Endpoints{
				Subsets: []EndpointSubset{
					{},
				},
			},
			want: false,
		},
		{
			name: "endpoints with no subsets",
			endpoints: Endpoints{
				Subsets: []EndpointSubset{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if got := tt.endpoints.HasEndpoints(); got != tt.want {
					t.Errorf("HasEndpoints() = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

func TestEndpoints_GetAllIPs(t *testing.T) {
	tests := []struct {
		name      string
		endpoints Endpoints
		want      []string
	}{
		{
			name: "single subset with ready addresses",
			endpoints: Endpoints{
				Subsets: []EndpointSubset{
					{
						Addresses: []EndpointAddress{
							{IP: "192.168.1.1"},
							{IP: "192.168.1.2"},
						},
					},
				},
			},
			want: []string{"192.168.1.1", "192.168.1.2"},
		},
		{
			name: "multiple subsets",
			endpoints: Endpoints{
				Subsets: []EndpointSubset{
					{
						Addresses: []EndpointAddress{
							{IP: "192.168.1.1"},
						},
					},
					{
						Addresses: []EndpointAddress{
							{IP: "192.168.1.2"},
							{IP: "192.168.1.3"},
						},
					},
				},
			},
			want: []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"},
		},
		{
			name: "only ready addresses (not ready excluded)",
			endpoints: Endpoints{
				Subsets: []EndpointSubset{
					{
						Addresses: []EndpointAddress{
							{IP: "192.168.1.1"},
						},
						NotReadyAddresses: []EndpointAddress{
							{IP: "192.168.1.2"},
						},
					},
				},
			},
			want: []string{"192.168.1.1"},
		},
		{
			name:      "empty endpoints",
			endpoints: Endpoints{},
			want:      []string{},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				got := tt.endpoints.GetAllIPs()
				if len(got) != len(tt.want) {
					t.Errorf("GetAllIPs() length = %v, want %v", len(got), len(tt.want))
					return
				}
				for i, ip := range got {
					if ip != tt.want[i] {
						t.Errorf("GetAllIPs()[%d] = %v, want %v", i, ip, tt.want[i])
					}
				}
			},
		)
	}
}

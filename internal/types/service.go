package types

import "time"

// ServiceType defines the type of service exposure
type ServiceType string

const (
	// ServiceTypeClusterIP exposes the service on a cluster-internal IP
	// This is the default service type
	ServiceTypeClusterIP ServiceType = "ClusterIP"

	// ServiceTypeNodePort exposes the service on each node's IP at a static port
	// A ClusterIP service is automatically created
	ServiceTypeNodePort ServiceType = "NodePort"

	// ServiceTypeLoadBalancer exposes the service externally using a load balancer
	// NodePort and ClusterIP services are automatically created
	ServiceTypeLoadBalancer ServiceType = "LoadBalancer"
)

// Service represents a stable endpoint for a set of pods
// Similar to Kubernetes Services, it provides service discovery and load balancing
type Service struct {
	// ServiceID is the unique identifier for the service
	ServiceID string `json:"serviceId"`

	// Name is a human-readable name for the service
	Name string `json:"name"`

	// Namespace is the logical grouping for the service
	Namespace string `json:"namespace,omitempty"`

	// Type determines how the service is exposed
	Type ServiceType `json:"type"`

	// ClusterIP is the virtual IP allocated to this service
	// Only valid for ClusterIP and derived types
	ClusterIP string `json:"clusterIp,omitempty"`

	// Selector is a label query to identify pods that belong to this service
	// Pods matching all labels in the selector will receive traffic
	Selector map[string]string `json:"selector,omitempty"`

	// Ports are the ports exposed by this service
	Ports []ServicePort `json:"ports"`

	// Labels are key-value pairs for organizing and selecting services
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are key-value pairs for storing arbitrary metadata
	Annotations map[string]string `json:"annotations,omitempty"`

	// SessionAffinity determines if traffic should stick to the same pod
	// Valid values: "None" (default), "ClientIP"
	SessionAffinity string `json:"sessionAffinity,omitempty"`

	// CreatedAt is when the service was created
	CreatedAt time.Time `json:"createdAt"`

	// UpdatedAt is when the service was last modified
	UpdatedAt time.Time `json:"updatedAt"`
}

// ServicePort represents a port exposed by a service
type ServicePort struct {
	// Name is an optional name for the port (must be unique within service)
	Name string `json:"name,omitempty"`

	// Protocol is the network protocol (TCP or UDP)
	Protocol string `json:"protocol,omitempty"` // Default: TCP

	// Port is the port that will be exposed by this service
	Port int `json:"port"`

	// TargetPort is the port to access on the pods selected by this service
	// If not specified, defaults to Port
	TargetPort int `json:"targetPort,omitempty"`

	// NodePort is the port on each node (only for NodePort/LoadBalancer types)
	NodePort int `json:"nodePort,omitempty"`
}

// Endpoints represents the actual pod IPs and ports backing a service
type Endpoints struct {
	// ServiceID is the service this endpoints object belongs to
	ServiceID string `json:"serviceId"`

	// ServiceName is the name of the service
	ServiceName string `json:"serviceName"`

	// Namespace is the namespace of the service
	Namespace string `json:"namespace"`

	// Subsets contains the actual endpoint addresses and ports
	Subsets []EndpointSubset `json:"subsets"`

	// UpdatedAt is when the endpoints were last updated
	UpdatedAt time.Time `json:"updatedAt"`
}

// EndpointSubset is a group of addresses with a common set of ports
type EndpointSubset struct {
	// Addresses are the IP addresses of pods that are ready
	Addresses []EndpointAddress `json:"addresses"`

	// NotReadyAddresses are IP addresses of pods that are not ready
	NotReadyAddresses []EndpointAddress `json:"notReadyAddresses,omitempty"`

	// Ports are the ports available on the endpoint addresses
	Ports []EndpointPort `json:"ports"`
}

// EndpointAddress represents a single IP address
type EndpointAddress struct {
	// IP is the IP address of the endpoint
	IP string `json:"ip"`

	// PodID is the reference to the pod
	PodID string `json:"podId"`

	// NodeID is the reference to the node hosting the pod
	NodeID string `json:"nodeId,omitempty"`
}

// EndpointPort represents a port on an endpoint
type EndpointPort struct {
	// Name is the name of the port (matches ServicePort.Name)
	Name string `json:"name,omitempty"`

	// Port is the port number
	Port int `json:"port"`

	// Protocol is the network protocol
	Protocol string `json:"protocol,omitempty"` // Default: TCP
}

// ServiceUpdate represents partial updates to a service
type ServiceUpdate struct {
	Selector        *map[string]string `json:"selector,omitempty"`
	Ports           *[]ServicePort     `json:"ports,omitempty"`
	Labels          *map[string]string `json:"labels,omitempty"`
	Annotations     *map[string]string `json:"annotations,omitempty"`
	SessionAffinity *string            `json:"sessionAffinity,omitempty"`
}

// GetPortByName returns a service port by name
func (s *Service) GetPortByName(name string) *ServicePort {
	for i := range s.Ports {
		if s.Ports[i].Name == name {
			return &s.Ports[i]
		}
	}
	return nil
}

// GetDNSName returns the fully qualified DNS name for this service
// Format: <service-name>.<namespace>.svc.cluster.local
func (s *Service) GetDNSName() string {
	namespace := s.Namespace
	if namespace == "" {
		namespace = "default"
	}
	return s.Name + "." + namespace + ".svc.cluster.local"
}

// HasEndpoints returns true if there are any ready endpoints
func (e *Endpoints) HasEndpoints() bool {
	for _, subset := range e.Subsets {
		if len(subset.Addresses) > 0 {
			return true
		}
	}
	return false
}

// GetAllIPs returns all ready IP addresses across all subsets
func (e *Endpoints) GetAllIPs() []string {
	var ips []string
	for _, subset := range e.Subsets {
		for _, addr := range subset.Addresses {
			ips = append(ips, addr.IP)
		}
	}
	return ips
}

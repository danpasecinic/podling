package types

import "time"

type ServiceType string

const (
	ServiceTypeClusterIP    ServiceType = "ClusterIP"
	ServiceTypeNodePort     ServiceType = "NodePort"
	ServiceTypeLoadBalancer ServiceType = "LoadBalancer"
)

type Service struct {
	ServiceID       string            `json:"serviceId"`
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace,omitempty"`
	Type            ServiceType       `json:"type"`
	ClusterIP       string            `json:"clusterIp,omitempty"`
	Selector        map[string]string `json:"selector,omitempty"`
	Ports           []ServicePort     `json:"ports"`
	Labels          map[string]string `json:"labels,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
	SessionAffinity string            `json:"sessionAffinity,omitempty"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
}

type ServicePort struct {
	Name       string `json:"name,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
	Port       int    `json:"port"`
	TargetPort int    `json:"targetPort,omitempty"`
	NodePort   int    `json:"nodePort,omitempty"`
}

type Endpoints struct {
	ServiceID   string           `json:"serviceId"`
	ServiceName string           `json:"serviceName"`
	Namespace   string           `json:"namespace"`
	Subsets     []EndpointSubset `json:"subsets"`
	UpdatedAt   time.Time        `json:"updatedAt"`
}

type EndpointSubset struct {
	Addresses         []EndpointAddress `json:"addresses"`
	NotReadyAddresses []EndpointAddress `json:"notReadyAddresses,omitempty"`
	Ports             []EndpointPort    `json:"ports"`
}

type EndpointAddress struct {
	IP     string `json:"ip"`
	PodID  string `json:"podId"`
	NodeID string `json:"nodeId,omitempty"`
}

type EndpointPort struct {
	Name     string `json:"name,omitempty"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol,omitempty"`
}

type ServiceUpdate struct {
	Selector        *map[string]string `json:"selector,omitempty"`
	Ports           *[]ServicePort     `json:"ports,omitempty"`
	Labels          *map[string]string `json:"labels,omitempty"`
	Annotations     *map[string]string `json:"annotations,omitempty"`
	SessionAffinity *string            `json:"sessionAffinity,omitempty"`
}

func (s *Service) GetPortByName(name string) *ServicePort {
	for i := range s.Ports {
		if s.Ports[i].Name == name {
			return &s.Ports[i]
		}
	}
	return nil
}

func (s *Service) GetDNSName() string {
	namespace := s.Namespace
	if namespace == "" {
		namespace = "default"
	}
	return s.Name + "." + namespace + ".svc.cluster.local"
}

func (e *Endpoints) HasEndpoints() bool {
	for _, subset := range e.Subsets {
		if len(subset.Addresses) > 0 {
			return true
		}
	}
	return false
}

func (e *Endpoints) GetAllIPs() []string {
	var ips []string
	for _, subset := range e.Subsets {
		for _, addr := range subset.Addresses {
			ips = append(ips, addr.IP)
		}
	}
	return ips
}

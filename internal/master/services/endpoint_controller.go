package services

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/danpasecinic/podling/internal/master/state"
	"github.com/danpasecinic/podling/internal/types"
)

// EndpointController watches for pod changes and updates service endpoints
type EndpointController struct {
	store        state.StateStore
	mu           sync.RWMutex
	stopChan     chan struct{}
	syncInterval time.Duration
	ipAllocator  *ClusterIPAllocator
}

// NewEndpointController creates a new endpoint controller
func NewEndpointController(store state.StateStore) *EndpointController {
	return &EndpointController{
		store:        store,
		stopChan:     make(chan struct{}),
		syncInterval: 10 * time.Second,
		ipAllocator:  NewClusterIPAllocator("10.96.0.0/12"),
	}
}

// Start begins the endpoint controller's reconciliation loop
func (ec *EndpointController) Start(ctx context.Context) error {
	log.Println("Starting endpoint controller...")

	ticker := time.NewTicker(ec.syncInterval)
	defer ticker.Stop()

	if err := ec.syncAllEndpoints(ctx); err != nil {
		log.Printf("Initial endpoint sync failed: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Endpoint controller stopping...")
			return nil
		case <-ec.stopChan:
			log.Println("Endpoint controller stopped")
			return nil
		case <-ticker.C:
			if err := ec.syncAllEndpoints(ctx); err != nil {
				log.Printf("Endpoint sync failed: %v", err)
			}
		}
	}
}

// Stop halts the endpoint controller
func (ec *EndpointController) Stop() {
	close(ec.stopChan)
}

// syncAllEndpoints reconciles all services with their matching pods
func (ec *EndpointController) syncAllEndpoints(ctx context.Context) error {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	services, err := ec.store.ListServices("")
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	for _, service := range services {
		if err := ec.syncServiceEndpoints(service); err != nil {
			log.Printf("Failed to sync endpoints for service %s: %v", service.Name, err)
		}
	}

	return nil
}

// syncServiceEndpoints updates endpoints for a single service
func (ec *EndpointController) syncServiceEndpoints(service types.Service) error {
	if len(service.Selector) == 0 {
		return nil
	}

	namespace := service.Namespace
	if namespace == "" {
		namespace = "default"
	}

	pods, err := ec.store.ListPodsByLabels(namespace, service.Selector)
	if err != nil {
		return fmt.Errorf("failed to list pods by labels: %w", err)
	}

	endpoints := ec.buildEndpoints(service, pods)

	if err := ec.store.SetEndpoints(endpoints); err != nil {
		return fmt.Errorf("failed to set endpoints: %w", err)
	}

	return nil
}

// buildEndpoints creates an Endpoints object from a service and its matching pods
func (ec *EndpointController) buildEndpoints(service types.Service, pods []types.Pod) types.Endpoints {
	namespace := service.Namespace
	if namespace == "" {
		namespace = "default"
	}

	endpoints := types.Endpoints{
		ServiceID:   service.ServiceID,
		ServiceName: service.Name,
		Namespace:   namespace,
		Subsets:     []types.EndpointSubset{},
	}

	var readyAddrs []types.EndpointAddress
	var notReadyAddrs []types.EndpointAddress

	for _, pod := range pods {
		if pod.Status != types.PodRunning || pod.NodeID == "" {
			continue
		}

		podIP := ec.getPodIP(pod)
		if podIP == "" {
			continue
		}

		addr := types.EndpointAddress{
			IP:     podIP,
			PodID:  pod.PodID,
			NodeID: pod.NodeID,
		}

		if ec.isPodReady(pod) {
			readyAddrs = append(readyAddrs, addr)
		} else {
			notReadyAddrs = append(notReadyAddrs, addr)
		}
	}

	var endpointPorts []types.EndpointPort
	for _, svcPort := range service.Ports {
		endpointPorts = append(
			endpointPorts, types.EndpointPort{
				Name:     svcPort.Name,
				Port:     svcPort.TargetPort,
				Protocol: svcPort.Protocol,
			},
		)
	}

	if len(readyAddrs) > 0 || len(notReadyAddrs) > 0 {
		endpoints.Subsets = append(
			endpoints.Subsets, types.EndpointSubset{
				Addresses:         readyAddrs,
				NotReadyAddresses: notReadyAddrs,
				Ports:             endpointPorts,
			},
		)
	}

	return endpoints
}

// getPodIP extracts the IP address from a pod's annotations
// The worker sets the "podling.io/pod-ip" annotation when the pod starts
func (ec *EndpointController) getPodIP(pod types.Pod) string {
	if pod.Annotations != nil {
		if ip, ok := pod.Annotations["podling.io/pod-ip"]; ok && ip != "" {
			return ip
		}
	}

	return ""
}

// isPodReady checks if all containers in a pod are ready
func (ec *EndpointController) isPodReady(pod types.Pod) bool {
	if pod.Status != types.PodRunning {
		return false
	}

	for _, container := range pod.Containers {
		if container.Status != types.ContainerRunning {
			return false
		}
		if container.ReadinessProbe != nil {
			if container.HealthStatus != types.HealthStatusHealthy {
				return false
			}
		}
	}

	return true
}

// AllocateClusterIP allocates a cluster IP for a new service
func (ec *EndpointController) AllocateClusterIP() (string, error) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	return ec.ipAllocator.Allocate()
}

// ReleaseClusterIP releases a cluster IP back to the pool
func (ec *EndpointController) ReleaseClusterIP(ip string) error {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	return ec.ipAllocator.Release(ip)
}

// ClusterIPAllocator manages allocation of cluster IPs for services
type ClusterIPAllocator struct {
	mu        sync.RWMutex
	cidr      *net.IPNet
	allocated map[string]bool
	lastIP    net.IP
}

// NewClusterIPAllocator creates a new IP allocator for the given CIDR
func NewClusterIPAllocator(cidr string) *ClusterIPAllocator {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		log.Fatalf("Invalid CIDR %s: %v", cidr, err)
	}

	allocator := &ClusterIPAllocator{
		cidr:      ipNet,
		allocated: make(map[string]bool),
		lastIP:    ipNet.IP,
	}

	return allocator
}

// Allocate returns a new available IP from the pool
func (a *ClusterIPAllocator) Allocate() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	ip := make(net.IP, len(a.lastIP))
	copy(ip, a.lastIP)

	for i := 0; i < 65536; i++ {
		ip = nextIP(ip)
		if !a.cidr.Contains(ip) {
			ip = make(net.IP, len(a.cidr.IP))
			copy(ip, a.cidr.IP)
			ip = nextIP(ip)
		}

		ipStr := ip.String()
		if !a.allocated[ipStr] {
			a.allocated[ipStr] = true
			a.lastIP = ip
			return ipStr, nil
		}
	}

	return "", fmt.Errorf("no available IPs in range %s", a.cidr.String())
}

// Release marks an IP as available
func (a *ClusterIPAllocator) Release(ip string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.allocated[ip] {
		return fmt.Errorf("IP %s was not allocated", ip)
	}

	delete(a.allocated, ip)
	return nil
}

// nextIP increments an IP address
func nextIP(ip net.IP) net.IP {
	next := make(net.IP, len(ip))
	copy(next, ip)

	for i := len(next) - 1; i >= 0; i-- {
		next[i]++
		if next[i] != 0 {
			break
		}
	}

	return next
}

package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/danpasecinic/podling/internal/types"
	"github.com/spf13/cobra"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage services",
	Long:  `Create, list, inspect, and delete services for pod discovery and load balancing.`,
}

// Service create command flags
var (
	serviceCreateNamespace string
	serviceCreateLabels    []string
	serviceCreateSelectors []string
	servicePorts           []string
	serviceType            string
	serviceSessionAffinity string
)

var serviceCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new service",
	Long: `Create a new service to expose pods.

Examples:
  # Create a service for nginx pods on port 80
  podling service create web \
    --selector app=nginx \
    --port 80

  # Create a service with named ports
  podling service create api \
    --selector app=backend \
    --port http:8080:80 \
    --port metrics:9090

  # Create a service with labels and namespace
  podling service create web \
    --namespace production \
    --selector app=nginx,env=prod \
    --port 80 \
    --label tier=frontend

Port format: [name:]port[:targetPort]
  - port: The port the service listens on
  - targetPort: The port on the pod (defaults to port if not specified)
  - name: Optional name for the port
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]

		if len(serviceCreateSelectors) == 0 {
			return fmt.Errorf("at least one selector is required (use --selector flag)")
		}

		if len(servicePorts) == 0 {
			return fmt.Errorf("at least one port is required (use --port flag)")
		}

		labels := make(map[string]string)
		for _, label := range serviceCreateLabels {
			parts := strings.SplitN(label, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid label format: %s (expected key=value)", label)
			}
			labels[parts[0]] = parts[1]
		}

		selector := make(map[string]string)
		for _, sel := range serviceCreateSelectors {
			parts := strings.SplitN(sel, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid selector format: %s (expected key=value)", sel)
			}
			selector[parts[0]] = parts[1]
		}

		ports := make([]types.ServicePort, 0, len(servicePorts))
		for _, portSpec := range servicePorts {
			port, err := parsePortSpec(portSpec)
			if err != nil {
				return fmt.Errorf("invalid port spec %q: %w", portSpec, err)
			}
			ports = append(ports, port)
		}

		client := NewClient(GetMasterURL())
		service, err := client.CreateService(serviceName, serviceCreateNamespace, selector, ports, labels, serviceType, serviceSessionAffinity)
		if err != nil {
			return fmt.Errorf("failed to create service: %w", err)
		}

		fmt.Println("Service created successfully:")
		fmt.Printf("  ID:         %s\n", service.ServiceID)
		fmt.Printf("  Name:       %s\n", service.Name)
		fmt.Printf("  Namespace:  %s\n", service.Namespace)
		fmt.Printf("  Type:       %s\n", service.Type)
		if service.ClusterIP != "" {
			fmt.Printf("  ClusterIP:  %s\n", service.ClusterIP)
			fmt.Printf("  DNS:        %s\n", service.GetDNSName())
		}
		fmt.Printf("  Ports:      %d\n", len(service.Ports))

		if IsVerbose() {
			fmt.Println("\nPorts:")
			for _, p := range service.Ports {
				if p.Name != "" {
					fmt.Printf("  - %s: %d -> %d/%s\n", p.Name, p.Port, p.TargetPort, p.Protocol)
				} else {
					fmt.Printf("  - %d -> %d/%s\n", p.Port, p.TargetPort, p.Protocol)
				}
			}

			if len(selector) > 0 {
				fmt.Println("\nSelector:")
				for k, v := range service.Selector {
					fmt.Printf("  %s=%s\n", k, v)
				}
			}

			if len(labels) > 0 {
				fmt.Println("\nLabels:")
				for k, v := range service.Labels {
					fmt.Printf("  %s=%s\n", k, v)
				}
			}
		}

		return nil
	},
}

var serviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all services",
	Long:  `List all services, optionally filtered by namespace.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := NewClient(GetMasterURL())
		services, err := client.ListServices(serviceCreateNamespace)
		if err != nil {
			return fmt.Errorf("failed to list services: %w", err)
		}

		if len(services) == 0 {
			fmt.Println("No services found")
			return nil
		}

		// Print header
		fmt.Printf("%-20s %-15s %-10s %-15s %-40s\n", "NAME", "NAMESPACE", "TYPE", "CLUSTER-IP", "PORTS")
		fmt.Println(strings.Repeat("-", 100))

		// Print services
		for _, svc := range services {
			namespace := svc.Namespace
			if namespace == "" {
				namespace = "default"
			}

			clusterIP := svc.ClusterIP
			if clusterIP == "" {
				clusterIP = "None"
			}

			ports := formatServicePorts(svc.Ports)

			fmt.Printf("%-20s %-15s %-10s %-15s %-40s\n",
				truncate(svc.Name, 20),
				truncate(namespace, 15),
				string(svc.Type),
				clusterIP,
				truncate(ports, 40),
			)
		}

		return nil
	},
}

var serviceGetCmd = &cobra.Command{
	Use:   "get [service-id]",
	Short: "Get service details",
	Long:  `Get detailed information about a specific service and its endpoints.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceID := args[0]

		client := NewClient(GetMasterURL())

		// Get service
		service, err := client.GetService(serviceID)
		if err != nil {
			return fmt.Errorf("failed to get service: %w", err)
		}

		// Get endpoints
		endpoints, err := client.GetEndpoints(serviceID)
		if err != nil {
			// Non-fatal, service might not have endpoints yet
			endpoints = nil
		}

		// Print service details
		fmt.Printf("Service: %s\n", service.Name)
		fmt.Printf("  ID:         %s\n", service.ServiceID)
		fmt.Printf("  Namespace:  %s\n", service.Namespace)
		fmt.Printf("  Type:       %s\n", service.Type)
		if service.ClusterIP != "" {
			fmt.Printf("  ClusterIP:  %s\n", service.ClusterIP)
			fmt.Printf("  DNS:        %s\n", service.GetDNSName())
		}
		fmt.Printf("  Created:    %s\n", service.CreatedAt.Format("2006-01-02 15:04:05"))

		if len(service.Selector) > 0 {
			fmt.Println("\nSelector:")
			for k, v := range service.Selector {
				fmt.Printf("  %s=%s\n", k, v)
			}
		}

		if len(service.Ports) > 0 {
			fmt.Println("\nPorts:")
			for _, p := range service.Ports {
				if p.Name != "" {
					fmt.Printf("  - %s: %d -> %d/%s\n", p.Name, p.Port, p.TargetPort, p.Protocol)
				} else {
					fmt.Printf("  - %d -> %d/%s\n", p.Port, p.TargetPort, p.Protocol)
				}
			}
		}

		if len(service.Labels) > 0 {
			fmt.Println("\nLabels:")
			for k, v := range service.Labels {
				fmt.Printf("  %s=%s\n", k, v)
			}
		}

		// Print endpoints
		if endpoints != nil && endpoints.HasEndpoints() {
			fmt.Println("\nEndpoints:")
			for _, subset := range endpoints.Subsets {
				if len(subset.Addresses) > 0 {
					fmt.Println("  Ready:")
					for _, addr := range subset.Addresses {
						fmt.Printf("    - %s (pod: %s)\n", addr.IP, addr.PodID)
					}
				}
				if len(subset.NotReadyAddresses) > 0 {
					fmt.Println("  Not Ready:")
					for _, addr := range subset.NotReadyAddresses {
						fmt.Printf("    - %s (pod: %s)\n", addr.IP, addr.PodID)
					}
				}
			}
		} else {
			fmt.Println("\nEndpoints: None")
		}

		return nil
	},
}

var serviceDeleteCmd = &cobra.Command{
	Use:   "delete [service-id]",
	Short: "Delete a service",
	Long:  `Delete a service by its ID.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceID := args[0]

		client := NewClient(GetMasterURL())
		if err := client.DeleteService(serviceID); err != nil {
			return fmt.Errorf("failed to delete service: %w", err)
		}

		fmt.Printf("Service %s deleted successfully\n", serviceID)
		return nil
	},
}

func init() {
	// Add service command to root
	rootCmd.AddCommand(serviceCmd)

	// Add subcommands
	serviceCmd.AddCommand(serviceCreateCmd)
	serviceCmd.AddCommand(serviceListCmd)
	serviceCmd.AddCommand(serviceGetCmd)
	serviceCmd.AddCommand(serviceDeleteCmd)

	// Create command flags
	serviceCreateCmd.Flags().StringVar(&serviceCreateNamespace, "namespace", "default", "Namespace for the service")
	serviceCreateCmd.Flags().StringSliceVar(&serviceCreateLabels, "label", []string{}, "Labels for the service (can be specified multiple times)")
	serviceCreateCmd.Flags().StringSliceVar(&serviceCreateSelectors, "selector", []string{}, "Pod selector (can be specified multiple times)")
	serviceCreateCmd.Flags().StringSliceVar(&servicePorts, "port", []string{}, "Service ports (can be specified multiple times)")
	serviceCreateCmd.Flags().StringVar(&serviceType, "type", "ClusterIP", "Service type (ClusterIP, NodePort, LoadBalancer)")
	serviceCreateCmd.Flags().StringVar(&serviceSessionAffinity, "session-affinity", "", "Session affinity (None or ClientIP)")

	// List command flags
	serviceListCmd.Flags().StringVar(&serviceCreateNamespace, "namespace", "", "Filter by namespace (empty for all)")
}

// parsePortSpec parses a port specification in the format [name:]port[:targetPort]
func parsePortSpec(spec string) (types.ServicePort, error) {
	parts := strings.Split(spec, ":")

	port := types.ServicePort{
		Protocol: "TCP", // Default protocol
	}

	switch len(parts) {
	case 1:
		p, err := strconv.Atoi(parts[0])
		if err != nil {
			return port, fmt.Errorf("invalid port number: %s", parts[0])
		}
		port.Port = p
		port.TargetPort = p

	case 2:
		if p, err := strconv.Atoi(parts[0]); err == nil {
			port.Port = p
			tp, err := strconv.Atoi(parts[1])
			if err != nil {
				return port, fmt.Errorf("invalid target port number: %s", parts[1])
			}
			port.TargetPort = tp
		} else {
			port.Name = parts[0]
			p, err := strconv.Atoi(parts[1])
			if err != nil {
				return port, fmt.Errorf("invalid port number: %s", parts[1])
			}
			port.Port = p
			port.TargetPort = p
		}

	case 3:
		port.Name = parts[0]
		p, err := strconv.Atoi(parts[1])
		if err != nil {
			return port, fmt.Errorf("invalid port number: %s", parts[1])
		}
		port.Port = p

		tp, err := strconv.Atoi(parts[2])
		if err != nil {
			return port, fmt.Errorf("invalid target port number: %s", parts[2])
		}
		port.TargetPort = tp

	default:
		return port, fmt.Errorf("invalid port format (expected [name:]port[:targetPort])")
	}

	return port, nil
}

// formatServicePorts formats service ports for display
func formatServicePorts(ports []types.ServicePort) string {
	if len(ports) == 0 {
		return "None"
	}

	parts := make([]string, 0, len(ports))
	for _, p := range ports {
		if p.Name != "" {
			parts = append(parts, fmt.Sprintf("%s:%d", p.Name, p.Port))
		} else {
			parts = append(parts, fmt.Sprintf("%d", p.Port))
		}
	}

	return strings.Join(parts, ",")
}

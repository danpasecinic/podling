package cli

import (
	"fmt"
	"strings"

	"github.com/danpasecinic/podling/internal/types"
	"github.com/spf13/cobra"
)

var podCmd = &cobra.Command{
	Use:   "pod",
	Short: "Manage pods",
	Long:  `Create, list, inspect, and delete pods (groups of containers).`,
}

// Pod create command
var (
	podCreateNamespace  string
	podCreateLabels     []string
	podCreateContainers []string
)

var podCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new pod",
	Long: `Create a new pod with one or more containers.

Examples:
  # Create a pod with a single nginx container
  podling pod create my-web --container nginx:nginx:latest

  # Create a pod with multiple containers
  podling pod create my-app \
    --container app:myapp:1.0 \
    --container sidecar:logging:latest

  # Create a pod with labels and namespace
  podling pod create my-app \
    --namespace production \
    --label app=myapp \
    --label version=1.0 \
    --container app:myapp:1.0

Container format: name:image[:env1=val1,env2=val2]
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		podName := args[0]

		if len(podCreateContainers) == 0 {
			return fmt.Errorf("at least one container is required (use --container flag)")
		}

		// Parse labels
		labels := make(map[string]string)
		for _, label := range podCreateLabels {
			parts := strings.SplitN(label, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid label format: %s (expected key=value)", label)
			}
			labels[parts[0]] = parts[1]
		}

		containers := make([]types.Container, 0, len(podCreateContainers))
		for _, containerSpec := range podCreateContainers {
			container, err := parseContainerSpec(containerSpec)
			if err != nil {
				return fmt.Errorf("invalid container spec %q: %w", containerSpec, err)
			}
			containers = append(containers, container)
		}

		client := NewClient(GetMasterURL())
		pod, err := client.CreatePod(podName, podCreateNamespace, labels, containers)
		if err != nil {
			return fmt.Errorf("failed to create pod: %w", err)
		}

		fmt.Println("Pod created successfully:")
		fmt.Printf("  ID:        %s\n", pod.PodID)
		fmt.Printf("  Name:      %s\n", pod.Name)
		fmt.Printf("  Namespace: %s\n", pod.Namespace)
		fmt.Printf("  Status:    %s\n", pod.Status)
		if pod.NodeID != "" {
			fmt.Printf("  Node:      %s\n", pod.NodeID)
		}
		fmt.Printf("  Containers: %d\n", len(pod.Containers))

		if IsVerbose() {
			fmt.Println("\nContainers:")
			for _, c := range pod.Containers {
				fmt.Printf("  - %s (%s)\n", c.Name, c.Image)
			}

			if len(labels) > 0 {
				fmt.Println("\nLabels:")
				for k, v := range pod.Labels {
					fmt.Printf("  %s=%s\n", k, v)
				}
			}
		}

		return nil
	},
}

// Pod list command
var podListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all pods",
	Long:  `List all pods in the system.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := NewClient(GetMasterURL())
		pods, err := client.ListPods()
		if err != nil {
			return fmt.Errorf("failed to list pods: %w", err)
		}

		if len(pods) == 0 {
			fmt.Println("No pods found")
			return nil
		}

		// Print header
		fmt.Printf("%-25s %-20s %-12s %-10s %s\n", "POD ID", "NAME", "NAMESPACE", "STATUS", "CONTAINERS")
		fmt.Println(strings.Repeat("-", 80))

		// Print pods
		for _, pod := range pods {
			namespace := pod.Namespace
			if namespace == "" {
				namespace = "default"
			}

			containerNames := make([]string, len(pod.Containers))
			for i, c := range pod.Containers {
				containerNames[i] = c.Name
			}

			fmt.Printf(
				"%-25s %-20s %-12s %-10s %s\n",
				pod.PodID,
				truncate(pod.Name, 20),
				truncate(namespace, 12),
				pod.Status,
				strings.Join(containerNames, ","),
			)
		}

		return nil
	},
}

// Pod get command
var podGetCmd = &cobra.Command{
	Use:   "get [pod-id]",
	Short: "Get detailed information about a pod",
	Long:  `Get detailed information about a specific pod including all containers.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		podID := args[0]

		client := NewClient(GetMasterURL())
		pod, err := client.GetPod(podID)
		if err != nil {
			return fmt.Errorf("failed to get pod: %w", err)
		}

		fmt.Printf("Pod ID:        %s\n", pod.PodID)
		fmt.Printf("Name:          %s\n", pod.Name)
		fmt.Printf("Namespace:     %s\n", pod.Namespace)
		fmt.Printf("Status:        %s\n", pod.Status)
		if pod.NodeID != "" {
			fmt.Printf("Node ID:       %s\n", pod.NodeID)
		}
		fmt.Printf("Created:       %s\n", pod.CreatedAt.Format("2006-01-02 15:04:05"))
		if pod.ScheduledAt != nil {
			fmt.Printf("Scheduled:     %s\n", pod.ScheduledAt.Format("2006-01-02 15:04:05"))
		}
		if pod.StartedAt != nil {
			fmt.Printf("Started:       %s\n", pod.StartedAt.Format("2006-01-02 15:04:05"))
		}
		if pod.FinishedAt != nil {
			fmt.Printf("Finished:      %s\n", pod.FinishedAt.Format("2006-01-02 15:04:05"))
		}
		if pod.Message != "" {
			fmt.Printf("Message:       %s\n", pod.Message)
		}
		if pod.Reason != "" {
			fmt.Printf("Reason:        %s\n", pod.Reason)
		}

		if len(pod.Labels) > 0 {
			fmt.Println("\nLabels:")
			for k, v := range pod.Labels {
				fmt.Printf("  %s=%s\n", k, v)
			}
		}

		fmt.Printf("\nContainers (%d):\n", len(pod.Containers))
		for i, container := range pod.Containers {
			fmt.Printf("\n  [%d] %s\n", i+1, container.Name)
			fmt.Printf("      Image:       %s\n", container.Image)
			fmt.Printf("      Status:      %s\n", container.Status)
			if container.ContainerID != "" {
				fmt.Printf("      Container ID: %s\n", truncate(container.ContainerID, 12))
			}
			if container.ExitCode != nil {
				fmt.Printf("      Exit Code:   %d\n", *container.ExitCode)
			}
			if container.Error != "" {
				fmt.Printf("      Error:       %s\n", container.Error)
			}
			if container.HealthStatus != "" {
				fmt.Printf("      Health:      %s\n", container.HealthStatus)
			}
			if len(container.Env) > 0 {
				fmt.Printf("      Environment:\n")
				for k, v := range container.Env {
					fmt.Printf("        %s=%s\n", k, v)
				}
			}
		}

		return nil
	},
}

// Pod delete command
var podDeleteCmd = &cobra.Command{
	Use:   "delete [pod-id]",
	Short: "Delete a pod",
	Long:  `Delete a pod by ID.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		podID := args[0]

		client := NewClient(GetMasterURL())
		if err := client.DeletePod(podID); err != nil {
			return fmt.Errorf("failed to delete pod: %w", err)
		}

		fmt.Printf("Pod %s deleted successfully\n", podID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(podCmd)

	podCmd.AddCommand(podCreateCmd)
	podCmd.AddCommand(podListCmd)
	podCmd.AddCommand(podGetCmd)
	podCmd.AddCommand(podDeleteCmd)
	podCmd.AddCommand(podLogsCmd)

	podCreateCmd.Flags().StringVar(&podCreateNamespace, "namespace", "default", "pod namespace")
	podCreateCmd.Flags().StringArrayVarP(&podCreateLabels, "label", "l", []string{}, "pod labels (key=value)")
	podCreateCmd.Flags().StringArrayVarP(
		&podCreateContainers, "container", "c", []string{}, "container spec (name:image[:env1=val1,env2=val2])",
	)
}

// parseContainerSpec parses a container specification string
// Format: name:image[:tag][:env1=val1,env2=val2]
func parseContainerSpec(spec string) (types.Container, error) {
	firstColon := strings.Index(spec, ":")
	if firstColon == -1 {
		return types.Container{}, fmt.Errorf("container spec must be in format 'name:image[:tag][:env]'")
	}

	name := spec[:firstColon]
	rest := spec[firstColon+1:]

	if name == "" {
		return types.Container{}, fmt.Errorf("container name cannot be empty")
	}

	image := ""
	envStr := ""

	lastColon := strings.LastIndex(rest, ":")
	if lastColon != -1 && strings.Contains(rest[lastColon+1:], "=") {
		image = rest[:lastColon]
		envStr = rest[lastColon+1:]
	} else {
		image = rest
	}

	if image == "" {
		return types.Container{}, fmt.Errorf("container image cannot be empty")
	}

	container := types.Container{
		Name:  name,
		Image: image,
		Env:   make(map[string]string),
	}

	if envStr != "" {
		envPairs := strings.Split(envStr, ",")
		for _, pair := range envPairs {
			if pair == "" {
				continue
			}
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) != 2 {
				return types.Container{}, fmt.Errorf("invalid env format: %s (expected key=value)", pair)
			}
			container.Env[kv[0]] = kv[1]
		}
	}

	return container, nil
}

// Pod logs command
var (
	podLogsContainer string
	podLogsTail      int
)

var podLogsCmd = &cobra.Command{
	Use:   "logs [pod-id]",
	Short: "Get logs from a pod's containers",
	Long: `Get logs from one or all containers in a pod.

Examples:
  # Get logs from all containers in a pod
  podling pod logs <pod-id>

  # Get logs from a specific container
  podling pod logs <pod-id> --container web

  # Get last 50 lines
  podling pod logs <pod-id> --tail 50
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		podID := args[0]

		client := NewClient(GetMasterURL())
		logs, err := client.GetPodLogs(podID, podLogsContainer, podLogsTail)
		if err != nil {
			return fmt.Errorf("failed to get pod logs: %w", err)
		}

		if podLogsContainer != "" {
			if log, ok := logs[podLogsContainer]; ok {
				fmt.Printf("==> Container: %s <==\n%s\n", podLogsContainer, log)
			} else {
				return fmt.Errorf("container %s not found", podLogsContainer)
			}
		} else {
			for containerName, log := range logs {
				fmt.Printf("==> Container: %s <==\n%s\n\n", containerName, log)
			}
		}

		return nil
	},
}

func init() {
	podLogsCmd.Flags().StringVarP(&podLogsContainer, "container", "c", "", "specific container name")
	podLogsCmd.Flags().IntVarP(&podLogsTail, "tail", "t", 100, "number of lines to show from the end of logs")
}

// truncate truncates a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/danpasecinic/podling/internal/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

var (
	pruneAll bool
)

type PruneResult = types.PruneResult

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Clean up resources",
	Long: `Remove old and unused resources from the system.

By default, removes:
- Completed/failed tasks and pods from database
- Offline nodes from database

Use --all to remove everything including Docker containers and networks.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiClient := NewClient(GetMasterURL())

		var result *PruneResult
		var err error

		if pruneAll {
			result, err = apiClient.PruneAll()
			if err != nil {
				return fmt.Errorf("failed to prune all: %w", err)
			}
		} else {
			result, err = apiClient.Prune()
			if err != nil {
				return fmt.Errorf("failed to prune: %w", err)
			}
		}

		fmt.Printf("Pruned resources from database:\n")
		fmt.Printf("  Pods:     %d\n", result.PodsRemoved)
		fmt.Printf("  Tasks:    %d\n", result.TasksRemoved)
		fmt.Printf("  Nodes:    %d\n", result.NodesRemoved)
		fmt.Printf("  Services: %d\n", result.ServicesRemoved)

		if pruneAll {
			fmt.Println("\nCleaning up Docker resources...")
			containersRemoved, networksRemoved, err := cleanupDockerResources()
			if err != nil {
				return fmt.Errorf("failed to cleanup docker resources: %w", err)
			}

			fmt.Printf("Docker resources cleaned:\n")
			fmt.Printf("  Containers: %d\n", containersRemoved)
			fmt.Printf("  Networks:   %d\n", networksRemoved)
		}

		return nil
	},
}

// cleanupDockerResources removes Docker containers and networks created by Podling
func cleanupDockerResources() (int, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create docker client: %w", err)
	}
	defer func() { _ = cli.Close() }()

	containersRemoved := 0
	networksRemoved := 0

	containerFilters := filters.NewArgs()
	containerFilters.Add("label", "podling.io/managed=true")

	containers, err := cli.ContainerList(
		ctx, container.ListOptions{
			All:     true,
			Filters: containerFilters,
		},
	)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list containers: %w", err)
	}

	for _, c := range containers {
		if err := cli.ContainerStop(ctx, c.ID, container.StopOptions{}); err != nil {
			fmt.Printf("Warning: failed to stop container %s: %v\n", c.ID[:12], err)
		}

		if err := cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true}); err != nil {
			fmt.Printf("Warning: failed to remove container %s: %v\n", c.ID[:12], err)
		} else {
			containersRemoved++
		}
	}

	networkFilters := filters.NewArgs()
	networkFilters.Add("label", "podling.io/type=pod-network")

	networks, err := cli.NetworkList(ctx, network.ListOptions{Filters: networkFilters})
	if err != nil {
		return containersRemoved, 0, fmt.Errorf("failed to list networks: %w", err)
	}

	for _, net := range networks {
		if err := cli.NetworkRemove(ctx, net.ID); err != nil {
			fmt.Printf("Warning: failed to remove network %s: %v\n", net.Name, err)
		} else {
			networksRemoved++
		}
	}

	return containersRemoved, networksRemoved, nil
}

// init initializes the prune command and its flags
func init() {
	rootCmd.AddCommand(pruneCmd)
	pruneCmd.Flags().BoolVar(&pruneAll, "all", false, "remove all resources including Docker containers and networks")
}

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	pruneAll     bool
	pruneOffline bool
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Clean up resources",
	Long: `Remove old and unused resources from the system.

By default, removes:
- Completed/failed pods
- Offline nodes

Use --all to remove everything.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := NewClient(GetMasterURL())

		if pruneAll {
			if err := client.PruneAll(); err != nil {
				return fmt.Errorf("failed to prune all: %w", err)
			}
			fmt.Println("Successfully pruned all resources")
			return nil
		}

		result, err := client.Prune()
		if err != nil {
			return fmt.Errorf("failed to prune: %w", err)
		}

		fmt.Printf("Pruned resources:\n")
		fmt.Printf("  Pods:     %d\n", result.PodsRemoved)
		fmt.Printf("  Nodes:    %d\n", result.NodesRemoved)
		fmt.Printf("  Services: %d\n", result.ServicesRemoved)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(pruneCmd)
	pruneCmd.Flags().BoolVar(&pruneAll, "all", false, "remove all resources (pods, nodes, services)")
}

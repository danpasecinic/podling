package cli

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var nodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "List worker nodes",
	Long:  `List all registered worker nodes and their status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := NewClient(GetMasterURL())

		nodes, err := client.ListNodes()
		if err != nil {
			return fmt.Errorf("failed to list nodes: %w", err)
		}

		if len(nodes) == 0 {
			fmt.Println("No worker nodes registered.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		_, _ = fmt.Fprint(w, "ID\tHOSTNAME\tPORT\tSTATUS\tCAPACITY\tTASKS\tLAST HEARTBEAT\n")

		for _, node := range nodes {
			lastHeartbeat := time.Since(node.LastHeartbeat)
			heartbeatStr := formatDuration(lastHeartbeat)

			_, _ = fmt.Fprintf(
				w, "%s\t%s\t%d\t%s\t%d\t%d\t%s ago\n",
				node.NodeID,
				node.Hostname,
				node.Port,
				node.Status,
				node.Capacity,
				node.RunningTasks,
				heartbeatStr,
			)
		}

		_ = w.Flush()

		if IsVerbose() {
			fmt.Printf("\nTotal nodes: %d\n", len(nodes))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(nodesCmd)
}

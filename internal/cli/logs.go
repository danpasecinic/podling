package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	logsTail int
)

var logsCmd = &cobra.Command{
	Use:   "logs [task-id]",
	Short: "Fetch container logs for a task",
	Long:  `Fetch and display container logs for a specific task.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]

		client := NewClient(GetMasterURL())

		task, err := client.GetTask(taskID)
		if err != nil {
			return fmt.Errorf("failed to get task: %w", err)
		}

		if task.NodeID == "" {
			return fmt.Errorf("task has not been scheduled to a node yet")
		}

		if task.Status == "pending" || task.Status == "scheduled" {
			return fmt.Errorf("task is not running yet (status: %s)", task.Status)
		}

		logs, err := client.GetTaskLogs(task, logsTail)
		if err != nil {
			return fmt.Errorf("failed to get logs: %w", err)
		}

		fmt.Print(logs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)

	logsCmd.Flags().IntVar(&logsTail, "tail", 100, "number of lines to show from the end of the logs")
}

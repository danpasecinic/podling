package cli

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var (
	psTaskID string
)

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List tasks",
	Long:  `List all tasks or get details of a specific task.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := NewClient(GetMasterURL())

		if psTaskID != "" {
			task, err := client.GetTask(psTaskID)
			if err != nil {
				return fmt.Errorf("failed to get task: %w", err)
			}

			fmt.Println("Task Details:")
			fmt.Printf("  ID:          %s\n", task.TaskID)
			fmt.Printf("  Name:        %s\n", task.Name)
			fmt.Printf("  Image:       %s\n", task.Image)
			fmt.Printf("  Status:      %s\n", task.Status)
			fmt.Printf("  Node:        %s\n", task.NodeID)
			fmt.Printf("  Container:   %s\n", task.ContainerID)
			fmt.Printf("  Created:     %s\n", task.CreatedAt.Format(time.RFC3339))
			if task.StartedAt != nil {
				fmt.Printf("  Started:     %s\n", task.StartedAt.Format(time.RFC3339))
			}
			if task.FinishedAt != nil {
				fmt.Printf("  Finished:    %s\n", task.FinishedAt.Format(time.RFC3339))
			}
			if task.Error != "" {
				fmt.Printf("  Error:       %s\n", task.Error)
			}

			if len(task.Env) > 0 {
				fmt.Println("\nEnvironment Variables:")
				for k, v := range task.Env {
					fmt.Printf("  %s=%s\n", k, v)
				}
			}

			return nil
		}

		tasks, err := client.ListTasks()
		if err != nil {
			return fmt.Errorf("failed to list tasks: %w", err)
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		_, _ = fmt.Fprintf(w, "ID\tNAME\tIMAGE\tSTATUS\tNODE\tCREATED\n")

		for _, task := range tasks {
			age := time.Since(task.CreatedAt)
			ageStr := formatDuration(age)

			nodeID := task.NodeID
			if nodeID == "" {
				nodeID = "-"
			}

			_, _ = fmt.Fprintf(
				w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				task.TaskID,
				task.Name,
				task.Image,
				task.Status,
				nodeID,
				ageStr,
			)
		}

		_ = w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(psCmd)

	psCmd.Flags().StringVarP(&psTaskID, "task", "t", "", "show details for specific task ID")
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

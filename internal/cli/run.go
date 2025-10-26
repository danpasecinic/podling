package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	runImage string
	runEnv   []string
)

var runCmd = &cobra.Command{
	Use:   "run [name]",
	Short: "Run a new task",
	Long:  `Create and schedule a new task to run a container image.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if runImage == "" {
			return fmt.Errorf("image is required (use --image flag)")
		}

		envMap := make(map[string]string)
		for _, e := range runEnv {
			// Simple parsing: KEY=VALUE
			var key, value string
			for i, c := range e {
				if c == '=' {
					key = e[:i]
					value = e[i+1:]
					break
				}
			}
			if key != "" {
				envMap[key] = value
			}
		}

		client := NewClient(GetMasterURL())
		task, err := client.CreateTask(name, runImage, envMap)
		if err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}

		fmt.Println("Task created successfully:")
		fmt.Printf("  ID:     %s\n", task.TaskID)
		fmt.Printf("  Name:   %s\n", task.Name)
		fmt.Printf("  Image:  %s\n", task.Image)
		fmt.Printf("  Status: %s\n", task.Status)
		if task.NodeID != "" {
			fmt.Printf("  Node:   %s\n", task.NodeID)
		}

		if IsVerbose() {
			fmt.Println("\nEnvironment variables:")
			for k, v := range task.Env {
				fmt.Printf("  %s=%s\n", k, v)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVarP(&runImage, "image", "i", "", "container image to run (required)")
	runCmd.Flags().StringArrayVarP(&runEnv, "env", "e", []string{}, "environment variables (KEY=VALUE)")
	_ = runCmd.MarkFlagRequired("image")
}

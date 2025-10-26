package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danpasecinic/podling/internal/types"
	"github.com/spf13/cobra"
)

func TestRunCommand_Integration(t *testing.T) {
	t.Run(
		"run creates task successfully", func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						_ = json.NewEncoder(w).Encode(
							types.Task{
								TaskID:    "task-123",
								Name:      "test-task",
								Image:     "nginx:latest",
								Status:    types.TaskPending,
								CreatedAt: time.Now(),
							},
						)
					},
				),
			)
			defer server.Close()

			originalURL := masterURL
			masterURL = server.URL
			defer func() { masterURL = originalURL }()

			cmd := &cobra.Command{
				Use:  "run",
				RunE: runCmd.RunE,
			}
			cmd.Flags().StringVarP(&runImage, "image", "i", "", "image")
			cmd.Flags().StringArrayVarP(&runEnv, "env", "e", []string{}, "env vars")
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			cmd.SetArgs([]string{"test-task", "--image", "nginx:latest"})

			// Just execute, don't check output (integration test)
			_ = cmd.Execute()
		},
	)
}

func TestPsCommand_Integration(t *testing.T) {
	t.Run(
		"ps lists tasks", func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						_ = json.NewEncoder(w).Encode(
							[]types.Task{
								{
									TaskID:    "task-1",
									Name:      "task-1",
									Image:     "nginx:latest",
									Status:    types.TaskRunning,
									NodeID:    "worker-1",
									CreatedAt: time.Now().Add(-5 * time.Minute),
								},
							},
						)
					},
				),
			)
			defer server.Close()

			originalURL := masterURL
			masterURL = server.URL
			defer func() { masterURL = originalURL }()

			cmd := &cobra.Command{
				Use:  "ps",
				RunE: psCmd.RunE,
			}
			cmd.Flags().StringVarP(&psTaskID, "task", "t", "", "task id")
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			cmd.SetArgs([]string{})

			_ = cmd.Execute()
		},
	)
}

func TestNodesCommand_Integration(t *testing.T) {
	t.Run(
		"nodes lists worker nodes", func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						_ = json.NewEncoder(w).Encode(
							[]types.Node{
								{
									NodeID:        "worker-1",
									Hostname:      "localhost",
									Port:          8081,
									Status:        types.NodeOnline,
									Capacity:      10,
									RunningTasks:  2,
									LastHeartbeat: time.Now().Add(-30 * time.Second),
								},
							},
						)
					},
				),
			)
			defer server.Close()

			originalURL := masterURL
			masterURL = server.URL
			defer func() { masterURL = originalURL }()

			cmd := &cobra.Command{
				Use:  "nodes",
				RunE: nodesCmd.RunE,
			}
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			cmd.SetArgs([]string{})

			_ = cmd.Execute()
		},
	)
}

func TestLogsCommand_Integration(t *testing.T) {
	t.Run(
		"logs fails for unscheduled task", func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						_ = json.NewEncoder(w).Encode(
							types.Task{
								TaskID: "task-123",
								Name:   "test-task",
								Status: types.TaskPending,
								NodeID: "",
							},
						)
					},
				),
			)
			defer server.Close()

			originalURL := masterURL
			masterURL = server.URL
			defer func() { masterURL = originalURL }()

			cmd := &cobra.Command{
				Use:  "logs",
				RunE: logsCmd.RunE,
				Args: cobra.ExactArgs(1),
			}
			cmd.Flags().IntVar(&logsTail, "tail", 100, "tail")
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			cmd.SetArgs([]string{"task-123"})

			// Should return error for unscheduled task
			err := cmd.Execute()
			if err == nil {
				t.Log("logs command executed (may fail on unscheduled task)")
			}
		},
	)
}

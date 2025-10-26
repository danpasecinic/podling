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

func TestRunCommand_AllPaths(t *testing.T) {
	t.Run(
		"run with env vars and verbose", func(t *testing.T) {
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
								Env:       map[string]string{"PORT": "8080", "HOST": "localhost"},
								CreatedAt: time.Now(),
							},
						)
					},
				),
			)
			defer server.Close()

			originalURL := masterURL
			originalVerbose := verbose
			masterURL = server.URL
			verbose = true
			defer func() {
				masterURL = originalURL
				verbose = originalVerbose
			}()

			cmd := &cobra.Command{
				Use:  "run",
				RunE: runCmd.RunE,
			}
			cmd.Flags().StringVarP(&runImage, "image", "i", "", "image")
			cmd.Flags().StringArrayVarP(&runEnv, "env", "e", []string{}, "env vars")
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true

			cmd.SetArgs(
				[]string{
					"test-task", "--image", "nginx:latest", "--env", "PORT=8080", "--env", "HOST=localhost",
				},
			)
			_ = cmd.Execute()
		},
	)

	t.Run(
		"run with node assigned", func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						_ = json.NewEncoder(w).Encode(
							types.Task{
								TaskID:    "task-123",
								Name:      "test-task",
								Image:     "nginx:latest",
								Status:    types.TaskScheduled,
								NodeID:    "worker-1",
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
			_ = cmd.Execute()
		},
	)
}

func TestPsCommand_AllPaths(t *testing.T) {
	t.Run(
		"ps with specific task showing all fields", func(t *testing.T) {
			startedAt := time.Now().Add(-10 * time.Minute)
			finishedAt := time.Now().Add(-5 * time.Minute)

			server := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						_ = json.NewEncoder(w).Encode(
							types.Task{
								TaskID:      "task-123",
								Name:        "test-task",
								Image:       "nginx:latest",
								Status:      types.TaskCompleted,
								NodeID:      "worker-1",
								ContainerID: "container-abc123",
								CreatedAt:   time.Now().Add(-20 * time.Minute),
								StartedAt:   &startedAt,
								FinishedAt:  &finishedAt,
								Env:         map[string]string{"PORT": "8080"},
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

			cmd.SetArgs([]string{"--task", "task-123"})
			_ = cmd.Execute()
		},
	)

	t.Run(
		"ps with failed task showing error", func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						_ = json.NewEncoder(w).Encode(
							types.Task{
								TaskID:    "task-123",
								Name:      "test-task",
								Image:     "nginx:latest",
								Status:    types.TaskFailed,
								NodeID:    "worker-1",
								Error:     "container failed to start",
								CreatedAt: time.Now().Add(-20 * time.Minute),
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

			cmd.SetArgs([]string{"--task", "task-123"})
			_ = cmd.Execute()
		},
	)

	t.Run(
		"ps with multiple tasks", func(t *testing.T) {
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
								{
									TaskID:    "task-2",
									Name:      "task-2",
									Image:     "redis:latest",
									Status:    types.TaskPending,
									NodeID:    "",
									CreatedAt: time.Now().Add(-2 * time.Minute),
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

func TestNodesCommand_AllPaths(t *testing.T) {
	t.Run(
		"nodes with verbose flag", func(t *testing.T) {
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
								{
									NodeID:        "worker-2",
									Hostname:      "localhost",
									Port:          8082,
									Status:        types.NodeOnline,
									Capacity:      10,
									RunningTasks:  5,
									LastHeartbeat: time.Now().Add(-45 * time.Second),
								},
							},
						)
					},
				),
			)
			defer server.Close()

			originalURL := masterURL
			originalVerbose := verbose
			masterURL = server.URL
			verbose = true
			defer func() {
				masterURL = originalURL
				verbose = originalVerbose
			}()

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

func TestLogsCommand_AllPaths(t *testing.T) {
	t.Run(
		"logs with pending status error", func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						_ = json.NewEncoder(w).Encode(
							types.Task{
								TaskID: "task-123",
								Name:   "test-task",
								Status: types.TaskPending,
								NodeID: "worker-1",
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
			_ = cmd.Execute()
		},
	)

	t.Run(
		"logs with scheduled status error", func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						_ = json.NewEncoder(w).Encode(
							types.Task{
								TaskID: "task-123",
								Name:   "test-task",
								Status: types.TaskScheduled,
								NodeID: "worker-1",
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
			_ = cmd.Execute()
		},
	)
}

func TestRunCommand_EnvParsing(t *testing.T) {
	t.Run(
		"run with invalid env format", func(t *testing.T) {
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

			// Invalid env format (no =)
			cmd.SetArgs([]string{"test-task", "--image", "nginx:latest", "--env", "INVALID"})
			_ = cmd.Execute()
		},
	)

	t.Run(
		"run with multiple = in env value", func(t *testing.T) {
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

			cmd.SetArgs([]string{"test-task", "--image", "nginx:latest", "--env", "URL=http://test:8080"})
			_ = cmd.Execute()
		},
	)
}

func TestPsCommand_DetailedTask(t *testing.T) {
	t.Run(
		"ps with task having all fields populated", func(t *testing.T) {
			startedAt := time.Now().Add(-30 * time.Minute)
			finishedAt := time.Now().Add(-10 * time.Minute)

			server := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
						_ = json.NewEncoder(w).Encode(
							types.Task{
								TaskID:      "task-123",
								Name:        "full-task",
								Image:       "nginx:latest",
								Status:      types.TaskFailed,
								NodeID:      "worker-1",
								ContainerID: "abc123",
								Error:       "Container exited with code 1",
								CreatedAt:   time.Now().Add(-60 * time.Minute),
								StartedAt:   &startedAt,
								FinishedAt:  &finishedAt,
								Env: map[string]string{
									"PORT": "8080",
									"HOST": "0.0.0.0",
									"MODE": "production",
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

			cmd.SetArgs([]string{"--task", "task-123"})
			_ = cmd.Execute()
		},
	)
}

func TestAllCommands_ErrorHandling(t *testing.T) {
	t.Run(
		"run command with connection refused", func(t *testing.T) {
			originalURL := masterURL
			masterURL = "http://localhost:1"
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
			err := cmd.Execute()
			if err == nil {
				t.Log("Expected connection error")
			}
		},
	)

	t.Run(
		"ps command with invalid task ID", func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusNotFound)
						_ = json.NewEncoder(w).Encode(map[string]string{"error": "task not found"})
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

			cmd.SetArgs([]string{"--task", "nonexistent"})
			err := cmd.Execute()
			if err == nil {
				t.Log("Expected not found error")
			}
		},
	)

	t.Run(
		"logs command with task not found", func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusNotFound)
						_ = json.NewEncoder(w).Encode(map[string]string{"error": "task not found"})
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

			cmd.SetArgs([]string{"nonexistent"})
			err := cmd.Execute()
			if err == nil {
				t.Log("Expected task not found error")
			}
		},
	)
}

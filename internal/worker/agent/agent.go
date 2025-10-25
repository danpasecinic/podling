package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/danpasecinic/podling/internal/types"
	"github.com/danpasecinic/podling/internal/worker/docker"
)

// Agent manages task execution and communication with the master.
type Agent struct {
	nodeID          string
	masterURL       string
	dockerClient    *docker.Client
	runningTasks    map[string]*types.Task
	mu              sync.RWMutex
	heartbeatTicker *time.Ticker
	stopChan        chan struct{}
}

// NewAgent creates a new worker agent.
func NewAgent(nodeID, masterURL string) (*Agent, error) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &Agent{
		nodeID:       nodeID,
		masterURL:    masterURL,
		dockerClient: dockerClient,
		runningTasks: make(map[string]*types.Task),
		stopChan:     make(chan struct{}),
	}, nil
}

// Start begins the agent's background operations (heartbeat).
func (a *Agent) Start(heartbeatInterval time.Duration) {
	a.heartbeatTicker = time.NewTicker(heartbeatInterval)
	go a.heartbeatLoop()
}

// Stop gracefully stops the agent.
func (a *Agent) Stop() {
	if a.heartbeatTicker != nil {
		a.heartbeatTicker.Stop()
	}
	close(a.stopChan)
	if a.dockerClient != nil {
		a.dockerClient.Close()
	}
}

// heartbeatLoop sends periodic heartbeats to the master.
func (a *Agent) heartbeatLoop() {
	for {
		select {
		case <-a.heartbeatTicker.C:
			if err := a.sendHeartbeat(); err != nil {
				log.Printf("heartbeat failed: %v", err)
			}
		case <-a.stopChan:
			return
		}
	}
}

// sendHeartbeat sends a heartbeat to the master node.
func (a *Agent) sendHeartbeat() error {
	url := fmt.Sprintf("%s/api/v1/nodes/%s/heartbeat", a.masterURL, a.nodeID)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create heartbeat request: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat returned status %d", resp.StatusCode)
	}

	return nil
}

// ExecuteTask executes a task by running it in a Docker container.
func (a *Agent) ExecuteTask(ctx context.Context, task *types.Task) error {
	a.mu.Lock()
	a.runningTasks[task.TaskID] = task
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		delete(a.runningTasks, task.TaskID)
		a.mu.Unlock()
	}()

	// Update status to running
	if err := a.updateTaskStatus(task.TaskID, types.TaskRunning, "", ""); err != nil {
		log.Printf("failed to update task status to running: %v", err)
	}

	// Pull image
	log.Printf("pulling image %s for task %s", task.Image, task.TaskID)
	if err := a.dockerClient.PullImage(ctx, task.Image); err != nil {
		a.updateTaskStatus(task.TaskID, types.TaskFailed, "", err.Error())
		return fmt.Errorf("failed to pull image: %w", err)
	}

	// Convert env map to slice
	env := make([]string, 0, len(task.Env))
	for k, v := range task.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Create container
	log.Printf("creating container for task %s", task.TaskID)
	containerID, err := a.dockerClient.CreateContainer(ctx, task.Image, env)
	if err != nil {
		a.updateTaskStatus(task.TaskID, types.TaskFailed, "", err.Error())
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	log.Printf("starting container %s for task %s", containerID, task.TaskID)
	if err := a.dockerClient.StartContainer(ctx, containerID); err != nil {
		a.updateTaskStatus(task.TaskID, types.TaskFailed, containerID, err.Error())
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Update status with container ID
	if err := a.updateTaskStatus(task.TaskID, types.TaskRunning, containerID, ""); err != nil {
		log.Printf("failed to update task with container ID: %v", err)
	}

	// Wait for container to finish
	exitCode, err := a.dockerClient.WaitContainer(ctx, containerID)
	if err != nil {
		a.updateTaskStatus(task.TaskID, types.TaskFailed, containerID, err.Error())
		return fmt.Errorf("error waiting for container: %w", err)
	}

	// Update final status
	if exitCode == 0 {
		log.Printf("task %s completed successfully", task.TaskID)
		a.updateTaskStatus(task.TaskID, types.TaskCompleted, containerID, "")
	} else {
		errMsg := fmt.Sprintf("container exited with code %d", exitCode)
		log.Printf("task %s failed: %s", task.TaskID, errMsg)
		a.updateTaskStatus(task.TaskID, types.TaskFailed, containerID, errMsg)
	}

	// Cleanup container
	if err := a.dockerClient.RemoveContainer(ctx, containerID); err != nil {
		log.Printf("failed to remove container %s: %v", containerID, err)
	}

	return nil
}

// updateTaskStatus sends a status update to the master.
func (a *Agent) updateTaskStatus(taskID string, status types.TaskStatus, containerID, errorMsg string) error {
	url := fmt.Sprintf("%s/api/v1/tasks/%s/status", a.masterURL, taskID)

	payload := map[string]interface{}{
		"status": status,
	}
	if containerID != "" {
		payload["containerId"] = containerID
	}
	if errorMsg != "" {
		payload["error"] = errorMsg
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal status update: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create status update request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send status update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status update returned status %d", resp.StatusCode)
	}

	return nil
}

// GetTask returns a running task by ID.
func (a *Agent) GetTask(taskID string) (*types.Task, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	task, ok := a.runningTasks[taskID]
	return task, ok
}

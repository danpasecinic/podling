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
	nodeID               string
	masterURL            string
	dockerClient         *docker.Client
	runningTasks         map[string]*types.Task
	mu                   sync.RWMutex
	heartbeatTicker      *time.Ticker
	stopChan             chan struct{}
	consecutiveFailures  int
	maxConsecutiveErrors int
}

// NewAgent creates a new worker agent.
func NewAgent(nodeID, masterURL string) (*Agent, error) {
	dockerClient, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &Agent{
		nodeID:               nodeID,
		masterURL:            masterURL,
		dockerClient:         dockerClient,
		runningTasks:         make(map[string]*types.Task),
		stopChan:             make(chan struct{}),
		consecutiveFailures:  0,
		maxConsecutiveErrors: 10,
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

// Shutdown performs a graceful shutdown waiting for running tasks to complete.
func (a *Agent) Shutdown(ctx context.Context) error {
	log.Printf("shutdown initiated, waiting for %d running tasks...", len(a.runningTasks))

	// Stop heartbeat immediately
	if a.heartbeatTicker != nil {
		a.heartbeatTicker.Stop()
	}

	// Wait for running tasks to complete or timeout
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		
		for {
			a.mu.RLock()
			count := len(a.runningTasks)
			a.mu.RUnlock()
			
			if count == 0 {
				close(done)
				return
			}
			
			select {
			case <-ticker.C:
				log.Printf("waiting for %d tasks to complete...", count)
			case <-ctx.Done():
				return
			}
		}
	}()

	select {
	case <-done:
		log.Println("all tasks completed successfully")
	case <-ctx.Done():
		a.mu.RLock()
		remaining := len(a.runningTasks)
		a.mu.RUnlock()
		log.Printf("shutdown timeout reached, %d tasks still running", remaining)
		
		// Force cleanup remaining containers
		a.cleanupRunningTasks(context.Background())
	}

	// Send final heartbeat (best effort)
	if err := a.sendHeartbeat(); err != nil {
		log.Printf("failed to send final heartbeat: %v", err)
	}

	// Close resources
	close(a.stopChan)
	if a.dockerClient != nil {
		a.dockerClient.Close()
	}

	return nil
}

// cleanupRunningTasks forcefully stops and removes running containers.
func (a *Agent) cleanupRunningTasks(ctx context.Context) {
	a.mu.Lock()
	tasks := make([]*types.Task, 0, len(a.runningTasks))
	for _, task := range a.runningTasks {
		tasks = append(tasks, task)
	}
	a.mu.Unlock()

	for _, task := range tasks {
		if task.ContainerID != "" {
			log.Printf("force stopping container %s for task %s", task.ContainerID, task.TaskID)
			if err := a.dockerClient.StopContainer(ctx, task.ContainerID); err != nil {
				log.Printf("error stopping container %s: %v", task.ContainerID, err)
			}
			if err := a.dockerClient.RemoveContainer(ctx, task.ContainerID); err != nil {
				log.Printf("error removing container %s: %v", task.ContainerID, err)
			}
		}
	}
}

// heartbeatLoop sends periodic heartbeats to the master with exponential backoff on failures.
func (a *Agent) heartbeatLoop() {
	for {
		select {
		case <-a.heartbeatTicker.C:
			if err := a.sendHeartbeatWithRetry(); err != nil {
				log.Printf("heartbeat failed after retries: %v", err)
			}
		case <-a.stopChan:
			return
		}
	}
}

// sendHeartbeatWithRetry sends a heartbeat with exponential backoff on failures.
func (a *Agent) sendHeartbeatWithRetry() error {
	backoff := 1 * time.Second
	maxBackoff := 30 * time.Second
	maxRetries := 5

	var lastErr error
	for i := 0; i < maxRetries; i++ {
		err := a.sendHeartbeat()
		if err == nil {
			// Success - reset failure counter
			if a.consecutiveFailures > 0 {
				log.Printf("heartbeat recovered after %d failures", a.consecutiveFailures)
				a.consecutiveFailures = 0
			}
			return nil
		}

		lastErr = err
		a.consecutiveFailures++

		if a.consecutiveFailures >= a.maxConsecutiveErrors {
			log.Printf("WARNING: %d consecutive heartbeat failures - worker may be marked dead", a.consecutiveFailures)
		}

		if i < maxRetries-1 {
			log.Printf("heartbeat attempt %d/%d failed: %v, retrying in %v", i+1, maxRetries, err, backoff)
			time.Sleep(backoff)
			
			// Exponential backoff with max cap
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}

	return fmt.Errorf("heartbeat failed after %d retries: %w", maxRetries, lastErr)
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

// GetTaskLogs retrieves container logs for a task.
func (a *Agent) GetTaskLogs(ctx context.Context, taskID string, tail int) (string, error) {
	a.mu.RLock()
	task, ok := a.runningTasks[taskID]
	a.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("task %s not found or not running", taskID)
	}

	if task.ContainerID == "" {
		return "", fmt.Errorf("task %s has no associated container", taskID)
	}

	return a.dockerClient.GetContainerLogs(ctx, task.ContainerID, tail)
}

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
	"github.com/danpasecinic/podling/internal/worker/health"
)

// Agent manages task and pod execution and communication with the master.
type Agent struct {
	nodeID               string
	masterURL            string
	dockerClient         *docker.Client
	runningTasks         map[string]*types.Task
	runningPods          map[string]*PodExecution
	healthCheckers       map[string]*health.Checker
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
		runningPods:          make(map[string]*PodExecution),
		healthCheckers:       make(map[string]*health.Checker),
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
		_ = a.dockerClient.Close()
	}
}

// Shutdown performs a graceful shutdown waiting for running tasks to complete.
func (a *Agent) Shutdown(ctx context.Context) error {
	a.mu.RLock()
	taskCount := len(a.runningTasks)
	podCount := len(a.runningPods)
	a.mu.RUnlock()

	log.Printf("shutdown initiated, waiting for %d running tasks and %d running pods...", taskCount, podCount)

	if a.heartbeatTicker != nil {
		a.heartbeatTicker.Stop()
	}

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			a.mu.RLock()
			taskCount := len(a.runningTasks)
			podCount := len(a.runningPods)
			a.mu.RUnlock()

			if taskCount == 0 && podCount == 0 {
				close(done)
				return
			}

			select {
			case <-ticker.C:
				log.Printf("waiting for %d tasks and %d pods to complete...", taskCount, podCount)
			case <-ctx.Done():
				return
			}
		}
	}()

	select {
	case <-done:
		log.Println("all tasks and pods completed successfully")
	case <-ctx.Done():
		a.mu.RLock()
		remainingTasks := len(a.runningTasks)
		remainingPods := len(a.runningPods)
		a.mu.RUnlock()
		log.Printf("shutdown timeout reached, %d tasks and %d pods still running", remainingTasks, remainingPods)

		a.cleanupRunningTasks(context.Background())
		a.cleanupRunningPods(context.Background())
	}

	if err := a.deregister(); err != nil {
		log.Printf("failed to deregister node: %v", err)
	}

	close(a.stopChan)
	if a.dockerClient != nil {
		if err := a.dockerClient.Close(); err != nil {
			log.Printf("failed to close docker client: %v", err)
		}
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

// cleanupRunningPods forcefully stops and removes all containers in running pods and their networks.
func (a *Agent) cleanupRunningPods(ctx context.Context) {
	a.mu.Lock()
	pods := make([]*PodExecution, 0, len(a.runningPods))
	for _, podExec := range a.runningPods {
		pods = append(pods, podExec)
	}
	a.mu.Unlock()

	for _, podExec := range pods {
		log.Printf("force stopping pod %s with %d containers", podExec.pod.PodID, len(podExec.pod.Containers))

		for _, container := range podExec.pod.Containers {
			if container.ContainerID != "" {
				log.Printf("force stopping container %s for pod %s", container.ContainerID, podExec.pod.PodID)
				if err := a.dockerClient.StopContainer(ctx, container.ContainerID); err != nil {
					log.Printf("error stopping container %s: %v", container.ContainerID, err)
				}
				if err := a.dockerClient.RemoveContainer(ctx, container.ContainerID); err != nil {
					log.Printf("error removing container %s: %v", container.ContainerID, err)
				}
			}
		}

		if podExec.networkID != "" {
			log.Printf("removing pod network %s", podExec.networkID)
			if err := a.dockerClient.RemovePodNetwork(ctx, podExec.networkID); err != nil {
				log.Printf("error removing pod network %s: %v", podExec.networkID, err)
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

			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}

	return fmt.Errorf("heartbeat failed after %d retries: %w", maxRetries, lastErr)
}

// Register registers the worker node with the master.
func (a *Agent) Register(hostname string, port int) error {
	url := fmt.Sprintf("%s/api/v1/nodes/register", a.masterURL)

	payload := map[string]interface{}{
		"hostname": hostname,
		"port":     port,
		"cpu":      "2",
		"memory":   "2Gi",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal registration payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create registration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send registration: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("registration returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode registration response: %w", err)
	}

	if nodeID, ok := result["nodeId"].(string); ok && nodeID != "" {
		a.nodeID = nodeID
		log.Printf("successfully registered with master as node %s", a.nodeID)
	}

	return nil
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat returned status %d", resp.StatusCode)
	}

	return nil
}

// deregister removes the worker node from the master.
func (a *Agent) deregister() error {
	log.Printf("deregistering node %s from master", a.nodeID)
	url := fmt.Sprintf("%s/api/v1/nodes/%s/deregister", a.masterURL, a.nodeID)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create deregister request: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to deregister: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("deregister returned status %d", resp.StatusCode)
	}

	log.Printf("node %s deregistered successfully", a.nodeID)
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

	if err := a.updateTaskStatus(task.TaskID, types.TaskRunning, "", ""); err != nil {
		log.Printf("failed to update task status to running: %v", err)
	}

	if err := a.dockerClient.PullImage(ctx, task.Image); err != nil {
		if updateErr := a.updateTaskStatus(task.TaskID, types.TaskFailed, "", err.Error()); updateErr != nil {
			log.Printf("failed to update task status: %v", updateErr)
		}
		return fmt.Errorf("failed to pull image: %w", err)
	}

	env := make([]string, 0, len(task.Env))
	for k, v := range task.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Create container with resource limits if specified
	var containerID string
	var err error
	if !task.Resources.Limits.IsZero() {
		cpuLimit := task.Resources.Limits.GetCPULimitForDocker()
		memoryLimit := task.Resources.Limits.GetMemoryLimitForDocker()
		containerID, err = a.dockerClient.CreateContainerWithResources(ctx, task.Image, env, cpuLimit, memoryLimit)
	} else {
		containerID, err = a.dockerClient.CreateContainer(ctx, task.Image, env)
	}

	if err != nil {
		if updateErr := a.updateTaskStatus(task.TaskID, types.TaskFailed, "", err.Error()); updateErr != nil {
			log.Printf("failed to update task status: %v", updateErr)
		}
		return fmt.Errorf("failed to create container: %w", err)
	}

	if err := a.dockerClient.StartContainer(ctx, containerID); err != nil {
		if updateErr := a.updateTaskStatus(task.TaskID, types.TaskFailed, containerID, err.Error()); updateErr != nil {
			log.Printf("failed to update task status: %v", updateErr)
		}
		return fmt.Errorf("failed to start container: %w", err)
	}

	if err := a.updateTaskStatus(task.TaskID, types.TaskRunning, containerID, ""); err != nil {
		log.Printf("failed to update task with container ID: %v", err)
	}

	if task.LivenessProbe != nil {
		restartPolicy := task.RestartPolicy
		if restartPolicy == "" {
			restartPolicy = types.RestartPolicyNever
		}

		checker := health.NewChecker(
			task.TaskID,
			containerID,
			task.LivenessProbe,
			restartPolicy,
			a.dockerClient,
			a.handleUnhealthyContainer,
		)

		a.mu.Lock()
		a.healthCheckers[task.TaskID] = checker
		a.mu.Unlock()

		go checker.Start(ctx)
		defer func() {
			checker.Stop()
			a.mu.Lock()
			delete(a.healthCheckers, task.TaskID)
			a.mu.Unlock()
		}()

		log.Printf("started liveness probe for task %s", task.TaskID)
	}

	exitCode, err := a.dockerClient.WaitContainer(ctx, containerID)
	if err != nil {
		if updateErr := a.updateTaskStatus(task.TaskID, types.TaskFailed, containerID, err.Error()); updateErr != nil {
			log.Printf("failed to update task status: %v", updateErr)
		}
		return fmt.Errorf("error waiting for container: %w", err)
	}

	if exitCode == 0 {
		if err := a.updateTaskStatus(task.TaskID, types.TaskCompleted, containerID, ""); err != nil {
			log.Printf("failed to update task status: %v", err)
		}
	} else {
		errMsg := fmt.Sprintf("container exited with code %d", exitCode)
		if err := a.updateTaskStatus(task.TaskID, types.TaskFailed, containerID, errMsg); err != nil {
			log.Printf("failed to update task status: %v", err)
		}

		restartPolicy := task.RestartPolicy
		if restartPolicy == "" {
			restartPolicy = types.RestartPolicyNever
		}

		if health.ShouldRestart(restartPolicy, exitCode) {
			log.Printf(
				"container exited with code %d, restart policy is %s - would restart (not implemented yet)",
				exitCode, restartPolicy,
			)
			// TODO: Implement actual container restart logic
			// This would require refactoring ExecuteTask into a loop or using a supervisor pattern
		}
	}

	if err := a.dockerClient.RemoveContainer(ctx, containerID); err != nil {
		log.Printf("failed to remove container %s: %v", containerID, err)
	}

	return nil
}

// handleUnhealthyContainer is called when a container becomes unhealthy
func (a *Agent) handleUnhealthyContainer(taskID string) {
	log.Printf("[health] container for task %s is unhealthy", taskID)

	a.mu.RLock()
	task, exists := a.runningTasks[taskID]
	a.mu.RUnlock()

	if !exists {
		log.Printf("[health] task %s not found in running tasks", taskID)
		return
	}

	restartPolicy := task.RestartPolicy
	if restartPolicy == "" {
		restartPolicy = types.RestartPolicyNever
	}

	log.Printf("[health] task %s restart policy: %s", taskID, restartPolicy)

	if restartPolicy == types.RestartPolicyAlways || restartPolicy == types.RestartPolicyOnFailure {
		log.Printf("[health] container restart not yet implemented - would restart task %s", taskID)
	}

	if err := a.updateTaskStatus(
		taskID, types.TaskFailed, task.ContainerID, "container failed health check",
	); err != nil {
		log.Printf("failed to update task status for unhealthy container: %v", err)
	}
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
	defer func() { _ = resp.Body.Close() }()

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

// GetPod retrieves a running pod by ID.
func (a *Agent) GetPod(podID string) (*types.Pod, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	podExec, ok := a.runningPods[podID]
	if !ok {
		return nil, false
	}
	return podExec.pod, true
}

// GetPodLogs retrieves logs from all containers in a pod.
func (a *Agent) GetPodLogs(ctx context.Context, podID string, containerName string, tail int) (
	map[string]string,
	error,
) {
	a.mu.RLock()
	podExec, ok := a.runningPods[podID]
	a.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("pod %s not found or not running", podID)
	}

	logs := make(map[string]string)

	podExec.mu.RLock()
	containerIDs := make(map[string]string)
	for name, id := range podExec.containerIDs {
		containerIDs[name] = id
	}
	podExec.mu.RUnlock()

	if containerName != "" {
		containerID, ok := containerIDs[containerName]
		if !ok {
			return nil, fmt.Errorf("container %s not found in pod %s", containerName, podID)
		}
		containerLogs, err := a.dockerClient.GetContainerLogs(ctx, containerID, tail)
		if err != nil {
			return nil, fmt.Errorf("failed to get logs for container %s: %w", containerName, err)
		}
		logs[containerName] = containerLogs
	} else {
		for name, containerID := range containerIDs {
			containerLogs, err := a.dockerClient.GetContainerLogs(ctx, containerID, tail)
			if err != nil {
				logs[name] = fmt.Sprintf("Error: %v", err)
			} else {
				logs[name] = containerLogs
			}
		}
	}

	return logs, nil
}

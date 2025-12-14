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

type Agent struct {
	nodeID               string
	masterURL            string
	apiKey               string
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

func (a *Agent) SetAPIKey(apiKey string) {
	a.apiKey = apiKey
}

func (a *Agent) SetDNSConfig(servers []string, searchDomains []string) {
	if len(servers) > 0 || len(searchDomains) > 0 {
		a.dockerClient.SetDNSConfig(
			&docker.DNSConfig{
				Servers: servers,
				Search:  searchDomains,
			},
		)
		log.Printf("DNS configured: servers=%v, search=%v", servers, searchDomains)
	}
}

func (a *Agent) addAuthHeader(req *http.Request) {
	if a.apiKey != "" {
		req.Header.Set("X-API-Key", a.apiKey)
	}
}

func (a *Agent) Start(heartbeatInterval time.Duration) {
	a.heartbeatTicker = time.NewTicker(heartbeatInterval)
	go a.heartbeatLoop()
}

func (a *Agent) Stop() {
	if a.heartbeatTicker != nil {
		a.heartbeatTicker.Stop()
	}
	close(a.stopChan)
	if a.dockerClient != nil {
		_ = a.dockerClient.Close()
	}
}

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
	a.addAuthHeader(req)

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

func (a *Agent) sendHeartbeat() error {
	url := fmt.Sprintf("%s/api/v1/nodes/%s/heartbeat", a.masterURL, a.nodeID)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create heartbeat request: %w", err)
	}
	a.addAuthHeader(req)

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

func (a *Agent) deregister() error {
	log.Printf("deregistering node %s from master", a.nodeID)
	url := fmt.Sprintf("%s/api/v1/nodes/%s/deregister", a.masterURL, a.nodeID)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create deregister request: %w", err)
	}
	a.addAuthHeader(req)

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
	a.addAuthHeader(req)

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

func (a *Agent) getTaskFromMaster(taskID string) (*types.Task, error) {
	url := fmt.Sprintf("%s/api/v1/tasks/%s", a.masterURL, taskID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	a.addAuthHeader(req)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch task: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("task not found (status %d)", resp.StatusCode)
	}

	var task types.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode task: %w", err)
	}

	if task.NodeID != a.nodeID {
		return nil, fmt.Errorf("task %s is not assigned to this node (assigned to %s)", taskID, task.NodeID)
	}

	return &task, nil
}

func (a *Agent) GetPod(podID string) (*types.Pod, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	podExec, ok := a.runningPods[podID]
	if !ok {
		return nil, false
	}
	return podExec.pod, true
}

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

func (a *Agent) CleanupPod(ctx context.Context, podID string) error {
	a.mu.RLock()
	podExec, ok := a.runningPods[podID]
	a.mu.RUnlock()

	if !ok {
		return fmt.Errorf("pod %s not found or not running on this worker", podID)
	}

	log.Printf("cleaning up pod %s on worker request", podID)

	if podExec.cancelFunc != nil {
		podExec.cancelFunc()
	}

	podExec.mu.RLock()
	containerIDs := make(map[string]string)
	for name, id := range podExec.containerIDs {
		containerIDs[name] = id
	}
	networkID := podExec.networkID
	podExec.mu.RUnlock()

	for name, containerID := range containerIDs {
		log.Printf("stopping container %s (id: %s)", name, containerID)
		if err := a.dockerClient.StopContainer(ctx, containerID); err != nil {
			log.Printf("error stopping container %s: %v", name, err)
		}

		log.Printf("removing container %s (id: %s)", name, containerID)
		if err := a.dockerClient.RemoveContainer(ctx, containerID); err != nil {
			log.Printf("error removing container %s: %v", name, err)
		}
	}

	if networkID != "" {
		log.Printf("removing pod network: %s", networkID)
		if err := a.dockerClient.RemovePodNetwork(ctx, networkID); err != nil {
			log.Printf("error removing pod network %s: %v", networkID, err)
		}
	}

	a.untrackPodExecution(podID)

	return nil
}

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

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
	"github.com/danpasecinic/podling/internal/worker/health"
)

// PodExecution tracks the state of a running pod
type PodExecution struct {
	pod            *types.Pod
	networkID      string
	containerIDs   map[string]string
	healthCheckers map[string]*health.Checker
	mu             sync.RWMutex
	cancelFunc     context.CancelFunc
}

// ExecutePod executes a pod by running all its containers with shared networking
func (a *Agent) ExecutePod(ctx context.Context, pod *types.Pod) error {
	log.Printf("starting pod execution: %s (id: %s) with %d containers", pod.Name, pod.PodID, len(pod.Containers))

	podCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	execution := &PodExecution{
		pod:            pod,
		containerIDs:   make(map[string]string),
		healthCheckers: make(map[string]*health.Checker),
		cancelFunc:     cancel,
	}

	// Track pod execution
	a.mu.Lock()
	if a.runningPods == nil {
		a.runningPods = make(map[string]*PodExecution)
	}
	a.runningPods[pod.PodID] = execution
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		delete(a.runningPods, pod.PodID)
		a.mu.Unlock()
	}()

	if err := a.updatePodStatus(pod.PodID, types.PodRunning, nil, "", ""); err != nil {
		log.Printf("failed to update pod status to running: %v", err)
	}

	// Create a dedicated network for this pod
	// All containers will share this network namespace
	log.Printf("creating pod network for pod %s", pod.PodID)
	networkID, err := a.dockerClient.CreatePodNetwork(podCtx, pod.PodID)
	if err != nil {
		errMsg := fmt.Sprintf("failed to create pod network: %v", err)
		if updateErr := a.updatePodStatus(
			pod.PodID, types.PodFailed, pod.Containers, errMsg, "NetworkCreateError",
		); updateErr != nil {
			log.Printf("failed to update pod status: %v", updateErr)
		}
		return fmt.Errorf("failed to create pod network: %w", err)
	}

	execution.mu.Lock()
	execution.networkID = networkID
	execution.mu.Unlock()

	log.Printf("pod network created: %s", networkID)

	for i := range pod.Containers {
		container := &pod.Containers[i]
		log.Printf("pulling image for container %s: %s", container.Name, container.Image)
		if err := a.dockerClient.PullImage(podCtx, container.Image); err != nil {
			errMsg := fmt.Sprintf("failed to pull image %s: %v", container.Image, err)
			a.cleanupPodResources(context.Background(), execution)
			if updateErr := a.updatePodStatus(
				pod.PodID, types.PodFailed, pod.Containers, errMsg, "ImagePullError",
			); updateErr != nil {
				log.Printf("failed to update pod status: %v", updateErr)
			}
			return fmt.Errorf("failed to pull image for container %s: %w", container.Name, err)
		}
	}

	// Create all containers in the pod network
	// All containers will share the same network namespace and can communicate via localhost
	for i := range pod.Containers {
		container := &pod.Containers[i]

		env := make([]string, 0, len(container.Env))
		for k, v := range container.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}

		log.Printf("creating container %s from image %s in pod network", container.Name, container.Image)

		var containerID string
		var err error
		if !container.Resources.Limits.IsZero() {
			cpuLimit := container.Resources.Limits.GetCPULimitForDocker()
			memoryLimit := container.Resources.Limits.GetMemoryLimitForDocker()
			containerID, err = a.dockerClient.CreateContainerInNetworkWithResources(
				podCtx, container.Image, env, networkID, cpuLimit, memoryLimit,
			)
		} else {
			containerID, err = a.dockerClient.CreateContainerInNetwork(
				podCtx, container.Image, env, networkID,
			)
		}

		if err != nil {
			errMsg := fmt.Sprintf("failed to create container %s: %v", container.Name, err)
			a.cleanupPodResources(context.Background(), execution)
			if updateErr := a.updatePodStatus(
				pod.PodID, types.PodFailed, pod.Containers, errMsg, "ContainerCreateError",
			); updateErr != nil {
				log.Printf("failed to update pod status: %v", updateErr)
			}
			return fmt.Errorf("failed to create container %s: %w", container.Name, err)
		}

		execution.mu.Lock()
		execution.containerIDs[container.Name] = containerID
		execution.mu.Unlock()

		container.ContainerID = containerID
		container.Status = types.ContainerRunning

		log.Printf("starting container %s (id: %s)", container.Name, containerID)
		if err := a.dockerClient.StartContainer(podCtx, containerID); err != nil {
			errMsg := fmt.Sprintf("failed to start container %s: %v", container.Name, err)
			container.Status = types.ContainerTerminated
			container.Error = err.Error()
			a.cleanupPodResources(context.Background(), execution)
			if updateErr := a.updatePodStatus(
				pod.PodID, types.PodFailed, pod.Containers, errMsg, "ContainerStartError",
			); updateErr != nil {
				log.Printf("failed to update pod status: %v", updateErr)
			}
			return fmt.Errorf("failed to start container %s: %w", container.Name, err)
		}

		now := time.Now()
		container.StartedAt = &now

		if container.LivenessProbe != nil {
			restartPolicy := pod.RestartPolicy
			if restartPolicy == "" {
				restartPolicy = types.RestartPolicyNever
			}

			onUnhealthy := func(cid string) {
				log.Printf("container %s in pod %s is unhealthy", container.Name, pod.PodID)
				for j := range pod.Containers {
					if pod.Containers[j].Name == container.Name {
						pod.Containers[j].HealthStatus = types.HealthStatusUnhealthy
						break
					}
				}
				if err := a.updatePodStatus(
					pod.PodID, pod.Status, pod.Containers,
					fmt.Sprintf("Container %s is unhealthy", container.Name), "Unhealthy",
				); err != nil {
					log.Printf("failed to update pod status: %v", err)
				}
			}

			checker := health.NewChecker(
				fmt.Sprintf("%s/%s", pod.PodID, container.Name),
				containerID,
				container.LivenessProbe,
				restartPolicy,
				a.dockerClient,
				onUnhealthy,
			)

			execution.mu.Lock()
			execution.healthCheckers[container.Name] = checker
			execution.mu.Unlock()

			go checker.Start(podCtx)

			log.Printf("started liveness probe for container %s in pod %s", container.Name, pod.PodID)
		}
	}

	// Get pod IP from the pod network
	// All containers in the pod share this IP address
	var podIP string
	if len(pod.Containers) > 0 && pod.Containers[0].ContainerID != "" {
		ip, err := a.dockerClient.GetNetworkIP(podCtx, pod.Containers[0].ContainerID, networkID)
		if err != nil {
			log.Printf("failed to get pod IP from network: %v", err)
		} else {
			podIP = ip
			log.Printf("pod %s assigned IP: %s (shared by all containers)", pod.PodID, podIP)
		}
	}

	// Update pod with container IDs and IP
	if err := a.updatePodStatusWithIP(pod.PodID, types.PodRunning, pod.Containers, podIP, "", ""); err != nil {
		log.Printf("failed to update pod with container IDs: %v", err)
	}

	log.Printf("all containers started for pod %s", pod.PodID)
	errChan := make(chan error, len(pod.Containers))
	var wg sync.WaitGroup

	for i := range pod.Containers {
		wg.Add(1)
		go func(container *types.Container) {
			defer wg.Done()

			containerID := container.ContainerID
			exitCode64, err := a.dockerClient.WaitContainer(podCtx, containerID)

			now := time.Now()
			container.FinishedAt = &now

			if err != nil {
				log.Printf("error waiting for container %s: %v", container.Name, err)
				container.Status = types.ContainerTerminated
				container.Error = err.Error()
				errChan <- fmt.Errorf("container %s failed: %w", container.Name, err)
				return
			}

			exitCode := int(exitCode64)
			container.Status = types.ContainerTerminated
			container.ExitCode = &exitCode

			if exitCode != 0 {
				log.Printf("container %s exited with code %d", container.Name, exitCode)
				errChan <- fmt.Errorf("container %s exited with code %d", container.Name, exitCode)
			} else {
				log.Printf("container %s completed successfully", container.Name)
			}
		}(&pod.Containers[i])
	}

	wg.Wait()
	close(errChan)

	execution.mu.Lock()
	for _, checker := range execution.healthCheckers {
		checker.Stop()
	}
	execution.mu.Unlock()

	var containerErrors []error
	for err := range errChan {
		if err != nil {
			containerErrors = append(containerErrors, err)
		}
	}

	a.cleanupPodResources(context.Background(), execution)

	finalStatus := types.PodSucceeded
	message := "All containers completed successfully"
	reason := "Completed"

	if len(containerErrors) > 0 {
		finalStatus = types.PodFailed
		message = fmt.Sprintf("%d container(s) failed", len(containerErrors))
		reason = "ContainerError"
	}

	if err := a.updatePodStatus(pod.PodID, finalStatus, pod.Containers, message, reason); err != nil {
		log.Printf("failed to update final pod status: %v", err)
	}

	if len(containerErrors) > 0 {
		return fmt.Errorf("pod failed: %s", message)
	}

	log.Printf("pod %s completed successfully", pod.PodID)
	return nil
}

// cleanupPodResources stops and removes all containers in a pod, and removes the pod network
func (a *Agent) cleanupPodResources(ctx context.Context, execution *PodExecution) {
	execution.mu.RLock()
	containerIDs := make(map[string]string)
	for name, id := range execution.containerIDs {
		containerIDs[name] = id
	}
	networkID := execution.networkID
	execution.mu.RUnlock()

	for name, containerID := range containerIDs {
		log.Printf("cleaning up container %s (id: %s)", name, containerID)

		if err := a.dockerClient.StopContainer(ctx, containerID); err != nil {
			log.Printf("error stopping container %s: %v", name, err)
		}

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
}

// updatePodStatus sends a pod status update to the master
func (a *Agent) updatePodStatus(
	podID string, status types.PodStatus, containers []types.Container, message, reason string,
) error {
	url := fmt.Sprintf("%s/api/v1/pods/%s/status", a.masterURL, podID)

	payload := map[string]interface{}{
		"status": status,
	}

	if containers != nil {
		payload["containers"] = containers
	}
	if message != "" {
		payload["message"] = message
	}
	if reason != "" {
		payload["reason"] = reason
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal pod status: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create pod status request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send pod status update: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pod status update returned status %d", resp.StatusCode)
	}

	return nil
}

// updatePodStatusWithIP sends a pod status update to the master including pod IP
func (a *Agent) updatePodStatusWithIP(
	podID string, status types.PodStatus, containers []types.Container, podIP, message, reason string,
) error {
	url := fmt.Sprintf("%s/api/v1/pods/%s/status", a.masterURL, podID)

	payload := map[string]interface{}{
		"status": status,
	}

	if containers != nil {
		payload["containers"] = containers
	}
	if message != "" {
		payload["message"] = message
	}
	if reason != "" {
		payload["reason"] = reason
	}

	// Add pod IP as annotation
	if podIP != "" {
		payload["annotations"] = map[string]string{
			"podling.io/pod-ip": podIP,
		}
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal pod status: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create pod status request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send pod status update: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pod status update returned status %d", resp.StatusCode)
	}

	return nil
}

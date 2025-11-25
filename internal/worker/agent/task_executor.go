package agent

import (
	"context"
	"fmt"
	"log"

	"github.com/danpasecinic/podling/internal/types"
	"github.com/danpasecinic/podling/internal/worker/docker"
	"github.com/danpasecinic/podling/internal/worker/health"
)

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

	var containerID string
	var err error

	if len(task.Ports) > 0 {
		ports := make([]docker.PortMapping, len(task.Ports))
		for i, port := range task.Ports {
			ports[i] = docker.PortMapping{
				ContainerPort: port.ContainerPort,
				HostPort:      port.HostPort,
				Protocol:      port.Protocol,
			}
		}

		cpuLimit := float64(0)
		memoryLimit := int64(0)
		if !task.Resources.Limits.IsZero() {
			cpuLimit = task.Resources.Limits.GetCPULimitForDocker()
			memoryLimit = task.Resources.Limits.GetMemoryLimitForDocker()
		}

		containerID, err = a.dockerClient.CreateContainerWithResourcesAndPorts(
			ctx, task.Image, env, cpuLimit, memoryLimit, ports,
		)
	} else if !task.Resources.Limits.IsZero() {
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
		}
	}

	if err := a.dockerClient.RemoveContainer(ctx, containerID); err != nil {
		log.Printf("failed to remove container %s: %v", containerID, err)
	}

	return nil
}

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

func (a *Agent) GetTask(taskID string) (*types.Task, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	task, ok := a.runningTasks[taskID]
	return task, ok
}

func (a *Agent) GetTaskLogs(ctx context.Context, taskID string, tail int) (string, error) {
	a.mu.RLock()
	task, ok := a.runningTasks[taskID]
	a.mu.RUnlock()

	if !ok {
		fetchedTask, err := a.getTaskFromMaster(taskID)
		if err != nil {
			return "", fmt.Errorf("task %s not found: %w", taskID, err)
		}
		task = fetchedTask
	}

	if task.ContainerID == "" {
		return "", fmt.Errorf("task %s has no associated container", taskID)
	}

	return a.dockerClient.GetContainerLogs(ctx, task.ContainerID, tail)
}

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

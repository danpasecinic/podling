package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danpasecinic/podling/internal/types"
)

// Client represents a Podling client
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Podling client with the given base URL
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateTask creates a new task with the given parameters
func (c *Client) CreateTask(name, image string, env map[string]string) (*types.Task, error) {
	payload := map[string]interface{}{
		"name":  name,
		"image": image,
		"env":   env,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/v1/tasks",
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return nil, fmt.Errorf("post request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var task types.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &task, nil
}

// CreateTaskWithPorts creates a new task with specified port mappings
func (c *Client) CreateTaskWithPorts(name, image string, env map[string]string, portSpecs []string) (
	*types.Task,
	error,
) {
	var ports []types.ContainerPort
	for _, portSpec := range portSpecs {
		parts := strings.Split(portSpec, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid port spec %q, expected format hostPort:containerPort", portSpec)
		}

		hostPort, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid host port %q: %w", parts[0], err)
		}

		containerPort, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid container port %q: %w", parts[1], err)
		}

		ports = append(
			ports, types.ContainerPort{
				ContainerPort: containerPort,
				HostPort:      hostPort,
				Protocol:      "TCP",
			},
		)
	}

	payload := map[string]interface{}{
		"name":  name,
		"image": image,
		"env":   env,
		"ports": ports,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/v1/tasks",
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return nil, fmt.Errorf("post request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var task types.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &task, nil
}

// ListTasks retrieves all tasks from the master
func (c *Client) ListTasks() ([]types.Task, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/tasks")
	if err != nil {
		return nil, fmt.Errorf("get request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var tasks []types.Task
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return tasks, nil
}

// GetTask retrieves a specific task by ID
func (c *Client) GetTask(taskID string) (*types.Task, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/tasks/" + taskID)
	if err != nil {
		return nil, fmt.Errorf("get request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var task types.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &task, nil
}

func (c *Client) ListNodes() ([]types.Node, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/nodes")
	if err != nil {
		return nil, fmt.Errorf("get request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var nodes []types.Node
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return nodes, nil
}

// GetNode retrieves a specific node by ID
func (c *Client) GetNode(nodeID string) (*types.Node, error) {
	nodes, err := c.ListNodes()
	if err != nil {
		return nil, err
	}

	for _, node := range nodes {
		if node.NodeID == nodeID {
			return &node, nil
		}
	}

	return nil, fmt.Errorf("node %s not found", nodeID)
}

func (c *Client) GetTaskLogs(task *types.Task, tail int) (string, error) {
	// Get the node to find the worker URL
	nodes, err := c.ListNodes()
	if err != nil {
		return "", fmt.Errorf("list nodes: %w", err)
	}

	var workerURL string
	for _, node := range nodes {
		if node.NodeID == task.NodeID {
			workerURL = fmt.Sprintf("http://%s:%d", node.Hostname, node.Port)
			break
		}
	}

	if workerURL == "" {
		return "", fmt.Errorf("worker node not found: %s", task.NodeID)
	}

	// Get logs from worker
	url := fmt.Sprintf("%s/api/v1/tasks/%s/logs?tail=%d", workerURL, task.TaskID, tail)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("get request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	logs, ok := result["logs"].(string)
	if !ok {
		return "", fmt.Errorf("invalid logs format in response")
	}

	return logs, nil
}

// CreatePod creates a new pod with the given containers
func (c *Client) CreatePod(name, namespace string, labels map[string]string, containers []types.Container) (
	*types.Pod,
	error,
) {
	payload := map[string]interface{}{
		"name":       name,
		"containers": containers,
	}

	if namespace != "" {
		payload["namespace"] = namespace
	}

	if len(labels) > 0 {
		payload["labels"] = labels
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/v1/pods",
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return nil, fmt.Errorf("post request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var pod types.Pod
	if err := json.NewDecoder(resp.Body).Decode(&pod); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &pod, nil
}

// ListPods retrieves all pods from the master
func (c *Client) ListPods() ([]types.Pod, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/pods")
	if err != nil {
		return nil, fmt.Errorf("get request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var pods []types.Pod
	if err := json.NewDecoder(resp.Body).Decode(&pods); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return pods, nil
}

// GetPod retrieves a specific pod by ID
func (c *Client) GetPod(podID string) (*types.Pod, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/pods/" + podID)
	if err != nil {
		return nil, fmt.Errorf("get request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var pod types.Pod
	if err := json.NewDecoder(resp.Body).Decode(&pod); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &pod, nil
}

// GetPodLogs retrieves logs from a pod's containers
func (c *Client) GetPodLogs(podID string, containerName string, tail int) (map[string]string, error) {
	pod, err := c.GetPod(podID)
	if err != nil {
		return nil, err
	}

	if pod.NodeID == "" {
		return nil, fmt.Errorf("pod is not scheduled to any node")
	}

	node, err := c.GetNode(pod.NodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get node info: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/api/v1/pods/%s/logs?tail=%d", node.Hostname, node.Port, podID, tail)
	if containerName != "" {
		url += "&container=" + containerName
	}

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("get request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Logs map[string]string `json:"logs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Logs, nil
}

// DeletePod deletes a pod by ID
func (c *Client) DeletePod(podID string) error {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+"/api/v1/pods/"+podID, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// CreateService creates a new service
func (c *Client) CreateService(
	name, namespace string, selector map[string]string, ports []types.ServicePort, labels map[string]string,
	serviceType, sessionAffinity string,
) (*types.Service, error) {
	payload := map[string]interface{}{
		"name":     name,
		"selector": selector,
		"ports":    ports,
	}

	if namespace != "" {
		payload["namespace"] = namespace
	}

	if len(labels) > 0 {
		payload["labels"] = labels
	}

	if serviceType != "" {
		payload["type"] = serviceType
	}

	if sessionAffinity != "" {
		payload["sessionAffinity"] = sessionAffinity
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/v1/services",
		"application/json",
		bytes.NewReader(data),
	)
	if err != nil {
		return nil, fmt.Errorf("post request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var service types.Service
	if err := json.NewDecoder(resp.Body).Decode(&service); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &service, nil
}

// ListServices retrieves all services, optionally filtered by namespace
func (c *Client) ListServices(namespace string) ([]types.Service, error) {
	url := c.baseURL + "/api/v1/services"
	if namespace != "" {
		url += "?namespace=" + namespace
	}

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("get request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var services []types.Service
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return services, nil
}

// GetService retrieves a specific service by ID
func (c *Client) GetService(serviceID string) (*types.Service, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/services/" + serviceID)
	if err != nil {
		return nil, fmt.Errorf("get request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var service types.Service
	if err := json.NewDecoder(resp.Body).Decode(&service); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &service, nil
}

// GetEndpoints retrieves endpoints for a specific service
func (c *Client) GetEndpoints(serviceID string) (*types.Endpoints, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/services/" + serviceID + "/endpoints")
	if err != nil {
		return nil, fmt.Errorf("get request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var endpoints types.Endpoints
	if err := json.NewDecoder(resp.Body).Decode(&endpoints); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &endpoints, nil
}

// DeleteService deletes a service by ID
func (c *Client) DeleteService(serviceID string) error {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+"/api/v1/services/"+serviceID, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Prune removes unused resources and returns a summary of the operation
func (c *Client) Prune() (*types.PruneResult, error) {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/prune", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prune request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result types.PruneResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// PruneAll removes all resources from the system
func (c *Client) PruneAll() (*types.PruneResult, error) {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/prune?all=true", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prune request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result types.PruneResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

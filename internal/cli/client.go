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

type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
	apiKey     string
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) SetAPIKey(apiKey string) {
	c.apiKey = apiKey
}

func (c *Client) addAuthHeader(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	} else if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
}

func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	c.addAuthHeader(req)
	return c.httpClient.Do(req)
}

func (c *Client) checkResponse(resp *http.Response, allowedStatuses ...int) error {
	if len(allowedStatuses) == 0 {
		allowedStatuses = []int{http.StatusOK}
	}
	for _, status := range allowedStatuses {
		if resp.StatusCode == status {
			return nil
		}
	}
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
}

func (c *Client) getJSON(url string, result interface{}) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return fmt.Errorf("get request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := c.checkResponse(resp); err != nil {
		return err
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) postJSON(url string, payload, result interface{}) error {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doRequest(req)
	if err != nil {
		return fmt.Errorf("post request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := c.checkResponse(resp, http.StatusOK, http.StatusCreated); err != nil {
		return err
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

func (c *Client) deleteRequest(url string) error {
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return fmt.Errorf("delete request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return c.checkResponse(resp)
}

func (c *Client) CreateTask(name, image string, env map[string]string) (*types.Task, error) {
	payload := map[string]interface{}{
		"name":  name,
		"image": image,
		"env":   env,
	}

	var task types.Task
	if err := c.postJSON(c.baseURL+"/api/v1/tasks", payload, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (c *Client) CreateTaskWithPorts(name, image string, env map[string]string, portSpecs []string) (
	*types.Task,
	error,
) {
	ports, err := parsePortSpecs(portSpecs)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"name":  name,
		"image": image,
		"env":   env,
		"ports": ports,
	}

	var task types.Task
	if err := c.postJSON(c.baseURL+"/api/v1/tasks", payload, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

func parsePortSpecs(portSpecs []string) ([]types.ContainerPort, error) {
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
	return ports, nil
}

func (c *Client) ListTasks() ([]types.Task, error) {
	var tasks []types.Task
	if err := c.getJSON(c.baseURL+"/api/v1/tasks", &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (c *Client) GetTask(taskID string) (*types.Task, error) {
	var task types.Task
	if err := c.getJSON(c.baseURL+"/api/v1/tasks/"+taskID, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (c *Client) ListNodes() ([]types.Node, error) {
	var nodes []types.Node
	if err := c.getJSON(c.baseURL+"/api/v1/nodes", &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

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
	node, err := c.GetNode(task.NodeID)
	if err != nil {
		return "", fmt.Errorf("worker node not found: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/api/v1/tasks/%s/logs?tail=%d", node.Hostname, node.Port, task.TaskID, tail)

	var result map[string]interface{}
	if err := c.getJSON(url, &result); err != nil {
		return "", err
	}

	logs, ok := result["logs"].(string)
	if !ok {
		return "", fmt.Errorf("invalid logs format in response")
	}
	return logs, nil
}

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

	var pod types.Pod
	if err := c.postJSON(c.baseURL+"/api/v1/pods", payload, &pod); err != nil {
		return nil, err
	}
	return &pod, nil
}

func (c *Client) ListPods() ([]types.Pod, error) {
	var pods []types.Pod
	if err := c.getJSON(c.baseURL+"/api/v1/pods", &pods); err != nil {
		return nil, err
	}
	return pods, nil
}

func (c *Client) GetPod(podID string) (*types.Pod, error) {
	var pod types.Pod
	if err := c.getJSON(c.baseURL+"/api/v1/pods/"+podID, &pod); err != nil {
		return nil, err
	}
	return &pod, nil
}

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

	var result struct {
		Logs map[string]string `json:"logs"`
	}
	if err := c.getJSON(url, &result); err != nil {
		return nil, err
	}
	return result.Logs, nil
}

func (c *Client) DeletePod(podID string) error {
	return c.deleteRequest(c.baseURL + "/api/v1/pods/" + podID)
}

func (c *Client) CreateService(
	name, namespace string, selector map[string]string, ports []types.ServicePort,
	labels map[string]string, serviceType, sessionAffinity string,
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

	var service types.Service
	if err := c.postJSON(c.baseURL+"/api/v1/services", payload, &service); err != nil {
		return nil, err
	}
	return &service, nil
}

func (c *Client) ListServices(namespace string) ([]types.Service, error) {
	url := c.baseURL + "/api/v1/services"
	if namespace != "" {
		url += "?namespace=" + namespace
	}

	var services []types.Service
	if err := c.getJSON(url, &services); err != nil {
		return nil, err
	}
	return services, nil
}

func (c *Client) GetService(serviceID string) (*types.Service, error) {
	var service types.Service
	if err := c.getJSON(c.baseURL+"/api/v1/services/"+serviceID, &service); err != nil {
		return nil, err
	}
	return &service, nil
}

func (c *Client) GetEndpoints(serviceID string) (*types.Endpoints, error) {
	var endpoints types.Endpoints
	if err := c.getJSON(c.baseURL+"/api/v1/services/"+serviceID+"/endpoints", &endpoints); err != nil {
		return nil, err
	}
	return &endpoints, nil
}

func (c *Client) DeleteService(serviceID string) error {
	return c.deleteRequest(c.baseURL + "/api/v1/services/" + serviceID)
}

func (c *Client) Prune() (*types.PruneResult, error) {
	var result types.PruneResult
	if err := c.postJSON(c.baseURL+"/api/v1/prune", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) PruneAll() (*types.PruneResult, error) {
	var result types.PruneResult
	if err := c.postJSON(c.baseURL+"/api/v1/prune?all=true", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

type LoginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt    string `json:"expiresAt"`
	User         struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
	} `json:"user"`
}

func (c *Client) Login(username, password string) (*LoginResponse, error) {
	payload := map[string]string{
		"username": username,
		"password": password,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/auth/login", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("login failed: %s", string(body))
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	c.token = loginResp.Token
	return &loginResp, nil
}

func (c *Client) GetCurrentUser() (map[string]interface{}, error) {
	var userInfo map[string]interface{}
	if err := c.getJSON(c.baseURL+"/api/v1/auth/me", &userInfo); err != nil {
		return nil, err
	}
	return userInfo, nil
}

package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/danpasecinic/podling/internal/types"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

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

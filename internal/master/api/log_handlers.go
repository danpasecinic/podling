package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

type nodeInfo struct {
	hostname string
	port     int
}

func (s *Server) getTaskNodeInfo(taskID string) (*nodeInfo, string, error) {
	task, err := s.store.GetTask(taskID)
	if err != nil {
		return nil, "task not found", err
	}
	if task.NodeID == "" {
		return nil, "task is not scheduled to any node", nil
	}
	return s.getNodeInfo(task.NodeID)
}

func (s *Server) getPodNodeInfo(podID string) (*nodeInfo, string, error) {
	pod, err := s.store.GetPod(podID)
	if err != nil {
		return nil, "pod not found", err
	}
	if pod.NodeID == "" {
		return nil, "pod is not scheduled to any node", nil
	}
	return s.getNodeInfo(pod.NodeID)
}

func (s *Server) getNodeInfo(nodeID string) (*nodeInfo, string, error) {
	node, err := s.store.GetNode(nodeID)
	if err != nil {
		return nil, "worker node not found", err
	}
	return &nodeInfo{hostname: node.Hostname, port: node.Port}, "", nil
}

func buildWorkerURL(node *nodeInfo, path string, params url.Values) string {
	u := url.URL{
		Scheme:   "http",
		Host:     fmt.Sprintf("%s:%d", node.hostname, node.port),
		Path:     path,
		RawQuery: params.Encode(),
	}
	return u.String()
}

func proxyJSONResponse(c echo.Context, workerURL string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(workerURL)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("failed to reach worker: %v", err)})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return c.JSON(resp.StatusCode, map[string]string{"error": string(body)})
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to decode worker response"})
	}

	return c.JSON(http.StatusOK, result)
}

func proxySSEStream(c echo.Context, ctx context.Context, workerURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, workerURL, nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create request"})
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("failed to reach worker: %v", err)})
	}
	defer resp.Body.Close()

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	reader := bufio.NewReader(resp.Body)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					fmt.Fprintf(c.Response(), "event: error\ndata: %s\n\n", err.Error())
					c.Response().Flush()
				}
				return nil
			}
			_, _ = fmt.Fprint(c.Response(), line)
			if strings.HasPrefix(line, "data:") || line == "\n" {
				c.Response().Flush()
			}
		}
	}
}

func getTailParam(c echo.Context) string {
	if tail := c.QueryParam("tail"); tail != "" {
		return tail
	}
	return "100"
}

func (s *Server) GetTaskLogs(c echo.Context) error {
	taskID := c.Param("id")
	node, errMsg, err := s.getTaskNodeInfo(taskID)
	if err != nil || node == nil {
		status := http.StatusNotFound
		if node == nil && err == nil {
			status = http.StatusBadRequest
		}
		return c.JSON(status, map[string]string{"error": errMsg})
	}

	params := url.Values{"tail": {getTailParam(c)}}
	workerURL := buildWorkerURL(node, fmt.Sprintf("/api/v1/tasks/%s/logs", taskID), params)
	return proxyJSONResponse(c, workerURL)
}

func (s *Server) StreamTaskLogs(c echo.Context) error {
	taskID := c.Param("id")
	node, errMsg, err := s.getTaskNodeInfo(taskID)
	if err != nil || node == nil {
		status := http.StatusNotFound
		if node == nil && err == nil {
			status = http.StatusBadRequest
		}
		return c.JSON(status, map[string]string{"error": errMsg})
	}

	params := url.Values{"tail": {getTailParam(c)}}
	workerURL := buildWorkerURL(node, fmt.Sprintf("/api/v1/tasks/%s/logs/stream", taskID), params)
	return proxySSEStream(c, c.Request().Context(), workerURL)
}

func (s *Server) GetPodLogs(c echo.Context) error {
	podID := c.Param("id")
	node, errMsg, err := s.getPodNodeInfo(podID)
	if err != nil || node == nil {
		status := http.StatusNotFound
		if node == nil && err == nil {
			status = http.StatusBadRequest
		}
		return c.JSON(status, map[string]string{"error": errMsg})
	}

	params := url.Values{"tail": {getTailParam(c)}}
	if container := c.QueryParam("container"); container != "" {
		params.Set("container", container)
	}
	workerURL := buildWorkerURL(node, fmt.Sprintf("/api/v1/pods/%s/logs", podID), params)
	return proxyJSONResponse(c, workerURL)
}

func (s *Server) StreamPodLogs(c echo.Context) error {
	podID := c.Param("id")
	node, errMsg, err := s.getPodNodeInfo(podID)
	if err != nil || node == nil {
		status := http.StatusNotFound
		if node == nil && err == nil {
			status = http.StatusBadRequest
		}
		return c.JSON(status, map[string]string{"error": errMsg})
	}

	params := url.Values{"tail": {getTailParam(c)}}
	if container := c.QueryParam("container"); container != "" {
		params.Set("container", container)
	}
	workerURL := buildWorkerURL(node, fmt.Sprintf("/api/v1/pods/%s/logs/stream", podID), params)
	return proxySSEStream(c, c.Request().Context(), workerURL)
}

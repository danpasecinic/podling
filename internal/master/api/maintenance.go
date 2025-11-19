package api

import (
	"log"
	"net/http"

	"github.com/danpasecinic/podling/internal/types"
	"github.com/labstack/echo/v4"
)

// Prune handles POST /api/v1/prune
// Removes old/completed resources from the system
func (s *Server) Prune(c echo.Context) error {
	all := c.QueryParam("all") == "true"

	result := &types.PruneResult{}

	if all {
		result = s.pruneAll()
	} else {
		result = s.pruneCompleted()
	}

	log.Printf(
		"Prune completed: %d pods, %d nodes, %d services, %d tasks removed",
		result.PodsRemoved, result.NodesRemoved, result.ServicesRemoved, result.TasksRemoved,
	)

	return c.JSON(http.StatusOK, result)
}

func (s *Server) pruneAll() *types.PruneResult {
	result := &types.PruneResult{}

	pods, err := s.store.ListPods()
	if err == nil {
		for _, pod := range pods {
			if err := s.store.DeletePod(pod.PodID); err == nil {
				result.PodsRemoved++
			}
		}
	}

	tasks, err := s.store.ListTasks()
	if err == nil {
		for range tasks {
			result.TasksRemoved++
		}
	}

	nodes, err := s.store.ListNodes()
	if err == nil {
		result.NodesRemoved = len(nodes)
	}

	services, err := s.store.ListServices("")
	if err == nil {
		for _, service := range services {
			if err := s.store.DeleteService(service.ServiceID); err == nil {
				result.ServicesRemoved++
				_ = s.store.DeleteEndpoints(service.ServiceID)
			}
		}
	}

	return result
}

func (s *Server) pruneCompleted() *types.PruneResult {
	result := &types.PruneResult{}

	pods, err := s.store.ListPods()
	if err == nil {
		for _, pod := range pods {
			if pod.Status == types.PodSucceeded || pod.Status == types.PodFailed {
				if err := s.store.DeletePod(pod.PodID); err == nil {
					result.PodsRemoved++
				}
			}
		}
	}

	tasks, err := s.store.ListTasks()
	if err == nil {
		for _, task := range tasks {
			if task.Status == types.TaskCompleted || task.Status == types.TaskFailed {
				result.TasksRemoved++
			}
		}
	}

	nodes, err := s.store.ListNodes()
	if err == nil {
		for _, node := range nodes {
			if node.Status == types.NodeOffline {
				result.NodesRemoved++
			}
		}
	}

	return result
}

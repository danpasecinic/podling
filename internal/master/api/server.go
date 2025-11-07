package api

import (
	"github.com/danpasecinic/podling/internal/master/scheduler"
	"github.com/danpasecinic/podling/internal/master/services"
	"github.com/danpasecinic/podling/internal/master/state"
	"github.com/labstack/echo/v4"
)

// Server handles HTTP requests for the master API.
type Server struct {
	store              state.StateStore
	scheduler          scheduler.Scheduler
	endpointController *services.EndpointController
}

// NewServer creates a new API server with the given state store and scheduler.
func NewServer(
	store state.StateStore, sched scheduler.Scheduler, endpointController *services.EndpointController,
) *Server {
	return &Server{
		store:              store,
		scheduler:          sched,
		endpointController: endpointController,
	}
}

// RegisterRoutes registers all API endpoints with the Echo router.
// Routes are grouped under /api/v1 for versioning.
func (s *Server) RegisterRoutes(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	// Task routes
	v1.POST("/tasks", s.CreateTask)
	v1.GET("/tasks", s.ListTasks)
	v1.GET("/tasks/:id", s.GetTask)
	v1.PUT("/tasks/:id/status", s.UpdateTaskStatus)

	// Pod routes
	v1.POST("/pods", s.CreatePod)
	v1.GET("/pods", s.ListPods)
	v1.GET("/pods/:id", s.GetPod)
	v1.PUT("/pods/:id/status", s.UpdatePodStatus)
	v1.DELETE("/pods/:id", s.DeletePod)

	// Node routes
	v1.POST("/nodes/register", s.RegisterNode)
	v1.POST("/nodes/:id/heartbeat", s.NodeHeartbeat)
	v1.GET("/nodes", s.ListNodes)

	// Service routes
	v1.POST("/services", s.CreateService)
	v1.GET("/services", s.ListServices)
	v1.GET("/services/:id", s.GetService)
	v1.PUT("/services/:id", s.UpdateService)
	v1.DELETE("/services/:id", s.DeleteService)
	v1.GET("/services/:id/endpoints", s.GetEndpoints)
}

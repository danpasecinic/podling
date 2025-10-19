package api

import (
	"github.com/danpasecinic/podling/internal/master/scheduler"
	"github.com/danpasecinic/podling/internal/master/state"
	"github.com/labstack/echo/v4"
)

type Server struct {
	store     state.StateStore
	scheduler scheduler.Scheduler
}

func NewServer(store state.StateStore, sched scheduler.Scheduler) *Server {
	return &Server{
		store:     store,
		scheduler: sched,
	}
}

func (s *Server) RegisterRoutes(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.POST("/tasks", s.CreateTask)
	v1.GET("/tasks", s.ListTasks)
	v1.GET("/tasks/:id", s.GetTask)
	v1.PUT("/tasks/:id/status", s.UpdateTaskStatus)

	v1.POST("/nodes/register", s.RegisterNode)
	v1.POST("/nodes/:id/heartbeat", s.NodeHeartbeat)
	v1.GET("/nodes", s.ListNodes)
}

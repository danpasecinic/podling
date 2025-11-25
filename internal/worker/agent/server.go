package agent

import (
	"github.com/labstack/echo/v4"
)

type Server struct {
	nodeID   string
	hostname string
	port     int
	agent    *Agent
}

func NewServer(nodeID, hostname string, port int, agent *Agent) *Server {
	return &Server{
		nodeID:   nodeID,
		hostname: hostname,
		port:     port,
		agent:    agent,
	}
}

func (s *Server) RegisterRoutes(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.POST("/tasks/:id/execute", s.ExecuteTask)
	v1.GET("/tasks/:id/status", s.GetTaskStatus)
	v1.GET("/tasks/:id/logs", s.GetTaskLogs)

	v1.POST("/pods/:id/execute", s.ExecutePod)
	v1.GET("/pods/:id/status", s.GetPodStatus)
	v1.GET("/pods/:id/logs", s.GetPodLogs)
	v1.DELETE("/pods/:id", s.DeletePod)
}

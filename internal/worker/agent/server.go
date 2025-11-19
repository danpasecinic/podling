package agent

import (
	"github.com/labstack/echo/v4"
)

// Server handles HTTP requests for the worker agent.
type Server struct {
	nodeID   string
	hostname string
	port     int
	agent    *Agent
}

// NewServer creates a new worker API server.
func NewServer(nodeID, hostname string, port int, agent *Agent) *Server {
	return &Server{
		nodeID:   nodeID,
		hostname: hostname,
		port:     port,
		agent:    agent,
	}
}

// RegisterRoutes registers all worker API endpoints.
func (s *Server) RegisterRoutes(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.POST("/tasks/:id/execute", s.ExecuteTask)
	v1.GET("/tasks/:id/status", s.GetTaskStatus)
	v1.GET("/tasks/:id/logs", s.GetTaskLogs)

	v1.POST("/pods/:id/execute", s.ExecutePod)
	v1.GET("/pods/:id/status", s.GetPodStatus)
	v1.GET("/pods/:id/logs", s.GetPodLogs)
}

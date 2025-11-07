package api

import (
	"net/http"
	"time"

	"github.com/danpasecinic/podling/internal/types"
	"github.com/labstack/echo/v4"
)

// CreateServiceRequest represents a request to create a new service
type CreateServiceRequest struct {
	Name            string              `json:"name" validate:"required"`
	Namespace       string              `json:"namespace"`
	Type            types.ServiceType   `json:"type"`
	Selector        map[string]string   `json:"selector" validate:"required"`
	Ports           []types.ServicePort `json:"ports" validate:"required"`
	Labels          map[string]string   `json:"labels"`
	Annotations     map[string]string   `json:"annotations"`
	SessionAffinity string              `json:"sessionAffinity"`
}

// UpdateServiceRequest represents a request to update a service
type UpdateServiceRequest struct {
	Selector        *map[string]string   `json:"selector"`
	Ports           *[]types.ServicePort `json:"ports"`
	Labels          *map[string]string   `json:"labels"`
	Annotations     *map[string]string   `json:"annotations"`
	SessionAffinity *string              `json:"sessionAffinity"`
}

// CreateService handles POST /api/v1/services
// Creates a new service and allocates a ClusterIP if needed
func (s *Server) CreateService(c echo.Context) error {
	var req CreateServiceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}

	if len(req.Selector) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "selector is required"})
	}

	if len(req.Ports) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "at least one port is required"})
	}

	namespace := req.Namespace
	if namespace == "" {
		namespace = "default"
	}

	serviceType := req.Type
	if serviceType == "" {
		serviceType = types.ServiceTypeClusterIP
	}

	var clusterIP string
	if serviceType == types.ServiceTypeClusterIP {
		ip, err := s.endpointController.AllocateClusterIP()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to allocate cluster IP"})
		}
		clusterIP = ip
	}

	for i := range req.Ports {
		if req.Ports[i].TargetPort == 0 {
			req.Ports[i].TargetPort = req.Ports[i].Port
		}
		if req.Ports[i].Protocol == "" {
			req.Ports[i].Protocol = "TCP"
		}
	}

	service := types.Service{
		ServiceID:       generateID(),
		Name:            req.Name,
		Namespace:       namespace,
		Type:            serviceType,
		ClusterIP:       clusterIP,
		Selector:        req.Selector,
		Ports:           req.Ports,
		Labels:          req.Labels,
		Annotations:     req.Annotations,
		SessionAffinity: req.SessionAffinity,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.store.AddService(service); err != nil {
		if clusterIP != "" {
			_ = s.endpointController.ReleaseClusterIP(clusterIP)
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, service)
}

// ListServices handles GET /api/v1/services
// Returns all services, optionally filtered by namespace
func (s *Server) ListServices(c echo.Context) error {
	namespace := c.QueryParam("namespace")

	services, err := s.store.ListServices(namespace)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, services)
}

// GetService handles GET /api/v1/services/:id
// Returns details for a specific service
func (s *Server) GetService(c echo.Context) error {
	serviceID := c.Param("id")

	service, err := s.store.GetService(serviceID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "service not found"})
	}

	return c.JSON(http.StatusOK, service)
}

// UpdateService handles PUT /api/v1/services/:id
// Updates a service's configuration
func (s *Server) UpdateService(c echo.Context) error {
	serviceID := c.Param("id")

	var req UpdateServiceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	update := types.ServiceUpdate{
		Selector:        req.Selector,
		Ports:           req.Ports,
		Labels:          req.Labels,
		Annotations:     req.Annotations,
		SessionAffinity: req.SessionAffinity,
	}

	if err := s.store.UpdateService(serviceID, update); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	}

	service, _ := s.store.GetService(serviceID)
	return c.JSON(http.StatusOK, service)
}

// DeleteService handles DELETE /api/v1/services/:id
// Deletes a service and its endpoints
func (s *Server) DeleteService(c echo.Context) error {
	serviceID := c.Param("id")

	service, err := s.store.GetService(serviceID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "service not found"})
	}

	if err := s.store.DeleteService(serviceID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	_ = s.store.DeleteEndpoints(serviceID)

	if service.ClusterIP != "" {
		_ = s.endpointController.ReleaseClusterIP(service.ClusterIP)
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "service deleted"})
}

// GetEndpoints handles GET /api/v1/services/:id/endpoints
// Returns the endpoints for a specific service
func (s *Server) GetEndpoints(c echo.Context) error {
	serviceID := c.Param("id")

	endpoints, err := s.store.GetEndpoints(serviceID)
	if err != nil {
		service, svcErr := s.store.GetService(serviceID)
		if svcErr != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "service not found"})
		}

		namespace := service.Namespace
		if namespace == "" {
			namespace = "default"
		}
		endpoints = types.Endpoints{
			ServiceID:   serviceID,
			ServiceName: service.Name,
			Namespace:   namespace,
			Subsets:     []types.EndpointSubset{},
		}
	}

	return c.JSON(http.StatusOK, endpoints)
}

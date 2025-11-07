package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danpasecinic/podling/internal/master/scheduler"
	"github.com/danpasecinic/podling/internal/master/services"
	"github.com/danpasecinic/podling/internal/master/state"
	"github.com/danpasecinic/podling/internal/types"
	"github.com/labstack/echo/v4"
)

func TestCreateService(t *testing.T) {
	e := echo.New()
	store := state.NewInMemoryStore()
	sched := scheduler.NewRoundRobin()
	endpointController := services.NewEndpointController(store)
	server := NewServer(store, sched, endpointController)

	t.Run(
		"create service with ClusterIP", func(t *testing.T) {
			payload := map[string]interface{}{
				"name":      "web-service",
				"namespace": "default",
				"selector": map[string]string{
					"app": "nginx",
				},
				"ports": []map[string]interface{}{
					{
						"port":       80,
						"targetPort": 8080,
						"protocol":   "TCP",
					},
				},
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/services", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := server.CreateService(c)
			if err != nil {
				t.Fatalf("CreateService failed: %v", err)
			}

			if rec.Code != http.StatusCreated {
				t.Errorf("expected status %d, got %d", http.StatusCreated, rec.Code)
			}

			var service types.Service
			if err := json.Unmarshal(rec.Body.Bytes(), &service); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if service.Name != "web-service" {
				t.Errorf("expected name web-service, got %s", service.Name)
			}
			if service.ClusterIP == "" {
				t.Error("expected ClusterIP to be allocated")
			}
			if service.Type != types.ServiceTypeClusterIP {
				t.Errorf("expected type ClusterIP, got %s", service.Type)
			}
		},
	)

	t.Run(
		"create service without name", func(t *testing.T) {
			payload := map[string]interface{}{
				"selector": map[string]string{"app": "nginx"},
				"ports":    []map[string]interface{}{{"port": 80}},
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/services", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			_ = server.CreateService(c)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		},
	)

	t.Run(
		"create service without selector", func(t *testing.T) {
			payload := map[string]interface{}{
				"name":  "test-service",
				"ports": []map[string]interface{}{{"port": 80}},
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/services", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			_ = server.CreateService(c)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		},
	)

	t.Run(
		"create service without ports", func(t *testing.T) {
			payload := map[string]interface{}{
				"name":     "test-service",
				"selector": map[string]string{"app": "nginx"},
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/services", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			_ = server.CreateService(c)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		},
	)

	t.Run(
		"create service with defaults", func(t *testing.T) {
			payload := map[string]interface{}{
				"name":     "default-service",
				"selector": map[string]string{"app": "test"},
				"ports": []map[string]interface{}{
					{"port": 80},
				},
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/services", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			_ = server.CreateService(c)

			var service types.Service
			_ = json.Unmarshal(rec.Body.Bytes(), &service)

			if service.Namespace != "default" {
				t.Errorf("expected namespace default, got %s", service.Namespace)
			}
			if service.Type != types.ServiceTypeClusterIP {
				t.Errorf("expected type ClusterIP, got %s", service.Type)
			}
			if service.Ports[0].Protocol != "TCP" {
				t.Errorf("expected protocol TCP, got %s", service.Ports[0].Protocol)
			}
			if service.Ports[0].TargetPort != 80 {
				t.Errorf("expected targetPort 80, got %d", service.Ports[0].TargetPort)
			}
		},
	)

	t.Run(
		"create service with invalid JSON", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/services", bytes.NewReader([]byte("invalid")))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			_ = server.CreateService(c)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		},
	)
}

func TestListServices(t *testing.T) {
	e := echo.New()
	store := state.NewInMemoryStore()
	sched := scheduler.NewRoundRobin()
	endpointController := services.NewEndpointController(store)
	server := NewServer(store, sched, endpointController)

	service1 := types.Service{
		ServiceID: "svc-1",
		Name:      "service-1",
		Namespace: "default",
		Type:      types.ServiceTypeClusterIP,
		ClusterIP: "10.96.0.1",
		Selector:  map[string]string{"app": "nginx"},
		Ports:     []types.ServicePort{{Port: 80, TargetPort: 80, Protocol: "TCP"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	service2 := types.Service{
		ServiceID: "svc-2",
		Name:      "service-2",
		Namespace: "production",
		Type:      types.ServiceTypeClusterIP,
		ClusterIP: "10.96.0.2",
		Selector:  map[string]string{"app": "redis"},
		Ports:     []types.ServicePort{{Port: 6379, TargetPort: 6379, Protocol: "TCP"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_ = store.AddService(service1)
	_ = store.AddService(service2)

	t.Run(
		"list all services", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/services", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := server.ListServices(c)
			if err != nil {
				t.Fatalf("ListServices failed: %v", err)
			}

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			var services []types.Service
			if err := json.Unmarshal(rec.Body.Bytes(), &services); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if len(services) != 2 {
				t.Errorf("expected 2 services, got %d", len(services))
			}
		},
	)

	t.Run(
		"list services by namespace", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/services?namespace=default", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			_ = server.ListServices(c)

			var services []types.Service
			_ = json.Unmarshal(rec.Body.Bytes(), &services)

			if len(services) != 1 {
				t.Errorf("expected 1 service, got %d", len(services))
			}
			if services[0].Namespace != "default" {
				t.Errorf("expected namespace default, got %s", services[0].Namespace)
			}
		},
	)
}

func TestGetService(t *testing.T) {
	e := echo.New()
	store := state.NewInMemoryStore()
	sched := scheduler.NewRoundRobin()
	endpointController := services.NewEndpointController(store)
	server := NewServer(store, sched, endpointController)

	service := types.Service{
		ServiceID: "svc-123",
		Name:      "test-service",
		Namespace: "default",
		Type:      types.ServiceTypeClusterIP,
		ClusterIP: "10.96.0.1",
		Selector:  map[string]string{"app": "nginx"},
		Ports:     []types.ServicePort{{Port: 80, TargetPort: 8080, Protocol: "TCP"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = store.AddService(service)

	t.Run(
		"get existing service", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/services/svc-123", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues("svc-123")

			err := server.GetService(c)
			if err != nil {
				t.Fatalf("GetService failed: %v", err)
			}

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			var retrieved types.Service
			_ = json.Unmarshal(rec.Body.Bytes(), &retrieved)

			if retrieved.ServiceID != "svc-123" {
				t.Errorf("expected ServiceID svc-123, got %s", retrieved.ServiceID)
			}
		},
	)

	t.Run(
		"get non-existent service", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/services/non-existent", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues("non-existent")

			_ = server.GetService(c)

			if rec.Code != http.StatusNotFound {
				t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
			}
		},
	)
}

func TestUpdateService(t *testing.T) {
	e := echo.New()
	store := state.NewInMemoryStore()
	sched := scheduler.NewRoundRobin()
	endpointController := services.NewEndpointController(store)
	server := NewServer(store, sched, endpointController)

	service := types.Service{
		ServiceID: "svc-123",
		Name:      "test-service",
		Namespace: "default",
		Type:      types.ServiceTypeClusterIP,
		ClusterIP: "10.96.0.1",
		Selector:  map[string]string{"app": "nginx"},
		Ports:     []types.ServicePort{{Port: 80, TargetPort: 8080, Protocol: "TCP"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = store.AddService(service)

	t.Run(
		"update service selector", func(t *testing.T) {
			newSelector := map[string]string{"app": "nginx", "version": "v2"}
			payload := map[string]interface{}{
				"selector": newSelector,
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/services/svc-123", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues("svc-123")

			err := server.UpdateService(c)
			if err != nil {
				t.Fatalf("UpdateService failed: %v", err)
			}

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			var updated types.Service
			_ = json.Unmarshal(rec.Body.Bytes(), &updated)

			if len(updated.Selector) != 2 {
				t.Errorf("expected 2 selector labels, got %d", len(updated.Selector))
			}
		},
	)

	t.Run(
		"update non-existent service", func(t *testing.T) {
			payload := map[string]interface{}{
				"selector": map[string]string{"app": "test"},
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/services/non-existent", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues("non-existent")

			_ = server.UpdateService(c)

			if rec.Code != http.StatusNotFound {
				t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
			}
		},
	)

	t.Run(
		"update with invalid JSON", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/api/v1/services/svc-123", bytes.NewReader([]byte("invalid")))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues("svc-123")

			_ = server.UpdateService(c)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
		},
	)
}

func TestDeleteService(t *testing.T) {
	e := echo.New()
	store := state.NewInMemoryStore()
	sched := scheduler.NewRoundRobin()
	endpointController := services.NewEndpointController(store)
	server := NewServer(store, sched, endpointController)

	service := types.Service{
		ServiceID: "svc-123",
		Name:      "test-service",
		Namespace: "default",
		Type:      types.ServiceTypeClusterIP,
		ClusterIP: "10.96.0.1",
		Selector:  map[string]string{"app": "nginx"},
		Ports:     []types.ServicePort{{Port: 80, TargetPort: 8080, Protocol: "TCP"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = store.AddService(service)

	t.Run(
		"delete existing service", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/services/svc-123", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues("svc-123")

			err := server.DeleteService(c)
			if err != nil {
				t.Fatalf("DeleteService failed: %v", err)
			}

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			_, err = store.GetService("svc-123")
			if err == nil {
				t.Error("expected service to be deleted")
			}
		},
	)

	t.Run(
		"delete non-existent service", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/services/non-existent", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues("non-existent")

			_ = server.DeleteService(c)

			if rec.Code != http.StatusNotFound {
				t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
			}
		},
	)
}

func TestGetEndpoints(t *testing.T) {
	e := echo.New()
	store := state.NewInMemoryStore()
	sched := scheduler.NewRoundRobin()
	endpointController := services.NewEndpointController(store)
	server := NewServer(store, sched, endpointController)

	service := types.Service{
		ServiceID: "svc-123",
		Name:      "test-service",
		Namespace: "default",
		Type:      types.ServiceTypeClusterIP,
		ClusterIP: "10.96.0.1",
		Selector:  map[string]string{"app": "nginx"},
		Ports:     []types.ServicePort{{Port: 80, TargetPort: 8080, Protocol: "TCP"}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_ = store.AddService(service)

	endpoints := types.Endpoints{
		ServiceID:   "svc-123",
		ServiceName: "test-service",
		Namespace:   "default",
		Subsets: []types.EndpointSubset{
			{
				Addresses: []types.EndpointAddress{
					{IP: "172.17.0.2", PodID: "pod-1", NodeID: "node-1"},
				},
				Ports: []types.EndpointPort{
					{Port: 8080, Protocol: "TCP"},
				},
			},
		},
	}
	_ = store.SetEndpoints(endpoints)

	t.Run(
		"get endpoints for service", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/services/svc-123/endpoints", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues("svc-123")

			err := server.GetEndpoints(c)
			if err != nil {
				t.Fatalf("GetEndpoints failed: %v", err)
			}

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			var retrieved types.Endpoints
			_ = json.Unmarshal(rec.Body.Bytes(), &retrieved)

			if len(retrieved.Subsets) != 1 {
				t.Errorf("expected 1 subset, got %d", len(retrieved.Subsets))
			}
		},
	)

	t.Run(
		"get endpoints for service without endpoints", func(t *testing.T) {
			service2 := types.Service{
				ServiceID: "svc-456",
				Name:      "empty-service",
				Namespace: "default",
				Type:      types.ServiceTypeClusterIP,
				ClusterIP: "10.96.0.2",
				Selector:  map[string]string{"app": "empty"},
				Ports:     []types.ServicePort{{Port: 80, TargetPort: 80, Protocol: "TCP"}},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			_ = store.AddService(service2)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/services/svc-456/endpoints", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues("svc-456")

			_ = server.GetEndpoints(c)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			var retrieved types.Endpoints
			_ = json.Unmarshal(rec.Body.Bytes(), &retrieved)

			if len(retrieved.Subsets) != 0 {
				t.Errorf("expected 0 subsets, got %d", len(retrieved.Subsets))
			}
		},
	)

	t.Run(
		"get endpoints for non-existent service", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/services/non-existent/endpoints", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues("non-existent")

			_ = server.GetEndpoints(c)

			if rec.Code != http.StatusNotFound {
				t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
			}
		},
	)
}

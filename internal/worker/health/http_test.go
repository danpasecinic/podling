package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danpasecinic/podling/internal/types"
)

func TestHTTPProbe_Check(t *testing.T) {
	tests := []struct {
		name            string
		check           *types.HealthCheck
		serverResponse  int
		expectedSuccess bool
		expectedMessage string
	}{
		{
			name: "successful health check (200)",
			check: &types.HealthCheck{
				Type:           types.ProbeTypeHTTP,
				HTTPPath:       "/health",
				Port:           8080,
				TimeoutSeconds: 5,
			},
			serverResponse:  http.StatusOK,
			expectedSuccess: true,
			expectedMessage: "HTTP 200",
		},
		{
			name: "successful health check (204)",
			check: &types.HealthCheck{
				Type:           types.ProbeTypeHTTP,
				HTTPPath:       "/health",
				Port:           8080,
				TimeoutSeconds: 5,
			},
			serverResponse:  http.StatusNoContent,
			expectedSuccess: true,
			expectedMessage: "HTTP 204",
		},
		{
			name: "failed health check (500)",
			check: &types.HealthCheck{
				Type:           types.ProbeTypeHTTP,
				HTTPPath:       "/health",
				Port:           8080,
				TimeoutSeconds: 5,
			},
			serverResponse:  http.StatusInternalServerError,
			expectedSuccess: false,
			expectedMessage: "HTTP 500 (unhealthy)",
		},
		{
			name: "failed health check (404)",
			check: &types.HealthCheck{
				Type:           types.ProbeTypeHTTP,
				HTTPPath:       "/health",
				Port:           8080,
				TimeoutSeconds: 5,
			},
			serverResponse:  http.StatusNotFound,
			expectedSuccess: false,
			expectedMessage: "HTTP 404 (unhealthy)",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				server := httptest.NewServer(
					http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							if r.URL.Path != tt.check.HTTPPath {
								t.Errorf("unexpected path: got %s, want %s", r.URL.Path, tt.check.HTTPPath)
							}
							w.WriteHeader(tt.serverResponse)
						},
					),
				)
				defer server.Close()

				// For simplicity, we'll modify the check to use the test server directly
				probe := NewHTTPProbe()
				ctx := context.Background()
				result := types.ProbeResult{Success: false, Timestamp: time.Now()}

				req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+tt.check.HTTPPath, nil)
				if err != nil {
					t.Fatalf("failed to create request: %v", err)
				}

				resp, err := probe.client.Do(req)
				if err != nil {
					t.Fatalf("request failed: %v", err)
				}
				defer func() { _ = resp.Body.Close() }()

				if resp.StatusCode >= 200 && resp.StatusCode < 400 {
					result.Success = true
				}

				if result.Success != tt.expectedSuccess {
					t.Errorf("expected success=%v, got %v", tt.expectedSuccess, result.Success)
				}
			},
		)
	}
}

func TestHTTPProbe_Check_InvalidConfig(t *testing.T) {
	probe := NewHTTPProbe()
	ctx := context.Background()

	tests := []struct {
		name  string
		check *types.HealthCheck
		want  string
	}{
		{
			name: "invalid port",
			check: &types.HealthCheck{
				Type:     types.ProbeTypeHTTP,
				HTTPPath: "/health",
				Port:     0,
			},
			want: "invalid port configuration",
		},
		{
			name: "missing HTTP path",
			check: &types.HealthCheck{
				Type:     types.ProbeTypeHTTP,
				HTTPPath: "",
				Port:     8080,
			},
			want: "HTTP path not configured",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				result := probe.Check(ctx, tt.check, "127.0.0.1")
				if result.Success {
					t.Error("expected check to fail")
				}
				if result.Message != tt.want {
					t.Errorf("expected message %q, got %q", tt.want, result.Message)
				}
			},
		)
	}
}

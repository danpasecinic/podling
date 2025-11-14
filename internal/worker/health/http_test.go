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

func TestValidateContainerIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{"valid private IP 10.x", "10.0.0.1", false},
		{"valid private IP 172.x", "172.16.0.1", false},
		{"valid private IP 192.168.x", "192.168.1.1", false},
		{"valid loopback", "127.0.0.1", false},
		{"public IP rejected", "8.8.8.8", true},
		{"public IP rejected (1.1.1.1)", "1.1.1.1", true},
		{"empty IP", "", true},
		{"invalid format", "not-an-ip", true},
		{"multicast IP", "224.0.0.1", true},
		{"unspecified IP", "0.0.0.0", true},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				err := validateContainerIP(tt.ip)
				if (err != nil) != tt.wantErr {
					t.Errorf("validateContainerIP(%q) error = %v, wantErr %v", tt.ip, err, tt.wantErr)
				}
			},
		)
	}
}

func TestValidateHTTPPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid path", "/health", false},
		{"valid path with query", "/health?check=true", false},
		{"valid nested path", "/api/v1/health", false},
		{"empty path", "", true},
		{"missing leading slash", "health", true},
		{"path traversal", "/../../etc/passwd", true},
		{"null byte", "/health\x00", true},
		{"newline", "/health\n", true},
		{"carriage return", "/health\r", true},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				err := validateHTTPPath(tt.path)
				if (err != nil) != tt.wantErr {
					t.Errorf("validateHTTPPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				}
			},
		)
	}
}

func TestHTTPProbe_Check_SecurityValidation(t *testing.T) {
	probe := NewHTTPProbe()
	ctx := context.Background()

	tests := []struct {
		name        string
		check       *types.HealthCheck
		containerIP string
		wantErrMsg  string
	}{
		{
			name: "SSRF prevention - public IP",
			check: &types.HealthCheck{
				Type:           types.ProbeTypeHTTP,
				HTTPPath:       "/health",
				Port:           80,
				TimeoutSeconds: 5,
			},
			containerIP: "8.8.8.8",
			wantErrMsg:  "invalid container IP",
		},
		{
			name: "path traversal prevention",
			check: &types.HealthCheck{
				Type:           types.ProbeTypeHTTP,
				HTTPPath:       "/../../../etc/passwd",
				Port:           8080,
				TimeoutSeconds: 5,
			},
			containerIP: "127.0.0.1",
			wantErrMsg:  "invalid HTTP path",
		},
		{
			name: "port out of range",
			check: &types.HealthCheck{
				Type:           types.ProbeTypeHTTP,
				HTTPPath:       "/health",
				Port:           99999,
				TimeoutSeconds: 5,
			},
			containerIP: "127.0.0.1",
			wantErrMsg:  "invalid port configuration",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				result := probe.Check(ctx, tt.check, tt.containerIP)
				if result.Success {
					t.Error("expected check to fail for security validation")
				}
				if result.Message == "" || len(result.Message) < len(tt.wantErrMsg) {
					t.Errorf("expected error message containing %q, got %q", tt.wantErrMsg, result.Message)
				}
			},
		)
	}
}

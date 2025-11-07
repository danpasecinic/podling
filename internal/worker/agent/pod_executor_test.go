package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danpasecinic/podling/internal/types"
)

func TestAgent_UpdatePodStatus(t *testing.T) {
	tests := []struct {
		name       string
		podID      string
		status     types.PodStatus
		containers []types.Container
		message    string
		reason     string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful status update",
			podID:      "pod-123",
			status:     types.PodRunning,
			containers: []types.Container{{Name: "nginx", Status: types.ContainerRunning}},
			message:    "Pod is running",
			reason:     "Started",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "status update without containers",
			podID:      "pod-123",
			status:     types.PodPending,
			containers: nil,
			message:    "",
			reason:     "",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "status update with error response",
			podID:      "pod-456",
			status:     types.PodFailed,
			containers: nil,
			message:    "Failed",
			reason:     "Error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				server := httptest.NewServer(
					http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							if r.Method != http.MethodPut {
								t.Errorf("Expected PUT method, got %s", r.Method)
							}

							expectedPath := "/api/v1/pods/" + tt.podID + "/status"
							if r.URL.Path != expectedPath {
								t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
							}

							var payload map[string]interface{}
							if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
								t.Errorf("Failed to decode request body: %v", err)
							}

							if payload["status"] != string(tt.status) {
								t.Errorf("Expected status %s, got %v", tt.status, payload["status"])
							}

							if tt.message != "" {
								if payload["message"] != tt.message {
									t.Errorf("Expected message %s, got %v", tt.message, payload["message"])
								}
							}

							if tt.reason != "" {
								if payload["reason"] != tt.reason {
									t.Errorf("Expected reason %s, got %v", tt.reason, payload["reason"])
								}
							}

							w.WriteHeader(tt.statusCode)
						},
					),
				)
				defer server.Close()

				agent := &Agent{
					masterURL: server.URL,
				}

				err := agent.updatePodStatus(tt.podID, tt.status, tt.containers, tt.message, tt.reason)

				if (err != nil) != tt.wantErr {
					t.Errorf("updatePodStatus() error = %v, wantErr %v", err, tt.wantErr)
				}
			},
		)
	}
}

func TestAgent_UpdatePodStatusWithIP(t *testing.T) {
	tests := []struct {
		name       string
		podID      string
		status     types.PodStatus
		containers []types.Container
		podIP      string
		message    string
		reason     string
		statusCode int
		wantErr    bool
	}{
		{
			name:   "successful status update with IP",
			podID:  "pod-123",
			status: types.PodRunning,
			containers: []types.Container{
				{Name: "nginx", Status: types.ContainerRunning},
			},
			podIP:      "172.17.0.2",
			message:    "Pod is running",
			reason:     "Started",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "status update without IP",
			podID:      "pod-123",
			status:     types.PodRunning,
			containers: nil,
			podIP:      "",
			message:    "",
			reason:     "",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "status update with error response",
			podID:      "pod-456",
			status:     types.PodFailed,
			containers: nil,
			podIP:      "172.17.0.3",
			message:    "Failed",
			reason:     "Error",
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				server := httptest.NewServer(
					http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							if r.Method != http.MethodPut {
								t.Errorf("Expected PUT method, got %s", r.Method)
							}

							expectedPath := "/api/v1/pods/" + tt.podID + "/status"
							if r.URL.Path != expectedPath {
								t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
							}

							var payload map[string]interface{}
							if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
								t.Errorf("Failed to decode request body: %v", err)
							}

							if payload["status"] != string(tt.status) {
								t.Errorf("Expected status %s, got %v", tt.status, payload["status"])
							}

							if tt.podIP != "" {
								annotations, ok := payload["annotations"].(map[string]interface{})
								if !ok {
									t.Error("Expected annotations in payload")
								} else {
									if annotations["podling.io/pod-ip"] != tt.podIP {
										t.Errorf(
											"Expected pod IP %s, got %v", tt.podIP, annotations["podling.io/pod-ip"],
										)
									}
								}
							}

							w.WriteHeader(tt.statusCode)
						},
					),
				)
				defer server.Close()

				agent := &Agent{
					masterURL: server.URL,
				}

				err := agent.updatePodStatusWithIP(tt.podID, tt.status, tt.containers, tt.podIP, tt.message, tt.reason)

				if (err != nil) != tt.wantErr {
					t.Errorf("updatePodStatusWithIP() error = %v, wantErr %v", err, tt.wantErr)
				}
			},
		)
	}
}

func TestUpdatePodStatus_ConnectionError(t *testing.T) {
	agent := &Agent{
		masterURL: "http://invalid-host-that-does-not-exist:99999",
	}

	err := agent.updatePodStatus("pod-123", types.PodRunning, nil, "", "")

	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
}

func TestUpdatePodStatusWithIP_ConnectionError(t *testing.T) {
	agent := &Agent{
		masterURL: "http://invalid-host-that-does-not-exist:99999",
	}

	err := agent.updatePodStatusWithIP("pod-123", types.PodRunning, nil, "172.17.0.2", "", "")

	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
}

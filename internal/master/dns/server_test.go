package dns

import (
	"testing"

	"github.com/danpasecinic/podling/internal/master/state"
	"github.com/danpasecinic/podling/internal/types"
	"github.com/miekg/dns"
)

func TestHandler_ParseServiceName(t *testing.T) {
	store := state.NewMockStateStore()
	handler := NewHandler(store, "cluster.local")

	tests := []struct {
		name          string
		input         string
		wantService   string
		wantNamespace string
		wantOk        bool
	}{
		{
			name:          "full service name with namespace",
			input:         "web.production.svc.cluster.local",
			wantService:   "web",
			wantNamespace: "production",
			wantOk:        true,
		},
		{
			name:          "service name with default namespace",
			input:         "nginx.default.svc.cluster.local",
			wantService:   "nginx",
			wantNamespace: "default",
			wantOk:        true,
		},
		{
			name:          "invalid - missing svc suffix",
			input:         "web.production.cluster.local",
			wantService:   "",
			wantNamespace: "",
			wantOk:        false,
		},
		{
			name:          "invalid - wrong domain",
			input:         "web.production.svc.example.com",
			wantService:   "",
			wantNamespace: "",
			wantOk:        false,
		},
		{
			name:          "invalid - just hostname",
			input:         "web",
			wantService:   "",
			wantNamespace: "",
			wantOk:        false,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				service, namespace, ok := handler.parseServiceName(tt.input)
				if ok != tt.wantOk {
					t.Errorf("parseServiceName() ok = %v, want %v", ok, tt.wantOk)
				}
				if service != tt.wantService {
					t.Errorf("parseServiceName() service = %v, want %v", service, tt.wantService)
				}
				if namespace != tt.wantNamespace {
					t.Errorf("parseServiceName() namespace = %v, want %v", namespace, tt.wantNamespace)
				}
			},
		)
	}
}

func TestHandler_HandleARecord(t *testing.T) {
	store := state.NewMockStateStore()

	err := store.AddService(
		types.Service{
			ServiceID: "svc-1",
			Name:      "web",
			Namespace: "default",
			ClusterIP: "10.96.0.1",
			Type:      types.ServiceTypeClusterIP,
		},
	)
	if err != nil {
		t.Fatalf("failed to add service: %v", err)
	}

	err = store.AddService(
		types.Service{
			ServiceID: "svc-2",
			Name:      "api",
			Namespace: "production",
			ClusterIP: "10.96.0.2",
			Type:      types.ServiceTypeClusterIP,
		},
	)
	if err != nil {
		t.Fatalf("failed to add service: %v", err)
	}

	handler := NewHandler(store, "cluster.local")

	tests := []struct {
		name       string
		query      string
		wantIP     string
		wantAnswer bool
	}{
		{
			name:       "existing service in default namespace",
			query:      "web.default.svc.cluster.local.",
			wantIP:     "10.96.0.1",
			wantAnswer: true,
		},
		{
			name:       "existing service in production namespace",
			query:      "api.production.svc.cluster.local.",
			wantIP:     "10.96.0.2",
			wantAnswer: true,
		},
		{
			name:       "non-existent service",
			query:      "unknown.default.svc.cluster.local.",
			wantIP:     "",
			wantAnswer: false,
		},
		{
			name:       "service in wrong namespace",
			query:      "web.production.svc.cluster.local.",
			wantIP:     "",
			wantAnswer: false,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				m := new(dns.Msg)
				q := dns.Question{
					Name:   tt.query,
					Qtype:  dns.TypeA,
					Qclass: dns.ClassINET,
				}

				handler.handleARecord(m, q)

				if tt.wantAnswer {
					if len(m.Answer) == 0 {
						t.Error("expected answer but got none")
						return
					}
					a, ok := m.Answer[0].(*dns.A)
					if !ok {
						t.Error("expected A record answer")
						return
					}
					if a.A.String() != tt.wantIP {
						t.Errorf("got IP %s, want %s", a.A.String(), tt.wantIP)
					}
				} else {
					if len(m.Answer) > 0 {
						t.Errorf("expected no answer but got %d", len(m.Answer))
					}
				}
			},
		)
	}
}

func TestConfig_Default(t *testing.T) {
	config := DefaultConfig()

	if config.Port != DefaultPort {
		t.Errorf("default port = %d, want %d", config.Port, DefaultPort)
	}
	if config.ClusterDomain != DefaultClusterDomain {
		t.Errorf("default cluster domain = %s, want %s", config.ClusterDomain, DefaultClusterDomain)
	}
	if !config.Enabled {
		t.Error("expected DNS to be enabled by default")
	}
}

func TestNewServer(t *testing.T) {
	store := state.NewMockStateStore()
	config := DefaultConfig()

	server := NewServer(store, config)

	if server == nil {
		t.Fatal("expected server to be created")
	}
	if server.Port() != DefaultPort {
		t.Errorf("port = %d, want %d", server.Port(), DefaultPort)
	}
	if server.Address() != ":5353" {
		t.Errorf("address = %s, want :5353", server.Address())
	}
}

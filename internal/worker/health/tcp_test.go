package health

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/danpasecinic/podling/internal/types"
)

func TestTCPProbe_Check(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}
	defer func(listener net.Listener) {
		_ = listener.Close()
	}(listener)

	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port

	// Accept connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	probe := NewTCPProbe()
	ctx := context.Background()

	t.Run(
		"successful connection", func(t *testing.T) {
			check := &types.HealthCheck{
				Type:           types.ProbeTypeTCP,
				Port:           port,
				TimeoutSeconds: 5,
			}

			result := probe.Check(ctx, check, "127.0.0.1")
			if !result.Success {
				t.Errorf("expected success, got failure: %s", result.Message)
			}
			if result.Message != "TCP connection successful" {
				t.Errorf("unexpected message: %s", result.Message)
			}
		},
	)

	t.Run(
		"failed connection - wrong port", func(t *testing.T) {
			check := &types.HealthCheck{
				Type:           types.ProbeTypeTCP,
				Port:           port + 1000,
				TimeoutSeconds: 1,
			}

			result := probe.Check(ctx, check, "127.0.0.1")
			if result.Success {
				t.Error("expected failure, got success")
			}
		},
	)

	t.Run(
		"invalid port", func(t *testing.T) {
			check := &types.HealthCheck{
				Type: types.ProbeTypeTCP,
				Port: 0,
			}

			result := probe.Check(ctx, check, "127.0.0.1")
			if result.Success {
				t.Error("expected failure, got success")
			}
			if result.Message != "invalid port configuration" {
				t.Errorf("unexpected message: %s", result.Message)
			}
		},
	)

	t.Run(
		"invalid container IP", func(t *testing.T) {
			check := &types.HealthCheck{
				Type:           types.ProbeTypeTCP,
				Port:           port,
				TimeoutSeconds: 1,
			}

			result := probe.Check(ctx, check, "invalid-ip")
			if result.Success {
				t.Error("expected failure, got success")
			}
		},
	)
}

func TestTCPProbe_CheckTimeout(t *testing.T) {
	probe := NewTCPProbe()
	ctx := context.Background()

	check := &types.HealthCheck{
		Type:           types.ProbeTypeTCP,
		Port:           12345,
		TimeoutSeconds: 1,
	}

	start := time.Now()
	result := probe.Check(ctx, check, "192.0.2.1")
	elapsed := time.Since(start)

	if result.Success {
		t.Error("expected failure for non-routable address")
	}

	// Should timeout around 1 second
	if elapsed > 3*time.Second {
		t.Errorf("timeout took too long: %v", elapsed)
	}
}

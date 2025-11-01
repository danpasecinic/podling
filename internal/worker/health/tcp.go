package health

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/danpasecinic/podling/internal/types"
)

// TCPProbe performs TCP health checks
type TCPProbe struct{}

// NewTCPProbe creates a new TCP probe
func NewTCPProbe() *TCPProbe {
	return &TCPProbe{}
}

// Check performs a TCP connection check
func (p *TCPProbe) Check(ctx context.Context, check *types.HealthCheck, containerIP string) types.ProbeResult {
	result := types.ProbeResult{
		Success:   false,
		Timestamp: time.Now(),
	}

	if check.Port <= 0 {
		result.Message = "invalid port configuration"
		return result
	}

	addr := fmt.Sprintf("%s:%d", containerIP, check.Port)
	dialer := net.Dialer{
		Timeout: check.GetTimeout(),
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		result.Message = fmt.Sprintf("connection failed: %v", err)
		return result
	}

	_ = conn.Close()

	result.Success = true
	result.Message = "TCP connection successful"
	return result
}

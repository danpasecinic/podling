package health

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/danpasecinic/podling/internal/types"
)

// HTTPProbe performs HTTP health checks
type HTTPProbe struct {
	client *http.Client
}

// NewHTTPProbe creates a new HTTP probe
func NewHTTPProbe() *HTTPProbe {
	return &HTTPProbe{
		client: &http.Client{
			// Default timeout will be overridden per-check
			Timeout: 30 * time.Second,
		},
	}
}

// Check performs an HTTP GET request to check container health
// containerIP is the IP address of the container
func (p *HTTPProbe) Check(ctx context.Context, check *types.HealthCheck, containerIP string) types.ProbeResult {
	result := types.ProbeResult{
		Success:   false,
		Timestamp: time.Now(),
	}

	if check.Port <= 0 {
		result.Message = "invalid port configuration"
		return result
	}

	if check.HTTPPath == "" {
		result.Message = "HTTP path not configured"
		return result
	}

	url := fmt.Sprintf("http://%s:%d%s", containerIP, check.Port, check.HTTPPath)
	reqCtx, cancel := context.WithTimeout(ctx, check.GetTimeout())
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		result.Message = fmt.Sprintf("failed to create request: %v", err)
		return result
	}

	resp, err := p.client.Do(req)
	if err != nil {
		result.Message = fmt.Sprintf("request failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.Success = true
		result.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
	} else {
		result.Message = fmt.Sprintf("HTTP %d (unhealthy)", resp.StatusCode)
	}

	return result
}

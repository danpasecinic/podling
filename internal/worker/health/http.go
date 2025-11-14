package health

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
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

	if check.Port <= 0 || check.Port > 65535 {
		result.Message = "invalid port configuration"
		return result
	}

	if check.HTTPPath == "" {
		result.Message = "HTTP path not configured"
		return result
	}

	// Validate containerIP to prevent SSRF attacks
	if err := validateContainerIP(containerIP); err != nil {
		result.Message = fmt.Sprintf("invalid container IP: %v", err)
		return result
	}

	// Validate HTTP path to prevent path traversal
	if err := validateHTTPPath(check.HTTPPath); err != nil {
		result.Message = fmt.Sprintf("invalid HTTP path: %v", err)
		return result
	}

	probeURL := fmt.Sprintf("http://%s:%d%s", containerIP, check.Port, check.HTTPPath)

	parsedURL, err := url.Parse(probeURL)
	if err != nil {
		result.Message = fmt.Sprintf("failed to parse URL: %v", err)
		return result
	}

	if parsedURL.Scheme != "http" {
		result.Message = "only HTTP scheme is allowed for health checks"
		return result
	}

	reqCtx, cancel := context.WithTimeout(ctx, check.GetTimeout())
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		result.Message = fmt.Sprintf("failed to create request: %v", err)
		return result
	}

	resp, err := p.client.Do(req)
	if err != nil {
		result.Message = fmt.Sprintf("request failed: %v", err)
		return result
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.Success = true
		result.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
	} else {
		result.Message = fmt.Sprintf("HTTP %d (unhealthy)", resp.StatusCode)
	}

	return result
}

// validateContainerIP validates that the IP address is a valid private IP
// This prevents SSRF attacks by ensuring we only probe container IPs
func validateContainerIP(ipStr string) error {
	if ipStr == "" {
		return fmt.Errorf("IP address is empty")
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Errorf("invalid IP address format")
	}

	// Only allow private IP ranges (RFC 1918) and Docker's default ranges
	// 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 127.0.0.0/8 (localhost)
	if !isPrivateIP(ip) && !ip.IsLoopback() {
		return fmt.Errorf("IP address must be private or loopback")
	}

	// Reject special addresses
	if ip.IsUnspecified() || ip.IsMulticast() {
		return fmt.Errorf("IP address is not a valid unicast address")
	}

	return nil
}

// isPrivateIP checks if an IP is in private ranges (RFC 1918)
func isPrivateIP(ip net.IP) bool {
	privateRanges := []string{
		"10.0.0.0/8",     // Class A private
		"172.16.0.0/12",  // Class B private
		"192.168.0.0/16", // Class C private
	}

	for _, cidr := range privateRanges {
		_, subnet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if subnet.Contains(ip) {
			return true
		}
	}

	return false
}

// validateHTTPPath validates the HTTP path to prevent path traversal
func validateHTTPPath(path string) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}

	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("path must start with /")
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal detected")
	}
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("null byte in path")
	}
	if strings.ContainsAny(path, "\r\n") {
		return fmt.Errorf("invalid characters in path")
	}

	_, err := url.ParseRequestURI(path)
	if err != nil {
		return fmt.Errorf("invalid URL path: %w", err)
	}

	return nil
}

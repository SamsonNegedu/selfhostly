package cloudflare

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/selfhostly/internal/constants"
)

// ExtractQuickTunnelURL fetches the cloudflared metrics endpoint and parses it
// to find the generated trycloudflare.com URL. Retries until the URL appears or maxRetries is reached.
// metricsEndpoint should be the full URL, e.g. "http://localhost:2000/metrics".
// If the endpoint uses localhost and fails, it will try alternative hosts (host.docker.internal, 172.17.0.1).
// The context is used to cancel the operation and respect deadlines.
func ExtractQuickTunnelURL(ctx context.Context, metricsEndpoint string, maxRetries int, interval time.Duration) (string, error) {
	if maxRetries <= 0 {
		maxRetries = constants.QuickTunnelURLMaxRetries
	}
	if interval <= 0 {
		interval = constants.QuickTunnelURLRetryInterval
	}

	// If endpoint uses localhost, prepare alternative hosts to try (for Docker environments)
	var alternativeEndpoints []string
	if strings.Contains(metricsEndpoint, "localhost") || strings.Contains(metricsEndpoint, "127.0.0.1") {
		// Extract port from endpoint (format: http://host:port/metrics)
		port := strconv.Itoa(constants.QuickTunnelMetricsPort)
		if strings.Contains(metricsEndpoint, ":") {
			parts := strings.Split(metricsEndpoint, ":")
			if len(parts) >= 3 {
				port = strings.Split(parts[2], "/")[0]
			}
		}
		// Try host.docker.internal (Mac/Windows Docker) and Docker bridge gateway (Linux)
		alternativeEndpoints = []string{
			fmt.Sprintf("http://%s:%s/metrics", constants.DockerHostInternal, port),
			fmt.Sprintf("http://%s:%s/metrics", constants.DockerBridgeGateway, port),
		}
	}

	var lastErr error
	endpointsToTry := []string{metricsEndpoint}
	endpointsToTry = append(endpointsToTry, alternativeEndpoints...)

	for i := 0; i < maxRetries; i++ {
		// Check if context is cancelled or deadline exceeded
		if ctx.Err() != nil {
			return "", fmt.Errorf("context cancelled or deadline exceeded: %w", ctx.Err())
		}

		// Try each endpoint in order
		for _, endpoint := range endpointsToTry {
			url, err := fetchMetricsAndParse(ctx, endpoint)
			if err == nil && url != "" {
				return url, nil
			}
			// Track error for the primary endpoint
			if endpoint == metricsEndpoint {
				lastErr = err
			}
			// Log errors at debug level to avoid spam, but log first attempt at info level
			if i == 0 && endpoint == metricsEndpoint {
				slog.Debug("failed to fetch metrics endpoint", "endpoint", endpoint, "error", err, "retry", i+1, "max_retries", maxRetries)
			}
		}

		// Check context again before sleeping
		if ctx.Err() != nil {
			return "", fmt.Errorf("context cancelled or deadline exceeded: %w", ctx.Err())
		}

		if i < maxRetries-1 {
			// Use context-aware sleep
			select {
			case <-ctx.Done():
				return "", fmt.Errorf("context cancelled or deadline exceeded: %w", ctx.Err())
			case <-time.After(interval):
				// Continue to next iteration
			}
		}
	}
	return "", fmt.Errorf("failed to extract Quick Tunnel URL after %d retries: %w", maxRetries, lastErr)
}

// fetchMetricsAndParse GETs the metrics endpoint and extracts the trycloudflare.com hostname.
// The context is used to cancel the HTTP request.
func fetchMetricsAndParse(ctx context.Context, endpoint string) (string, error) {
	// Create HTTP client with timeout that respects context
	// Use a shorter timeout to fail fast if endpoint is unreachable
	client := &http.Client{
		Timeout: constants.HTTPClientTimeout,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		// Check if error is due to context cancellation
		if ctx.Err() != nil {
			return "", fmt.Errorf("request cancelled: %w", ctx.Err())
		}
		return "", fmt.Errorf("failed to fetch metrics endpoint %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("metrics endpoint returned status %d for %s", resp.StatusCode, endpoint)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read metrics response: %w", err)
	}

	if len(body) == 0 {
		return "", fmt.Errorf("metrics endpoint returned empty response from %s", endpoint)
	}

	return ParseQuickTunnelURLFromMetrics(string(body))
}

// ParseQuickTunnelURLFromMetrics looks for userHostname="...trycloudflare.com" in Prometheus-style metrics.
// Cloudflared may expose either hostname only (e.g. "foo.trycloudflare.com") or full URL (e.g. "https://foo.trycloudflare.com").
// Tries multiple regex patterns to handle different metric formats.
func ParseQuickTunnelURLFromMetrics(body string) (string, error) {
	if body == "" {
		return "", fmt.Errorf("metrics response body is empty")
	}

	// Try multiple regex patterns to match different possible formats
	patterns := []*regexp.Regexp{
		// Standard format: userHostname="hostname.trycloudflare.com"
		regexp.MustCompile(`userHostname="([^"]+\.trycloudflare\.com)"`),
		// Alternative format: userHostname=hostname.trycloudflare.com (without quotes)
		regexp.MustCompile(`userHostname=([^\s,}]+\.trycloudflare\.com)`),
		// Look for any trycloudflare.com hostname in the body
		regexp.MustCompile(`([a-zA-Z0-9-]+\.trycloudflare\.com)`),
	}

	for _, re := range patterns {
		matches := re.FindStringSubmatch(body)
		if len(matches) >= 2 {
			val := strings.TrimSpace(matches[1])
			if val != "" && strings.Contains(val, "trycloudflare.com") {
				// Avoid double "https://" if cloudflared already returns full URL
				if strings.HasPrefix(val, "https://") || strings.HasPrefix(val, "http://") {
					return val, nil
				}
				return "https://" + val, nil
			}
		}
	}

	// Log a sample of the response for debugging (first 500 chars)
	bodySample := body
	if len(bodySample) > 500 {
		bodySample = bodySample[:500] + "..."
	}
	slog.Debug("Quick Tunnel URL not found in metrics", "body_sample", bodySample)
	return "", fmt.Errorf("Quick Tunnel URL not found in metrics (checked %d patterns). Response sample: %s", len(patterns), bodySample)
}

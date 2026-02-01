package cloudflare

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ExtractQuickTunnelURL fetches the cloudflared metrics endpoint and parses it
// to find the generated trycloudflare.com URL. Retries until the URL appears or maxRetries is reached.
// metricsEndpoint should be the full URL, e.g. "http://localhost:2000/metrics".
// If the endpoint uses localhost and fails, it will try alternative hosts (host.docker.internal, 172.17.0.1).
func ExtractQuickTunnelURL(metricsEndpoint string, maxRetries int, interval time.Duration) (string, error) {
	if maxRetries <= 0 {
		maxRetries = 30
	}
	if interval <= 0 {
		interval = 2 * time.Second
	}
	
	// If endpoint uses localhost, prepare alternative hosts to try (for Docker environments)
	var alternativeEndpoints []string
	if strings.Contains(metricsEndpoint, "localhost") || strings.Contains(metricsEndpoint, "127.0.0.1") {
		// Extract port from endpoint (format: http://host:port/metrics)
		port := "2000"
		if strings.Contains(metricsEndpoint, ":") {
			parts := strings.Split(metricsEndpoint, ":")
			if len(parts) >= 3 {
				port = strings.Split(parts[2], "/")[0]
			}
		}
		// Try host.docker.internal (Mac/Windows Docker) and Docker bridge gateway (Linux)
		alternativeEndpoints = []string{
			fmt.Sprintf("http://host.docker.internal:%s/metrics", port),
			fmt.Sprintf("http://172.17.0.1:%s/metrics", port),
		}
	}
	
	var lastErr error
	endpointsToTry := []string{metricsEndpoint}
	endpointsToTry = append(endpointsToTry, alternativeEndpoints...)
	
	for i := 0; i < maxRetries; i++ {
		// Try each endpoint in order
		for _, endpoint := range endpointsToTry {
			url, err := fetchMetricsAndParse(endpoint)
			if err == nil && url != "" {
				return url, nil
			}
			// Only log error for the primary endpoint to avoid spam
			if endpoint == metricsEndpoint {
				lastErr = err
			}
		}
		if i < maxRetries-1 {
			time.Sleep(interval)
		}
	}
	return "", fmt.Errorf("failed to extract Quick Tunnel URL after %d retries: %w", maxRetries, lastErr)
}

// fetchMetricsAndParse GETs the metrics endpoint and extracts the trycloudflare.com hostname.
func fetchMetricsAndParse(endpoint string) (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to fetch metrics endpoint: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("metrics endpoint returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read metrics response: %w", err)
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

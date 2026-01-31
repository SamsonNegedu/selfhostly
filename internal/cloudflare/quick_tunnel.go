package cloudflare

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ExtractQuickTunnelURL fetches the cloudflared metrics endpoint and parses it
// to find the generated trycloudflare.com URL. Retries until the URL appears or maxRetries is reached.
// metricsEndpoint should be the full URL, e.g. "http://localhost:2000/metrics".
func ExtractQuickTunnelURL(metricsEndpoint string, maxRetries int, interval time.Duration) (string, error) {
	if maxRetries <= 0 {
		maxRetries = 30
	}
	if interval <= 0 {
		interval = 2 * time.Second
	}
	for i := 0; i < maxRetries; i++ {
		url, err := fetchMetricsAndParse(metricsEndpoint)
		if err == nil && url != "" {
			return url, nil
		}
		if i < maxRetries-1 {
			time.Sleep(interval)
		}
	}
	return "", fmt.Errorf("failed to extract Quick Tunnel URL after %d retries", maxRetries)
}

// fetchMetricsAndParse GETs the metrics endpoint and extracts the trycloudflare.com hostname.
func fetchMetricsAndParse(endpoint string) (string, error) {
	resp, err := http.Get(endpoint)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("metrics endpoint returned %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return parseQuickTunnelURLFromMetrics(string(body))
}

// parseQuickTunnelURLFromMetrics looks for userHostname="...trycloudflare.com" in Prometheus-style metrics.
// Cloudflared may expose either hostname only (e.g. "foo.trycloudflare.com") or full URL (e.g. "https://foo.trycloudflare.com").
func parseQuickTunnelURLFromMetrics(body string) (string, error) {
	// Match userHostname="...trycloudflare.com" or full URL in metrics output
	re := regexp.MustCompile(`userHostname="([^"]+\.trycloudflare\.com)"`)
	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		return "", fmt.Errorf("Quick Tunnel URL not found in metrics")
	}
	val := strings.TrimSpace(matches[1])
	if val == "" {
		return "", fmt.Errorf("Quick Tunnel URL not found in metrics")
	}
	// Avoid double "https://" if cloudflared already returns full URL
	if strings.HasPrefix(val, "https://") || strings.HasPrefix(val, "http://") {
		return val, nil
	}
	return "https://" + val, nil
}

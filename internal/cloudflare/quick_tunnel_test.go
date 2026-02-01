package cloudflare

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseQuickTunnelURLFromMetrics(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantURL string
		wantErr bool
	}{
		{
			name:    "finds userHostname",
			body:    `cloudflared_tunnel_user_hostnames_counts{userHostname="foo-bar-1234.trycloudflare.com"} 1`,
			wantURL: "https://foo-bar-1234.trycloudflare.com",
			wantErr: false,
		},
		{
			name:    "no trycloudflare in body",
			body:    `some_other_metric 1`,
			wantURL: "",
			wantErr: true,
		},
		{
			name:    "empty body",
			body:    "",
			wantURL: "",
			wantErr: true,
		},
		{
			name:    "full URL in userHostname (no double https)",
			body:    `cloudflared_tunnel_user_hostnames_counts{userHostname="https://messaging-nor-dts-maps.trycloudflare.com"} 1`,
			wantURL: "https://messaging-nor-dts-maps.trycloudflare.com",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseQuickTunnelURLFromMetrics(tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseQuickTunnelURLFromMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantURL {
				t.Errorf("ParseQuickTunnelURLFromMetrics() = %v, want %v", got, tt.wantURL)
			}
		})
	}
}

func TestFetchMetricsAndParse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`cloudflared_tunnel_user_hostnames_counts{userHostname="test-abc.trycloudflare.com"} 1`))
	}))
	defer server.Close()

	ctx := context.Background()
	url, err := fetchMetricsAndParse(ctx, server.URL+"/metrics")
	if err != nil {
		t.Fatalf("fetchMetricsAndParse() error = %v", err)
	}
	if url != "https://test-abc.trycloudflare.com" {
		t.Errorf("fetchMetricsAndParse() = %v, want https://test-abc.trycloudflare.com", url)
	}
}

func TestExtractQuickTunnelURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`cloudflared_tunnel_user_hostnames_counts{userHostname="quick-xyz.trycloudflare.com"} 1`))
	}))
	defer server.Close()

	ctx := context.Background()
	url, err := ExtractQuickTunnelURL(ctx, server.URL+"/metrics", 3, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("ExtractQuickTunnelURL() error = %v", err)
	}
	if url != "https://quick-xyz.trycloudflare.com" {
		t.Errorf("ExtractQuickTunnelURL() = %v, want https://quick-xyz.trycloudflare.com", url)
	}
}

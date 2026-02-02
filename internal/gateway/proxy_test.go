package gateway

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func setupTestProxy(t *testing.T) (*Proxy, *NodeRegistry, *Config) {
	logger := slog.Default()
	cfg := &Config{
		PrimaryBackendURL: "http://primary:8082",
		GatewayAPIKey:     "test-api-key",
		ListenAddress:     ":8080",
		AuthEnabled:       false,
		RegistryTTL:       60 * time.Second,
	}

	registry := NewNodeRegistry(cfg.PrimaryBackendURL, cfg.GatewayAPIKey, cfg.RegistryTTL, logger)
	// Mark registry as initialized for tests (simulate successful refresh)
	registry.mu.Lock()
	registry.initialized = true
	registry.mu.Unlock()
	
	router := NewRouter(registry, logger)
	proxy := NewProxy(router, registry, cfg, logger)

	return proxy, registry, cfg
}

func TestProxy_HealthCheck(t *testing.T) {
	proxy, _, _ := setupTestProxy(t)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	expectedBody := `{"status":"healthy","service":"gateway"}`
	if w.Body.String() != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}
}

func TestProxy_HealthCheck_NoAuthRequired(t *testing.T) {
	// Test that health check works even when auth is enabled
	proxy, _, cfg := setupTestProxy(t)
	cfg.AuthEnabled = true

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestProxy_HealthCheck_OnlyGET(t *testing.T) {
	proxy, _, _ := setupTestProxy(t)

	// GET and HEAD requests are handled directly by the gateway
	// Other methods will be routed to primary (which may fail, but that's expected)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	expectedBody := `{"status":"healthy","service":"gateway"}`
	if w.Body.String() != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, w.Body.String())
	}
}

func TestProxy_HealthCheck_HEAD(t *testing.T) {
	proxy, _, _ := setupTestProxy(t)

	// Docker healthcheck uses HEAD requests
	req := httptest.NewRequest(http.MethodHead, "/api/health", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// HEAD requests should still set proper headers
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}
}

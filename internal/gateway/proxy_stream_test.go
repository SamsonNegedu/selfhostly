package gateway

import (
	"bufio"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// gatewayFor starts a real gateway in front of upstream, with login disabled
func gatewayFor(t *testing.T, upstream http.Handler) (gatewayURL string, upstreamAddr string) {
	t.Helper()
	up := httptest.NewServer(upstream)
	t.Cleanup(up.Close)

	cfg := &Config{PrimaryBackendURL: up.URL, GatewayAPIKey: "test-api-key", RegistryTTL: 60 * time.Second}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	registry := NewNodeRegistry(cfg.PrimaryBackendURL, cfg.GatewayAPIKey, cfg.RegistryTTL, logger)
	registry.mu.Lock()
	registry.initialized = true
	registry.mu.Unlock()
	proxy := NewProxy(NewRouter(registry, logger), registry, cfg, logger)

	gw := httptest.NewServer(proxy)
	t.Cleanup(gw.Close)
	return gw.URL, strings.TrimPrefix(gw.URL, "http://")
}

func TestStreamingResponsesAreNotHeldBack(t *testing.T) {
	release := make(chan struct{})
	gw, _ := gatewayFor(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: first\n\n"))
		w.(http.Flusher).Flush()
		select {
		case <-release:
		case <-r.Context().Done():
		}
		_, _ = w.Write([]byte("data: second\n\n"))
	}))
	// Registered after the servers' cleanups, so it runs before them: the upstream handler waits on
	// this channel, and shutting the upstream down first would wait on it forever.
	t.Cleanup(func() { close(release) })

	resp, err := http.Get(gw + "/api/apps")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	got := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(resp.Body).ReadString('\n')
		got <- line
	}()
	select {
	case line := <-got:
		if !strings.Contains(line, "first") {
			t.Fatalf("unexpected first event %q", line)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the first event was held back while the upstream was still streaming")
	}
}

func TestUpgradeRequestsPassThrough(t *testing.T) {
	gw, addr := gatewayFor(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			http.Error(w, "expected an upgrade", http.StatusBadRequest)
			return
		}
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "no hijack", http.StatusInternalServerError)
			return
		}
		conn, rw, err := hj.Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
		_ = rw.Flush()
		buf := make([]byte, 4)
		if _, err := io.ReadFull(rw, buf); err == nil {
			_, _ = rw.Write(append([]byte("echo:"), buf...))
			_ = rw.Flush()
		}
	}))
	_ = gw

	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	_, _ = conn.Write([]byte("GET /api/nodes/connect HTTP/1.1\r\nHost: gw\r\nConnection: Upgrade\r\nUpgrade: websocket\r\n\r\n"))

	br := bufio.NewReader(conn)
	status, err := br.ReadString('\n')
	if err != nil || !strings.Contains(status, "101") {
		t.Fatalf("the upgrade must be answered with 101, got %q (%v)", status, err)
	}
	for { // skip the response headers
		line, err := br.ReadString('\n')
		if err != nil || line == "\r\n" {
			break
		}
	}
	_, _ = conn.Write([]byte("ping"))
	reply := make([]byte, 9)
	if _, err := io.ReadFull(br, reply); err != nil || string(reply) != "echo:ping" {
		t.Fatalf("bytes must flow both ways after the upgrade: %q (%v)", reply, err)
	}
}

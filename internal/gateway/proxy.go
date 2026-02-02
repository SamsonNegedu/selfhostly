package gateway

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

// Proxy forwards requests to the target node and returns the response as-is
type Proxy struct {
	router        *Router
	registry      *NodeRegistry
	gatewayAPIKey string
	config        *Config
	transport     http.RoundTripper
	logger        *slog.Logger
}

// NewProxy creates a proxy that uses the router and adds gateway auth
func NewProxy(router *Router, registry *NodeRegistry, cfg *Config, logger *slog.Logger) *Proxy {
	return &Proxy{
		router:        router,
		registry:      registry,
		gatewayAPIKey: cfg.GatewayAPIKey,
		config:        cfg,
		transport:     http.DefaultTransport,
		logger:        logger,
	}
}

// ServeHTTP validates auth, resolves target, and forwards the request
func (p *Proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Handle gateway health check directly (don't route to primary)
	// Support both GET and HEAD methods (Docker healthcheck uses HEAD)
	if (req.Method == http.MethodGet || req.Method == http.MethodHead) && req.URL.Path == "/api/health" {
		w.Header().Set("Content-Type", "application/json")
		// Check if registry is ready (has successfully connected to primary at least once)
		if !p.registry.IsReady() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"initializing","service":"gateway","message":"waiting for node registry"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","service":"gateway"}`))
		return
	}

	hasReqCookie := req.Header.Get("Cookie") != ""
	p.logger.InfoContext(req.Context(), "gateway: incoming request",
		"method", req.Method,
		"path", req.URL.Path,
		"host", req.Host,
		"referer", req.Header.Get("Referer"),
		"has_cookie", hasReqCookie,
	)

	if !p.config.ValidateRequest(req) {
		p.logger.WarnContext(req.Context(), "gateway: auth required",
			"path", req.URL.Path,
			"has_cookie", req.Header.Get("Cookie") != "",
			"has_auth_header", req.Header.Get("Authorization") != "",
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"Authentication required"}`))
		return
	}

	baseURL, ok := p.router.Target(req)
	if !ok {
		p.logger.WarnContext(req.Context(), "gateway: could not resolve target",
			"path", req.URL.Path,
			"node_id", req.URL.Query().Get("node_id"),
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"node_id is required for this operation"}`))
		return
	}

	p.logger.DebugContext(req.Context(), "gateway: routing request",
		"path", req.URL.Path,
		"target", baseURL,
	)

	targetURL, err := url.Parse(baseURL)
	if err != nil {
		p.logger.ErrorContext(req.Context(), "gateway: invalid target URL",
			"base", baseURL,
			"error", err,
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Build outgoing request: same method, path, query, body
	outReq := req.Clone(req.Context())
	outReq.URL.Scheme = targetURL.Scheme
	outReq.URL.Host = targetURL.Host
	outReq.URL.Path = req.URL.Path
	outReq.URL.RawQuery = req.URL.RawQuery
	if req.Body != nil {
		outReq.Body = req.Body
		outReq.ContentLength = req.ContentLength
		outReq.GetBody = req.GetBody
	}
	if outReq.URL.RawQuery != "" {
		outReq.URL.RawPath = req.URL.Path + "?" + req.URL.RawQuery
	}
	// Strip Hop-by-hop headers that ReverseProxy would strip
	outReq.Header.Del("Connection")
	outReq.Header.Del("Proxy-Connection")
	outReq.Header.Del("Keep-Alive")
	outReq.Header.Del("Transfer-Encoding")
	outReq.Header.Del("Te")
	outReq.Header.Del("Trailer")

	// Strip Cloudflare-specific headers to prevent Error 1000 loops
	// These headers should NOT be forwarded to upstream as they can cause:
	// - Error 1000 if CF-Connecting-IP is present
	// - Error 1000 if X-Forwarded-For exceeds 100 chars or appears twice
	outReq.Header.Del("CF-Connecting-IP")
	outReq.Header.Del("CF-Ray")
	outReq.Header.Del("CF-Visitor")
	outReq.Header.Del("CF-IPCountry")
	outReq.Header.Del("CF-Request-ID")
	
	// Replace X-Forwarded-For with just the original client IP to prevent length issues
	// Cloudflare adds each hop to X-Forwarded-For which can exceed 100 chars
	if cfIP := req.Header.Get("CF-Connecting-IP"); cfIP != "" {
		// Use Cloudflare's original client IP
		outReq.Header.Set("X-Forwarded-For", cfIP)
		outReq.Header.Set("X-Real-IP", cfIP)
	} else if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		// If no CF header, use the first IP from X-Forwarded-For
		// to avoid accumulating a long chain
		if firstIP := strings.Split(xff, ",")[0]; firstIP != "" {
			outReq.Header.Set("X-Forwarded-For", strings.TrimSpace(firstIP))
		}
	}

	// Add gateway auth only for node registry/management endpoints.
	// Don't add it for user-facing endpoints (like /api/me, /api/apps, etc.)
	// because gateway auth bypasses user authentication.
	isNodeManagementEndpoint := strings.HasPrefix(req.URL.Path, "/api/nodes") &&
		!strings.HasSuffix(req.URL.Path, "/register")

	if isNodeManagementEndpoint {
		outReq.Header.Set("X-Gateway-API-Key", p.gatewayAPIKey)
	}
	// So primary can rewrite OAuth redirects to the public URL (where the user actually is).
	// Use incoming X-Forwarded-Host if set, else derive from Referer (for dev with Vite proxy),
	// else use req.Host.
	forwardedHost := req.Header.Get("X-Forwarded-Host")
	isAuthRoute := strings.HasPrefix(req.URL.Path, "/auth/") || strings.HasPrefix(req.URL.Path, "/avatar/")

	p.logger.DebugContext(req.Context(), "gateway: determining forwarded host",
		"x_forwarded_host_set", forwardedHost != "",
		"has_referer", req.Header.Get("Referer") != "",
		"req_host", req.Host,
		"is_auth_route", isAuthRoute,
	)

	// For auth routes, check the origin cookie FIRST before using Referer
	// (OAuth callbacks from GitHub will have Referer=github.com which is wrong)
	if forwardedHost == "" && strings.HasPrefix(req.URL.Path, "/auth/") {
		if cookie, err := req.Cookie("_gateway_origin"); err == nil && cookie.Value != "" {
			forwardedHost = cookie.Value
			p.logger.DebugContext(req.Context(), "gateway: retrieved host from origin cookie",
				"host", forwardedHost,
			)
		}
	}

	// If no X-Forwarded-Host and no cookie, try to extract from Referer (handles Vite proxy scenario)
	if forwardedHost == "" {
		if referer := req.Header.Get("Referer"); referer != "" {
			if refURL, err := url.Parse(referer); err == nil && refURL.Host != "" {
				forwardedHost = refURL.Host
				p.logger.DebugContext(req.Context(), "gateway: extracted host from Referer",
					"extracted_host", refURL.Host,
				)
			} else {
				p.logger.WarnContext(req.Context(), "gateway: failed to parse Referer",
					"error", err,
				)
			}
		}
	}

	if forwardedHost == "" {
		forwardedHost = req.Host
		p.logger.DebugContext(req.Context(), "gateway: using request host as forwarded host",
			"host", req.Host,
		)
	}
	outReq.Header.Set("X-Forwarded-Host", forwardedHost)

	// Always set outReq.Host to forwardedHost so go-pkgz/auth validates cookies correctly.
	// The auth library uses request.Host (not X-Forwarded-Host) to validate cookies,
	// so this must match the Host used when the cookie was issued during auth.
	outReq.Host = forwardedHost

	if isAuthRoute {
		p.logger.InfoContext(req.Context(), "gateway: rewriting Host for auth route",
			"original_host", req.Host,
			"new_host", outReq.Host,
			"path", req.URL.Path,
		)

		// For /auth/github/login, store the origin host in a cookie so we can retrieve it
		// during callback (when there's no Referer header from GitHub redirect)
		if strings.HasPrefix(req.URL.Path, "/auth/") && strings.Contains(req.URL.Path, "/login") {
			// Set a short-lived cookie to track the originating host
			http.SetCookie(w, &http.Cookie{
				Name:     "_gateway_origin",
				Value:    forwardedHost,
				Path:     "/",
				MaxAge:   300, // 5 minutes (just long enough for OAuth flow)
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
			p.logger.DebugContext(req.Context(), "gateway: stored origin host for OAuth flow",
				"host", forwardedHost,
			)
		}
	}

	p.logger.DebugContext(req.Context(), "gateway: forwarding request",
		"forwarded_host", forwardedHost,
		"forwarded_proto", outReq.Header.Get("X-Forwarded-Proto"),
		"target_host", outReq.Host,
		"is_auth_route", isAuthRoute,
		"has_gateway_key", isNodeManagementEndpoint,
	)
	if proto := req.Header.Get("X-Forwarded-Proto"); proto != "" {
		outReq.Header.Set("X-Forwarded-Proto", proto)
	} else if req.TLS != nil {
		outReq.Header.Set("X-Forwarded-Proto", "https")
	} else {
		outReq.Header.Set("X-Forwarded-Proto", "http")
	}

	// Use ReverseProxy-style response handling
	p.logger.DebugContext(req.Context(), "gateway: sending upstream request",
		"target", baseURL,
		"path", req.URL.Path,
	)

	resp, err := p.transport.RoundTrip(outReq)
	if err != nil {
		// Client disconnect (context canceled) is normal; avoid noisy ERROR logs
		if errors.Is(err, context.Canceled) || req.Context().Err() == context.Canceled {
			p.logger.DebugContext(req.Context(), "gateway: upstream request canceled by client",
				"target", baseURL,
			)
		} else {
			p.logger.ErrorContext(req.Context(), "gateway: upstream request failed",
				"target", baseURL,
				"path", req.URL.Path,
				"error", err,
			)
		}
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	hasCookie := resp.Header.Get("Set-Cookie") != ""
	p.logger.DebugContext(req.Context(), "gateway: upstream response received",
		"status", resp.StatusCode,
		"target", baseURL,
		"path", req.URL.Path,
		"has_set_cookie", hasCookie,
		"location", resp.Header.Get("Location"),
	)

	// Log Set-Cookie details for auth routes (masked for security)
	if isAuthRoute && hasCookie {
		cookies := resp.Header["Set-Cookie"]
		p.logger.InfoContext(req.Context(), "gateway: auth response with cookies",
			"cookie_count", len(cookies),
			"has_jwt", containsCookieName(cookies, "JWT"),
			"has_xsrf", containsCookieName(cookies, "XSRF-TOKEN"),
		)
	}

	// Copy response headers (exclude hop-by-hop)
	for k, vv := range resp.Header {
		kk := strings.ToLower(k)
		if kk == "connection" || kk == "keep-alive" || kk == "proxy-authenticate" ||
			kk == "proxy-authorization" || kk == "te" || kk == "trailers" || kk == "transfer-encoding" {
			continue
		}
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// containsCookieName checks if any Set-Cookie header contains the given cookie name
func containsCookieName(cookies []string, name string) bool {
	for _, cookie := range cookies {
		if strings.HasPrefix(cookie, name+"=") {
			return true
		}
	}
	return false
}

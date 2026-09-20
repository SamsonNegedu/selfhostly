package gateway

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/selfhostly/internal/constants"
)

// Proxy authenticates a request, resolves which backend it belongs to, and forwards it with the
// standard library's reverse proxy. That gives correct hop-by-hop handling, immediate flushing of
// streaming responses (server-sent events) and WebSocket upgrades without any code of our own.
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

	p.logger.DebugContext(req.Context(), "gateway: incoming request",
		"method", req.Method,
		"path", req.URL.Path,
		"host", req.Host,
		"has_cookie", req.Header.Get("Cookie") != "",
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

	targetURL, err := url.Parse(baseURL)
	if err != nil {
		p.logger.ErrorContext(req.Context(), "gateway: invalid target URL", "base", baseURL, "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	p.logger.DebugContext(req.Context(), "gateway: routing request", "path", req.URL.Path, "target", baseURL)

	isAuthRoute := strings.HasPrefix(req.URL.Path, "/auth/") || strings.HasPrefix(req.URL.Path, "/avatar/")
	forwardedHost := p.publicHost(req)

	// For /auth/.../login, remember the originating host in a short-lived cookie: the OAuth callback
	// comes back from GitHub with no useful Referer, and needs to know which host the user was on.
	if strings.HasPrefix(req.URL.Path, "/auth/") && strings.Contains(req.URL.Path, "/login") {
		http.SetCookie(w, &http.Cookie{
			Name:     "_gateway_origin",
			Value:    forwardedHost,
			Path:     "/",
			MaxAge:   300,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}

	rp := &httputil.ReverseProxy{
		Transport: p.transport,
		Rewrite: func(pr *httputil.ProxyRequest) {
			p.rewrite(pr, targetURL, forwardedHost)
		},
		ModifyResponse: func(resp *http.Response) error {
			if isAuthRoute {
				if cookies := resp.Header["Set-Cookie"]; len(cookies) > 0 {
					p.logger.InfoContext(req.Context(), "gateway: auth response with cookies",
						"cookie_count", len(cookies),
						"has_jwt", containsCookieName(cookies, "JWT"),
						"has_xsrf", containsCookieName(cookies, "XSRF-TOKEN"),
					)
				}
			}
			return nil
		},
		ErrorHandler: func(rw http.ResponseWriter, r *http.Request, err error) {
			// A client that disconnected is normal; avoid noisy ERROR logs
			if errors.Is(err, context.Canceled) || r.Context().Err() == context.Canceled {
				p.logger.DebugContext(r.Context(), "gateway: upstream request canceled by client", "target", baseURL)
			} else {
				p.logger.ErrorContext(r.Context(), "gateway: upstream request failed",
					"target", baseURL, "path", r.URL.Path, "error", err)
			}
			rw.WriteHeader(http.StatusBadGateway)
		},
	}
	rp.ServeHTTP(w, req)
}

// publicHost decides which hostname the backend should treat as the one the user is on. With
// PUBLIC_HOSTS configured only those are ever used, so a forged X-Forwarded-Host, Referer or cookie
// cannot steer OAuth redirects or cookie scoping to another host.
func (p *Proxy) publicHost(req *http.Request) string {
	host := req.Header.Get("X-Forwarded-Host")

	// OAuth callbacks from GitHub carry Referer=github.com, which is wrong: prefer the origin cookie
	if host == "" && strings.HasPrefix(req.URL.Path, "/auth/") {
		if cookie, err := req.Cookie("_gateway_origin"); err == nil && cookie.Value != "" {
			host = cookie.Value
		}
	}
	// A Referer host is only trusted when no PUBLIC_HOSTS are configured (Vite dev proxy)
	if host == "" && len(p.config.PublicHosts) == 0 {
		if referer := req.Header.Get("Referer"); referer != "" {
			if u, err := url.Parse(referer); err == nil && u.Host != "" {
				host = u.Host
			}
		}
	}
	if host == "" {
		host = req.Host
	}
	if pinned, ok := p.config.pinPublicHost(host); !ok {
		p.logger.WarnContext(req.Context(), "gateway: forwarded host is not a configured public host; using the primary one",
			"received", host, "using", pinned)
		host = pinned
	}
	return host
}

// rewrite builds the outgoing request. The reverse proxy has already removed hop-by-hop headers and
// any X-Forwarded-* the client sent, so those are set here from what the gateway itself knows.
func (p *Proxy) rewrite(pr *httputil.ProxyRequest, target *url.URL, forwardedHost string) {
	in, out := pr.In, pr.Out
	out.URL.Scheme = target.Scheme
	out.URL.Host = target.Host

	// Credentials only the gateway or a backend node may present must never be accepted from a
	// client: the gateway adds its own key below when it intends to.
	out.Header.Del(constants.HeaderGatewayAPIKey)
	out.Header.Del(constants.HeaderNodeID)
	out.Header.Del(constants.HeaderNodeAPIKey)

	// Cloudflare-specific headers cause Error 1000 loops if forwarded upstream
	for _, h := range []string{"CF-Connecting-IP", "CF-Ray", "CF-Visitor", "CF-IPCountry", "CF-Request-ID"} {
		out.Header.Del(h)
	}

	// Pass on just the original client address: Cloudflare appends every hop to X-Forwarded-For, which
	// can exceed the length Cloudflare accepts on the next hop.
	if cfIP := in.Header.Get("CF-Connecting-IP"); cfIP != "" {
		out.Header.Set("X-Forwarded-For", cfIP)
		out.Header.Set("X-Real-IP", cfIP)
	} else if xff := in.Header.Get("X-Forwarded-For"); xff != "" {
		if first := strings.TrimSpace(strings.Split(xff, ",")[0]); first != "" {
			out.Header.Set("X-Forwarded-For", first)
		}
	} else if host, _, err := net.SplitHostPort(in.RemoteAddr); err == nil {
		out.Header.Set("X-Forwarded-For", host)
	}

	// The gateway key is added only for node registry/management endpoints. It is deliberately not
	// added for user-facing endpoints, because it bypasses user authentication on the backend.
	if strings.HasPrefix(in.URL.Path, "/api/nodes") && !strings.HasSuffix(in.URL.Path, "/register") &&
		in.URL.Path != constants.LinkPath {
		out.Header.Set(constants.HeaderGatewayAPIKey, p.gatewayAPIKey)
	}

	out.Header.Set("X-Forwarded-Host", forwardedHost)
	// The auth library validates cookies against request.Host (not X-Forwarded-Host), so it must match
	// the host used when the cookie was issued.
	out.Host = forwardedHost

	switch {
	case in.Header.Get("X-Forwarded-Proto") != "":
		out.Header.Set("X-Forwarded-Proto", in.Header.Get("X-Forwarded-Proto"))
	case in.TLS != nil:
		out.Header.Set("X-Forwarded-Proto", "https")
	default:
		out.Header.Set("X-Forwarded-Proto", "http")
	}
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

package http

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
)

// secretsEqual compares two secrets without leaking their contents through timing.
func secretsEqual(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func isMutating(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

// originGuardMiddleware defends cookie-authenticated endpoints against cross-site requests. The
// session cookie is SameSite=Lax, and on top of that a state-changing request must either carry no
// Origin (non-browser client) or one that belongs to this deployment, and a request body must be
// JSON (which a cross-site HTML form cannot send).
func (s *Server) originGuardMiddleware() gin.HandlerFunc {
	trusted := map[string]bool{}
	for _, o := range s.config.CORS.AllowedOrigins {
		if u, err := url.Parse(o); err == nil && u.Host != "" {
			trusted[strings.ToLower(u.Host)] = true
		}
	}
	if u, err := url.Parse(s.config.Auth.BaseURL); err == nil && u.Host != "" {
		trusted[strings.ToLower(u.Host)] = true
	}

	return func(c *gin.Context) {
		if !isMutating(c.Request.Method) {
			c.Next()
			return
		}
		violation := ""
		if origin := c.GetHeader("Origin"); origin != "" {
			u, err := url.Parse(origin)
			host := ""
			if err == nil {
				host = strings.ToLower(u.Host)
			}
			fwd := strings.ToLower(strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Host"), ",")[0]))
			if host == "" || (!trusted[host] && host != strings.ToLower(c.Request.Host) && host != fwd) {
				violation = "cross-origin request from " + origin
			}
		}
		if violation == "" && c.Request.ContentLength != 0 {
			if !strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
				violation = "request body is not application/json"
			}
		}
		if violation != "" {
			if s.config.Enforcing() {
				slog.WarnContext(c.Request.Context(), "request rejected", "reason", violation, "path", c.Request.URL.Path)
				c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Error: "Request rejected", Details: violation})
				return
			}
			slog.WarnContext(c.Request.Context(), "request would be rejected in enforce mode", "reason", violation, "path", c.Request.URL.Path)
		}
		c.Next()
	}
}

// bodyLimitMiddleware caps every state-changing request body, whatever its content type.
func bodyLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isMutating(c.Request.Method) {
			if c.Request.ContentLength > maxBytes {
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Request body too large"})
				return
			}
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}

// actorFor describes who made the request, for the audit trail. It uses the stable user ID, never
// the display name, which the user controls.
func actorFor(c *gin.Context) string {
	if u, ok := getUserFromContext(c); ok && u.ID != "" {
		return u.ID
	}
	if email, ok := c.Get("cf_email"); ok {
		if e, _ := email.(string); e != "" {
			return "cf:" + e
		}
	}
	if id, ok := c.Get("node_id"); ok {
		if s, _ := id.(string); s != "" {
			return "node:" + s
		}
	}
	if scope, ok := c.Get("request_scope"); ok && scope == "local" {
		return "gateway"
	}
	return "anonymous"
}

// auditMiddleware records every state-changing API request with its outcome.
func (s *Server) auditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		record := isMutating(c.Request.Method) && strings.HasPrefix(path, "/api/") && !strings.HasSuffix(path, "/heartbeat")
		var route auditRoute
		var target auditTarget
		if record {
			route, _, target = s.describeAudit(c)
		}
		c.Next()
		if !record {
			return
		}
		if named, ok := c.Get(auditTargetKey); ok {
			if t, ok := named.(auditTarget); ok {
				target = t
			}
		}
		entry := db.AuditEntry{
			Action:     route.Action,
			TargetType: target.Type,
			TargetID:   target.ID,
			TargetName: target.Name,
			Time:       time.Now(),
			Actor:      actorFor(c),
			Method:     c.Request.Method,
			Path:       path,
			Status:     c.Writer.Status(),
			RemoteAddr: c.ClientIP(),
		}
		slog.InfoContext(c.Request.Context(), "audit",
			"actor", entry.Actor, "method", entry.Method, "path", entry.Path, "status", entry.Status)
		if err := s.securityService.RecordAudit(c.Request.Context(), entry); err != nil {
			slog.Warn("could not write audit record", "error", err)
		}
	}
}

// attemptLimiter is a small per-key sliding-window limiter for unauthenticated endpoints.
type attemptLimiter struct {
	mu       sync.Mutex
	window   time.Duration
	max      int
	attempts map[string][]time.Time
}

func newAttemptLimiter(max int, window time.Duration) *attemptLimiter {
	return &attemptLimiter{window: window, max: max, attempts: map[string][]time.Time{}}
}

// allow records an attempt and reports whether the key is still within its budget.
func (l *attemptLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := now.Add(-l.window)
	kept := l.attempts[key][:0]
	for _, t := range l.attempts[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.max {
		l.attempts[key] = kept
		return false
	}
	l.attempts[key] = append(kept, now)
	if len(l.attempts) > 4096 { // bound memory under a spray of distinct keys
		for k, ts := range l.attempts {
			if len(ts) == 0 || !ts[len(ts)-1].After(cutoff) {
				delete(l.attempts, k)
			}
		}
	}
	return true
}

func (s *Server) registerRateLimitMiddleware() gin.HandlerFunc {
	limiter := newAttemptLimiter(constants.RegisterRateLimitAttempts, constants.RegisterRateLimitWindow)
	return func(c *gin.Context) {
		if !limiter.allow(c.ClientIP(), time.Now()) {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, ErrorResponse{Error: "Too many attempts", Details: "try again in a minute"})
			return
		}
		c.Next()
	}
}

// cfAccessMiddleware requires a valid Cloudflare Access token when GitHub auth is not in use, so
// the backend does not rely on network position alone.
func (s *Server) cfAccessMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		email, err := s.cfAccess.Verify(c.Request.Context(), c.GetHeader(constants.HeaderCFAccessJWT))
		if err != nil {
			slog.WarnContext(c.Request.Context(), "Cloudflare Access verification failed", "error", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "Authentication required"})
			return
		}
		c.Set("cf_email", email)
		c.Next()
	}
}

// denyNodeAuthMiddleware keeps user-level administration away from node credentials: a secondary
// node's key must never be able to mint join tokens, revoke sessions or read the audit log.
func (s *Server) denyNodeAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, isNode := c.Get("node_id"); isNode {
			c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Error: "Node credentials cannot perform this action"})
			return
		}
		c.Next()
	}
}

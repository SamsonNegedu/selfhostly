package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// IngressRule represents a single ingress rule for a Cloudflare tunnel
type IngressRule struct {
	Hostname      *string                `json:"hostname"`
	Service       string                 `json:"service"`
	Path          *string                `json:"path"`
	OriginRequest map[string]interface{} `json:"originRequest"`
}

// CloudflareTunnelResponse represents a Cloudflare tunnel API response
type CloudflareTunnelResponse struct {
	ID           string        `json:"id"`
	AppID        string        `json:"app_id"`
	TunnelID     string        `json:"tunnel_id"`
	TunnelName   string        `json:"tunnel_name"`
	Status       string        `json:"status"`
	IsActive     bool          `json:"is_active"`
	PublicURL    string        `json:"public_url"`
	IngressRules []IngressRule `json:"ingress_rules,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	LastSyncedAt *time.Time    `json:"last_synced_at"`
	ErrorDetails string        `json:"error_details,omitempty"`
}

// listLocalTunnels returns only tunnels from the local node (for inter-node calls)
func (s *Server) listLocalTunnels(c *gin.Context) {
	tunnels, err := s.database.ListActiveCloudflareTunnels()
	if err != nil {
		s.handleServiceError(c, "list local tunnels", err)
		return
	}

	c.JSON(http.StatusOK, tunnels)
}

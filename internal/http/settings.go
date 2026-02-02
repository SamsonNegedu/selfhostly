package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// UpdateSettingsRequest represents an update settings request
type UpdateSettingsRequest struct {
	AutoStartApps        bool   `json:"auto_start_apps"`
	ActiveTunnelProvider string `json:"active_tunnel_provider"`
	TunnelProviderConfig string `json:"tunnel_provider_config"`
}

// getSettingsDispatch returns settings: when node auth (request_scope=local) calls getSettingsForNode, else getSettings
func (s *Server) getSettingsDispatch(c *gin.Context) {
	if scope, ok := c.Get("request_scope"); ok && scope == "local" {
		s.getSettingsForNode(c)
		return
	}
	s.getSettings(c)
}

// getSettings returns current settings
func (s *Server) getSettings(c *gin.Context) {
	settings, err := s.database.GetSettings()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to retrieve settings", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve settings"})
		return
	}

	// Don't expose sensitive data - mask tokens in tunnel_provider_config
	activeTunnelProvider := ""
	if settings.ActiveTunnelProvider != nil {
		activeTunnelProvider = *settings.ActiveTunnelProvider
	}
	tunnelProviderConfig := ""
	if settings.TunnelProviderConfig != nil {
		tunnelProviderConfig = maskTokensInProviderConfig(*settings.TunnelProviderConfig)
	}
	response := gin.H{
		"id":                     settings.ID,
		"auto_start_apps":        settings.AutoStartApps,
		"active_tunnel_provider": activeTunnelProvider,
		"tunnel_provider_config": tunnelProviderConfig,
		"updated_at":             settings.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// updateSettings updates settings
func (s *Server) updateSettings(c *gin.Context) {
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.WarnContext(c.Request.Context(), "invalid update settings request", "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request format"})
		return
	}

	// Get current settings to preserve unset values
	settings, err := s.database.GetSettings()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to retrieve settings for update", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve settings"})
		return
	}

	// Update only provided fields - only use tunnel_provider_config, no legacy fields
	settings.AutoStartApps = req.AutoStartApps

	// Update new provider fields
	if req.ActiveTunnelProvider != "" {
		settings.ActiveTunnelProvider = &req.ActiveTunnelProvider
	}
	if req.TunnelProviderConfig != "" {
		settings.TunnelProviderConfig = &req.TunnelProviderConfig
	}

	if err := s.database.UpdateSettings(settings); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to update settings", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update settings"})
		return
	}

	slog.InfoContext(c.Request.Context(), "settings updated successfully")

	// Return updated settings with masked tokens
	activeTunnelProvider := ""
	if settings.ActiveTunnelProvider != nil {
		activeTunnelProvider = *settings.ActiveTunnelProvider
	}
	tunnelProviderConfig := ""
	if settings.TunnelProviderConfig != nil {
		tunnelProviderConfig = maskTokensInProviderConfig(*settings.TunnelProviderConfig)
	}
	response := gin.H{
		"id":                     settings.ID,
		"auto_start_apps":        settings.AutoStartApps,
		"active_tunnel_provider": activeTunnelProvider,
		"tunnel_provider_config": tunnelProviderConfig,
		"updated_at":             settings.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// maskToken masks sensitive token data
func maskToken(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 8 {
		return "********"
	}
	return token[:4] + "****" + token[len(token)-4:]
}

// maskTokensInProviderConfig masks all api_token fields in the provider config JSON
func maskTokensInProviderConfig(configJSON string) string {
	if configJSON == "" {
		return configJSON
	}

	var providerConfigs map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &providerConfigs); err != nil {
		// If parsing fails, return as-is (shouldn't happen, but be safe)
		return configJSON
	}

	// Mask tokens in all providers
	for providerName, providerConfig := range providerConfigs {
		if configMap, ok := providerConfig.(map[string]interface{}); ok {
			if apiToken, ok := configMap["api_token"].(string); ok && apiToken != "" {
				// Only mask if not already masked
				if !strings.Contains(apiToken, "****") {
					configMap["api_token"] = maskToken(apiToken)
					providerConfigs[providerName] = configMap
				}
			}
		}
	}

	// Re-marshal back to JSON
	maskedJSON, err := json.Marshal(providerConfigs)
	if err != nil {
		// If marshaling fails, return original
		return configJSON
	}

	return string(maskedJSON)
}

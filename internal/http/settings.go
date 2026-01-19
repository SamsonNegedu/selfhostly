package http

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// UpdateSettingsRequest represents an update settings request
type UpdateSettingsRequest struct {
	CloudflareAPIToken  string `json:"cloudflare_api_token"`
	CloudflareAccountID string `json:"cloudflare_account_id"`
	AutoStartApps       bool   `json:"auto_start_apps"`
}

// getSettings returns current settings
func (s *Server) getSettings(c *gin.Context) {
	settings, err := s.database.GetSettings()
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to retrieve settings", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to retrieve settings"})
		return
	}

	// Don't expose sensitive data
	tokenValue := ""
	if settings.CloudflareAPIToken != nil {
		tokenValue = *settings.CloudflareAPIToken
	}
	accountIDValue := ""
	if settings.CloudflareAccountID != nil {
		accountIDValue = *settings.CloudflareAccountID
	}
	response := gin.H{
		"id":                    settings.ID,
		"cloudflare_api_token":  maskToken(tokenValue),
		"cloudflare_account_id": accountIDValue,
		"auto_start_apps":       settings.AutoStartApps,
		"updated_at":            settings.UpdatedAt,
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

	// Update only provided fields
	if req.CloudflareAPIToken != "" {
		settings.CloudflareAPIToken = &req.CloudflareAPIToken
	}
	if req.CloudflareAccountID != "" {
		settings.CloudflareAccountID = &req.CloudflareAccountID
	}
	settings.AutoStartApps = req.AutoStartApps

	if err := s.database.UpdateSettings(settings); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to update settings", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update settings"})
		return
	}

	slog.InfoContext(c.Request.Context(), "settings updated successfully")

	// Return updated settings (masked token)
	tokenValue := ""
	if settings.CloudflareAPIToken != nil {
		tokenValue = *settings.CloudflareAPIToken
	}
	accountIDValue := ""
	if settings.CloudflareAccountID != nil {
		accountIDValue = *settings.CloudflareAccountID
	}
	response := gin.H{
		"id":                    settings.ID,
		"cloudflare_api_token":  maskToken(tokenValue),
		"cloudflare_account_id": accountIDValue,
		"auto_start_apps":       settings.AutoStartApps,
		"updated_at":            settings.UpdatedAt,
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

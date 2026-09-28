package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/domain"
)

// CreateDeployHookRequest names the new hook. The source kind is not client-supplied: every hook
// made through this route is constants.DeploySourceGeneric until a second kind exists to choose from.
type CreateDeployHookRequest struct {
	Name string `json:"name" binding:"required"`
}

// createdDeployHookResponse embeds the same shape listDeployHooks returns, plus the plaintext token
// - the one field list never includes. Embedding, rather than hand-listing fields again, keeps the
// two responses from drifting apart as domain.DeployHook gains fields.
type createdDeployHookResponse struct {
	domain.DeployHook
	Token string `json:"token"`
}

// createDeployHook issues a new deploy hook for an app. Session-authed like any other app route;
// the token it returns, not this route, is what an external pipeline uses.
func (s *Server) createDeployHook(c *gin.Context) {
	appID := c.Param("id")
	var req CreateDeployHookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body", Details: domain.PublicMessage(err)})
		return
	}

	token, hook, err := s.deployHookService.CreateDeployHook(c.Request.Context(), appID, req.Name)
	if err != nil {
		s.handleServiceError(c, "create deploy hook", err)
		return
	}
	// The plaintext token is returned exactly once, here, and never again: the database only ever
	// holds its hash.
	c.JSON(http.StatusCreated, createdDeployHookResponse{DeployHook: *hook, Token: token})
}

// listDeployHooks returns an app's hooks. Tokens are never included; only metadata is.
func (s *Server) listDeployHooks(c *gin.Context) {
	appID := c.Param("id")
	hooks, err := s.deployHookService.ListDeployHooks(c.Request.Context(), appID)
	if err != nil {
		s.handleServiceError(c, "list deploy hooks", err)
		return
	}
	c.JSON(http.StatusOK, hooks)
}

// revokeDeployHook deletes one hook. It stops working immediately.
func (s *Server) revokeDeployHook(c *gin.Context) {
	appID := c.Param("id")
	hookID := c.Param("hookId")
	if err := s.deployHookService.RevokeDeployHook(c.Request.Context(), appID, hookID); err != nil {
		s.handleServiceError(c, "revoke deploy hook", err)
		return
	}
	c.Status(http.StatusNoContent)
}

// bearerToken extracts the token from an "Authorization: Bearer <token>" header, or "" when the
// header is missing or a different scheme.
func bearerToken(header string) string {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return header[len(prefix):]
}

// triggerDeployUpdate is the endpoint an external pipeline calls, authenticated purely by the
// per-app deploy hook token in the Authorization header. It needs no session and no node
// credentials, so it is mounted outside the normal /api auth group (see routes.go); the token is
// the only thing that authorizes it, and it authorizes nothing beyond triggering an update for the
// one app named in the path, the same job the dashboard's own Update button starts.
func (s *Server) triggerDeployUpdate(c *gin.Context) {
	appID := c.Param("id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid app ID"})
		return
	}
	token := bearerToken(c.GetHeader("Authorization"))
	if token == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Missing deploy hook token", Details: "send it as Authorization: Bearer <token>"})
		return
	}

	job, hook, err := s.deployHookService.TriggerDeploy(c.Request.Context(), appID, token, c.ClientIP())
	if errors.Is(err, domain.ErrDeployTriggerUnauthorized) {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "Invalid deploy hook token"})
		return
	}
	if err != nil {
		s.handleServiceError(c, "trigger deploy", err)
		return
	}

	// Named here so the audit entry's actor reads "deploy-hook:<name>" instead of "anonymous" -
	// there is no session or node identity on this request for actorFor to fall back to. The
	// target stays the app itself (resolved automatically from the URL, like every other app route).
	c.Set("deploy_hook_actor", "deploy-hook:"+hook.Name)
	c.JSON(http.StatusAccepted, gin.H{
		"job_id":  job.ID,
		"status":  job.Status,
		"message": "App update started in background",
	})
}

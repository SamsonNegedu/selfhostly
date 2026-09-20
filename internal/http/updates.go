package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/update"
)

// Codes the browser can react to without matching text.
const (
	codeUpdatesUnavailable = "updates_unavailable"
	codeUpdateInProgress   = "update_in_progress"
	codeReviewRequired     = "review_required"
	codeUnknownVersion     = "unknown_version"
	codeComposeNotApproved = "compose_not_approved"
	codeUpdateBlocked      = "update_blocked"
	codeInvalidInput       = "invalid_input"
)

// updateBlockedResponse is the answer when an update cannot start: the reasons, so the UI can show each one.
type updateBlockedResponse struct {
	Error    string           `json:"error"`
	Code     string           `json:"code"`
	Blockers []update.Blocker `json:"blockers"`
}

// getUpdate reports what the update feature knows: the running version, a newer release if one was found, the
// review of it, and the last run.
func (s *Server) getUpdate(c *gin.Context) {
	c.JSON(http.StatusOK, s.updateService.View(c.Request.Context()))
}

// checkUpdate looks for a new release now.
func (s *Server) checkUpdate(c *gin.Context) {
	if err := s.updateService.Check(c.Request.Context()); err != nil && errors.Is(err, update.ErrDisabled) {
		s.updateError(c, err)
		return
	}
	// a failed check is not a failed request: the reason is in the view, next to what was known before
	c.JSON(http.StatusOK, s.updateService.View(c.Request.Context()))
}

type planUpdateRequest struct {
	Version string `json:"version" binding:"required"`
}

// planUpdate starts the review of a release. The review pulls an image, so it runs in the background.
func (s *Server) planUpdate(c *gin.Context) {
	var req planUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request format", Code: codeInvalidInput})
		return
	}
	if err := s.updateService.Review(req.Version); err != nil {
		s.updateError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, s.updateService.View(c.Request.Context()))
}

// applyUpdate starts the updater. The browser names a version and gives values for the settings the release
// asks for; everything else comes from the signed manifest.
func (s *Server) applyUpdate(c *gin.Context) {
	var req update.ApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Version == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request format", Code: codeInvalidInput})
		return
	}
	run, err := s.updateService.Apply(c.Request.Context(), req)
	if err != nil {
		s.updateError(c, err)
		return
	}
	setAuditTarget(c, targetUpdate, run.ID, run.ToVersion)
	c.JSON(http.StatusAccepted, s.updateService.View(c.Request.Context()))
}

// rollbackUpdate starts an updater that restores the state saved before the last update.
func (s *Server) rollbackUpdate(c *gin.Context) {
	run, err := s.updateService.Rollback(c.Request.Context())
	if err != nil {
		s.updateError(c, err)
		return
	}
	setAuditTarget(c, targetUpdate, run.ID, "rollback")
	c.JSON(http.StatusAccepted, s.updateService.View(c.Request.Context()))
}

// updateError maps the update errors to responses. The messages are written for the operator and hold no
// internals: docker output is cut to its first line before it gets here.
func (s *Server) updateError(c *gin.Context, err error) {
	var blocked *update.BlockedError
	switch {
	case errors.As(err, &blocked):
		c.JSON(http.StatusConflict, updateBlockedResponse{Error: err.Error(), Code: codeUpdateBlocked, Blockers: blocked.Blockers})
	case errors.Is(err, update.ErrDisabled):
		c.JSON(http.StatusForbidden, ErrorResponse{Error: err.Error(), Code: codeUpdatesUnavailable})
	case errors.Is(err, update.ErrInProgress):
		c.JSON(http.StatusConflict, ErrorResponse{Error: err.Error(), Code: codeUpdateInProgress})
	case errors.Is(err, update.ErrNoPlan):
		c.JSON(http.StatusConflict, ErrorResponse{Error: err.Error(), Code: codeReviewRequired})
	case errors.Is(err, update.ErrNotApproved):
		c.JSON(http.StatusConflict, ErrorResponse{Error: err.Error(), Code: codeComposeNotApproved})
	case errors.Is(err, update.ErrUnknownVersion):
		c.JSON(http.StatusConflict, ErrorResponse{Error: err.Error(), Code: codeUnknownVersion})
	case errors.Is(err, update.ErrBadInput):
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error(), Code: codeInvalidInput})
	default:
		slog.ErrorContext(c.Request.Context(), "update failed", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}
}

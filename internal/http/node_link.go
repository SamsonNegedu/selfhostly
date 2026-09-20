package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
)

var linkNodeID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

// linkEndpoint is how a node reached over its link is recorded: tunnel://<node-id>
func linkEndpoint(nodeID string) string { return constants.LinkEndpointScheme + "://" + nodeID }

// nodeLinkChanged keeps a node's recorded status in step with its connection: down is offline at
// once (no waiting for the next poll), up triggers an immediate health check over the new link.
func (s *Server) nodeLinkChanged(nodeID string, connected bool) {
	if connected {
		go func() {
			if err := s.nodeService.HealthCheckNode(s.shutdownCtx, nodeID); err != nil {
				slog.Warn("health check over the new node link failed", "node_id", nodeID, "error", err)
			}
		}()
		slog.Info("node link up", "node_id", nodeID)
		return
	}
	slog.Info("node link down", "node_id", nodeID)
	if err := s.nodeLinkService.MarkOffline(s.shutdownCtx, nodeID); err != nil {
		slog.Warn("could not mark a node offline after its link dropped", "node_id", nodeID, "error", err)
	}
}

// nodeConnect is the endpoint a secondary dials out to. It validates the request, has the node link
// service authenticate and record the node, then turns the connection into that node's link and holds
// it open. The rules (known node by key, new node by token, direct node switching) live in the service.
func (s *Server) nodeConnect(c *gin.Context) {
	if !s.config.Node.IsPrimary {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Not found"})
		return
	}
	req := domain.LinkRequest{
		ID:    c.GetHeader(constants.HeaderLinkNodeID),
		Name:  c.GetHeader(constants.HeaderLinkNodeName),
		Key:   c.GetHeader(constants.HeaderLinkNodeKey),
		Token: c.GetHeader(constants.HeaderLinkJoinToken),
	}
	if !linkNodeID.MatchString(req.ID) || len(req.Key) < minLinkKeyLength {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid node identity"})
		return
	}
	if req.Name == "" {
		req.Name = req.ID
	}

	admission, err := s.nodeLinkService.Admit(c.Request.Context(), req)
	switch {
	case errors.Is(err, domain.ErrLinkUnauthorized):
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Invalid node credentials",
			Details: "The token is wrong, expired, or was already used, or the node key does not match",
		})
		return
	case errors.Is(err, domain.ErrLinkNameTaken):
		c.JSON(http.StatusConflict, ErrorResponse{Error: "Node name already exists"})
		return
	case err != nil:
		s.handleServiceError(c, "connect node", err)
		return
	}
	if admission.Created || admission.Switched {
		s.auditNodeLink(c, admission)
	}

	if err := s.nodeLinks.Serve(c.Writer, c.Request, req.ID); err != nil {
		slog.Warn("node link could not be established", "node_id", req.ID, "error", err)
	}
}

// auditNodeLink records a node being registered or moved to its link. Reconnects are not recorded:
// they change nothing. The link endpoint is a GET, so the request middleware does not see it.
func (s *Server) auditNodeLink(c *gin.Context, a *domain.LinkAdmission) {
	action := "node.link_switch"
	if a.Created {
		action = "node.join"
	}
	entry := db.AuditEntry{
		Action: action, TargetType: targetNode, TargetID: a.Node.ID, TargetName: a.Node.Name,
		Time: time.Now(), Actor: "node:" + a.Node.ID, Method: c.Request.Method, Path: c.Request.URL.Path,
		Status: http.StatusSwitchingProtocols, RemoteAddr: c.ClientIP(),
	}
	if err := s.securityService.RecordAudit(c.Request.Context(), entry); err != nil {
		slog.Warn("could not write audit record", "error", err)
	}
}

const minLinkKeyLength = 16

// forwardToLinkedNode sends a user's request for a linked node down that node's link.
//
// By-id operations run against the local database of whichever backend receives them, and the
// gateway normally sends them straight to the right node. A linked node has no address the gateway
// can reach, so the primary does the forwarding, using the node's own credentials: the user's
// cookies, tokens and browser Origin are removed and never reach the secondary.
func (s *Server) forwardToLinkedNode() gin.HandlerFunc {
	return func(c *gin.Context) {
		// requests authenticated as a node or the gateway are already meant for this backend
		if scope, ok := c.Get("request_scope"); ok && scope == "local" {
			c.Next()
			return
		}
		nodeID := c.Query("node_id")
		if nodeID == "" && c.Request.Method == http.MethodPost && c.Request.URL.Path == "/api/apps" {
			nodeID = peekNodeID(c)
		}
		if nodeID == "" || nodeID == s.config.Node.ID {
			c.Next()
			return
		}
		node, _ := s.nodeLinkService.LinkedNode(c.Request.Context(), nodeID)
		if node == nil {
			c.Next() // unknown or directly reached: the existing handling applies
			return
		}
		if !s.nodeLinks.Connected(nodeID) {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, ErrorResponse{
				Error:   "Node is not connected",
				Details: "the node " + node.Name + " has no live link to the primary right now",
			})
			return
		}

		proxy := &httputil.ReverseProxy{
			Transport: s.nodeLinks.RoundTripper(),
			Rewrite: func(pr *httputil.ProxyRequest) {
				out := pr.Out
				out.URL.Scheme = constants.LinkEndpointScheme
				out.URL.Host = node.ID
				out.Host = ""
				for _, h := range []string{"Cookie", "Authorization", "Origin", "Referer", "X-Xsrf-Token",
					constants.HeaderGatewayAPIKey, constants.HeaderCFAccessJWT} {
					out.Header.Del(h)
				}
				out.Header.Set(constants.HeaderNodeID, node.ID)
				out.Header.Set(constants.HeaderNodeAPIKey, node.APIKey)
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				slog.Warn("forwarding to a linked node failed", "node_id", nodeID, "error", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadGateway)
				_, _ = w.Write([]byte(`{"error":"Node did not answer"}`))
			},
		}
		proxy.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}

// peekNodeID reads node_id from a JSON body and puts the body back for the handler
func peekNodeID(c *gin.Context) string {
	if !strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		return ""
	}
	body, err := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return ""
	}
	var v struct {
		NodeID string `json:"node_id"`
	}
	_ = json.Unmarshal(body, &v)
	return v.NodeID
}

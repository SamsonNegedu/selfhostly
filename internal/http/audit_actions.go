package http

import (
	"github.com/gin-gonic/gin"
	"github.com/selfhostly/internal/domain"
)

// auditRoute says what a route does, for the audit log: an action name, the kind of thing it acts on, and the
// name of the URL parameter that identifies it.
type auditRoute struct {
	Action     string
	TargetType string
	Param      string
}

// Target kinds.
const (
	targetApp       = domain.AuditTargetApp
	targetNode      = domain.AuditTargetNode
	targetContainer = "container"
	targetSettings  = "settings"
	targetSessions  = "sessions"
	targetUpdate    = "update"
)

// auditRoutes maps "METHOD route-pattern" to what it does. A route that is not listed is still recorded, with
// its method, path and result, only without an action.
var auditRoutes = map[string]auditRoute{
	"POST /api/apps":                                 {"app.create", targetApp, ""},
	"PUT /api/apps/:id":                              {"app.edit", targetApp, "id"},
	"DELETE /api/apps/:id":                           {"app.delete", targetApp, "id"},
	"POST /api/apps/:id/start":                       {"app.start", targetApp, "id"},
	"POST /api/apps/:id/stop":                        {"app.stop", targetApp, "id"},
	"POST /api/apps/:id/update":                      {"app.update", targetApp, "id"},
	"POST /api/apps/:id/services/:service/restart":   {"app.service_restart", targetApp, "id"},
	"POST /api/apps/:id/quick-tunnel":                {"tunnel.quick_create", targetApp, "id"},
	"POST /api/apps/:id/schedule":                    {"schedule.save", targetApp, "id"},
	"DELETE /api/apps/:id/schedule":                  {"schedule.delete", targetApp, "id"},
	"POST /api/apps/:id/compose/rollback/:version":   {"app.restore_version", targetApp, "id"},
	"POST /api/tunnels/apps/:appId":                  {"tunnel.create", targetApp, "appId"},
	"POST /api/tunnels/apps/:appId/switch-to-custom": {"tunnel.switch_to_custom", targetApp, "appId"},
	"POST /api/tunnels/apps/:appId/sync":             {"tunnel.sync", targetApp, "appId"},
	"PUT /api/tunnels/apps/:appId/ingress":           {"tunnel.routes_edit", targetApp, "appId"},
	"POST /api/tunnels/apps/:appId/dns":              {"tunnel.dns_create", targetApp, "appId"},
	"DELETE /api/tunnels/apps/:appId":                {"tunnel.delete", targetApp, "appId"},
	"PUT /api/settings":                              {"settings.update", targetSettings, ""},
	"POST /api/system/containers/:id/restart":        {"container.restart", targetContainer, "id"},
	"POST /api/system/containers/:id/stop":           {"container.stop", targetContainer, "id"},
	"DELETE /api/system/containers/:id":              {"container.delete", targetContainer, "id"},
	"POST /api/nodes":                                {"node.register", targetNode, ""},
	"PUT /api/nodes/:id":                             {"node.edit", targetNode, "id"},
	"DELETE /api/nodes/:id":                          {"node.remove", targetNode, "id"},
	"POST /api/nodes/join-tokens":                    {"node.join_token", targetNode, ""},
	"POST /api/nodes/:id/check":                      {"node.check", targetNode, "id"},
	"POST /api/nodes/register":                       {"node.join", targetNode, ""},
	"POST /api/security/revoke-sessions":             {"session.revoke_all", targetSessions, ""},
	"POST /api/system/update/check":                  {"update.check", targetUpdate, ""},
	"POST /api/system/update/plan":                   {"update.plan", targetUpdate, ""},
	"POST /api/system/update/apply":                  {"update.apply", targetUpdate, ""},
	"POST /api/system/update/rollback":               {"update.rollback", targetUpdate, ""},
}

const auditTargetKey = "audit_target"

type auditTarget struct{ Type, ID, Name string }

// setAuditTarget lets a handler name what it created, when the URL cannot: a new app or node has no id in the
// address. The name is a label such as an app name, never a value from the request body.
func setAuditTarget(c *gin.Context, targetType, id, name string) {
	c.Set(auditTargetKey, auditTarget{Type: targetType, ID: id, Name: name})
}

// describeAudit works out the action and target of a request. It runs before the handler, so a target that the
// request deletes can still be named.
func (s *Server) describeAudit(c *gin.Context) (route auditRoute, known bool, target auditTarget) {
	route, known = auditRoutes[c.Request.Method+" "+c.FullPath()]
	if !known {
		return route, false, target
	}
	target.Type = route.TargetType
	if route.Param != "" {
		target.ID = c.Param(route.Param)
	}
	if target.ID != "" {
		target.Name = s.securityService.AuditTargetName(c.Request.Context(), route.TargetType, target.ID)
	}
	return route, true, target
}

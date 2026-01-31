package apipaths

import "fmt"

// Single API surface paths (no /api/internal). Used by routes and by node/heartbeat clients.

const (
	Apps         = "/api/apps"
	Settings     = "/api/settings"
	SystemStats  = "/api/system/stats"
	TunnelsList  = "/api/tunnels"
	NodeRegister = "/api/nodes/register"
	Health       = "/api/health"
)

func AppByID(appID string) string              { return "/api/apps/" + appID }
func AppStart(appID string) string             { return "/api/apps/" + appID + "/start" }
func AppStop(appID string) string              { return "/api/apps/" + appID + "/stop" }
func AppUpdateContainers(appID string) string  { return "/api/apps/" + appID + "/update" }
func AppComposeVersions(appID string) string   { return "/api/apps/" + appID + "/compose/versions" }
func AppComposeVersion(appID string, v int) string { return fmt.Sprintf("/api/apps/%s/compose/versions/%d", appID, v) }
func AppComposeRollback(appID string, v int) string { return fmt.Sprintf("/api/apps/%s/compose/rollback/%d", appID, v) }
func AppLogs(appID string) string              { return "/api/apps/" + appID + "/logs" }
func AppStats(appID string) string             { return "/api/apps/" + appID + "/stats" }
func TunnelByApp(appID string) string          { return "/api/tunnels/apps/" + appID }
func TunnelSync(appID string) string           { return "/api/tunnels/apps/" + appID + "/sync" }
func TunnelIngress(appID string) string        { return "/api/tunnels/apps/" + appID + "/ingress" }
func TunnelDNS(appID string) string            { return "/api/tunnels/apps/" + appID + "/dns" }
func NodeHeartbeat(nodeID string) string       { return "/api/nodes/" + nodeID + "/heartbeat" }
func ContainerRestart(containerID string) string { return "/api/system/containers/" + containerID + "/restart" }
func ContainerStop(containerID string) string    { return "/api/system/containers/" + containerID + "/stop" }
func Container(containerID string) string        { return "/api/system/containers/" + containerID }

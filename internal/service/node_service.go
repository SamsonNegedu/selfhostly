package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
	"github.com/selfhostly/internal/node"
	nodepkg "github.com/selfhostly/internal/node"
)

// nodeService implements node management operations
type nodeService struct {
	database   *db.DB
	nodeClient *node.Client
	config     *config.Config
	logger     *slog.Logger
}

// NewNodeService creates a new node service
func NewNodeService(
	database *db.DB,
	cfg *config.Config,
	logger *slog.Logger,
) domain.NodeService {
	return &nodeService{
		database:   database,
		nodeClient: node.NewClient(),
		config:     cfg,
		logger:     logger,
	}
}

// RegisterNode registers a new node in the cluster
func (s *nodeService) RegisterNode(ctx context.Context, req domain.RegisterNodeRequest) (*db.Node, error) {
	s.logger.InfoContext(ctx, "registering new node", "name", req.Name, "id", req.ID)

	// Validate node ID is provided
	if req.ID == "" {
		return nil, domain.WrapValidationError("id", fmt.Errorf("node ID is required"))
	}
	if err := node.ValidateEndpoint(req.APIEndpoint); err != nil {
		return nil, domain.WrapValidationError("api_endpoint", err)
	}

	// Check if node with this ID already exists
	existingNodeByID, err := s.database.GetNode(req.ID)
	if err == nil && existingNodeByID != nil {
		return nil, domain.WrapConflict(domain.CodeNodeIDTaken, "id", fmt.Sprintf("a node with ID %s is already in the cluster", req.ID))
	}

	// Check if node with this name already exists
	existingNode, err := s.database.GetNodeByName(req.Name)
	if err == nil && existingNode != nil {
		return nil, domain.WrapConflict(domain.CodeNodeNameTaken, "name", fmt.Sprintf("a node named %s is already in the cluster", req.Name))
	}

	// Create new node with the provided ID
	newNode := db.NewNodeWithID(req.ID, req.Name, req.APIEndpoint, req.APIKey, false)

	// The node is saved as soon as it is known. Its first health check runs in the background, because an
	// address that does not answer would otherwise make the caller wait for the full network timeout.
	newNode.Status = "unknown"
	newNode.LastSeen = nil

	// Save to database
	if err := s.database.CreateNode(newNode); err != nil {
		s.logger.ErrorContext(ctx, "failed to create node in database", "name", req.Name, "error", err)
		return nil, domain.WrapDatabaseOperation("create node", err)
	}

	s.logger.InfoContext(ctx, "node registered successfully", "name", req.Name, "id", newNode.ID)

	go func(nodeID string) {
		checkCtx, cancel := context.WithTimeout(context.Background(), constants.HTTPClientTimeout*2)
		defer cancel()
		if err := s.HealthCheckNode(checkCtx, nodeID); err != nil {
			s.logger.WarnContext(checkCtx, "first health check of new node failed", "nodeID", nodeID, "error", err)
		}
	}(newNode.ID)

	return newNode, nil
}

// GetNode retrieves a node by ID
func (s *nodeService) GetNode(ctx context.Context, nodeID string) (*db.Node, error) {
	s.logger.DebugContext(ctx, "getting node", "nodeID", nodeID)

	node, err := s.database.GetNode(nodeID)
	if err != nil {
		return nil, domain.WrapNodeNotFound(nodeID, err)
	}

	return node, nil
}

// ListNodes retrieves all nodes in the cluster
func (s *nodeService) ListNodes(ctx context.Context) ([]*db.Node, error) {
	s.logger.DebugContext(ctx, "listing all nodes")

	nodes, err := s.database.GetAllNodes()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list nodes", "error", err)
		return nil, domain.WrapDatabaseOperation("list nodes", err)
	}

	return nodes, nil
}

// UpdateNode updates a node's information
func (s *nodeService) UpdateNode(ctx context.Context, nodeID string, req domain.UpdateNodeRequest) (*db.Node, error) {
	s.logger.InfoContext(ctx, "updating node", "nodeID", nodeID)

	node, err := s.database.GetNode(nodeID)
	if err != nil {
		return nil, domain.WrapNodeNotFound(nodeID, err)
	}

	// Update fields
	if req.Name != "" {
		node.Name = req.Name
	}
	if req.APIEndpoint != "" {
		if err := nodepkg.ValidateEndpoint(req.APIEndpoint); err != nil {
			return nil, domain.WrapValidationError("api_endpoint", err)
		}
		node.APIEndpoint = req.APIEndpoint
	}
	if req.APIKey != "" {
		node.APIKey = req.APIKey
	}

	node.UpdatedAt = time.Now()

	if err := s.database.UpdateNode(node); err != nil {
		s.logger.ErrorContext(ctx, "failed to update node", "nodeID", nodeID, "error", err)
		return nil, domain.WrapDatabaseOperation("update node", err)
	}

	s.logger.InfoContext(ctx, "node updated successfully", "nodeID", nodeID)
	return node, nil
}

// DeleteNode removes a node from the cluster
func (s *nodeService) DeleteNode(ctx context.Context, nodeID string, force bool) error {
	s.logger.InfoContext(ctx, "deleting node", "nodeID", nodeID, "force", force)

	node, err := s.database.GetNode(nodeID)
	if err != nil {
		return domain.WrapNodeNotFound(nodeID, err)
	}

	// The machine this server runs on cannot be removed from itself. Any other node can, including a primary
	// record left over from a machine that no longer exists.
	if nodeID == s.config.Node.ID {
		return domain.WrapConflict(domain.CodeCurrentNode, "", fmt.Sprintf("%s is the machine you are connected to, so it cannot be removed from here", node.Name))
	}

	apps, err := s.database.GetAllApps()
	if err != nil {
		s.logger.WarnContext(ctx, "failed to check for apps on node", "nodeID", nodeID, "error", err)
	}
	var onNode []*db.App
	for _, app := range apps {
		if app.NodeID == nodeID {
			onNode = append(onNode, app)
		}
	}

	if len(onNode) > 0 {
		noun := "apps"
		if len(onNode) == 1 {
			noun = "app"
		}
		if !force {
			return domain.WrapConflict(domain.CodeNodeHasApps, "", fmt.Sprintf("%s still has %d %s. Delete them first", node.Name, len(onNode), noun))
		}
		// Forcing is for a node that cannot answer. One that answers can have its apps deleted properly.
		if node.Status == "online" {
			return domain.WrapConflict(domain.CodeNodeOnline, "", fmt.Sprintf("%s is online, so delete its %s the normal way first", node.Name, noun))
		}
		for _, app := range onNode {
			if err := s.database.DeleteApp(app.ID); err != nil {
				return domain.WrapDatabaseOperation("delete app record", err)
			}
			s.logger.WarnContext(ctx, "dropped the record of an app on an unreachable node", "nodeID", nodeID, "appID", app.ID, "app", app.Name)
		}
	}

	if err := s.database.DeleteNode(nodeID); err != nil {
		s.logger.ErrorContext(ctx, "failed to delete node", "nodeID", nodeID, "error", err)
		return domain.WrapDatabaseOperation("delete node", err)
	}

	s.logger.InfoContext(ctx, "node deleted successfully", "nodeID", nodeID)
	return nil
}

// HealthCheckNode performs a health check on a specific node
func (s *nodeService) HealthCheckNode(ctx context.Context, nodeID string) error {
	s.logger.DebugContext(ctx, "health checking node", "nodeID", nodeID)

	node, err := s.database.GetNode(nodeID)
	if err != nil {
		return domain.WrapNodeNotFound(nodeID, err)
	}

	// Perform health check
	latency, err := s.nodeClient.HealthCheckTimed(node)
	now := time.Now()

	if err != nil {
		// Health check failed
		node.ConsecutiveFailures++
		node.LastHealthCheck = &now

		// After 3 consecutive failures, mark as offline
		// After 10 consecutive failures, mark as unreachable (will be checked less frequently)
		oldStatus := node.Status
		if node.ConsecutiveFailures >= 10 {
			node.Status = "unreachable"
		} else if node.ConsecutiveFailures >= 3 {
			node.Status = "offline"
		}
		// A node that has never answered is not merely slow to be marked offline, it is not reachable.
		if oldStatus == "unknown" {
			node.Status = "unreachable"
		}

		s.logger.WarnContext(ctx, "node health check failed",
			"nodeID", nodeID,
			"nodeName", node.Name,
			"consecutive_failures", node.ConsecutiveFailures,
			"old_status", oldStatus,
			"new_status", node.Status,
			"error", err)
	} else {
		// Health check succeeded - reset failure counter
		node.ConsecutiveFailures = 0
		node.Status = "online"
		node.LastSeen = &now
		node.LastHealthCheck = &now
		node.LastLatencyMs = int(latency.Milliseconds())
		s.logger.DebugContext(ctx, "node health check succeeded", "nodeID", nodeID, "latency_ms", node.LastLatencyMs)
	}

	node.UpdatedAt = now

	// Update node status in database
	if dbErr := s.database.UpdateNode(node); dbErr != nil {
		s.logger.ErrorContext(ctx, "failed to update node status", "nodeID", nodeID, "error", dbErr)
	}

	return err
}

// HealthCheckAllNodes performs health checks on all nodes with exponential backoff
func (s *nodeService) HealthCheckAllNodes(ctx context.Context) error {
	nodes, err := s.database.GetAllNodes()
	if err != nil {
		return err
	}

	now := time.Now()

	for _, node := range nodes {
		// Update current node's status as online (it's alive if we're running this)
		if node.ID == s.config.Node.ID {
			node.Status = "online"
			node.LastSeen = &now
			node.LastHealthCheck = &now
			node.ConsecutiveFailures = 0
			node.UpdatedAt = now
			if dbErr := s.database.UpdateNode(node); dbErr != nil {
				s.logger.WarnContext(ctx, "failed to update current node status", "nodeID", node.ID, "error", dbErr)
			}
			continue
		}

		// Implement exponential backoff based on consecutive failures
		// 0 failures (online): check every cycle (30s)
		// 1-2 failures (offline): check every cycle (30s)
		// 3-5 failures (offline->unreachable at 5): check every 2 minutes
		// 6-9 failures (unreachable): check every 5 minutes
		// 10+ failures (unreachable): check every 15 minutes
		shouldCheck := s.shouldCheckNode(node, now)

		if shouldCheck {
			// Perform health check on remote nodes
			// HealthCheckNode updates the database even on failure, so we log but don't fail the entire operation
			if err := s.HealthCheckNode(ctx, node.ID); err != nil {
				// Error is already logged in HealthCheckNode, but we log here too for visibility
				s.logger.DebugContext(ctx, "health check completed with error (status updated in database)",
					"nodeID", node.ID,
					"nodeName", node.Name,
					"error", err)
			}
		} else {
			s.logger.DebugContext(ctx, "skipping health check due to backoff",
				"nodeID", node.ID,
				"nodeName", node.Name,
				"consecutive_failures", node.ConsecutiveFailures,
				"last_health_check", node.LastHealthCheck)
		}
	}

	return nil
}

// shouldCheckNode determines if a node should be checked based on its failure history
func (s *nodeService) shouldCheckNode(node *db.Node, now time.Time) bool {
	// If never checked, always check
	if node.LastHealthCheck == nil {
		return true
	}

	timeSinceLastCheck := now.Sub(*node.LastHealthCheck)

	// Exponential backoff based on consecutive failures
	switch {
	case node.ConsecutiveFailures == 0:
		// Online node - check every cycle
		return true
	case node.ConsecutiveFailures <= 2:
		// Recently failed - check every cycle (30s)
		return true
	case node.ConsecutiveFailures <= 5:
		// Multiple failures - check every 2 minutes
		return timeSinceLastCheck >= constants.NodeHealthCheckIntervalMedium
	case node.ConsecutiveFailures <= 9:
		// Many failures - check every 5 minutes
		return timeSinceLastCheck >= constants.NodeHealthCheckIntervalLong
	default:
		// Persistent failures - check every 15 minutes
		return timeSinceLastCheck >= 15*time.Minute
	}
}

// SyncSettingsFromPrimary fetches settings from primary node and updates local settings
// This is called periodically on secondary nodes
func (s *nodeService) SyncSettingsFromPrimary(ctx context.Context) error {
	// Only secondary nodes should sync settings
	if s.config.Node.IsPrimary {
		return fmt.Errorf("primary node should not sync settings")
	}

	if s.config.Node.PrimaryNodeURL == "" {
		return fmt.Errorf("PRIMARY_NODE_URL not configured")
	}

	s.logger.InfoContext(ctx, "syncing settings from primary node", "primaryURL", s.config.Node.PrimaryNodeURL)

	// Construct primary node info from config
	// Use secondary's own credentials to authenticate with primary
	primaryNode := &db.Node{
		ID:          s.config.Node.ID, // Secondary's ID
		APIEndpoint: s.config.Node.PrimaryNodeURL,
		APIKey:      s.config.Node.APIKey, // Secondary's API key
	}

	// Fetch settings from primary
	settings, err := s.nodeClient.GetSettings(primaryNode)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to fetch settings from primary", "error", err)
		return err
	}

	// Update local settings (only Cloudflare credentials, not auto_start_apps)
	localSettings, err := s.database.GetSettings()
	if err != nil {
		return fmt.Errorf("failed to get local settings: %w", err)
	}

	localSettings.CloudflareAPIToken = settings.CloudflareAPIToken
	localSettings.CloudflareAccountID = settings.CloudflareAccountID
	localSettings.ActiveTunnelProvider = settings.ActiveTunnelProvider
	localSettings.TunnelProviderConfig = settings.TunnelProviderConfig
	localSettings.AutoStartApps = settings.AutoStartApps
	localSettings.UpdatedAt = time.Now()

	if err := s.database.UpdateSettings(localSettings); err != nil {
		s.logger.ErrorContext(ctx, "failed to update local settings", "error", err)
		return domain.WrapDatabaseOperation("update settings", err)
	}

	s.logger.InfoContext(ctx, "settings synced successfully from primary node")
	return nil
}

// GetCurrentNodeInfo returns information about the current node
func (s *nodeService) GetCurrentNodeInfo(ctx context.Context) (*db.Node, error) {
	// Try to find the current node in the database
	nodes, err := s.database.GetAllNodes()
	if err != nil {
		return nil, err
	}

	// Find node matching current config
	for _, node := range nodes {
		if node.Name == s.config.Node.Name || node.ID == s.config.Node.ID {
			return node, nil
		}
	}

	// If not found, return a virtual node from config
	// This can happen on secondary nodes that haven't been registered yet
	now := time.Now()
	return &db.Node{
		ID:          s.config.Node.ID,
		Name:        s.config.Node.Name,
		APIEndpoint: s.config.Node.APIEndpoint,
		APIKey:      s.config.Node.APIKey,
		IsPrimary:   s.config.Node.IsPrimary,
		Status:      "online",
		LastSeen:    &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// NodeHeartbeat handles a heartbeat from a node announcing it's online
// This resets the failure counter and triggers an immediate health check
func (s *nodeService) NodeHeartbeat(ctx context.Context, nodeID string) error {
	s.logger.InfoContext(ctx, "received heartbeat from node", "nodeID", nodeID)

	node, err := s.database.GetNode(nodeID)
	if err != nil {
		return domain.WrapNodeNotFound(nodeID, err)
	}

	// Reset failure counter and mark as online
	now := time.Now()
	node.ConsecutiveFailures = 0
	node.Status = "online"
	node.LastSeen = &now
	node.LastHealthCheck = &now
	node.UpdatedAt = now

	if err := s.database.UpdateNode(node); err != nil {
		s.logger.ErrorContext(ctx, "failed to update node after heartbeat", "nodeID", nodeID, "error", err)
		return err
	}

	s.logger.InfoContext(ctx, "node heartbeat processed successfully", "nodeID", nodeID, "nodeName", node.Name)
	return nil
}

// GetSettings returns the cluster settings a secondary syncs from the primary
func (s *nodeService) GetSettings(_ context.Context) (*db.Settings, error) {
	settings, err := s.database.GetSettings()
	if err != nil {
		return nil, domain.WrapDatabaseOperation("get settings", err)
	}
	return settings, nil
}

// UpdateSettings applies the fields a user set. Provider fields left empty keep their stored value.
func (s *nodeService) UpdateSettings(ctx context.Context, req domain.UpdateSettingsRequest) (*db.Settings, error) {
	settings, err := s.database.GetSettings()
	if err != nil {
		return nil, domain.WrapDatabaseOperation("get settings", err)
	}
	settings.AutoStartApps = req.AutoStartApps
	if req.ActiveTunnelProvider != "" {
		settings.ActiveTunnelProvider = &req.ActiveTunnelProvider
	}
	if req.TunnelProviderConfig != "" {
		settings.TunnelProviderConfig = &req.TunnelProviderConfig
	}
	if err := s.database.UpdateSettings(settings); err != nil {
		return nil, domain.WrapDatabaseOperation("update settings", err)
	}
	s.logger.InfoContext(ctx, "settings updated")
	return settings, nil
}

// AutoRegisterNode authenticates a secondary with the shared registration token or a single-use join
// token and records it. The node is saved as unreachable first, because the health check looks it up in
// the database: checking before the insert reported every node unreachable however reachable it was.
func (s *nodeService) AutoRegisterNode(ctx context.Context, req domain.AutoRegisterRequest) (*domain.AutoRegistration, error) {
	authenticated := constantTimeEqual(req.Token, s.config.Node.RegistrationToken)
	if !authenticated {
		ok, err := s.database.ConsumeJoinToken(req.Token, req.ID)
		if err != nil {
			return nil, domain.WrapDatabaseOperation("check join token", err)
		}
		authenticated = ok
	}
	if !authenticated {
		s.logger.WarnContext(ctx, "node registration rejected: invalid token", "node_name", req.Name)
		return nil, domain.ErrRegistrationUnauthorized
	}
	if err := node.ValidateEndpoint(req.APIEndpoint); err != nil {
		return nil, domain.WrapValidationError("api_endpoint", err)
	}

	if existing, err := s.database.GetNode(req.ID); err == nil && existing != nil {
		// Re-registration (for example after a restart) may refresh the endpoint but must present the key
		// already on record; replacing a node's key needs an explicit update by an admin.
		if !constantTimeEqual(existing.APIKey, req.APIKey) {
			s.logger.WarnContext(ctx, "node re-registration rejected: API key differs from the one on record", "node_id", req.ID)
			return nil, domain.WrapConflict(domain.CodeNodeIDTaken, "id",
				"this node is already registered with a different API key; update its key from the primary instead of re-registering")
		}
		existing.Name = req.Name
		existing.APIEndpoint = req.APIEndpoint
		existing.Status = constants.NodeStatusOnline
		if err := s.database.UpdateNode(existing); err != nil {
			return nil, domain.WrapDatabaseOperation("update existing node", err)
		}
		return &domain.AutoRegistration{Node: existing}, nil
	}
	if other, err := s.database.GetNodeByName(req.Name); err == nil && other != nil {
		return nil, domain.WrapConflict(domain.CodeNodeNameTaken, "name",
			fmt.Sprintf("a node named %s is already registered with a different ID", req.Name))
	}

	n := db.NewNodeWithID(req.ID, req.Name, req.APIEndpoint, req.APIKey, false)
	n.Status = constants.NodeStatusUnreachable
	if err := s.database.CreateNode(n); err != nil {
		return nil, domain.WrapDatabaseOperation("register node", err)
	}
	if err := s.HealthCheckNode(ctx, n.ID); err != nil {
		s.logger.WarnContext(ctx, "health check failed for auto-registered node", "name", req.Name, "error", err)
	} else {
		n.Status = constants.NodeStatusOnline
	}
	s.logger.InfoContext(ctx, "node auto-registered", "id", req.ID, "name", req.Name, "status", n.Status)
	return &domain.AutoRegistration{Node: n, Created: true}, nil
}

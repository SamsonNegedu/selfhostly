package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
	"github.com/selfhostly/internal/node"
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
		return nil, fmt.Errorf("node ID is required for registration")
	}

	// Check if node with this ID already exists
	existingNodeByID, err := s.database.GetNode(req.ID)
	if err == nil && existingNodeByID != nil {
		return nil, fmt.Errorf("node with ID %s already exists", req.ID)
	}

	// Check if node with this name already exists
	existingNode, err := s.database.GetNodeByName(req.Name)
	if err == nil && existingNode != nil {
		return nil, fmt.Errorf("node with name %s already exists", req.Name)
	}

	// Create new node with the provided ID
	newNode := db.NewNodeWithID(req.ID, req.Name, req.APIEndpoint, req.APIKey, false)

	// Perform initial health check
	if err := s.nodeClient.HealthCheck(newNode); err != nil {
		s.logger.WarnContext(ctx, "health check failed for new node", "name", req.Name, "error", err)
		newNode.Status = "unreachable"
	} else {
		newNode.Status = "online"
		now := time.Now()
		newNode.LastSeen = &now
	}

	// Save to database
	if err := s.database.CreateNode(newNode); err != nil {
		s.logger.ErrorContext(ctx, "failed to create node in database", "name", req.Name, "error", err)
		return nil, domain.WrapDatabaseOperation("create node", err)
	}

	s.logger.InfoContext(ctx, "node registered successfully", "name", req.Name, "id", newNode.ID)
	return newNode, nil
}

// GetNode retrieves a node by ID
func (s *nodeService) GetNode(ctx context.Context, nodeID string) (*db.Node, error) {
	s.logger.DebugContext(ctx, "getting node", "nodeID", nodeID)

	node, err := s.database.GetNode(nodeID)
	if err != nil {
		return nil, fmt.Errorf("node not found: %w", err)
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
		return nil, fmt.Errorf("node not found: %w", err)
	}

	// Update fields
	if req.Name != "" {
		node.Name = req.Name
	}
	if req.APIEndpoint != "" {
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
func (s *nodeService) DeleteNode(ctx context.Context, nodeID string) error {
	s.logger.InfoContext(ctx, "deleting node", "nodeID", nodeID)

	node, err := s.database.GetNode(nodeID)
	if err != nil {
		return fmt.Errorf("node not found: %w", err)
	}

	// Prevent deletion of primary node
	if node.IsPrimary {
		return fmt.Errorf("cannot delete primary node")
	}

	// Check if node has apps
	apps, err := s.database.GetAllApps()
	if err != nil {
		s.logger.WarnContext(ctx, "failed to check for apps on node", "nodeID", nodeID, "error", err)
	} else {
		appsOnNode := 0
		for _, app := range apps {
			if app.NodeID == nodeID {
				appsOnNode++
			}
		}
		if appsOnNode > 0 {
			return fmt.Errorf("cannot delete node with %d apps still deployed", appsOnNode)
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
		return fmt.Errorf("node not found: %w", err)
	}

	// Perform health check
	err = s.nodeClient.HealthCheck(node)
	now := time.Now()

	if err != nil {
		// Health check failed
		node.ConsecutiveFailures++
		node.LastHealthCheck = &now
		
		// After 3 consecutive failures, mark as offline
		// After 10 consecutive failures, mark as unreachable (will be checked less frequently)
		if node.ConsecutiveFailures >= 10 {
			node.Status = "unreachable"
		} else if node.ConsecutiveFailures >= 3 {
			node.Status = "offline"
		}
		
		s.logger.WarnContext(ctx, "node health check failed", 
			"nodeID", nodeID, 
			"consecutive_failures", node.ConsecutiveFailures,
			"status", node.Status,
			"error", err)
	} else {
		// Health check succeeded - reset failure counter
		node.ConsecutiveFailures = 0
		node.Status = "online"
		node.LastSeen = &now
		node.LastHealthCheck = &now
		s.logger.DebugContext(ctx, "node health check succeeded", "nodeID", nodeID)
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
		// 0 failures: check every cycle (30s)
		// 1-2 failures: check every cycle (30s)
		// 3-5 failures: check every 2 minutes
		// 6-9 failures: check every 5 minutes
		// 10+ failures: check every 15 minutes
		shouldCheck := s.shouldCheckNode(node, now)
		
		if shouldCheck {
			// Perform health check on remote nodes (ignore individual errors)
			_ = s.HealthCheckNode(ctx, node.ID)
		} else {
			s.logger.DebugContext(ctx, "skipping health check due to backoff", 
				"nodeID", node.ID,
				"nodeName", node.Name,
				"consecutive_failures", node.ConsecutiveFailures)
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
		return timeSinceLastCheck >= 2*time.Minute
	case node.ConsecutiveFailures <= 9:
		// Many failures - check every 5 minutes
		return timeSinceLastCheck >= 5*time.Minute
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

	// Get primary node from database
	primaryNode, err := s.database.GetPrimaryNode()
	if err != nil {
		return fmt.Errorf("failed to get primary node: %w", err)
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

	// Update only Cloudflare settings
	localSettings.CloudflareAPIToken = settings.CloudflareAPIToken
	localSettings.CloudflareAccountID = settings.CloudflareAccountID
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
		return fmt.Errorf("node not found: %w", err)
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

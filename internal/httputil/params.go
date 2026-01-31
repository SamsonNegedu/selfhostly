package httputil

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

// ParseNodeIDs extracts and parses comma-separated node_ids from query parameter
func ParseNodeIDs(c *gin.Context) []string {
	nodeIDsParam := c.Query("node_ids")
	if nodeIDsParam == "" {
		return nil
	}

	// Split comma-separated node IDs
	nodeIDs := strings.Split(nodeIDsParam, ",")
	
	// Trim whitespace from each ID
	for i := range nodeIDs {
		nodeIDs[i] = strings.TrimSpace(nodeIDs[i])
	}

	return nodeIDs
}

// ValidateAndGetAppID validates and returns app ID from URL parameter
func ValidateAndGetAppID(c *gin.Context) (string, error) {
	id := c.Param("id")
	if id == "" {
		return "", fmt.Errorf("invalid app ID")
	}
	return id, nil
}

// ValidateAndGetContainerID validates and returns container ID from URL parameter
func ValidateAndGetContainerID(c *gin.Context) (string, error) {
	id := c.Param("id")
	if id == "" {
		return "", fmt.Errorf("invalid container ID")
	}
	return id, nil
}

// ValidateAndGetVersion validates and returns version number from URL parameter
func ValidateAndGetVersion(c *gin.Context) (int, error) {
	versionParam := c.Param("version")
	if versionParam == "" {
		return 0, fmt.Errorf("invalid version")
	}

	var version int
	if _, err := fmt.Sscanf(versionParam, "%d", &version); err != nil {
		return 0, fmt.Errorf("invalid version number")
	}

	return version, nil
}

// GetNodeIDOrDefault gets node_id from query parameter or returns default
func GetNodeIDOrDefault(c *gin.Context, defaultNodeID string) string {
	nodeID := c.Query("node_id")
	if nodeID == "" {
		return defaultNodeID
	}
	return nodeID
}

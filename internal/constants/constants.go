package constants

import "time"

// App status values
const (
	AppStatusRunning  = "running"
	AppStatusStopped  = "stopped"
	AppStatusUpdating = "updating"
	AppStatusError    = "error"
	AppStatusPending  = "pending" // Used when app creation is queued
)

// Job status values
const (
	JobStatusPending   = "pending"
	JobStatusRunning   = "running"
	JobStatusCompleted = "completed"
	JobStatusFailed    = "failed"
)

// Job type values
const (
	JobTypeAppCreate     = "app_create"
	JobTypeAppUpdate     = "app_update"
	JobTypeTunnelCreate  = "tunnel_create"
	JobTypeTunnelDelete  = "tunnel_delete"
	JobTypeQuickTunnel   = "quick_tunnel"
)

// Tunnel mode values
const (
	TunnelModeCustom = "custom"
	TunnelModeQuick  = "quick"
	TunnelModeNone   = "" // Empty string means no tunnel
)

// Tunnel status values
const (
	TunnelStatusActive   = "active"
	TunnelStatusInactive = "inactive"
	TunnelStatusError    = "error"
	TunnelStatusDeleted  = "deleted"
)

// Node status values
const (
	NodeStatusOnline      = "online"
	NodeStatusOffline     = "offline"
	NodeStatusUnreachable = "unreachable"
)

// Tunnel provider names
const (
	ProviderCloudflare = "cloudflare"
)

// Port constants
const (
	// QuickTunnelMetricsPort is the container port for cloudflared metrics endpoint
	QuickTunnelMetricsPort = 2000

	// QuickTunnelMetricsPortMin is the minimum host port for Quick Tunnel metrics
	QuickTunnelMetricsPortMin = 2000

	// QuickTunnelMetricsPortMax is the maximum host port for Quick Tunnel metrics
	QuickTunnelMetricsPortMax = 2999

	// MaxPort is the maximum valid port number
	MaxPort = 65535

	// MinPort is the minimum valid port number
	MinPort = 1

	// QuickTunnelMetricsEndpointFormat is the format string for metrics endpoint address
	// Use fmt.Sprintf(QuickTunnelMetricsEndpointFormat, port) to build endpoint (e.g., "0.0.0.0:2000")
	QuickTunnelMetricsEndpointFormat = "0.0.0.0:%d"
)

// Timeout and interval constants
const (
	// QuickTunnelURLRetryInterval is the interval between retries when extracting Quick Tunnel URL
	QuickTunnelURLRetryInterval = 2 * time.Second

	// QuickTunnelURLMaxRetries is the maximum number of retries when extracting Quick Tunnel URL
	QuickTunnelURLMaxRetries = 30

	// QuickTunnelStartupDelay is the delay after starting Quick Tunnel before extracting URL
	// Cloudflared needs time to establish connection and expose metrics endpoint
	// Increased to 15s to account for container startup + cloudflare connection establishment
	QuickTunnelStartupDelay = 15 * time.Second

	// HTTPClientTimeout is the timeout for HTTP client requests
	HTTPClientTimeout = 5 * time.Second

	// ServerReadTimeout is the HTTP server read timeout
	ServerReadTimeout = 30 * time.Second

	// ServerWriteTimeout is the HTTP server write timeout
	ServerWriteTimeout = 120 * time.Second

	// ServerIdleTimeout is the HTTP server idle timeout
	ServerIdleTimeout = 120 * time.Second

	// HealthCheckInterval is the interval for periodic health checks
	HealthCheckInterval = 30 * time.Second

	// HeartbeatDelay is the delay before sending heartbeat
	HeartbeatDelay = 2 * time.Second

	// AutoRegistrationDelay is the delay before attempting auto-registration
	AutoRegistrationDelay = 2 * time.Second

	// AutoRegistrationMaxRetries is the maximum number of retries for auto-registration
	AutoRegistrationMaxRetries = 5

	// AutoRegistrationRetryDelay is the delay between auto-registration retries
	AutoRegistrationRetryDelay = 5 * time.Second

	// DatabaseLockTimeout is the timeout when database is locked
	DatabaseLockTimeout = 5 * time.Second
)

// Compose version change reasons
const (
	ComposeVersionReasonInitial       = "Initial version"
	ComposeVersionReasonUpdated       = "Compose file updated"
	ComposeVersionReasonQuickTunnel   = "Quick Tunnel added"
	ComposeVersionReasonTunnelAdded   = "Tunnel added"
	ComposeVersionReasonTunnelRemoved = "Tunnel removed"
)

// URL scheme constants
const (
	URLSchemeHTTPS = "https://"
	URLSchemeHTTP  = "http://"
)

// Docker network constants
const (
	// CoreAPINetwork is the external Docker network used for cross-app communication
	// All tunnel containers join this network to be reachable by the core API backend
	CoreAPINetwork = "selfhostly-network"

	// DockerBridgeGateway is the default Docker bridge gateway IP (Linux)
	DockerBridgeGateway = "172.17.0.1"

	// DockerHostInternal is the host.docker.internal hostname (Mac/Windows)
	DockerHostInternal = "host.docker.internal"
)

// Circuit breaker constants
const (
	// CircuitBreakerFailureThreshold is the number of consecutive failures before opening circuit
	CircuitBreakerFailureThreshold = 5

	// CircuitBreakerHalfOpenSuccesses is the number of successes needed to close circuit from half-open
	CircuitBreakerHalfOpenSuccesses = 2
)

// Node health check constants
const (
	// NodeHealthCheckFailureThreshold marks node as offline after this many consecutive failures
	NodeHealthCheckFailureThreshold = 3

	// NodeHealthCheckUnreachableThreshold marks node as unreachable after this many consecutive failures
	NodeHealthCheckUnreachableThreshold = 5

	// NodeHealthCheckIntervalShort is the interval for nodes with 1-2 failures
	NodeHealthCheckIntervalShort = 30 * time.Second

	// NodeHealthCheckIntervalMedium is the interval for nodes with 3-5 failures
	NodeHealthCheckIntervalMedium = 2 * time.Minute

	// NodeHealthCheckIntervalLong is the interval for nodes with 6+ failures
	NodeHealthCheckIntervalLong = 5 * time.Minute
)

// Backoff constants for retry logic
const (
	// BackoffInitialInterval is the initial interval for exponential backoff
	BackoffInitialInterval = 2 * time.Second

	// BackoffMaxInterval is the maximum interval for exponential backoff
	BackoffMaxInterval = 5 * time.Minute
)

// Job processing constants
const (
	// JobWorkerPollInterval is how often the worker checks for pending jobs
	JobWorkerPollInterval = 2 * time.Second

	// JobStaleThreshold is how long a job can be in "running" state before considered stale
	JobStaleThreshold = 30 * time.Minute

	// JobGracefulShutdownTimeout is how long to wait for current job during shutdown
	JobGracefulShutdownTimeout = 5 * time.Minute

	// JobHistoryKeepCount is how many completed/failed jobs to keep per app
	JobHistoryKeepCount = 20

	// JobHistoryCleanupInterval is how often to clean up old job records
	JobHistoryCleanupInterval = 1 * time.Hour
)

// Default provider name (for backward compatibility)
const DefaultProviderName = ProviderCloudflare

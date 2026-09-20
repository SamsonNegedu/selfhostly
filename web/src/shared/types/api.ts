export interface Node {
    id: string
    name: string
    api_endpoint: string
    // api_key is excluded from API responses for security - never exposed to frontend
    is_primary: boolean
    status: 'online' | 'offline' | 'unreachable' | 'unknown'
    last_seen?: string
    /** When the node was last checked, and how the checks are going. Latency is 0 until one has succeeded. */
    last_health_check?: string | null
    last_latency_ms?: number
    consecutive_failures?: number
    created_at: string
    updated_at: string
}

export interface App {
    id: string
    name: string
    description: string
    compose_content: string
    tunnel_token: string
    tunnel_id: string
    tunnel_domain: string
    public_url: string
    status: 'running' | 'stopped' | 'updating' | 'error' | 'pending'
    error_message: string
    node_id: string
    node_name?: string // For display purposes (added by backend)
    tunnel_mode?: '' | 'custom' | 'quick' // '' = none, custom = named tunnel, quick = trycloudflare.com
    created_at: string
    updated_at: string
    schedule?: AppSchedule // Optional schedule for this app
}

export interface AppSchedule {
    id: string
    app_id: string
    start_cron: string
    stop_cron: string
    timezone: string
    enabled: boolean
    created_at: string
    updated_at: string
}

export interface ScheduleNextRuns {
    app_id: string
    next_start?: string // ISO date string
    next_stop?: string // ISO date string
    /** The next few starts and stops together, earliest first. */
    upcoming?: { action: 'start' | 'stop'; at: string }[]
}

export interface UpdateScheduleRequest {
    start_cron: string
    stop_cron: string
    timezone: string
    enabled: boolean
}

export interface JobLogLine {
    seq: number
    text: string
}

export interface Job {
    id: string
    type: 'app_create' | 'app_update' | 'app_start' | 'tunnel_create' | 'tunnel_delete' | 'quick_tunnel'
    app_id: string
    status: 'pending' | 'running' | 'completed' | 'failed'
    payload?: string
    progress: number
    progress_message?: string
    result?: string
    error_message?: string
    started_at?: string
    completed_at?: string
    created_at: string
    updated_at: string
}

export interface JobResponse {
    job_id: string
    status: 'pending'
    message: string
}

export interface CreateAppRequest {
    name: string
    description: string
    compose_content: string
    ingress_rules?: IngressRule[]
    node_id?: string // Target node for app deployment
    tunnel_mode?: '' | 'custom' | 'quick'
    quick_tunnel_service?: string // Required when tunnel_mode='quick'
    quick_tunnel_port?: number // Required when tunnel_mode='quick'
}

export interface RegisterNodeRequest {
    id: string // Required: Secondary's existing node ID for heartbeat authentication
    name: string
    api_endpoint: string
    api_key: string
}

export interface UpdateAppRequest {
    name?: string
    description?: string
    compose_content?: string
}

export interface Settings {
    id: string
    active_tunnel_provider?: string
    tunnel_provider_config?: string // JSON string with masked tokens
    auto_start_apps: boolean
    updated_at: string
}

export interface IngressRule {
    hostname?: string | null
    service: string
    path?: string | null
    originRequest?: Record<string, any>
}

export interface UpdateSettingsRequest {
    active_tunnel_provider?: string
    tunnel_provider_config?: string
    auto_start_apps?: boolean
}

export interface CloudflareTunnel {
    id: string
    app_id: string
    tunnel_id: string
    tunnel_name: string
    status: 'active' | 'inactive' | 'error' | 'deleted'
    is_active: boolean
    public_url: string
    ingress_rules?: IngressRule[]
    created_at: string
    updated_at: string
    last_synced_at?: string
    error_details?: string
}

/** GET /api/tunnels/apps/:appId - single envelope for primary and secondary; tunnel is null when no named tunnel (e.g. Quick Tunnel or none) */
export interface TunnelByAppResponse {
    tunnel: CloudflareTunnel | null
    app_id: string
    tunnel_mode: string
    node_id: string
    public_url?: string
}

export interface CloudflareTunnelResponse {
    tunnels: CloudflareTunnel[]
    count: number
}

// New provider-agnostic tunnel types
interface TunnelProvider {
    name: string
    display_name: string
    is_configured: boolean
}

export interface ProviderFeatures {
    provider: string
    display_name: string
    is_configured: boolean
    features: {
        ingress: boolean
        dns: boolean
        status_sync: boolean
        container: boolean
        list: boolean
    }
}

export interface TunnelProvidersResponse {
    providers: TunnelProvider[]
    active: string
}

export interface ComposeVersion {
    id: string
    app_id: string
    version: number
    compose_content: string
    change_reason?: string | null
    changed_by?: string | null
    is_current: boolean
    created_at: string
    rolled_back_from?: number | null
}

export interface RollbackRequest {
    change_reason?: string
}

// System monitoring types
export interface SystemStats {
    node_id: string
    node_name: string
    cpu: CPUStats
    memory: MemoryStats
    disk: DiskStats
    docker: DockerStats
    containers: ContainerInfo[]
    timestamp: string
    error?: string // Error message if stats couldn't be fetched
    status: 'online' | 'offline' | 'error' // Node connectivity status
}

interface CPUStats {
    usage_percent: number
    cores: number
}

interface MemoryStats {
    total_bytes: number
    used_bytes: number
    free_bytes: number
    available_bytes: number
    usage_percent: number
}

interface DiskStats {
    total_bytes: number
    used_bytes: number
    free_bytes: number
    usage_percent: number
    path: string
}

interface DockerStats {
    total_containers: number
    running: number
    stopped: number
    paused: number
    images: number
    version: string
}

export interface ContainerInfo {
    id: string
    name: string
    app_name: string
    node_id: string // ID of the node this container is running on
    is_managed: boolean // Whether container belongs to an app managed by our system
    status: string
    state: 'running' | 'stopped' | 'paused'
    cpu_percent: number
    memory_usage_bytes: number
    memory_limit_bytes: number
    network_rx_bytes: number
    network_tx_bytes: number
    block_read_bytes: number
    block_write_bytes: number
    created_at: string
    restart_count: number
}

// UI-driven update types

export type UpdateSettingKind = 'optional' | 'generated' | 'required'

export interface UpdateSettingInfo {
    key: string
    kind: UpdateSettingKind
    description: string
    secret: boolean
}

export interface UpdateAvailable {
    version: string
    published_at: string
    notes: string
    settings: UpdateSettingInfo[]
}

export type UpdatePlanState = 'preparing' | 'ready' | 'blocked' | 'failed'

export type UpdateBlockerCode =
    | 'job_running'
    | 'compose_customized'
    | 'missing_required'
    | 'unsupported_from'
    | 'socket_proxy'
    | 'disk_low'
    | 'doctor_failed'
    | 'not_admin'
    | 'update_in_progress'

export interface UpdateBlocker {
    code: UpdateBlockerCode
    message: string
}

export type UpdateComposeState = 'current' | 'behind' | 'customized'

export interface UpdatePlanCompose {
    state: UpdateComposeState
    diff: string
    // What compose-diff says would change or carry over, in plain text.
    report: string
    approval_token: string
}

export interface UpdateRequiredSetting {
    key: string
    description: string
    secret: boolean
}

export interface UpdatePlanSettings {
    required_missing: UpdateRequiredSetting[]
    generated: string[]
    optional: string[]
}

export interface UpdatePlan {
    version: string
    state: UpdatePlanState
    error: string
    blockers: UpdateBlocker[]
    compose: UpdatePlanCompose
    settings: UpdatePlanSettings
}

export type UpdateRunState = 'pending' | 'running' | 'succeeded' | 'rolled_back' | 'failed' | 'interrupted'

export type UpdatePhase =
    | 'verify'
    | 'rollback_point'
    | 'pull'
    | 'configure'
    | 'dry_start'
    | 'database'
    | 'primary'
    | 'gateway'
    | 'frontend'
    | 'apps'
    | 'done'

export type UpdateStepState = 'pending' | 'running' | 'done' | 'failed'

export interface UpdateStep {
    name: string
    state: UpdateStepState
    message: string
}

export type UpdateRunKind = 'update' | 'rollback'

export interface UpdateRun {
    id: string
    // A status file written before runs had a kind is an update
    kind?: UpdateRunKind
    state: UpdateRunState
    phase: UpdatePhase
    from_version: string
    to_version: string
    message: string
    steps: UpdateStep[]
    warnings: string[]
    started_at: string
    finished_at: string
}

export interface UpdateStatus {
    enabled: boolean
    disabled_reason: string
    current_version: string
    checked_at: string
    check_error: string
    available: UpdateAvailable | null
    plan: UpdatePlan | null
    run: UpdateRun | null
}

export interface PlanUpdateRequest {
    version: string
}

export interface ApplyUpdateRequest {
    version: string
    inputs: Record<string, string>
    // The approval_token of the reviewed compose diff. Left out when the compose file is already current.
    approve_compose?: string
}

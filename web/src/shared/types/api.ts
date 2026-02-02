export interface Node {
  id: string;
  name: string;
  api_endpoint: string;
  // api_key is excluded from API responses for security - never exposed to frontend
  is_primary: boolean;
  status: 'online' | 'offline' | 'unreachable';
  last_seen?: string;
  created_at: string;
  updated_at: string;
}

export interface App {
  id: string;
  name: string;
  description: string;
  compose_content: string;
  tunnel_token: string;
  tunnel_id: string;
  tunnel_domain: string;
  public_url: string;
  status: 'running' | 'stopped' | 'updating' | 'error' | 'pending';
  error_message: string;
  node_id: string;
  node_name?: string; // For display purposes (added by backend)
  tunnel_mode?: '' | 'custom' | 'quick'; // '' = none, custom = named tunnel, quick = trycloudflare.com
  created_at: string;
  updated_at: string;
}

export interface Job {
  id: string;
  type: 'app_create' | 'app_update' | 'tunnel_create' | 'tunnel_delete' | 'quick_tunnel';
  app_id: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  payload?: string;
  progress: number;
  progress_message?: string;
  result?: string;
  error_message?: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface JobResponse {
  job_id: string;
  status: 'pending';
  message: string;
}

export interface CreateAppRequest {
  name: string;
  description: string;
  compose_content: string;
  ingress_rules?: IngressRule[];
  node_id?: string; // Target node for app deployment
  tunnel_mode?: '' | 'custom' | 'quick';
  quick_tunnel_service?: string; // Required when tunnel_mode='quick'
  quick_tunnel_port?: number; // Required when tunnel_mode='quick'
}

export interface RegisterNodeRequest {
  id: string; // Required: Secondary's existing node ID for heartbeat authentication
  name: string;
  api_endpoint: string;
  api_key: string;
}

export interface UpdateNodeRequest {
  name?: string;
  api_endpoint?: string;
  api_key?: string;
}

export interface UpdateAppRequest {
  name?: string;
  description?: string;
  compose_content?: string;
}

export interface Settings {
  id: string;
  active_tunnel_provider?: string;
  tunnel_provider_config?: string; // JSON string with masked tokens
  auto_start_apps: boolean;
  updated_at: string;
}

export interface IngressRule {
  hostname?: string | null;
  service: string;
  path?: string | null;
  originRequest?: Record<string, any>;
}

export interface UpdateSettingsRequest {
  active_tunnel_provider?: string;
  tunnel_provider_config?: string;
  auto_start_apps?: boolean;
}

export interface CloudflareTunnel {
  id: string;
  app_id: string;
  tunnel_id: string;
  tunnel_name: string;
  status: 'active' | 'inactive' | 'error' | 'deleted';
  is_active: boolean;
  public_url: string;
  ingress_rules?: IngressRule[];
  created_at: string;
  updated_at: string;
  last_synced_at?: string;
  error_details?: string;
}

/** GET /api/tunnels/apps/:appId - single envelope for primary and secondary; tunnel is null when no named tunnel (e.g. Quick Tunnel or none) */
export interface TunnelByAppResponse {
  tunnel: CloudflareTunnel | null;
  app_id: string;
  tunnel_mode: string;
  node_id: string;
  public_url?: string;
}

export interface CloudflareTunnelResponse {
  tunnels: CloudflareTunnel[];
  count: number;
}

// New provider-agnostic tunnel types
export interface TunnelProvider {
  name: string;
  display_name: string;
  is_configured: boolean;
}

export interface ProviderFeatures {
  provider: string;
  display_name: string;
  is_configured: boolean;
  features: {
    ingress: boolean;
    dns: boolean;
    status_sync: boolean;
    container: boolean;
    list: boolean;
  };
}

export interface TunnelProvidersResponse {
  providers: TunnelProvider[];
  active: string;
}

// Generic tunnel type (for future use with provider abstraction)
// CloudflareTunnel will eventually be replaced by this
export interface Tunnel {
  id: string;
  app_id: string;
  provider_type: string;
  tunnel_id: string;
  tunnel_name: string;
  public_url: string;
  status: 'active' | 'inactive' | 'error' | 'deleted';
  is_active: boolean;
  ingress_rules?: IngressRule[] | null;
  created_at: string;
  updated_at: string;
  last_synced_at?: string;
  error_details?: string;
}

export interface ApiResponse<T = unknown> {
  data?: T;
  error?: string;
  message?: string;
}

export interface ComposeVersion {
  id: string;
  app_id: string;
  version: number;
  compose_content: string;
  change_reason?: string | null;
  changed_by?: string | null;
  is_current: boolean;
  created_at: string;
  rolled_back_from?: number | null;
}

export interface RollbackRequest {
  change_reason?: string;
}

export interface AppStats {
  app_name: string;
  total_cpu_percent: number;
  total_memory_bytes: number;
  memory_limit_bytes: number;
  containers: ContainerStat[];
  timestamp: string;
  status?: string;
  message?: string;
}

export interface ContainerStat {
  container_id: string;
  container_name: string;
  cpu_percent: number;
  memory_usage_bytes: number;
  memory_limit_bytes: number;
  network_rx_bytes: number;
  network_tx_bytes: number;
  block_read_bytes: number;
  block_write_bytes: number;
}

// System monitoring types
export interface SystemStats {
  node_id: string;
  node_name: string;
  cpu: CPUStats;
  memory: MemoryStats;
  disk: DiskStats;
  docker: DockerStats;
  containers: ContainerInfo[];
  timestamp: string;
  error?: string; // Error message if stats couldn't be fetched
  status: 'online' | 'offline' | 'error'; // Node connectivity status
}

export interface CPUStats {
  usage_percent: number;
  cores: number;
}

export interface MemoryStats {
  total_bytes: number;
  used_bytes: number;
  free_bytes: number;
  available_bytes: number;
  usage_percent: number;
}

export interface DiskStats {
  total_bytes: number;
  used_bytes: number;
  free_bytes: number;
  usage_percent: number;
  path: string;
}

export interface DockerStats {
  total_containers: number;
  running: number;
  stopped: number;
  paused: number;
  images: number;
  version: string;
}

export interface ContainerInfo {
  id: string;
  name: string;
  app_name: string;
  node_id: string; // ID of the node this container is running on
  is_managed: boolean; // Whether container belongs to an app managed by our system
  status: string;
  state: 'running' | 'stopped' | 'paused';
  cpu_percent: number;
  memory_usage_bytes: number;
  memory_limit_bytes: number;
  network_rx_bytes: number;
  network_tx_bytes: number;
  block_read_bytes: number;
  block_write_bytes: number;
  created_at: string;
  restart_count: number;
}

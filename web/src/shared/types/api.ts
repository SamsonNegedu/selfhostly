export interface App {
  id: string;
  name: string;
  description: string;
  compose_content: string;
  tunnel_token: string;
  tunnel_id: string;
  tunnel_domain: string;
  public_url: string;
  status: 'running' | 'stopped' | 'updating' | 'error';
  error_message: string;
  created_at: string;
  updated_at: string;
}

export interface CreateAppRequest {
  name: string;
  description: string;
  compose_content: string;
  ingress_rules?: IngressRule[];
}

export interface UpdateAppRequest {
  name?: string;
  description?: string;
  compose_content?: string;
}

export interface Settings {
  id: string;
  cloudflare_api_token: string;
  cloudflare_account_id: string;
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
  cloudflare_api_token?: string;
  cloudflare_account_id?: string;
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

export interface CloudflareTunnelResponse {
  tunnels: CloudflareTunnel[];
  count: number;
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

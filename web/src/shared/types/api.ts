export interface App {
  id: number;
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
}

export interface UpdateAppRequest {
  name?: string;
  description?: string;
  compose_content?: string;
}

export interface Settings {
  id: number;
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
  id: number;
  app_id: number;
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

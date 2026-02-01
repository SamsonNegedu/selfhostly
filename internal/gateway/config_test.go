package gateway

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Save original env vars
	originalEnv := map[string]string{
		"PRIMARY_BACKEND_URL":     os.Getenv("PRIMARY_BACKEND_URL"),
		"GATEWAY_API_KEY":         os.Getenv("GATEWAY_API_KEY"),
		"GATEWAY_LISTEN_ADDRESS":   os.Getenv("GATEWAY_LISTEN_ADDRESS"),
		"JWT_SECRET":              os.Getenv("JWT_SECRET"),
		"AUTH_ENABLED":            os.Getenv("AUTH_ENABLED"),
		"GATEWAY_REGISTRY_TTL_SEC": os.Getenv("GATEWAY_REGISTRY_TTL_SEC"),
	}

	// Cleanup: restore original env vars
	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	tests := []struct {
		name        string
		env         map[string]string
		wantErr     bool
		wantConfig  *Config
		checkFields func(*testing.T, *Config)
	}{
		{
			name: "all fields set",
			env: map[string]string{
				"PRIMARY_BACKEND_URL":     "http://primary:8082",
				"GATEWAY_API_KEY":         "test-api-key",
				"GATEWAY_LISTEN_ADDRESS":   ":8080",
				"JWT_SECRET":              "test-jwt-secret",
				"AUTH_ENABLED":            "true",
				"GATEWAY_REGISTRY_TTL_SEC": "120",
			},
			wantErr: false,
			checkFields: func(t *testing.T, cfg *Config) {
				if cfg.PrimaryBackendURL != "http://primary:8082" {
					t.Errorf("PrimaryBackendURL = %q, want %q", cfg.PrimaryBackendURL, "http://primary:8082")
				}
				if cfg.GatewayAPIKey != "test-api-key" {
					t.Errorf("GatewayAPIKey = %q, want %q", cfg.GatewayAPIKey, "test-api-key")
				}
				if cfg.ListenAddress != ":8080" {
					t.Errorf("ListenAddress = %q, want %q", cfg.ListenAddress, ":8080")
				}
				if cfg.JWTSecret != "test-jwt-secret" {
					t.Errorf("JWTSecret = %q, want %q", cfg.JWTSecret, "test-jwt-secret")
				}
				if !cfg.AuthEnabled {
					t.Error("AuthEnabled = false, want true")
				}
				if cfg.RegistryTTL != 120*time.Second {
					t.Errorf("RegistryTTL = %v, want %v", cfg.RegistryTTL, 120*time.Second)
				}
			},
		},
		{
			name: "defaults",
			env: map[string]string{
				"GATEWAY_API_KEY": "test-api-key",
			},
			wantErr: false,
			checkFields: func(t *testing.T, cfg *Config) {
				if cfg.PrimaryBackendURL != "http://localhost:8082" {
					t.Errorf("PrimaryBackendURL = %q, want %q", cfg.PrimaryBackendURL, "http://localhost:8082")
				}
				if cfg.ListenAddress != ":8080" {
					t.Errorf("ListenAddress = %q, want %q", cfg.ListenAddress, ":8080")
				}
				if cfg.AuthEnabled {
					t.Error("AuthEnabled = true, want false")
				}
				if cfg.RegistryTTL != 60*time.Second {
					t.Errorf("RegistryTTL = %v, want %v", cfg.RegistryTTL, 60*time.Second)
				}
			},
		},
		{
			name: "missing GATEWAY_API_KEY",
			env:  map[string]string{},
			wantErr: true,
		},
		{
			name: "auth disabled",
			env: map[string]string{
				"GATEWAY_API_KEY": "test-api-key",
				"AUTH_ENABLED":    "false",
			},
			wantErr: false,
			checkFields: func(t *testing.T, cfg *Config) {
				if cfg.AuthEnabled {
					t.Error("AuthEnabled = true, want false")
				}
			},
		},
		{
			name: "invalid TTL defaults to 60",
			env: map[string]string{
				"GATEWAY_API_KEY":         "test-api-key",
				"GATEWAY_REGISTRY_TTL_SEC": "invalid",
			},
			wantErr: false,
			checkFields: func(t *testing.T, cfg *Config) {
				if cfg.RegistryTTL != 60*time.Second {
					t.Errorf("RegistryTTL = %v, want %v", cfg.RegistryTTL, 60*time.Second)
				}
			},
		},
		{
			name: "zero TTL defaults to 60",
			env: map[string]string{
				"GATEWAY_API_KEY":         "test-api-key",
				"GATEWAY_REGISTRY_TTL_SEC": "0",
			},
			wantErr: false,
			checkFields: func(t *testing.T, cfg *Config) {
				if cfg.RegistryTTL != 60*time.Second {
					t.Errorf("RegistryTTL = %v, want %v", cfg.RegistryTTL, 60*time.Second)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars first
			for k := range originalEnv {
				os.Unsetenv(k)
			}

			// Set test env vars
			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			cfg, err := LoadConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if cfg == nil {
				t.Fatal("LoadConfig() returned nil config")
			}

			if tt.checkFields != nil {
				tt.checkFields(t, cfg)
			}
		})
	}
}

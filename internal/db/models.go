package db

import (
	"time"

	"github.com/google/uuid"
)

// App represents a self-hosted application
type App struct {
	ID             string    `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	Description    string    `json:"description" db:"description"`
	ComposeContent string    `json:"compose_content" db:"compose_content"`
	TunnelToken    string    `json:"tunnel_token" db:"tunnel_token"`
	TunnelID       string    `json:"tunnel_id" db:"tunnel_id"`
	TunnelDomain   string    `json:"tunnel_domain" db:"tunnel_domain"`
	PublicURL      string    `json:"public_url" db:"public_url"`
	Status         string    `json:"status" db:"status"`               // running, stopped, updating, error
	ErrorMessage   *string   `json:"error_message" db:"error_message"` // Make nullable to handle NULL values
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// CloudflareTunnel represents Cloudflare tunnel configuration and metadata
type CloudflareTunnel struct {
	ID           string         `json:"id" db:"id"`
	AppID        string         `json:"app_id" db:"app_id"`
	TunnelID     string         `json:"tunnel_id" db:"tunnel_id"`
	TunnelName   string         `json:"tunnel_name" db:"tunnel_name"`
	TunnelToken  string         `json:"tunnel_token" db:"tunnel_token"`
	AccountID    string         `json:"account_id" db:"account_id"`
	IsActive     bool           `json:"is_active" db:"is_active"`
	Status       string         `json:"status" db:"status"`               // active, inactive, error, deleted
	IngressRules *[]IngressRule `json:"ingress_rules" db:"ingress_rules"` // Make nullable to handle NULL values
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
	LastSyncedAt *time.Time     `json:"last_synced_at" db:"last_synced_at"` // Make nullable to handle NULL values
	ErrorDetails *string        `json:"error_details" db:"error_details"`   // Make nullable to handle NULL values
}

// IngressRule represents a single ingress rule for a Cloudflare tunnel
type IngressRule struct {
	Hostname     *string                 `json:"hostname" db:"hostname"`
	Service      string                 `json:"service" db:"service"`
	Path         *string                `json:"path" db:"path"`
	OriginRequest map[string]interface{} `json:"originRequest" db:"originRequest"`
}

// User represents a user for authentication
type User struct {
	ID        string    `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	Password  string    `json:"-" db:"password"` // Never expose password in JSON
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Settings holds application settings
type Settings struct {
	ID                  string    `json:"id" db:"id"`
	CloudflareAPIToken  *string   `json:"cloudflare_api_token" db:"cloudflare_api_token"`
	CloudflareAccountID *string   `json:"cloudflare_account_id" db:"cloudflare_account_id"`
	AutoStartApps       bool      `json:"auto_start_apps" db:"auto_start_apps"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// NewApp creates a new App with a generated UUID
func NewApp(name, description, composeContent string) *App {
	return &App{
		ID:             uuid.New().String(),
		Name:           name,
		Description:    description,
		ComposeContent: composeContent,
		Status:         "stopped",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// NewCloudflareTunnel creates a new CloudflareTunnel with a generated UUID
func NewCloudflareTunnel(appID, tunnelID, tunnelName, tunnelToken, accountID string) *CloudflareTunnel {
	return &CloudflareTunnel{
		ID:          uuid.New().String(),
		AppID:       appID,
		TunnelID:    tunnelID,
		TunnelName:  tunnelName,
		TunnelToken: tunnelToken,
		AccountID:   accountID,
		IsActive:    true,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// NewUser creates a new User with a generated UUID
func NewUser(username, password string) *User {
	return &User{
		ID:        uuid.New().String(),
		Username:  username,
		Password:  password,
		CreatedAt: time.Now(),
	}
}

// NewSettings creates a new Settings with a generated UUID
func NewSettings() *Settings {
	return &Settings{
		ID:            uuid.New().String(),
		AutoStartApps: false,
		UpdatedAt:     time.Now(),
	}
}

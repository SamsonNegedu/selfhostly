package validation

import (
	"strings"
	"testing"
)

func TestValidateAppName(t *testing.T) {
	tests := []struct {
		name      string
		appName   string
		shouldErr bool
	}{
		// Valid names
		{"valid simple name", "myapp", false},
		{"valid with numbers", "my-app-123", false},
		{"valid with underscore", "my_app_test", false},
		{"valid with hyphen", "my-app-name", false},
		{"valid mixed", "App_Name-123", false},
		
		// Invalid names - path traversal
		{"path traversal dots", "../../../etc", true},
		{"path traversal relative", "../../passwd", true},
		{"double dots", "my..app", true},
		{"slash forward", "my/app", true},
		{"slash backward", "my\\app", true},
		
		// Invalid names - reserved
		{"reserved dot", ".", true},
		{"reserved double dot", "..", true},
		{"reserved tilde", "~", true},
		{"reserved tmp", "tmp", true},
		{"reserved temp", "temp", true},
		
		// Invalid names - length
		{"empty", "", true},
		{"too long", strings.Repeat("a", 65), true},
		
		// Invalid names - special chars
		{"starts with hyphen", "-myapp", true},
		{"starts with underscore", "_myapp", true},
		{"ends with hyphen", "myapp-", true},
		{"ends with underscore", "myapp_", true},
		{"special characters", "my@app", true},
		{"spaces", "my app", true},
		{"unicode", "my-app-ü", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAppName(tt.appName)
			if tt.shouldErr && err == nil {
				t.Errorf("expected error but got none for app name: %s", tt.appName)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error for valid app name %s: %v", tt.appName, err)
			}
		})
	}
}

func TestValidateContainerID(t *testing.T) {
	tests := []struct {
		name        string
		containerID string
		shouldErr   bool
	}{
		// Valid IDs
		{"valid short ID", "abc123def456", false},
		{"valid long ID", "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", false},
		{"valid mixed case", "AbCdEf123456", false},
		
		// Invalid IDs
		{"empty", "", true},
		{"too short", "abc123", true},
		{"too long", strings.Repeat("a", 65), true},
		{"non-hex characters", "xyz123abc456", true},
		{"special characters", "abc-123-def", true},
		{"spaces", "abc 123 def", true},
		{"uppercase hex allowed", "ABCDEF123456", false},
		{"with slashes", "abc/123/def", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContainerID(tt.containerID)
			if tt.shouldErr && err == nil {
				t.Errorf("expected error but got none for container ID: %s", tt.containerID)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error for valid container ID %s: %v", tt.containerID, err)
			}
		})
	}
}

func TestValidateComposeContent(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		shouldErr bool
		errMsg    string
	}{
		// Valid content
		{"valid with services", "version: '3'\nservices:\n  web:\n    image: nginx", false, ""},
		{"valid minimal", "services:\n  app:\n    image: test", false, ""},
		{"valid with version", "version: '3.8'\nservices:\n  web:\n    image: nginx", false, ""},
		{
			"valid with defined network",
			"services:\n  web:\n    image: nginx\n    networks:\n      - mynet\nnetworks:\n  mynet:",
			false,
			"",
		},
		{
			"valid with default network",
			"services:\n  web:\n    image: nginx\n    networks:\n      - default",
			false,
			"",
		},
		{
			"valid with host volume",
			"services:\n  web:\n    image: nginx\n    volumes:\n      - ./data:/data",
			false,
			"",
		},
		{
			"valid with named volume",
			"services:\n  web:\n    image: nginx\n    volumes:\n      - mydata:/data\nvolumes:\n  mydata:",
			false,
			"",
		},
		
		// Invalid content - basic
		{"empty", "", true, "cannot be empty"},
		{"too large", strings.Repeat("x", (1<<20)+1), true, "too large"},
		{"no services", "version: '3.8'", true, "no services"},
		
		// Invalid content - undefined networks
		{
			"undefined network",
			"services:\n  web:\n    image: nginx\n    networks:\n      - undefined-network",
			true,
			"undefined network",
		},
		{
			"multiple services undefined network",
			"services:\n  web:\n    image: nginx\n  db:\n    image: postgres\n    networks:\n      - missing-net",
			true,
			"undefined network",
		},
		
		// Invalid YAML
		{
			"invalid yaml syntax",
			"services:\n  web:\n    image: nginx\n    ports:\n      - \"80:80\n      - \"443:443\"",
			true,
			"YAML syntax error",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateComposeContent(tt.content)
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.shouldErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error message to contain %q, but got: %v", tt.errMsg, err)
				}
			}
		})
	}
}

func TestValidateDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
		shouldErr   bool
	}{
		// Valid descriptions
		{"empty allowed", "", false},
		{"short description", "My app", false},
		{"long description", strings.Repeat("a", 500), false},
		
		// Invalid descriptions
		{"too long", strings.Repeat("a", 501), true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDescription(tt.description)
			if tt.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}


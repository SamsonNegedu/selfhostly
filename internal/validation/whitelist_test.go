package validation

import (
	"strings"
	"testing"
)

func TestVolumeWhitelist(t *testing.T) {
	tests := []struct {
		name           string
		composeContent string
		whitelist      []string
		shouldPass     bool
		errorContains  string
	}{
		{
			name: "home path blocked without whitelist",
			composeContent: `
version: '3.8'
services:
  app:
    image: nginx
    volumes:
      - /home/user/data:/data
`,
			whitelist:     []string{},
			shouldPass:    false,
			errorContains: "mounting /home paths is not allowed",
		},
		{
			name: "home path allowed with exact whitelist",
			composeContent: `
version: '3.8'
services:
  app:
    image: nginx
    volumes:
      - /home/user/Documents/apps:/data
`,
			whitelist:  []string{"/home/user/Documents/apps"},
			shouldPass: true,
		},
		{
			name: "home path allowed with parent whitelist",
			composeContent: `
version: '3.8'
services:
  app:
    image: nginx
    volumes:
      - /home/user/Documents/apps/app1:/data
`,
			whitelist:  []string{"/home/user/Documents/apps"},
			shouldPass: true,
		},
		{
			name: "multiple whitelisted paths",
			composeContent: `
version: '3.8'
services:
  app1:
    image: nginx
    volumes:
      - /home/user/Documents/apps/app1:/data
  app2:
    image: postgres
    volumes:
      - /home/user/backup/postgres:/var/lib/postgresql/data
`,
			whitelist:  []string{"/home/user/Documents/apps", "/home/user/backup"},
			shouldPass: true,
		},
		{
			name: "one path whitelisted, one not",
			composeContent: `
version: '3.8'
services:
  app1:
    image: nginx
    volumes:
      - /home/user/Documents/apps/app1:/data
  app2:
    image: postgres
    volumes:
      - /home/user/other/data:/data
`,
			whitelist:     []string{"/home/user/Documents/apps"},
			shouldPass:    false,
			errorContains: "mounting /home paths is not allowed",
		},
		{
			name: "whitelist does not override critical paths",
			composeContent: `
version: '3.8'
services:
  app:
    image: nginx
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
`,
			whitelist:     []string{"/var/run/docker.sock"},
			shouldPass:    false,
			errorContains: "mounting \"/var/run/docker.sock\" is not allowed",
		},
		{
			name: "whitelist with trailing slash",
			composeContent: `
version: '3.8'
services:
  app:
    image: nginx
    volumes:
      - /home/user/Documents/apps/app1:/data
`,
			whitelist:  []string{"/home/user/Documents/apps/"},
			shouldPass: true,
		},
		{
			name: "path traversal attempt blocked",
			composeContent: `
version: '3.8'
services:
  app:
    image: nginx
    volumes:
      - /home/user/Documents/apps/../../../etc:/data
`,
			whitelist:     []string{"/home/user/Documents/apps"},
			shouldPass:    false,
			errorContains: "mounting /home paths is not allowed",
		},
		{
			name: "path traversal to critical path blocked",
			composeContent: `
version: '3.8'
services:
  app:
    image: nginx
    volumes:
      - /opt/myapp/../../../../etc:/data
`,
			whitelist:     []string{"/opt/myapp"},
			shouldPass:    false,
			errorContains: "is not allowed",
		},
		{
			name: "regular safe paths still work without whitelist",
			composeContent: `
version: '3.8'
services:
  app:
    image: nginx
    volumes:
      - /opt/myapp/data:/data
      - /mnt/storage:/storage
      - /data/files:/files
`,
			whitelist:  []string{},
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			securityConfig := &SecurityConfig{
				AllowedVolumePaths: tt.whitelist,
			}

			err := ValidateComposeContentWithConfig(tt.composeContent, securityConfig)

			if tt.shouldPass {
				if err != nil {
					t.Errorf("expected validation to pass, but got error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected validation to fail, but it passed")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain %q, but got: %v", tt.errorContains, err)
				}
			}
		})
	}
}

func TestVolumeWhitelistRealWorldScenario(t *testing.T) {
	// Simulate the user's exact use case
	composeContent := `
version: '3.8'
services:
  finkit:
    image: myapp/finkit:latest
    volumes:
      - /home/user/Documents/opq/finkit/data:/data
      - /home/user/Documents/opq/finkit/config:/config
    ports:
      - "8080:8080"
`

	t.Run("blocked without whitelist", func(t *testing.T) {
		err := ValidateComposeContent(composeContent)
		if err == nil {
			t.Error("expected validation to fail without whitelist")
		}
		if !strings.Contains(err.Error(), "mounting /home paths is not allowed") {
			t.Errorf("expected /home error, got: %v", err)
		}
	})

	t.Run("allowed with whitelist", func(t *testing.T) {
		securityConfig := &SecurityConfig{
			AllowedVolumePaths: []string{"/home/user/Documents/opq"},
		}
		err := ValidateComposeContentWithConfig(composeContent, securityConfig)
		if err != nil {
			t.Errorf("expected validation to pass with whitelist, but got error: %v", err)
		}
	})
}

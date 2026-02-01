package docker

import (
	"reflect"
	"testing"
)

func TestComposeUpCommand(t *testing.T) {
	cmd := ComposeUpCommand()
	expected := []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandUp, ComposeFlagDetached}
	if !reflect.DeepEqual(cmd, expected) {
		t.Errorf("ComposeUpCommand() = %v, want %v", cmd, expected)
	}
}

func TestComposeUpWithBuildCommand(t *testing.T) {
	cmd := ComposeUpWithBuildCommand()
	expected := []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandUp, ComposeFlagDetached, ComposeFlagBuild}
	if !reflect.DeepEqual(cmd, expected) {
		t.Errorf("ComposeUpWithBuildCommand() = %v, want %v", cmd, expected)
	}
}

func TestComposeUpWithRemoveOrphansCommand(t *testing.T) {
	cmd := ComposeUpWithRemoveOrphansCommand()
	expected := []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandUp, ComposeFlagDetached, ComposeFlagRemoveOrphans}
	if !reflect.DeepEqual(cmd, expected) {
		t.Errorf("ComposeUpWithRemoveOrphansCommand() = %v, want %v", cmd, expected)
	}
}

func TestComposeDownCommand(t *testing.T) {
	cmd := ComposeDownCommand()
	expected := []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandDown}
	if !reflect.DeepEqual(cmd, expected) {
		t.Errorf("ComposeDownCommand() = %v, want %v", cmd, expected)
	}
}

func TestComposePullCommand(t *testing.T) {
	cmd := ComposePullCommand()
	expected := []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandPull, ComposeFlagIgnoreBuildable}
	if !reflect.DeepEqual(cmd, expected) {
		t.Errorf("ComposePullCommand() = %v, want %v", cmd, expected)
	}
}

func TestComposePsCommand(t *testing.T) {
	cmd := ComposePsCommand()
	expected := []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandPs}
	if !reflect.DeepEqual(cmd, expected) {
		t.Errorf("ComposePsCommand() = %v, want %v", cmd, expected)
	}
}

func TestComposePsQuietCommand(t *testing.T) {
	cmd := ComposePsQuietCommand()
	expected := []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandPs, "-q"}
	if !reflect.DeepEqual(cmd, expected) {
		t.Errorf("ComposePsQuietCommand() = %v, want %v", cmd, expected)
	}
}

func TestComposeLogsCommand(t *testing.T) {
	tests := []struct {
		name      string
		tailLines int
		want      []string
	}{
		{
			name:      "default tail",
			tailLines: 100,
			want:      []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandLogs, "--tail=100"},
		},
		{
			name:      "custom tail",
			tailLines: 50,
			want:      []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandLogs, "--tail=50"},
		},
		{
			name:      "zero tail",
			tailLines: 0,
			want:      []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandLogs, "--tail=0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := ComposeLogsCommand(tt.tailLines)
			if !reflect.DeepEqual(cmd, tt.want) {
				t.Errorf("ComposeLogsCommand(%d) = %v, want %v", tt.tailLines, cmd, tt.want)
			}
		})
	}
}

func TestComposeRestartServiceCommand(t *testing.T) {
	tests := []struct {
		name    string
		service string
		want    []string
	}{
		{
			name:    "restart tunnel",
			service: ServiceTunnel,
			want:    []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandRestart, ServiceTunnel},
		},
		{
			name:    "restart cloudflared",
			service: ServiceCloudflared,
			want:    []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandRestart, ServiceCloudflared},
		},
		{
			name:    "restart custom service",
			service: "web",
			want:    []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandRestart, "web"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := ComposeRestartServiceCommand(tt.service)
			if !reflect.DeepEqual(cmd, tt.want) {
				t.Errorf("ComposeRestartServiceCommand(%q) = %v, want %v", tt.service, cmd, tt.want)
			}
		})
	}
}

func TestComposeForceRecreateServiceCommand(t *testing.T) {
	tests := []struct {
		name    string
		service string
		want    []string
	}{
		{
			name:    "force recreate tunnel",
			service: ServiceTunnel,
			want:    []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandUp, ComposeFlagDetached, ComposeFlagForceRecreate, ServiceTunnel},
		},
		{
			name:    "force recreate cloudflared",
			service: ServiceCloudflared,
			want:    []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandUp, ComposeFlagDetached, ComposeFlagForceRecreate, ServiceCloudflared},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := ComposeForceRecreateServiceCommand(tt.service)
			if !reflect.DeepEqual(cmd, tt.want) {
				t.Errorf("ComposeForceRecreateServiceCommand(%q) = %v, want %v", tt.service, cmd, tt.want)
			}
		})
	}
}

func TestDockerRestartCommand(t *testing.T) {
	containerID := "abc123def456"
	cmd := DockerRestartCommand(containerID)
	expected := []string{DockerCommand, DockerSubcommandRestart, containerID}
	if !reflect.DeepEqual(cmd, expected) {
		t.Errorf("DockerRestartCommand(%q) = %v, want %v", containerID, cmd, expected)
	}
}

func TestDockerStopCommand(t *testing.T) {
	containerID := "abc123def456"
	cmd := DockerStopCommand(containerID)
	expected := []string{DockerCommand, DockerSubcommandStop, containerID}
	if !reflect.DeepEqual(cmd, expected) {
		t.Errorf("DockerStopCommand(%q) = %v, want %v", containerID, cmd, expected)
	}
}

func TestDockerRmCommand(t *testing.T) {
	containerID := "abc123def456"
	cmd := DockerRmCommand(containerID)
	expected := []string{DockerCommand, DockerSubcommandRm, DockerFlagForce, containerID}
	if !reflect.DeepEqual(cmd, expected) {
		t.Errorf("DockerRmCommand(%q) = %v, want %v", containerID, cmd, expected)
	}
}

func TestComposeCommandBuilder(t *testing.T) {
	tests := []struct {
		name     string
		builder  func() *ComposeCommandBuilder
		expected []string
	}{
		{
			name: "basic command",
			builder: func() *ComposeCommandBuilder {
				return NewComposeCommand(ComposeSubcommandUp)
			},
			expected: []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandUp},
		},
		{
			name: "command with single flag",
			builder: func() *ComposeCommandBuilder {
				return NewComposeCommand(ComposeSubcommandUp).
					WithFlag(ComposeFlagDetached)
			},
			expected: []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandUp, ComposeFlagDetached},
		},
		{
			name: "command with multiple flags",
			builder: func() *ComposeCommandBuilder {
				return NewComposeCommand(ComposeSubcommandUp).
					WithFlag(ComposeFlagDetached).
					WithFlag(ComposeFlagBuild)
			},
			expected: []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandUp, ComposeFlagDetached, ComposeFlagBuild},
		},
		{
			name: "command with service",
			builder: func() *ComposeCommandBuilder {
				return NewComposeCommand(ComposeSubcommandRestart).
					WithService(ServiceTunnel)
			},
			expected: []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandRestart, ServiceTunnel},
		},
		{
			name: "command with flags and service",
			builder: func() *ComposeCommandBuilder {
				return NewComposeCommand(ComposeSubcommandUp).
					WithFlag(ComposeFlagDetached).
					WithFlag(ComposeFlagForceRecreate).
					WithService(ServiceTunnel)
			},
			expected: []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandUp, ComposeFlagDetached, ComposeFlagForceRecreate, ServiceTunnel},
		},
		{
			name: "command with multiple services",
			builder: func() *ComposeCommandBuilder {
				return NewComposeCommand(ComposeSubcommandRestart).
					WithService(ServiceTunnel).
					WithService(ServiceCloudflared)
			},
			expected: []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandRestart, ServiceTunnel, ServiceCloudflared},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.builder().Build()
			if !reflect.DeepEqual(cmd, tt.expected) {
				t.Errorf("Builder.Build() = %v, want %v", cmd, tt.expected)
			}
		})
	}
}

func TestComposeCommandBuilder_Chaining(t *testing.T) {
	// Test that builder methods can be chained
	builder := NewComposeCommand(ComposeSubcommandUp).
		WithFlag(ComposeFlagDetached).
		WithFlag(ComposeFlagBuild).
		WithService("web")

	cmd := builder.Build()
	expected := []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandUp, ComposeFlagDetached, ComposeFlagBuild, "web"}
	if !reflect.DeepEqual(cmd, expected) {
		t.Errorf("Chained builder.Build() = %v, want %v", cmd, expected)
	}
}

func TestComposeCommandBuilder_EmptyCommand(t *testing.T) {
	// Test that an empty command (no flags, no services) still builds correctly
	cmd := NewComposeCommand(ComposeSubcommandPs).Build()
	expected := []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, ComposeSubcommandPs}
	if !reflect.DeepEqual(cmd, expected) {
		t.Errorf("Empty builder.Build() = %v, want %v", cmd, expected)
	}
}

func TestConstants(t *testing.T) {
	// Verify all constants are non-empty
	constants := map[string]string{
		"DockerCommand":              DockerCommand,
		"ComposeCommand":             ComposeCommand,
		"ComposeFileFlag":           ComposeFileFlag,
		"ComposeFileName":           ComposeFileName,
		"ComposeSubcommandUp":       ComposeSubcommandUp,
		"ComposeSubcommandDown":     ComposeSubcommandDown,
		"ComposeSubcommandPull":      ComposeSubcommandPull,
		"ComposeSubcommandRestart":   ComposeSubcommandRestart,
		"ComposeSubcommandPs":        ComposeSubcommandPs,
		"ComposeSubcommandLogs":      ComposeSubcommandLogs,
		"ComposeFlagDetached":       ComposeFlagDetached,
		"ComposeFlagBuild":          ComposeFlagBuild,
		"ComposeFlagRemoveOrphans":   ComposeFlagRemoveOrphans,
		"ComposeFlagForceRecreate":   ComposeFlagForceRecreate,
		"ComposeFlagIgnoreBuildable": ComposeFlagIgnoreBuildable,
		"ComposeFlagTail":           ComposeFlagTail,
		"ServiceTunnel":             ServiceTunnel,
		"ServiceCloudflared":        ServiceCloudflared,
		"DockerSubcommandRestart":   DockerSubcommandRestart,
		"DockerSubcommandStop":      DockerSubcommandStop,
		"DockerSubcommandRm":        DockerSubcommandRm,
		"DockerFlagForce":          DockerFlagForce,
	}

	for name, value := range constants {
		if value == "" {
			t.Errorf("Constant %s is empty", name)
		}
	}
}

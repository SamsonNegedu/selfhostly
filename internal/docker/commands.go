package docker

import "fmt"

// Docker command constants for better discoverability and maintainability

// Docker Compose base command parts
const (
	DockerCommand   = "docker"
	ComposeCommand  = "compose"
	ComposeFileFlag = "-f"
	ComposeFileName = "docker-compose.yml"
)

// Docker Compose subcommands
const (
	ComposeSubcommandUp      = "up"
	ComposeSubcommandDown    = "down"
	ComposeSubcommandPull    = "pull"
	ComposeSubcommandRestart = "restart"
	ComposeSubcommandStop    = "stop"
	ComposeSubcommandRm      = "rm"
	ComposeSubcommandPs      = "ps"
	ComposeSubcommandLogs    = "logs"
)

// Docker Compose flags
const (
	ComposeFlagDetached        = "-d"
	ComposeFlagBuild           = "--build"
	ComposeFlagRemoveOrphans   = "--remove-orphans"
	ComposeFlagForceRecreate   = "--force-recreate"
	ComposeFlagIgnoreBuildable = "--ignore-buildable"
	ComposeFlagTail            = "--tail"
)

// Docker Compose service names
const (
	ServiceTunnel      = "tunnel"
	ServiceCloudflared = "cloudflared"
)

// Docker command parts (for direct docker commands, not compose)
const (
	DockerSubcommandRestart = "restart"
	DockerSubcommandStop    = "stop"
	DockerSubcommandRm      = "rm"
	DockerFlagForce         = "-f"
)

// ComposeCommandBuilder helps build docker compose commands
type ComposeCommandBuilder struct {
	subcommand string
	flags      []string
	services   []string
}

// NewComposeCommand creates a new compose command builder
func NewComposeCommand(subcommand string) *ComposeCommandBuilder {
	return &ComposeCommandBuilder{
		subcommand: subcommand,
		flags:      []string{},
		services:   []string{},
	}
}

// WithFlag adds a flag to the command
func (b *ComposeCommandBuilder) WithFlag(flag string) *ComposeCommandBuilder {
	b.flags = append(b.flags, flag)
	return b
}

// WithService adds a service name to the command
func (b *ComposeCommandBuilder) WithService(service string) *ComposeCommandBuilder {
	b.services = append(b.services, service)
	return b
}

// Build returns the command as a slice of strings ready for ExecuteCommandInDir
func (b *ComposeCommandBuilder) Build() []string {
	cmd := []string{DockerCommand, ComposeCommand, ComposeFileFlag, ComposeFileName, b.subcommand}
	cmd = append(cmd, b.flags...)
	cmd = append(cmd, b.services...)
	return cmd
}

// Helper functions for common compose commands

// ComposeUpCommand returns command for "docker compose -f docker-compose.yml up -d"
func ComposeUpCommand() []string {
	return NewComposeCommand(ComposeSubcommandUp).
		WithFlag(ComposeFlagDetached).
		Build()
}

// ComposeUpWithBuildCommand returns command for "docker compose -f docker-compose.yml up -d --build"
func ComposeUpWithBuildCommand() []string {
	return NewComposeCommand(ComposeSubcommandUp).
		WithFlag(ComposeFlagDetached).
		WithFlag(ComposeFlagBuild).
		Build()
}

// ComposeUpWithRemoveOrphansCommand returns command for "docker compose -f docker-compose.yml up -d --remove-orphans"
func ComposeUpWithRemoveOrphansCommand() []string {
	return NewComposeCommand(ComposeSubcommandUp).
		WithFlag(ComposeFlagDetached).
		WithFlag(ComposeFlagRemoveOrphans).
		Build()
}

// ComposeDownCommand returns command for "docker compose -f docker-compose.yml down"
func ComposeDownCommand() []string {
	return NewComposeCommand(ComposeSubcommandDown).Build()
}

// ComposePullCommand returns command for "docker compose -f docker-compose.yml pull --ignore-buildable"
func ComposePullCommand() []string {
	return NewComposeCommand(ComposeSubcommandPull).
		WithFlag(ComposeFlagIgnoreBuildable).
		Build()
}

// ComposePsCommand returns command for "docker compose -f docker-compose.yml ps"
func ComposePsCommand() []string {
	return NewComposeCommand(ComposeSubcommandPs).Build()
}

// ComposePsQuietCommand returns command for "docker compose -f docker-compose.yml ps -q"
func ComposePsQuietCommand() []string {
	return NewComposeCommand(ComposeSubcommandPs).
		WithFlag("-q").
		Build()
}

// ComposeLogsCommand returns command for "docker compose -f docker-compose.yml logs --tail=100"
func ComposeLogsCommand(tailLines int) []string {
	return NewComposeCommand(ComposeSubcommandLogs).
		WithFlag(ComposeFlagTail + "=" + fmt.Sprintf("%d", tailLines)).
		Build()
}

// ComposeRestartServiceCommand returns command for "docker compose -f docker-compose.yml restart <service>"
func ComposeRestartServiceCommand(service string) []string {
	return NewComposeCommand(ComposeSubcommandRestart).
		WithService(service).
		Build()
}

// ComposeStopServiceCommand returns command for "docker compose -f docker-compose.yml stop <service>"
func ComposeStopServiceCommand(service string) []string {
	return NewComposeCommand(ComposeSubcommandStop).
		WithService(service).
		Build()
}

// ComposeRemoveServiceCommand returns command for "docker compose -f docker-compose.yml rm -f -s <service>"
// -f: Don't ask for confirmation
// -s: Stop the container if it's running before removing
func ComposeRemoveServiceCommand(service string) []string {
	return NewComposeCommand(ComposeSubcommandRm).
		WithFlag("-f").
		WithFlag("-s").
		WithService(service).
		Build()
}

// ComposeForceRecreateServiceCommand returns command for "docker compose -f docker-compose.yml up -d --force-recreate <service>"
func ComposeForceRecreateServiceCommand(service string) []string {
	return NewComposeCommand(ComposeSubcommandUp).
		WithFlag(ComposeFlagDetached).
		WithFlag(ComposeFlagForceRecreate).
		WithService(service).
		Build()
}

// Direct Docker commands (not compose)

// DockerRestartCommand returns command for "docker restart <containerID>"
func DockerRestartCommand(containerID string) []string {
	return []string{DockerCommand, DockerSubcommandRestart, containerID}
}

// DockerStopCommand returns command for "docker stop <containerID>"
func DockerStopCommand(containerID string) []string {
	return []string{DockerCommand, DockerSubcommandStop, containerID}
}

// DockerRmCommand returns command for "docker rm -f <containerID>"
func DockerRmCommand(containerID string) []string {
	return []string{DockerCommand, DockerSubcommandRm, DockerFlagForce, containerID}
}

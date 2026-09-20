package ctl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const (
	lockName      = ".lock"
	lockOwnerFile = "owner"
	ownerPID      = "pid"
	ownerBox      = "container"
	dockerEnvFile = "/.dockerenv"
)

// lockOwner names who holds the upgrade lock, so a lock left by a process that died can be told from a live one.
// A process id only means something on the machine that wrote it, so the host name travels with it.
func lockOwner() string {
	host, _ := os.Hostname()
	if _, err := os.Stat(dockerEnvFile); err == nil {
		return ownerBox + ":" + host
	}
	return fmt.Sprintf("%s:%d@%s", ownerPID, os.Getpid(), host)
}

// acquireLock takes the upgrade lock. A lock whose owner is provably gone is removed; anything uncertain stays.
func (a *App) acquireLock(ctx context.Context, lock string) error {
	for attempt := 0; attempt < 2; attempt++ {
		if err := os.Mkdir(lock, 0o700); err == nil {
			return os.WriteFile(filepath.Join(lock, lockOwnerFile), []byte(lockOwner()+"\n"), 0o600)
		}
		if attempt == 0 && a.lockIsStale(ctx, lock) {
			a.say("removing a stale upgrade lock left by a run that stopped")
			_ = os.RemoveAll(lock)
			continue
		}
	}
	return fmt.Errorf("another upgrade is running (remove %s if it is stale)", lock)
}

func (a *App) lockIsStale(ctx context.Context, lock string) bool {
	b, err := os.ReadFile(filepath.Join(lock, lockOwnerFile))
	if err != nil {
		return false // an older lock without an owner: never guess
	}
	kind, rest, _ := strings.Cut(strings.TrimSpace(string(b)), ":")
	switch kind {
	case ownerBox:
		out, _, err := a.Run.Output(ctx, "docker", "inspect", "--format", "{{.State.Running}}", rest)
		return err != nil || strings.TrimSpace(out) != "true"
	case ownerPID:
		pidText, host, _ := strings.Cut(rest, "@")
		if here, _ := os.Hostname(); host != here {
			return false
		}
		pid, err := strconv.Atoi(pidText)
		if err != nil {
			return false
		}
		proc, err := os.FindProcess(pid)
		if err != nil {
			return true
		}
		return proc.Signal(syscall.Signal(0)) != nil
	}
	return false
}

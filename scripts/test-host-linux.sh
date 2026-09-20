#!/usr/bin/env bash
# Runs the host-check tests against real Linux (unix sockets, file ownership, /proc fixtures) from any
# machine with Docker: builds the test binary for Linux and runs it in a clean Ubuntu container.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
T="$(mktemp -d)"; trap 'rm -rf "$T"' EXIT
arch="$(uname -m)"; case "$arch" in x86_64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; esac
(cd "$ROOT" && GOOS=linux GOARCH="$arch" CGO_ENABLED=0 go test -c -o "$T/hostcheck.test" ./internal/ctl/hostcheck)
docker run --rm -v "$T":/t:ro ubuntu:24.04 /t/hostcheck.test -test.v 2>&1 | grep -E "^(--- |PASS|FAIL|ok)" 

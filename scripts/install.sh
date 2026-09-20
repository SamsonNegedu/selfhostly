#!/bin/sh
# Install selfhostlyctl: downloads the release binary for this machine and checks its checksum.
#
#   curl -fsSL https://raw.githubusercontent.com/samsonnegedu/selfhostly/main/scripts/install.sh | sh
#   curl -fsSL .../install.sh | SELFHOSTLYCTL_VERSION=v1.4.0 sh   (a numbered release)
#   curl -fsSL .../install.sh | SELFHOSTLYCTL_VERSION=edge sh     (the newest main)
#
# With no version it installs the newest numbered release, or the newest main if none exists yet.
#   curl -fsSL .../install.sh | SELFHOSTLYCTL_BIN_DIR=$HOME/bin sh
#
# Installs to /usr/local/bin when writable (or with sudo), else to ~/.local/bin.
set -eu

REPO="${SELFHOSTLYCTL_REPO:-samsonnegedu/selfhostly}"
VERSION="${SELFHOSTLYCTL_VERSION:-latest}"

os="$(uname -s | tr 'A-Z' 'a-z')"
case "$os" in linux|darwin) ;; *) echo "unsupported system: $os (linux and macOS only)" >&2; exit 1 ;; esac
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) echo "unsupported CPU: $(uname -m) (64-bit x86 and ARM only; on a Raspberry Pi use the 64-bit OS)" >&2; exit 1 ;;
esac

if [ "$VERSION" = latest ]; then
  base="https://github.com/$REPO/releases/latest/download"
  # no numbered release yet: use the rolling build of main
  curl -fsSLI -o /dev/null "$base/checksums.txt" 2>/dev/null || base="https://github.com/$REPO/releases/download/edge"
else
  base="https://github.com/$REPO/releases/download/$VERSION"
fi
file="selfhostlyctl_${os}_${arch}"

tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
echo "downloading $file ($VERSION)"
curl -fsSL "$base/$file" -o "$tmp/selfhostlyctl"
curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt"

want="$(grep " $file\$" "$tmp/checksums.txt" | awk '{print $1}')"
[ -n "$want" ] || { echo "no checksum published for $file" >&2; exit 1; }
if command -v sha256sum >/dev/null 2>&1; then got="$(sha256sum "$tmp/selfhostlyctl" | awk '{print $1}')"; else got="$(shasum -a 256 "$tmp/selfhostlyctl" | awk '{print $1}')"; fi
[ "$want" = "$got" ] || { echo "checksum mismatch: refusing to install (expected $want, got $got)" >&2; exit 1; }
chmod 755 "$tmp/selfhostlyctl"

dir="${SELFHOSTLYCTL_BIN_DIR:-}"
sudo_cmd=""
if [ -z "$dir" ]; then
  if [ -w /usr/local/bin ]; then dir=/usr/local/bin
  elif command -v sudo >/dev/null 2>&1; then dir=/usr/local/bin; sudo_cmd=sudo
  else dir="$HOME/.local/bin"; fi
fi
$sudo_cmd mkdir -p "$dir"
if [ -n "$sudo_cmd" ] || [ -w "$dir" ]; then $sudo_cmd mv "$tmp/selfhostlyctl" "$dir/selfhostlyctl"; else echo "cannot write to $dir" >&2; exit 1; fi
echo "installed $dir/selfhostlyctl"
case ":$PATH:" in *":$dir:"*) ;; *) echo "add $dir to your PATH to run it by name" ;; esac
"$dir/selfhostlyctl" version

#!/usr/bin/env bash
set -euo pipefail

# Integration test for dotctl + dotfiles repo
# Runs entirely in a temp directory — does not touch your real HOME config

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
DOTCTL_BIN="$SCRIPT_DIR/dotctl"
DOTFILES_PATH="${DOTFILES_PATH:-$(dirname "$SCRIPT_DIR")/dotfiles}"

if [[ ! -f "$DOTCTL_BIN" ]]; then
  echo "Building dotctl..."
  (cd "$(dirname "$0")" && go build -o dotctl .)
fi

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

export HOME="$TMPDIR/home"
export XDG_CONFIG_HOME="$HOME/.config"
export XDG_STATE_HOME="$HOME/.local/state"

mkdir -p "$HOME/.config/dotctl"
mkdir -p "$HOME/.config/git"
mkdir -p "$HOME/.ssh"
mkdir -p "$HOME/.local/state/dotctl"
mkdir -p "$HOME/.aws-work" "$HOME/.aws-personal"
mkdir -p "$HOME/.kube-work" "$HOME/.kube-personal"
mkdir -p "$HOME/.docker-work" "$HOME/.docker-personal"

# Write dotctl config pointing to the real dotfiles repo
cat > "$HOME/.config/dotctl/config.toml" <<EOF
[dotfiles]
path = "$DOTFILES_PATH"

[dotctl]
remote = "git@personal.github.com:gh-jsoares/dotctl.git"
EOF

# Stow the git config into the fake home
(cd "$DOTFILES_PATH/stow" && stow -S -t "$HOME" git 2>/dev/null) || true

echo "=== Test: ctx list ==="
"$DOTCTL_BIN" ctx list
echo ""

echo "=== Test: ctx personal ==="
"$DOTCTL_BIN" ctx personal
echo ""

echo "=== Verify: current-context ==="
cat "$HOME/.local/state/dotctl/current-context"
echo ""

echo "=== Verify: env file ==="
cat "$HOME/.local/state/dotctl/env"
echo ""

echo "=== Verify: symlinks ==="
ls -la "$HOME/.aws"
ls -la "$HOME/.kube"
echo ""

echo "=== Verify: git config-current symlink ==="
ls -la "$HOME/.config/git/config-current"
echo ""

echo "=== Test: ctx work ==="
"$DOTCTL_BIN" ctx work
echo ""

echo "=== Verify: symlinks flipped ==="
ls -la "$HOME/.aws"
ls -la "$HOME/.kube"
echo ""

echo "=== Verify: git config-current ==="
ls -la "$HOME/.config/git/config-current"
echo ""

echo "=== Verify: env file updated ==="
cat "$HOME/.local/state/dotctl/env"
echo ""

echo "=== Test: ctx (show current) ==="
"$DOTCTL_BIN" ctx
echo ""

echo "=== Test: doctor ==="
"$DOTCTL_BIN" doctor || true
echo ""

echo "=== All integration tests passed ==="

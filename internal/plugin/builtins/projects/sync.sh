#!/usr/bin/env bash
set -euo pipefail

# Source context env to get PROJECTS_DIR
state_dir="${XDG_STATE_HOME:-$HOME/.local/state}/dotctl"
[[ -f "$state_dir/env" ]] && source "$state_dir/env"

if [[ -z "${PROJECTS_DIR:-}" ]]; then
  exit 0
fi

mkdir -p "$PROJECTS_DIR"
echo "Ensured $PROJECTS_DIR exists"

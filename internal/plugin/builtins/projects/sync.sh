#!/usr/bin/env bash
set -euo pipefail

contexts_dir="${DOTCTL_DOTFILES_PATH}/contexts"
if [[ ! -d "$contexts_dir" ]]; then
  exit 0
fi

for toml in "$contexts_dir"/*.toml; do
  [[ -f "$toml" ]] || continue
  ctx=$(basename "$toml" .toml)
  dir=$(grep -E '^PROJECTS_DIR\s*=' "$toml" | sed 's/.*=\s*"//;s/"//' | sed "s|^~/|$HOME/|")
  [[ -z "$dir" ]] && continue
  if [[ ! -d "$dir" ]]; then
    mkdir -p "$dir"
    echo "Created $dir ($ctx)"
  else
    echo "OK $dir ($ctx)"
  fi
done

#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${PROJECTS_DIR:-}" ]]; then
  exit 0
fi

mkdir -p "$PROJECTS_DIR"
echo "Ensured $PROJECTS_DIR exists"

#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
cd "$REPO_ROOT"

echo "Recording demo..."
vhs demo.tape

echo "Pushing to assets branch..."
TMPDIR=$(mktemp -d)
git clone --no-checkout --depth 1 "$(git remote get-url origin)" "$TMPDIR"
cd "$TMPDIR"
git checkout --orphan assets
git rm -rf . 2>/dev/null || true
cp "$REPO_ROOT/demo.gif" demo.gif
git add demo.gif
git commit -m "update demo gif"
git push origin assets --force
cd "$REPO_ROOT"
rm -rf "$TMPDIR" demo.gif

echo "Done — gif live at https://raw.githubusercontent.com/gh-jsoares/dotctl/assets/demo.gif"

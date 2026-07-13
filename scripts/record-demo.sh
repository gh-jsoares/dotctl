#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
cd "$REPO_ROOT"

echo "Recording demo..."
vhs demo.tape

echo "Pushing to assets branch..."
cp demo.gif /tmp/demo.gif
git stash -u
git fetch origin assets 2>/dev/null || true
git checkout --orphan assets-tmp
git rm -rf .
cp /tmp/demo.gif demo.gif
git add demo.gif
git commit -m "update demo gif"
git branch -M assets-tmp assets
git push origin assets --force
git checkout main
git stash pop 2>/dev/null || true
rm -f /tmp/demo.gif

echo "Done — gif live at https://raw.githubusercontent.com/gh-jsoares/dotctl/assets/demo.gif"

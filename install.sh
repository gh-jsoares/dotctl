#!/usr/bin/env bash
set -euo pipefail

REPO_OWNER="gh-jsoares"
REPO_NAME="dotctl"
INSTALL_DIR="${DOTCTL_INSTALL_DIR:-/usr/local/bin}"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

ASSET_NAME="dotctl_${OS}_${ARCH}"

echo "Detecting latest release..."
RELEASE_URL="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
DOWNLOAD_URL=$(curl -sSf "$RELEASE_URL" | grep "browser_download_url.*${ASSET_NAME}" | cut -d '"' -f 4)

if [ -z "$DOWNLOAD_URL" ]; then
  echo "Error: Could not find release asset for ${OS}/${ARCH}" >&2
  echo "Check: https://github.com/${REPO_OWNER}/${REPO_NAME}/releases" >&2
  exit 1
fi

echo "Downloading dotctl for ${OS}/${ARCH}..."
TMP=$(mktemp)
curl -sSfL -o "$TMP" "$DOWNLOAD_URL"
chmod +x "$TMP"

echo "Installing to ${INSTALL_DIR}/dotctl..."
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP" "${INSTALL_DIR}/dotctl"
else
  sudo mv "$TMP" "${INSTALL_DIR}/dotctl"
fi

echo "dotctl installed successfully."
echo "Run 'dotctl bootstrap' to set up your machine."

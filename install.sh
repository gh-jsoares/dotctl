#!/usr/bin/env bash
set -euo pipefail

REPO_OWNER="gh-jsoares"
REPO_NAME="dotctl"
INSTALL_DIR="${DOTCTL_INSTALL_DIR:-/usr/local/bin}"
MAN_DIR="${DOTCTL_MAN_DIR:-/usr/local/share/man/man1}"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

ASSET_NAME="dotctl_${OS}_${ARCH}.tar.gz"

echo "Detecting latest release..."
RELEASE_URL="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
DOWNLOAD_URL=$(curl -sSf "$RELEASE_URL" | grep "browser_download_url.*${ASSET_NAME}" | cut -d '"' -f 4)

if [ -z "$DOWNLOAD_URL" ]; then
  echo "Error: Could not find release asset for ${OS}/${ARCH}" >&2
  echo "Check: https://github.com/${REPO_OWNER}/${REPO_NAME}/releases" >&2
  exit 1
fi

echo "Downloading dotctl for ${OS}/${ARCH}..."
TMP_DIR=$(mktemp -d)
curl -sSfL -o "${TMP_DIR}/dotctl.tar.gz" "$DOWNLOAD_URL"
tar xzf "${TMP_DIR}/dotctl.tar.gz" -C "$TMP_DIR"

BINARY="${TMP_DIR}/dotctl_${OS}_${ARCH}"
chmod +x "$BINARY"

echo "Installing to ${INSTALL_DIR}/dotctl..."
if [ -w "$INSTALL_DIR" ]; then
  mv "$BINARY" "${INSTALL_DIR}/dotctl"
else
  sudo mv "$BINARY" "${INSTALL_DIR}/dotctl"
fi

if [ -d "${TMP_DIR}/man" ]; then
  echo "Installing man pages to ${MAN_DIR}..."
  if [ -w "$(dirname "$MAN_DIR")" ]; then
    mkdir -p "$MAN_DIR"
    cp "${TMP_DIR}"/man/*.1 "$MAN_DIR/"
  else
    sudo mkdir -p "$MAN_DIR"
    sudo cp "${TMP_DIR}"/man/*.1 "$MAN_DIR/"
  fi
fi

rm -rf "$TMP_DIR"

echo "dotctl installed successfully."
echo "Run 'dotctl bootstrap' to set up your machine."

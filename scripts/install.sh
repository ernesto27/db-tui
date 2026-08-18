#!/usr/bin/env bash

set -euo pipefail

REPO="ernesto27/db-tui"
BINARY="db-tui"
INSTALL_DIR="${HOME}/.local/bin"

# --- OS check ---
OS="$(uname -s)"
if [ "$OS" != "Linux" ]; then
	echo "error: only Linux is supported at this time (detected: $OS)" >&2
	exit 1
fi

# --- Architecture check ---
ARCH="$(uname -m)"
case "$ARCH" in
	x86_64) ASSET="db-tui_linux_amd64.tar.gz" ;;
	*)
		echo "error: unsupported architecture '$ARCH' (only x86_64 is available)" >&2
		exit 1
		;;
esac

# --- Downloader ---
if command -v curl >/dev/null 2>&1; then
	DOWNLOAD="curl -fsSL"
elif command -v wget >/dev/null 2>&1; then
	DOWNLOAD="wget -qO-"
else
	echo "error: curl or wget is required" >&2
	exit 1
fi

# --- Resolve the latest release asset through the GitHub API ---
API_URL="https://api.github.com/repos/${REPO}/releases/latest"
echo "Resolving latest release for ${REPO} ..."

DOWNLOAD_URL="$($DOWNLOAD "$API_URL" \
	| grep -oE "\"browser_download_url\"[[:space:]]*:[[:space:]]*\"[^\"]*${ASSET}\"" \
	| head -n1 \
	| sed -E 's/.*"(https:[^"]+)".*/\1/')"

if [ -z "$DOWNLOAD_URL" ]; then
	echo "error: could not find asset '${ASSET}' in the latest release of ${REPO}" >&2
	exit 1
fi

# --- Download and extract ---
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading ${BINARY} from ${DOWNLOAD_URL} ..."
$DOWNLOAD "$DOWNLOAD_URL" | tar -xz -C "$TMP_DIR"

# --- Install ---
mkdir -p "$INSTALL_DIR"
install -m 755 "$TMP_DIR/${BINARY}" "$INSTALL_DIR/${BINARY}"

echo
echo "${BINARY} installed to ${INSTALL_DIR}/${BINARY}"

if ! echo ":${PATH}:" | grep -q ":${INSTALL_DIR}:"; then
	echo
	echo "Add this to your shell profile:"
	echo "export PATH=\"\$HOME/.local/bin:\$PATH\""
fi

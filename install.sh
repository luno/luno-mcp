#!/usr/bin/env sh
set -e

REPO="luno/luno-mcp"
BINARY="luno-mcp"
DEFAULT_INSTALL_DIR="/usr/local/bin"
CLAUDE_DESKTOP_CONFIG="${HOME}/Library/Application Support/Claude/claude_desktop_config.json"

# --- Detect platform ---

OS="$(uname -s)"
case "$OS" in
  Darwin) OS="darwin" ;;
  Linux)  OS="linux" ;;
  *)      echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)          ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
  *)               echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# --- Resolve version ---

if [ -z "$VERSION" ]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' \
    | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
  if [ -z "$VERSION" ]; then
    echo "error: could not determine latest version" >&2
    exit 1
  fi
fi
VERSION="${VERSION#v}"

# --- Download and verify ---

TARBALL="${BINARY}-${OS}-${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/v${VERSION}"

echo "Installing ${BINARY} v${VERSION} (${OS}/${ARCH})..."

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

curl -fsSL "${BASE_URL}/${TARBALL}"      -o "${TMP}/${TARBALL}"
curl -fsSL "${BASE_URL}/checksums.txt"   -o "${TMP}/checksums.txt"

# Verify checksum using whichever tool is available
cd "$TMP"
if command -v sha256sum >/dev/null 2>&1; then
  grep "${TARBALL}" checksums.txt | sha256sum -c --quiet
elif command -v shasum >/dev/null 2>&1; then
  grep "${TARBALL}" checksums.txt | shasum -a 256 -c --quiet
else
  echo "warning: no sha256 tool found, skipping checksum verification" >&2
fi

tar xzf "${TARBALL}"

# --- Install binary ---

INSTALL_DIR="$DEFAULT_INSTALL_DIR"

if [ -w "$INSTALL_DIR" ]; then
  mv "${BINARY}" "${INSTALL_DIR}/${BINARY}"
elif command -v sudo >/dev/null 2>&1; then
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mv "${BINARY}" "${INSTALL_DIR}/${BINARY}"
  sudo chmod +x "${INSTALL_DIR}/${BINARY}"
else
  INSTALL_DIR="${HOME}/.local/bin"
  mkdir -p "$INSTALL_DIR"
  mv "${BINARY}" "${INSTALL_DIR}/${BINARY}"
  echo "note: installed to ${INSTALL_DIR} — add it to your PATH if not already present"
fi
chmod +x "${INSTALL_DIR}/${BINARY}" 2>/dev/null || true

echo "${BINARY} v${VERSION} installed to ${INSTALL_DIR}/${BINARY}"

# --- Configure Claude Desktop (macOS only, if API keys are provided) ---

if [ "$OS" = "darwin" ] && [ -n "$LUNO_API_KEY_ID" ] && [ -n "$LUNO_API_SECRET" ]; then
  if [ -f "$CLAUDE_DESKTOP_CONFIG" ]; then
    # Merge into existing config using Python if available, otherwise append a warning
    if command -v python3 >/dev/null 2>&1; then
      python3 - <<PYEOF
import json, os
p = os.path.expanduser("$CLAUDE_DESKTOP_CONFIG")
with open(p) as f:
    c = json.load(f)
c.setdefault("mcpServers", {})["luno"] = {
    "command": "$BINARY",
    "args": ["--transport", "stdio"],
    "env": {
        "LUNO_API_KEY_ID": "$LUNO_API_KEY_ID",
        "LUNO_API_SECRET": "$LUNO_API_SECRET"
    }
}
with open(p, "w") as f:
    json.dump(c, f, indent=2)
    f.write("\n")
PYEOF
      echo "Claude Desktop configured."
    else
      echo "warning: python3 not found — Claude Desktop config not updated automatically" >&2
    fi
  else
    mkdir -p "$(dirname "$CLAUDE_DESKTOP_CONFIG")"
    cat > "$CLAUDE_DESKTOP_CONFIG" <<EOF
{
  "mcpServers": {
    "luno": {
      "command": "${BINARY}",
      "args": ["--transport", "stdio"],
      "env": {
        "LUNO_API_KEY_ID": "${LUNO_API_KEY_ID}",
        "LUNO_API_SECRET": "${LUNO_API_SECRET}"
      }
    }
  }
}
EOF
    echo "Claude Desktop config created."
  fi
  echo "Restart Claude Desktop to apply changes."
fi

# --- Print next steps if keys were not provided ---

if [ -z "$LUNO_API_KEY_ID" ] || [ -z "$LUNO_API_SECRET" ]; then
  echo ""
  echo "To configure Claude Desktop, re-run with your Luno API credentials:"
  echo ""
  echo "  LUNO_API_KEY_ID=<key> LUNO_API_SECRET=<secret> \\"
  echo "    curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh | sh"
  echo ""
  echo "Get API keys from: https://www.luno.com/wallet/security/api"
fi

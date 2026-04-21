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
  VERSION="$(curl --proto '=https' --tlsv1.2 -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
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

curl --proto '=https' --tlsv1.2 -fsSL "${BASE_URL}/${TARBALL}"    -o "${TMP}/${TARBALL}"
curl --proto '=https' --tlsv1.2 -fsSL "${BASE_URL}/checksums.txt" -o "${TMP}/checksums.txt"

# Verify checksum — macOS ships BSD sha256sum which lacks -c; use shasum there
cd "$TMP"
if [ "$OS" = "darwin" ] && command -v shasum >/dev/null 2>&1; then
  grep "${TARBALL}" checksums.txt | shasum -a 256 -c --quiet
elif command -v sha256sum >/dev/null 2>&1; then
  grep "${TARBALL}" checksums.txt | sha256sum -c --quiet
elif command -v shasum >/dev/null 2>&1; then
  grep "${TARBALL}" checksums.txt | shasum -a 256 -c --quiet
else
  echo "error: no SHA-256 verification tool found; refusing to install unverified download" >&2
  exit 1
fi

tar xzf "${TARBALL}"

# --- Install binary ---

INSTALL_DIR="$DEFAULT_INSTALL_DIR"

if [ -w "$INSTALL_DIR" ]; then
  mv "${BINARY}" "${INSTALL_DIR}/${BINARY}"
elif command -v sudo >/dev/null 2>&1 && sudo -v; then
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mkdir -p "$INSTALL_DIR"
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
  if command -v python3 >/dev/null 2>&1; then
    mkdir -p "$(dirname "$CLAUDE_DESKTOP_CONFIG")"
    CLAUDE_DESKTOP_CONFIG="$CLAUDE_DESKTOP_CONFIG" \
    LUNO_API_KEY_ID="$LUNO_API_KEY_ID" \
    LUNO_API_SECRET="$LUNO_API_SECRET" \
    LUNO_API_DOMAIN="${LUNO_API_DOMAIN:-}" \
    ALLOW_WRITE_OPERATIONS="${ALLOW_WRITE_OPERATIONS:-}" \
    INSTALL_PATH="${INSTALL_DIR}/${BINARY}" \
    python3 - <<'PYEOF'
import json, os

p = os.environ["CLAUDE_DESKTOP_CONFIG"]
try:
    with open(p) as f:
        c = json.load(f)
except FileNotFoundError:
    c = {}
except json.JSONDecodeError as e:
    raise SystemExit(f"error: {p} contains invalid JSON ({e}); fix it before re-running")

if not isinstance(c, dict):
    raise SystemExit(f"error: {p} must contain a JSON object at the top level")

env = {
    "LUNO_API_KEY_ID": os.environ["LUNO_API_KEY_ID"],
    "LUNO_API_SECRET": os.environ["LUNO_API_SECRET"],
}
if os.environ.get("LUNO_API_DOMAIN"):
    env["LUNO_API_DOMAIN"] = os.environ["LUNO_API_DOMAIN"]
if os.environ.get("ALLOW_WRITE_OPERATIONS"):
    env["ALLOW_WRITE_OPERATIONS"] = os.environ["ALLOW_WRITE_OPERATIONS"]

mcp_servers = c.setdefault("mcpServers", {})
if not isinstance(mcp_servers, dict):
    raise SystemExit(f"error: mcpServers in {p} must be a JSON object")

mcp_servers["luno"] = {
    "command": os.environ["INSTALL_PATH"],
    "args": ["--transport", "stdio"],
    "env": env,
}

fd = os.open(p, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
with os.fdopen(fd, "w") as f:
    json.dump(c, f, indent=2)
    f.write("\n")
os.chmod(p, 0o600)
PYEOF
    echo "Claude Desktop configured. Restart Claude Desktop to apply changes."
  else
    echo "error: python3 is required to configure Claude Desktop but was not found" >&2
    echo "Install Python 3 from https://www.python.org and re-run." >&2
    exit 1
  fi
fi

# --- Print next steps if keys were not provided ---

if [ -z "$LUNO_API_KEY_ID" ] || [ -z "$LUNO_API_SECRET" ]; then
  echo ""
  echo "To configure Claude Desktop, re-run with your Luno API credentials:"
  echo ""
  echo "  curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/${REPO}/main/claude-desktop-install.sh | \\"
  echo "    LUNO_API_KEY_ID=<key> LUNO_API_SECRET=<secret> sh"
  echo ""
  echo "Get API keys from: https://www.luno.com/wallet/security/api"
fi

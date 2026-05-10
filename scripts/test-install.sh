#!/usr/bin/env sh
# Integration test for claude-desktop-install.sh.
# Builds the binary, creates a fake release archive, runs the install script
# with a stub curl, verifies the result, then cleans up everything it touched.

set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
INSTALL_SCRIPT="$REPO_ROOT/claude-desktop-install.sh"

# --- Build ---

echo "==> Building binary..."
make -C "$REPO_ROOT" build

# --- Platform detection (must match install script logic) ---

OS="$(uname -s)"
case "$OS" in
  Darwin) OS="darwin" ;;
  Linux)  OS="linux" ;;
  *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)          ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
  *) echo "Unsupported arch: $ARCH" >&2; exit 1 ;;
esac

BINARY="luno-mcp"
TARBALL="${BINARY}-${OS}-${ARCH}.tar.gz"

# --- Expected install location (mirrors installer logic) ---
# On CI runners /usr/local/bin is often writable, so the installer goes there
# instead of falling back to ~/.local/bin. Detect this upfront so verification
# and cleanup both agree on where the binary should land.

if [ -w "/usr/local/bin" ]; then
  EXPECTED_INSTALL_PATH="/usr/local/bin/$BINARY"
  USED_FALLBACK=false
else
  USED_FALLBACK=true
fi

# --- Temp dirs ---

TMP_RELEASE="$(mktemp -d)"
TMP_HOME="$(mktemp -d)"
TMP_BIN="$(mktemp -d)"

if [ "$USED_FALLBACK" = "true" ]; then
  EXPECTED_INSTALL_PATH="$TMP_HOME/.local/bin/$BINARY"
fi

cleanup() {
  rm -rf "$TMP_RELEASE" "$TMP_HOME" "$TMP_BIN"
  # Remove system binary if the installer wrote it there (common on CI runners)
  [ "$USED_FALLBACK" = "false" ] && rm -f "/usr/local/bin/$BINARY" || true
}
trap cleanup EXIT

# --- Fake release ---

echo "==> Creating fake release archive..."
cp "$REPO_ROOT/$BINARY" "$TMP_RELEASE/$BINARY"
(cd "$TMP_RELEASE" && tar czf "$TARBALL" "$BINARY" && rm "$BINARY")
if command -v sha256sum >/dev/null 2>&1; then
  (cd "$TMP_RELEASE" && sha256sum "$TARBALL" > checksums.txt)
elif command -v shasum >/dev/null 2>&1; then
  (cd "$TMP_RELEASE" && shasum -a 256 "$TARBALL" > checksums.txt)
else
  echo "error: no sha256 tool found" >&2; exit 1
fi

# --- Stub curl ---
# Parses -o OUTPUT and https://... URL from the argument list, then serves
# the matching file from TMP_RELEASE instead of hitting GitHub.

cat > "$TMP_BIN/curl" << STUB
#!/bin/sh
prev=""
url=""
output=""
for arg; do
  case "\$prev" in
    -o) output="\$arg" ;;
  esac
  case "\$arg" in
    https://*) url="\$arg" ;;
  esac
  prev="\$arg"
done
filename="\$(basename "\$url")"
src="${TMP_RELEASE}/\$filename"
if [ ! -f "\$src" ]; then
  echo "stub curl: file not found: \$src" >&2
  exit 1
fi
cp "\$src" "\$output"
STUB
chmod +x "$TMP_BIN/curl"

# --- Run ---

echo "==> Running install script..."
PATH="$TMP_BIN:$PATH" \
HOME="$TMP_HOME" \
SHELL="/bin/sh" \
VERSION="0.0.0-test" \
  sh "$INSTALL_SCRIPT"

# --- Verify ---

if [ ! -x "$EXPECTED_INSTALL_PATH" ]; then
  echo "FAIL: binary not installed at $EXPECTED_INSTALL_PATH" >&2
  exit 1
fi
echo "PASS: binary installed at $EXPECTED_INSTALL_PATH"

if [ "$USED_FALLBACK" = "true" ]; then
  PROFILE="$TMP_HOME/.profile"
  if ! grep -qF ".local/bin" "$PROFILE" 2>/dev/null; then
    echo "FAIL: PATH not added to $PROFILE" >&2
    exit 1
  fi
  echo "PASS: PATH updated in $PROFILE"
fi

# --- Cleanup handled by trap ---

echo "==> All checks passed."

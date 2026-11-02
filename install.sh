#!/usr/bin/env bash
# BramandScript Smart Universal Installer
# Compatible with Android (Termux), Linux, macOS, and WSL/MSYS
# Usage: curl -fsSL https://raw.githubusercontent.com/Codeleafly/bramandScript/master/install.sh | bash

set -e

REPO="Codeleafly/bramandScript"
COLOR_RESET="\033[0m"
COLOR_BOLD="\033[1m"
COLOR_CYAN="\033[1;36m"
COLOR_GREEN="\033[1;32m"
COLOR_YELLOW="\033[1;33m"
COLOR_RED="\033[1;31m"
COLOR_DIM="\033[2m"

info() {
  printf "${COLOR_CYAN}✦ [INFO]${COLOR_RESET} %s\n" "$1"
}

success() {
  printf "${COLOR_GREEN}✔ [OK]${COLOR_RESET} %s\n" "$1"
}

warn() {
  printf "${COLOR_YELLOW}▲ [WARN]${COLOR_RESET} %s\n" "$1"
}

error() {
  printf "${COLOR_RED}✖ [ERROR]${COLOR_RESET} %s\n" "$1" >&2
  exit 1
}

# 1. Detect Operating System & Environment
OS_RAW="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH_RAW="$(uname -m | tr '[:upper:]' '[:lower:]')"

IS_TERMUX=false
if [ -n "$PREFIX" ] && [ -d "$PREFIX" ] && echo "$PREFIX" | grep -q "com.termux"; then
  IS_TERMUX=true
fi

case "$OS_RAW" in
  linux*)
    if [ "$IS_TERMUX" = true ]; then
      OS="android"
    else
      OS="linux"
    fi
    ;;
  darwin*)
    OS="darwin"
    ;;
  msys*|mingw*|cygwin*)
    OS="windows"
    ;;
  *)
    error "Unsupported operating system: $OS_RAW"
    ;;
esac

# 2. Detect Architecture
case "$ARCH_RAW" in
  x86_64|amd64)
    ARCH="amd64"
    ;;
  aarch64|arm64)
    ARCH="arm64"
    ;;
  armv7l|armv7)
    ARCH="armv7"
    ;;
  *)
    error "Unsupported CPU architecture: $ARCH_RAW"
    ;;
esac

info "Detected environment: ${OS}-${ARCH}"

# 3. Choose Install Location
if [ "$IS_TERMUX" = true ]; then
  INSTALL_DIR="${PREFIX}/bin"
elif [ "$EUID" -eq 0 ]; then
  INSTALL_DIR="/usr/local/bin"
else
  if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
  fi
fi

mkdir -p "$INSTALL_DIR"

# 4. Check If Building from Local Source or Downloading Release
BUILD_FROM_SOURCE=false
if [ -f "./main.go" ] && [ -f "./go.mod" ] && command -v go >/dev/null 2>&1; then
  info "Local source tree and Go compiler found. Compiling native binary..."
  BUILD_FROM_SOURCE=true
fi

if [ "$BUILD_FROM_SOURCE" = true ]; then
  go build -ldflags="-s -w" -o "bramand" main.go
  rm -f "${INSTALL_DIR}/bramand" "${INSTALL_DIR}/bmd"
  cp bramand "${INSTALL_DIR}/bramand"
  chmod +x "${INSTALL_DIR}/bramand"
  ln -sf "${INSTALL_DIR}/bramand" "${INSTALL_DIR}/bmd"

  if [ -f "tools/task.js" ]; then
    chmod +x tools/task.js
    ln -sf "$(pwd)/tools/task.js" "${INSTALL_DIR}/bmd-task"
  fi
else
  # 5. Fetch Latest Release Asset from GitHub
  info "Fetching latest release information from GitHub (${REPO})..."

  LATEST_JSON="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null || true)"
  TAG="$(echo "$LATEST_JSON" | grep '"tag_name":' | head -n1 | sed -E 's/.*"tag_name": "([^"]+)".*/\1/' || true)"

  if [ -z "$TAG" ]; then
    TAG="v0.1.0"
    warn "Could not determine latest release tag via API (rate limit or no releases yet). Falling back to tag ${TAG}."
  fi

  ARCHIVE_NAME="bramand-${OS}-${ARCH}-${TAG}.tar.gz"
  if [ "$OS" = "windows" ]; then
    ARCHIVE_NAME="bramand-${OS}-${ARCH}-${TAG}.zip"
  fi

  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE_NAME}"
  info "Downloading binary archive: ${DOWNLOAD_URL}..."

  TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t 'bmd_install')"
  trap 'rm -rf "$TMP_DIR"' EXIT

  HTTP_STATUS="$(curl -SL --write-out "%{http_code}" -o "${TMP_DIR}/${ARCHIVE_NAME}" "$DOWNLOAD_URL" 2>/dev/null || true)"

  if [ "$HTTP_STATUS" != "200" ] && [ "$HTTP_STATUS" != "302" ]; then
    warn "Release asset not found on GitHub releases (HTTP $HTTP_STATUS)."
    if command -v go >/dev/null 2>&1; then
      info "Go toolchain detected! Cloning and compiling from master..."
      git clone "https://github.com/${REPO}.git" "${TMP_DIR}/src"
      (cd "${TMP_DIR}/src" && go build -ldflags="-s -w" -o bramand main.go)
      cp "${TMP_DIR}/src/bramand" "${INSTALL_DIR}/bramand"
    else
      error "Could not download pre-built binary and 'go' is not installed. Please install Go or check release assets at https://github.com/${REPO}/releases"
    fi
  else
    cd "$TMP_DIR"
    if [ "$OS" = "windows" ]; then
      unzip -q "$ARCHIVE_NAME"
    else
      tar -xzf "$ARCHIVE_NAME"
    fi

    rm -f "${INSTALL_DIR}/bramand" "${INSTALL_DIR}/bmd"
    cp bramand* "${INSTALL_DIR}/bramand"
  fi

  chmod +x "${INSTALL_DIR}/bramand"
  ln -sf "${INSTALL_DIR}/bramand" "${INSTALL_DIR}/bmd"
fi

# 6. Verify Installation
if ! command -v bmd >/dev/null 2>&1 && ! [ -x "${INSTALL_DIR}/bmd" ]; then
  warn "Binary installed to ${INSTALL_DIR}, but ${INSTALL_DIR} is not in your current PATH."
  echo "     Add it to your shell configuration:"
  echo "     export PATH=\"${INSTALL_DIR}:\$PATH\""
fi

success "BramandScript successfully installed to ${INSTALL_DIR}!"
printf "\n"
printf "${COLOR_BOLD}Commands Available:${COLOR_RESET}\n"
printf "  ${COLOR_CYAN}bramand${COLOR_RESET} / ${COLOR_CYAN}bmd${COLOR_RESET}       Interactive REPL & CLI\n"
printf "  ${COLOR_CYAN}bmd run <file>${COLOR_RESET}   Run source script\n"
printf "  ${COLOR_CYAN}bmd --help${COLOR_RESET}       Full command help\n\n"

if [ -x "${INSTALL_DIR}/bmd" ]; then
  "${INSTALL_DIR}/bmd" version || true
fi

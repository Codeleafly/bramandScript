#!/usr/bin/env bash
# BramandScript Smart Universal Installer
# Compatible with Android (Termux), Linux, macOS, and WSL/MSYS
# Usage: curl -fsSL https://raw.githubusercontent.com/Codeleafly/bramandScript/main/install.sh | bash

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

BINARY_NAME="bramand"
if [ "$OS" = "windows" ]; then
  BINARY_NAME="bramand.exe"
fi

# 4. Determine Installation Mode (Pre-built Release Binary vs Source Build)
BUILD_FROM_SOURCE=false
if [ "$1" = "--build" ] || [ "$1" = "-b" ]; then
  info "Source build requested via CLI argument..."
  BUILD_FROM_SOURCE=true
fi

if [ "$BUILD_FROM_SOURCE" = true ]; then
  if command -v go >/dev/null 2>&1; then
    info "Go compiler found. Compiling native binary..."
    go build -ldflags="-s -w" -o "bramand" main.go
    cat bramand > "${INSTALL_DIR}/${BINARY_NAME}.new"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}.new"
    mv -f "${INSTALL_DIR}/${BINARY_NAME}.new" "${INSTALL_DIR}/${BINARY_NAME}"
    ln -sf "${INSTALL_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/bmd"

    if [ -f "tools/task.js" ]; then
      chmod +x tools/task.js
      ln -sf "$(pwd)/tools/task.js" "${INSTALL_DIR}/bmd-task"
    fi
  else
    error "Go compiler not found to build from source."
  fi
else
  # 5. Fetch Latest Pre-built Release Binary from GitHub
  info "Fetching latest release information from GitHub (${REPO})..."

  LATEST_JSON="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null || true)"
  TAG="$(echo "$LATEST_JSON" | grep '"tag_name":' | head -n1 | sed -E 's/.*"tag_name": "([^"]+)".*/\1/' || true)"

  if [ -z "$TAG" ]; then
    TAG="v0.1.1"
    warn "Could not determine latest release tag via GitHub API. Falling back to tag ${TAG}."
  fi

  ARCHIVE_NAME="bramand-${OS}-${ARCH}-${TAG}.tar.gz"
  if [ "$OS" = "windows" ]; then
    ARCHIVE_NAME="bramand-${OS}-${ARCH}-${TAG}.zip"
  fi

  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE_NAME}"
  info "Downloading pre-compiled server binary: ${DOWNLOAD_URL}..."

  TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t 'bmd_install')"
  trap 'rm -rf "$TMP_DIR"' EXIT

  HTTP_STATUS="$(curl -sSL --write-out "%{http_code}" -o "${TMP_DIR}/${ARCHIVE_NAME}" "$DOWNLOAD_URL" 2>/dev/null || true)"

  if [ "$HTTP_STATUS" != "200" ] && [ "$HTTP_STATUS" != "302" ]; then
    warn "Release asset ${ARCHIVE_NAME} returned HTTP $HTTP_STATUS from GitHub."
    if command -v go >/dev/null 2>&1 && [ -f "./main.go" ]; then
      info "Falling back to local compilation with Go..."
      go build -ldflags="-s -w" -o "bramand" main.go
      cat bramand > "${INSTALL_DIR}/${BINARY_NAME}.new"
      chmod +x "${INSTALL_DIR}/${BINARY_NAME}.new"
      mv -f "${INSTALL_DIR}/${BINARY_NAME}.new" "${INSTALL_DIR}/${BINARY_NAME}"
      ln -sf "${INSTALL_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/bmd"
    else
      error "Could not download pre-built release binary for ${OS}-${ARCH} and local Go compiler is unavailable. Please check https://github.com/${REPO}/releases"
    fi
  else
    mkdir -p "${TMP_DIR}/extracted"
    if [ "$OS" = "windows" ]; then
      unzip -q "${TMP_DIR}/${ARCHIVE_NAME}" -d "${TMP_DIR}/extracted"
    else
      tar -xzf "${TMP_DIR}/${ARCHIVE_NAME}" -C "${TMP_DIR}/extracted"
    fi

    # Install binary atomically
    cat "${TMP_DIR}/extracted/${BINARY_NAME}" > "${INSTALL_DIR}/${BINARY_NAME}.new"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}.new"
    mv -f "${INSTALL_DIR}/${BINARY_NAME}.new" "${INSTALL_DIR}/${BINARY_NAME}"
    ln -sf "${INSTALL_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/bmd"

    if [ -d "${TMP_DIR}/extracted/stdlib" ]; then
      mkdir -p "${HOME}/.bramand/stdlib"
      cp -r "${TMP_DIR}/extracted/stdlib/"* "${HOME}/.bramand/stdlib/" 2>/dev/null || true
    fi
  fi
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

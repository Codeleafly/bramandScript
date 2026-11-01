#!/bin/bash
set -e

echo "[INFO] Building BramandScript native binary..."
go build -o bramand main.go

INSTALL_DIR="${PREFIX:-/usr/local}/bin"
echo "[INFO] Installing to ${INSTALL_DIR}..."

mkdir -p "${INSTALL_DIR}"
cp bramand "${INSTALL_DIR}/bramand"
chmod +x "${INSTALL_DIR}/bramand"
ln -sf "${INSTALL_DIR}/bramand" "${INSTALL_DIR}/bmd"

echo "[OK] BramandScript successfully installed globally!"
echo "     Available commands: 'bramand' or 'bmd'"

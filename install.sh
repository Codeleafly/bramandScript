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

chmod +x tools/task.js
ln -sf "$(pwd)/tools/task.js" "${INSTALL_DIR}/bmd-task"

echo "[OK] BramandScript successfully installed globally!"
echo "     Available commands: 'bramand', 'bmd', and 'bmd-task'"

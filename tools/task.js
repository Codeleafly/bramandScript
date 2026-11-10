#!/bin/sh
# BramandScript Native Task Runner Wrapper
REPO_DIR="/data/data/com.termux/files/home/bramandScript"
exec "$REPO_DIR/bramand" run --allow-all "$REPO_DIR/tools/task.bmd" "$@"

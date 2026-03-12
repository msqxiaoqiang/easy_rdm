#!/bin/bash
# Wails dev watcher 脚本：启动 Vite 开发服务器
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
WEB_DIR="$SCRIPT_DIR/../../web"
cd "$WEB_DIR" && VITE_PLATFORM=desktop npx vite --host --port 5173

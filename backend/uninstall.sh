#!/bin/bash
APP_DIR="$(cd "$(dirname "$0")" && pwd)"
APP_BIN="$APP_DIR/app/bin/main"

# 停止应用服务
echo "正在停止应用服务..."
PID=$(pgrep -f "$APP_BIN" 2>/dev/null)
if [ -n "$PID" ]; then
    kill "$PID" 2>/dev/null
    sleep 2
    kill -9 "$PID" 2>/dev/null || true
    echo "服务已停止 (PID: $PID)"
fi

# 清理运行时文件
rm -f "$APP_DIR/tmp/app.sock"
rm -f "$APP_DIR/tmp/app.port"

# 根据 IsClean 环境变量决定是否清除数据
if [ "$IsClean" = "true" ]; then
    echo "清除应用数据..."
    rm -rf "$APP_DIR/config/"
    rm -rf "$APP_DIR/logs/"
    rm -rf "$APP_DIR/data/"
else
    echo "保留应用数据"
fi

echo "应用卸载完成"

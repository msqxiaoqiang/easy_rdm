#!/bin/bash
set -e
cd "$(dirname "$0")"

# 安装依赖
if [ ! -d "node_modules" ]; then
    npm install
fi

# 构建
npm run build

echo "前端构建完成: dist/"

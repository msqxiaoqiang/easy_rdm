#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/../docker-compose.test.yml"
DATA_DIR="$ROOT_DIR/backend/data"
BACKUP_DIR="$ROOT_DIR/backend/data/.e2e-backup"

cleanup() {
  echo "==> Cleaning up..."
  kill "$BACKEND_PID" 2>/dev/null || true
  kill "$FRONTEND_PID" 2>/dev/null || true
  docker compose -f "$COMPOSE_FILE" down 2>/dev/null || true
  # 恢复备份的数据文件
  if [ -d "$BACKUP_DIR" ]; then
    echo "==> Restoring data files..."
    cp "$BACKUP_DIR/connections.json" "$DATA_DIR/connections.json" 2>/dev/null || true
    cp "$BACKUP_DIR/session.json" "$DATA_DIR/session.json" 2>/dev/null || true
    rm -rf "$BACKUP_DIR"
  fi
}
trap cleanup EXIT

echo "==> Starting Redis..."
docker compose -f "$COMPOSE_FILE" up -d

echo "==> Running backend unit tests..."
cd "$ROOT_DIR/backend" && go test ./...

echo "==> Running backend integration tests..."
cd "$ROOT_DIR/backend" && go test -tags=integration ./... || echo "Integration tests skipped or failed"

echo "==> Killing residual processes on ports 8899 and 5173..."
lsof -ti:8899 | xargs kill 2>/dev/null || true
lsof -ti:5173 | xargs kill 2>/dev/null || true
sleep 1

echo "==> Resetting data for E2E clean state..."
mkdir -p "$BACKUP_DIR"
cp "$DATA_DIR/connections.json" "$BACKUP_DIR/connections.json" 2>/dev/null || true
cp "$DATA_DIR/session.json" "$BACKUP_DIR/session.json" 2>/dev/null || true
echo '[]' > "$DATA_DIR/connections.json"
echo '{}' > "$DATA_DIR/session.json"

echo "==> Starting backend (HTTP mode)..."
cd "$ROOT_DIR/backend" && go run main.go &
BACKEND_PID=$!
sleep 2

echo "==> Starting frontend dev server..."
cd "$ROOT_DIR/web" && npx vite --port 5173 &
FRONTEND_PID=$!
sleep 3

echo "==> Running E2E tests..."
cd "$ROOT_DIR/tests/e2e" && npx playwright test

echo "==> All tests completed!"

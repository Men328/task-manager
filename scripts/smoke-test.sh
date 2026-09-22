#!/usr/bin/env bash
# Smoke test end-to-end: build 3 service (mỗi service gồm gRPC server + gateway),
# chạy ở port mặc định, gọi API thật, rồi tắt.
# Yêu cầu: go, node >= 20 (dùng fetch), curl. Port 8081/8082/8083/9081/9082/9083 phải đang trống.
set -euo pipefail

cd "$(dirname "$0")/.."

BIN_DIR="${TMPDIR:-/tmp}/task-manager-smoke"
mkdir -p "$BIN_DIR"

log() { printf '\033[36m==>\033[0m %s\n' "$*"; }

log "build service"
go build -o "$BIN_DIR/identity-grpc" ./service/identity/cmd/grpc
go build -o "$BIN_DIR/identity-http" ./service/identity/cmd/http
go build -o "$BIN_DIR/task-grpc" ./service/task/cmd/grpc
go build -o "$BIN_DIR/task-http" ./service/task/cmd/http
go build -o "$BIN_DIR/workspace-grpc" ./service/workspace/cmd/grpc
go build -o "$BIN_DIR/workspace-http" ./service/workspace/cmd/http

log "start service"
"$BIN_DIR/identity-grpc" >"$BIN_DIR/identity-grpc.log" 2>&1 &
IDENTITY_GRPC_PID=$!
"$BIN_DIR/identity-http" >"$BIN_DIR/identity-http.log" 2>&1 &
IDENTITY_HTTP_PID=$!
"$BIN_DIR/task-grpc" >"$BIN_DIR/task-grpc.log" 2>&1 &
TASK_GRPC_PID=$!
"$BIN_DIR/task-http" >"$BIN_DIR/task-http.log" 2>&1 &
TASK_HTTP_PID=$!
"$BIN_DIR/workspace-grpc" >"$BIN_DIR/workspace-grpc.log" 2>&1 &
WORKSPACE_GRPC_PID=$!
"$BIN_DIR/workspace-http" >"$BIN_DIR/workspace-http.log" 2>&1 &
WORKSPACE_HTTP_PID=$!

cleanup() {
  kill "$IDENTITY_HTTP_PID" "$IDENTITY_GRPC_PID" \
    "$TASK_HTTP_PID" "$TASK_GRPC_PID" \
    "$WORKSPACE_HTTP_PID" "$WORKSPACE_GRPC_PID" 2>/dev/null || true
  wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM

for _ in $(seq 1 60); do
  if curl -sf localhost:8081/healthz >/dev/null 2>&1 \
    && curl -sf localhost:8082/healthz >/dev/null 2>&1 \
    && curl -sf localhost:8083/healthz >/dev/null 2>&1; then
    break
  fi
  sleep 0.25
done

if ! curl -sf localhost:8081/healthz >/dev/null 2>&1 \
  || ! curl -sf localhost:8082/healthz >/dev/null 2>&1 \
  || ! curl -sf localhost:8083/healthz >/dev/null 2>&1; then
  echo "Service không khởi động được. Log:" >&2
  tail -20 \
    "$BIN_DIR/identity-grpc.log" "$BIN_DIR/identity-http.log" \
    "$BIN_DIR/task-grpc.log" "$BIN_DIR/task-http.log" \
    "$BIN_DIR/workspace-grpc.log" "$BIN_DIR/workspace-http.log" >&2
  exit 1
fi

log "chạy smoke test"
node scripts/smoke-test.mjs

#!/usr/bin/env bash
# Chạy đồng thời tất cả service Go ở local (Ctrl+C để dừng cả hai).
# Mỗi service gồm 2 process: gRPC server (cmd/grpc) + HTTP gateway (cmd/http).
# Port: identity gRPC :9081 / HTTP :8081, task gRPC :9082 / HTTP :8082
set -euo pipefail

cd "$(dirname "$0")/.."

pids=()
cleanup() {
  echo
  echo "==> Đang dừng service..."
  for pid in "${pids[@]}"; do kill "$pid" 2>/dev/null || true; done
  wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM

go run ./service/identity/cmd/grpc &
pids+=("$!")
go run ./service/identity/cmd/http &
pids+=("$!")

go run ./service/task/cmd/grpc &
pids+=("$!")
go run ./service/task/cmd/http &
pids+=("$!")

echo "==> identity: http://localhost:8081  (gRPC :9081)"
echo "==> task:     http://localhost:8082  (gRPC :9082)"
wait

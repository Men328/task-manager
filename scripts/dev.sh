#!/usr/bin/env bash
# Chạy đồng thời tất cả service Go ở local (Ctrl+C để dừng cả hai).
# Mỗi service gồm 2 process: gRPC server (cmd/grpc) + HTTP gateway (cmd/http).
# Port: identity gRPC :9081 / HTTP :8081, task gRPC :9082 / HTTP :8082,
#       calendar gRPC :9083 / HTTP :8083, mail-provider gRPC :9084 / HTTP :8084
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

go run ./service/calendar/cmd/grpc &
pids+=("$!")
go run ./service/calendar/cmd/http &
pids+=("$!")

go run ./service/mail-provider/cmd/grpc &
pids+=("$!")
go run ./service/mail-provider/cmd/http &
pids+=("$!")

echo "==> identity:  http://localhost:8081  (gRPC :9081)"
echo "==> task:      http://localhost:8082  (gRPC :9082)"
echo "==> calendar:  http://localhost:8083  (gRPC :9083)"
echo "==> mail:      http://localhost:8084  (gRPC :9084)"
wait

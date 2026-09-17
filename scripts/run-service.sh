#!/usr/bin/env bash
# Chạy 1 service Go ở local: gRPC server (cmd/grpc) + HTTP gateway (cmd/http).
# Usage: scripts/run-service.sh <identity|task>
set -euo pipefail

cd "$(dirname "$0")/.."

service="${1:-}"
case "$service" in
  identity) grpc_port=9081; http_port=8081 ;;
  task)     grpc_port=9082; http_port=8082 ;;
  *) echo "usage: $0 <identity|task>" >&2; exit 1 ;;
esac

pids=()
cleanup() {
  echo
  echo "==> Đang dừng $service..."
  for pid in "${pids[@]}"; do kill "$pid" 2>/dev/null || true; done
  wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM

( cd "service/$service" && exec go run ./cmd/grpc ) &
pids+=("$!")
( cd "service/$service" && exec go run ./cmd/http ) &
pids+=("$!")

echo "==> $service: gRPC :$grpc_port / HTTP :$http_port"
wait

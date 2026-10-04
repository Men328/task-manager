#!/usr/bin/env bash
# Chạy 1 service Go ở local: gRPC server (cmd/grpc) + HTTP gateway (cmd/http).
# Usage: scripts/run-service.sh <attachment|backlog|calendar|event|identity|mail-provider|notification|report|task>
set -euo pipefail

cd "$(dirname "$0")/.."

service="${1:-}"
case "$service" in
  attachment) grpc_port=9089; http_port=8089 ;;
  backlog)  grpc_port=9086; http_port=8086 ;;
  calendar) grpc_port=9083; http_port=8083 ;;
  event)    grpc_port=9085; http_port=8085 ;;
  identity) grpc_port=9081; http_port=8081 ;;
  mail-provider) grpc_port=9084; http_port=8084 ;;
  notification) grpc_port=9088; http_port=8088 ;;
  report)   grpc_port=9087; http_port=8087 ;;
  task)     grpc_port=9082; http_port=8082 ;;
  *) echo "usage: $0 <attachment|backlog|calendar|event|identity|mail-provider|notification|report|task>" >&2; exit 1 ;;
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

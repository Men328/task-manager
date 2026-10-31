#!/usr/bin/env bash
# Build image theo batch thay vì build toàn bộ cùng lúc (tránh ngốn CPU/RAM).
#
# Mỗi service Go chỉ build 1 lần qua container `<svc>-grpc`; gateway `<svc>`
# dùng chung tag image `task-manager/<svc>:latest` (xem docker-compose.yml).
#
# Biến môi trường:
#   BATCH_SIZE  số image build song song trong 1 batch (mặc định 3)
#   SERVICES    danh sách service compose cần build, cách nhau bởi dấu cách
#               (mặc định: toàn bộ image của stack)
#
# Ví dụ:
#   bash scripts/build-images.sh
#   BATCH_SIZE=1 bash scripts/build-images.sh
#   SERVICES="task-grpc frontend" bash scripts/build-images.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE=(docker compose -f "$ROOT/deployments/docker/docker-compose.yml")

BATCH_SIZE="${BATCH_SIZE:-3}"
if ! [[ "$BATCH_SIZE" =~ ^[1-9][0-9]*$ ]]; then
  echo "BATCH_SIZE phải là số nguyên dương (đang là '$BATCH_SIZE')" >&2
  exit 1
fi

# Thứ tự: project trước vì nhiều service phụ thuộc, frontend (npm build nặng) để cuối.
DEFAULT_SERVICES="project-grpc identity-grpc task-grpc calendar-grpc event-grpc backlog-grpc attachment-grpc report-grpc mail-provider-grpc notification-grpc frontend"
read -r -a services <<<"${SERVICES:-$DEFAULT_SERVICES}"

total=${#services[@]}
batches=$(( (total + BATCH_SIZE - 1) / BATCH_SIZE ))
start=$SECONDS

for (( i = 0; i < total; i += BATCH_SIZE )); do
  batch=("${services[@]:i:BATCH_SIZE}")
  echo "==> Batch $(( i / BATCH_SIZE + 1 ))/$batches: ${batch[*]}"
  "${COMPOSE[@]}" build "${batch[@]}"
done

echo "==> Xong $total image trong $(( SECONDS - start ))s"

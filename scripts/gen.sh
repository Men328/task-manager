#!/usr/bin/env bash
# Sinh code từ common/proto -> common/gen (go + openapi) và cập nhật go.mod từng module.
set -euo pipefail

cd "$(dirname "$0")/.."

MODULES="common service/identity service/task"

log() { printf '\033[36m==>\033[0m %s\n' "$*"; }

command -v buf >/dev/null 2>&1 || {
  echo "Chưa có 'buf'. Chạy trước: make tools" >&2
  exit 1
}

for plugin in protoc-gen-go protoc-gen-go-grpc protoc-gen-grpc-gateway protoc-gen-openapiv2; do
  command -v "$plugin" >/dev/null 2>&1 || {
    echo "Thiếu plugin '$plugin'. Chạy trước: make tools" >&2
    exit 1
  }
done

log "buf dep update (tải google/api/annotations.proto)"
buf dep update

log "buf lint"
buf lint

log "buf generate"
buf generate

# common/gen nằm trong module `common`; các service import qua module đó.
log "go mod tidy từng module"
for m in $MODULES; do
  echo "    - $m"
  (cd "$m" && go mod tidy)
done

log "Xong. Code gen:"
find common/gen -type f \( -name '*.go' -o -name '*.json' \) | sort

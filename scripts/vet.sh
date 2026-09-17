#!/usr/bin/env bash
# go vet cho từng module, BỎ QUA diagnostic phát sinh trong source của dependency.
#
# Vì sao cần: Go 1.26 chạy analyzer `unreachable` lên cả package dependency, nên
# `go vet ./...` báo lỗi trong module cache, ví dụ:
#   .../pkg/mod/google.golang.org/protobuf@v1.36.12/encoding/protojson/decode.go:575:2: unreachable code
# Đó không phải code của mình. (Go 1.27 không còn hiện tượng này.)
# Cách xử lý: vẫn chạy đầy đủ analyzer, chỉ lọc các dòng trỏ vào module cache.
set -uo pipefail

cd "$(dirname "$0")/.."

MODULES="common service/identity service/task tools/quality"
GOMODCACHE="$(go env GOMODCACHE 2>/dev/null || true)"

status=0
for m in $MODULES; do
  echo "==> $m"

  out="$( cd "$m" && go vet ./... 2>&1 )"

  # Lọc: bỏ dòng thuộc module cache (dependency) và dòng trống
  filtered="$(printf '%s\n' "$out" \
    | grep -v -e '/pkg/mod/' -e '^$' || true)"

  if [ -n "$GOMODCACHE" ]; then
    filtered="$(printf '%s\n' "$filtered" | grep -v -F "$GOMODCACHE" || true)"
  fi

  if [ -n "$filtered" ]; then
    printf '%s\n' "$filtered" >&2
    status=1
  fi
done

if [ "$status" -eq 0 ]; then
  echo "vet OK (đã bỏ qua cảnh báo trong dependency)"
fi
exit "$status"

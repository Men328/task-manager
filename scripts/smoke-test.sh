#!/usr/bin/env bash
# Smoke test end-to-end: build 8 service (mỗi service gồm gRPC server + gateway),
# chạy ở port mặc định, gọi API thật, rồi tắt.
# Yêu cầu: go, node >= 20 (dùng fetch), curl.
# Port 8081-8088 và 9081-9088 phải đang trống.
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
go build -o "$BIN_DIR/calendar-grpc" ./service/calendar/cmd/grpc
go build -o "$BIN_DIR/calendar-http" ./service/calendar/cmd/http
go build -o "$BIN_DIR/event-grpc" ./service/event/cmd/grpc
go build -o "$BIN_DIR/event-http" ./service/event/cmd/http
go build -o "$BIN_DIR/backlog-grpc" ./service/backlog/cmd/grpc
go build -o "$BIN_DIR/backlog-http" ./service/backlog/cmd/http
go build -o "$BIN_DIR/mail-provider-grpc" ./service/mail-provider/cmd/grpc
go build -o "$BIN_DIR/mail-provider-http" ./service/mail-provider/cmd/http
go build -o "$BIN_DIR/report-grpc" ./service/report/cmd/grpc
go build -o "$BIN_DIR/report-http" ./service/report/cmd/http
go build -o "$BIN_DIR/notification-grpc" ./service/notification/cmd/grpc
go build -o "$BIN_DIR/notification-http" ./service/notification/cmd/http

log "start service"
"$BIN_DIR/identity-grpc" >"$BIN_DIR/identity-grpc.log" 2>&1 &
IDENTITY_GRPC_PID=$!
"$BIN_DIR/identity-http" >"$BIN_DIR/identity-http.log" 2>&1 &
IDENTITY_HTTP_PID=$!
"$BIN_DIR/task-grpc" >"$BIN_DIR/task-grpc.log" 2>&1 &
TASK_GRPC_PID=$!
"$BIN_DIR/task-http" >"$BIN_DIR/task-http.log" 2>&1 &
TASK_HTTP_PID=$!
"$BIN_DIR/calendar-grpc" >"$BIN_DIR/calendar-grpc.log" 2>&1 &
CALENDAR_GRPC_PID=$!
"$BIN_DIR/calendar-http" >"$BIN_DIR/calendar-http.log" 2>&1 &
CALENDAR_HTTP_PID=$!
"$BIN_DIR/event-grpc" >"$BIN_DIR/event-grpc.log" 2>&1 &
EVENT_GRPC_PID=$!
"$BIN_DIR/event-http" >"$BIN_DIR/event-http.log" 2>&1 &
EVENT_HTTP_PID=$!
"$BIN_DIR/backlog-grpc" >"$BIN_DIR/backlog-grpc.log" 2>&1 &
BACKLOG_GRPC_PID=$!
"$BIN_DIR/backlog-http" >"$BIN_DIR/backlog-http.log" 2>&1 &
BACKLOG_HTTP_PID=$!
"$BIN_DIR/mail-provider-grpc" >"$BIN_DIR/mail-provider-grpc.log" 2>&1 &
MAIL_GRPC_PID=$!
"$BIN_DIR/mail-provider-http" >"$BIN_DIR/mail-provider-http.log" 2>&1 &
MAIL_HTTP_PID=$!
"$BIN_DIR/report-grpc" >"$BIN_DIR/report-grpc.log" 2>&1 &
REPORT_GRPC_PID=$!
"$BIN_DIR/report-http" >"$BIN_DIR/report-http.log" 2>&1 &
REPORT_HTTP_PID=$!
"$BIN_DIR/notification-grpc" >"$BIN_DIR/notification-grpc.log" 2>&1 &
NOTIFICATION_GRPC_PID=$!
"$BIN_DIR/notification-http" >"$BIN_DIR/notification-http.log" 2>&1 &
NOTIFICATION_HTTP_PID=$!

cleanup() {
  kill "$IDENTITY_HTTP_PID" "$IDENTITY_GRPC_PID" \
    "$TASK_HTTP_PID" "$TASK_GRPC_PID" \
    "$CALENDAR_HTTP_PID" "$CALENDAR_GRPC_PID" \
    "$EVENT_HTTP_PID" "$EVENT_GRPC_PID" \
    "$BACKLOG_HTTP_PID" "$BACKLOG_GRPC_PID" \
    "$MAIL_HTTP_PID" "$MAIL_GRPC_PID" \
    "$REPORT_HTTP_PID" "$REPORT_GRPC_PID" \
    "$NOTIFICATION_HTTP_PID" "$NOTIFICATION_GRPC_PID" 2>/dev/null || true
  wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM

for _ in $(seq 1 60); do
  if curl -sf localhost:8081/healthz >/dev/null 2>&1 \
    && curl -sf localhost:8082/healthz >/dev/null 2>&1 \
    && curl -sf localhost:8083/healthz >/dev/null 2>&1 \
    && curl -sf localhost:8085/healthz >/dev/null 2>&1 \
    && curl -sf localhost:8086/healthz >/dev/null 2>&1 \
    && curl -sf localhost:8087/healthz >/dev/null 2>&1 \
    && curl -sf localhost:8084/healthz >/dev/null 2>&1 \
    && curl -sf localhost:8088/healthz >/dev/null 2>&1; then
    break
  fi
  sleep 0.25
done

if ! curl -sf localhost:8081/healthz >/dev/null 2>&1 \
  || ! curl -sf localhost:8082/healthz >/dev/null 2>&1 \
  || ! curl -sf localhost:8083/healthz >/dev/null 2>&1 \
  || ! curl -sf localhost:8085/healthz >/dev/null 2>&1 \
  || ! curl -sf localhost:8086/healthz >/dev/null 2>&1 \
  || ! curl -sf localhost:8087/healthz >/dev/null 2>&1 \
  || ! curl -sf localhost:8084/healthz >/dev/null 2>&1 \
  || ! curl -sf localhost:8088/healthz >/dev/null 2>&1; then
  echo "Service không khởi động được. Log:" >&2
  tail -20 \
    "$BIN_DIR/identity-grpc.log" "$BIN_DIR/identity-http.log" \
    "$BIN_DIR/task-grpc.log" "$BIN_DIR/task-http.log" \
    "$BIN_DIR/calendar-grpc.log" "$BIN_DIR/calendar-http.log" \
    "$BIN_DIR/event-grpc.log" "$BIN_DIR/event-http.log" \
    "$BIN_DIR/backlog-grpc.log" "$BIN_DIR/backlog-http.log" \
    "$BIN_DIR/mail-provider-grpc.log" "$BIN_DIR/mail-provider-http.log" \
    "$BIN_DIR/report-grpc.log" "$BIN_DIR/report-http.log" \
    "$BIN_DIR/notification-grpc.log" "$BIN_DIR/notification-http.log" >&2
  exit 1
fi

log "chạy smoke test"
node scripts/smoke-test.mjs

# service/calendar

Service quản lý **lịch (schedule)** cá nhân của từng profile — CRUD cơ bản. Là **1 Go module
riêng** (`taskmanager/service/calendar`), dùng code chung qua module `taskmanager/common`
(xem `replace` trong `go.mod`).

- gRPC: `:9083` — HTTP/JSON gateway: `:8083`
- Proto: `common/proto/calendar/v1/schedule.proto`
- Bảng DB: `calendar.schedules` (xem `design/db_schema.dbml` + `deployments/migrations/000006_init_calendar.*.sql`)

## Cấu trúc

```
Dockerfile                       # build 2 binary grpc + http (build context = root repo)
cmd/grpc/                        # fx app: gRPC server (:9083) + health + reflection + chọn repo
cmd/http/                        # fx app: grpc-gateway (:8083) + lifecycle, dial qua GRPC_DIAL_ADDR
internal/config/config.go
internal/model/                  # CORE: Schedule + ScheduleFilter/ScheduleUpdate + lỗi domain (chỉ stdlib)
internal/service/                # NGHIỆP VỤ + interface contract (interfaces.go): 1 port + 1 service
internal/repository/             # adapter I/O: postgres_schedule_repository.go + in-memory fallback
internal/dependency/             # validate + mapping (model <-> proto) + map lỗi -> mã lỗi chuẩn
internal/handler/                # transport gRPC mỏng
```

**Dependency inversion**: `service/interfaces.go` khai báo port `ScheduleRepository` và contract
`ScheduleService`. `service` chỉ import `model` — không import `repository`/`dependency`. Adapter ở
`repository` thoả mãn port nhờ structural typing (không import `service`). `cmd` là nơi duy nhất
wiring concrete vào interface.

## Nghiệp vụ đã có trong base

- **CRUD lịch**: `POST/GET/PATCH/DELETE /v1/schedules`, list `GET /v1/schedules?profile_id=...`.
- **List theo profile + khoảng thời gian**: `from`/`to` (optional) trả các lịch giao với `[from, to)`
  — UI calendar chỉ tải đúng khoảng đang hiển thị.
- **Validate**: `title` bắt buộc, `profile_id` bắt buộc, `start_at` bắt buộc; `end_at` (nếu có)
  không được sớm hơn `start_at` (`CALENDAR_TIME_RANGE_INVALID`). DB cũng có CHECK tương ứng.
- **All-day + màu**: `all_day` map thẳng sang `allDay` của FullCalendar; `color` nhận `#RRGGBB`
  hoặc `#RRGGBBAA`, rỗng = dùng màu mặc định của UI.
- **Chưa có** (ngoài scope base): lịch lặp/RRULE, người tham gia, nhắc hẹn, đồng bộ Google Calendar.

## Chạy

```bash
make run-calendar
```

| Biến | Default |
|---|---|
| `SERVICE_NAME` | `calendar` |
| `GRPC_ADDR` | `:9083` (cmd/grpc listen) |
| `HTTP_ADDR` | `:8083` (cmd/http listen) |
| `GRPC_DIAL_ADDR` | rỗng -> suy ra `127.0.0.1:9083` (cmd/http dial target) |
| `LOG_LEVEL` | `info` |
| `DATABASE_URL` | rỗng -> in-memory stub; set thì dùng PostgreSQL |

## Thử nhanh

```bash
P=<profile_id>

# tạo lịch
curl -s -X POST localhost:8083/v1/schedules -H 'Content-Type: application/json' \
  -d "{\"profile_id\":\"$P\",\"title\":\"Họp nhóm\",\"start_at\":\"2026-10-05T01:00:00Z\",\"end_at\":\"2026-10-05T02:00:00Z\"}"

# list theo khoảng thời gian
curl -s "localhost:8083/v1/schedules?profile_id=$P&from=2026-10-01T00:00:00Z&to=2026-11-01T00:00:00Z"

# sửa / xoá
curl -s -X PATCH localhost:8083/v1/schedules/<id> -H 'Content-Type: application/json' -d '{"all_day":true}'
curl -s -X DELETE localhost:8083/v1/schedules/<id>
```

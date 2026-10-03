# service/event

Service quản lý **sự kiện (event)** cá nhân của từng profile — CRUD cơ bản. Là **1 Go module
riêng** (`taskmanager/service/event`), dùng code chung qua module `taskmanager/common`
(xem `replace` trong `go.mod`).

- gRPC: `:9085` — HTTP/JSON gateway: `:8085`
- Proto: `common/proto/event/v1/event.proto`
- Bảng DB: `event.events` (xem `design/db_schema.dbml` + `deployments/migrations/000007_init_event.*.sql`)

## Vì sao có `event` bên cạnh `calendar`

`calendar.schedules` là **lịch** (khối thời gian trên calendar, không có vòng đời).
`event.events` là **sự kiện** có:

- `status`: `PLANNED` → `CONFIRMED` → `CANCELLED`;
- `source`: id email nguồn khi sự kiện được mail worker tạo tự động.

Nhờ vậy UI phân biệt được "lịch trình" với "sự kiện cần xác nhận", và truy vết được sự kiện nào
sinh ra từ email nào.

## Cấu trúc

```
Dockerfile                       # build 2 binary grpc + http (build context = root repo)
cmd/grpc/                        # fx app: gRPC server (:9085) + health + reflection + chọn repo
cmd/http/                        # fx app: grpc-gateway (:8085), dial qua GRPC_DIAL_ADDR
internal/config/config.go
internal/model/                  # CORE: Event + Filter/Update + Status + lỗi domain (chỉ stdlib)
internal/service/                # NGHIỆP VỤ + interface contract (interfaces.go)
internal/repository/             # adapter I/O: postgres_event_repository.go + in-memory fallback
internal/dependency/             # validate + mapping (model <-> proto) + map lỗi -> mã lỗi chuẩn
internal/handler/                # transport gRPC mỏng
```

## Nghiệp vụ

- **CRUD sự kiện**: `POST/GET/PATCH/DELETE /v1/events`, list `GET /v1/events?profile_id=...`.
- **List theo profile + khoảng thời gian + trạng thái**: `from`/`to` (giao với `[from, to)`) và
  `status` (optional).
- **Validate**: `title`, `profile_id`, `start_at` bắt buộc; `end_at` (nếu có) không sớm hơn
  `start_at` (`EVENT_TIME_RANGE_INVALID`); `status` chỉ nhận 3 giá trị đã biết.
- **Chưa có** (ngoài scope base): người tham gia, lặp/RRULE, nhắc hẹn, đồng bộ Google Calendar.

## Chạy

```bash
make run-event
```

| Biến | Default |
|---|---|
| `SERVICE_NAME` | `event` |
| `GRPC_ADDR` | `:9085` (cmd/grpc listen) |
| `HTTP_ADDR` | `:8085` (cmd/http listen) |
| `GRPC_DIAL_ADDR` | rỗng -> suy ra `127.0.0.1:9085` (cmd/http dial target) |
| `LOG_LEVEL` | `info` |
| `DATABASE_URL` | rỗng -> in-memory stub; set thì dùng PostgreSQL |

## Thử nhanh

```bash
P=<profile_id>

curl -s -X POST localhost:8085/v1/events -H 'Content-Type: application/json' \
  -d "{\"profile_id\":\"$P\",\"title\":\"Hội thảo\",\"start_at\":\"2026-10-05T01:00:00Z\",\"status\":\"EVENT_STATUS_CONFIRMED\"}"

curl -s "localhost:8085/v1/events?profile_id=$P&from=2026-10-01T00:00:00Z&to=2026-11-01T00:00:00Z"

curl -s -X PATCH localhost:8085/v1/events/<id> -H 'Content-Type: application/json' -d '{"status":"EVENT_STATUS_CANCELLED"}'
curl -s -X DELETE localhost:8085/v1/events/<id>
```

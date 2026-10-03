# service/backlog

Service **backlog** — nơi mail worker đẩy mọi email KHÔNG phân loại được thành
`task`/`schedule`/`event` (category `other`), cùng các email thiếu dữ kiện để tạo bản ghi.
Người dùng xem lại, sửa và chuyển trạng thái trên UI. Là **1 Go module riêng**
(`taskmanager/service/backlog`).

- gRPC: `:9086` — HTTP/JSON gateway: `:8086`
- Proto: `common/proto/backlog/v1/backlog.proto`
- Bảng DB: `backlog.backlogs` (xem `deployments/migrations/000008_init_backlog.*.sql`)

## Vì sao có backlog

Luồng mail worker: mỗi email được phân loại thành 1 trong 4 nhóm `task | schedule | event | other`.
- `task` → task service
- `schedule` → calendar service
- `event` → event service
- `other` (không actionable / newsletter / không rõ) → **backlog service**

Nhờ vậy không có email nào bị "rơi" âm thầm: nếu bộ phân loại không chắc, dữ liệu vẫn nằm ở backlog
kèm `reason` (lý do) và `object_key` (email gốc lưu trên object storage) để người dùng xử lý lại.

## Cấu trúc

```
Dockerfile                       # build 2 binary grpc + http (build context = root repo)
cmd/grpc/                        # fx app: gRPC server (:9086) + health + reflection + chọn repo
cmd/http/                        # fx app: grpc-gateway (:8086), dial qua GRPC_DIAL_ADDR
internal/config/config.go
internal/model/                  # CORE: Item + Filter/Update + Status + lỗi domain (chỉ stdlib)
internal/service/                # NGHIỆP VỤ + interface contract (interfaces.go)
internal/repository/             # adapter I/O: postgres_backlog_repository.go + in-memory fallback
internal/dependency/             # validate + mapping (model <-> proto) + map lỗi -> mã lỗi chuẩn
internal/handler/                # transport gRPC mỏng
```

## Nghiệp vụ

- **CRUD mục backlog**: `POST/GET/PATCH/DELETE /v1/backlogs`, list `GET /v1/backlogs?profile_id=...`.
- **List lọc**: `status` (NEW/TRIAGED/ARCHIVED) và `category` (task|schedule|event|other), optional.
- **Trường dữ liệu**: `title`, `description`, `sender`, `source` (message id), `category`, `reason`,
  `object_key` (object storage S3), `status`.
- **Mặc định**: tạo mới không truyền `status` → `NEW`; không truyền `category` → `other`.

## Chạy

```bash
make run-backlog
```

| Biến | Default |
|---|---|
| `SERVICE_NAME` | `backlog` |
| `GRPC_ADDR` | `:9086` (cmd/grpc listen) |
| `HTTP_ADDR` | `:8086` (cmd/http listen) |
| `GRPC_DIAL_ADDR` | rỗng -> suy ra `127.0.0.1:9086` |
| `LOG_LEVEL` | `info` |
| `DATABASE_URL` | rỗng -> in-memory stub; set thì dùng PostgreSQL |

## Thử nhanh

```bash
P=<profile_id>

curl -s -X POST localhost:8086/v1/backlogs -H 'Content-Type: application/json' \
  -d "{\"profile_id\":\"$P\",\"title\":\"Newsletter tháng 10\",\"category\":\"other\",\"reason\":\"not_actionable\"}"

curl -s "localhost:8086/v1/backlogs?profile_id=$P&status=BACKLOG_STATUS_NEW"

curl -s -X PATCH localhost:8086/v1/backlogs/<id> -H 'Content-Type: application/json' -d '{"status":"BACKLOG_STATUS_TRIAGED"}'
curl -s -X DELETE localhost:8086/v1/backlogs/<id>
```

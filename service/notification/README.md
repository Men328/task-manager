# service/notification

Service **notification** — lưu thông báo của user (`notification.notices`) và đẩy realtime qua
**soketi** (WebSocket, giao thức Pusher). Là **1 Go module riêng**
(`taskmanager/service/notification`).

- gRPC: `:9088` — HTTP/JSON gateway: `:8088`
- Proto: `common/proto/notification/v1/notice.proto`
- Bảng DB: `notification.notices` (xem `deployments/migrations/000009_init_notification.*.sql`)
- Kênh soketi: `private-noti-internal-<profile_id>` (mỗi user 1 kênh private)

## Luồng

```
mail-provider (worker)
  └─ tạo task/lịch/sự kiện/backlog thành công
       └─ gRPC CreateNotice ──► notification service
                                   ├─ INSERT notification.notices (is_read = false)
                                   └─ POST /apps/<id>/events ──► soketi
                                                                   └─ WS ──► browser
                                                                              └─ GET /v1/notices/unread-count
```

- `CreateNotice` là **RPC nội bộ** (không khai báo `google.api.http`) nên không lộ ra Internet.
- Nếu publish soketi lỗi, notice **vẫn được lưu** (chỉ log lỗi): tránh mất thông báo và tránh
  việc mail-provider retry tạo trùng.
- Browser nhận event `notice.created` → gọi API đếm notice chưa đọc để cập nhật badge.

## API HTTP (yêu cầu session JWT của identity)

Tất cả endpoint đều xác thực `Authorization: Bearer <session token>` (HS256, cùng `SESSION_SECRET`
với identity). `profile_id` được suy ra từ token, **không** nhận từ client — nhờ vậy mọi thao tác chỉ
tác động notice của chính user đang sở hữu.

| Method | Path | Việc |
|---|---|---|
| `GET` | `/v1/notices?limit=20&unread_only=true` | Danh sách notice mới nhất của user |
| `GET` | `/v1/notices/unread-count` | `{ "count": N }` — số notice chưa đọc |
| `POST` | `/v1/notices/read` | Đánh dấu đã đọc; body `{ "ids": ["<id>"] }` hoặc `{ "ids": ["*"] }` |

- `"*"` = đánh dấu đã đọc **tất cả** notice chưa đọc của user.
- Chỉ notice thuộc `profile_id` trong token mới bị tác động (scope ở cả gateway lẫn câu SQL).

## Uỷ quyền soketi (ở identity)

Browser subscribe kênh private nên phải xin chữ ký uỷ quyền. Endpoint nằm ở **identity**:

```
POST /v1/auth/soketi
Content-Type: application/x-www-form-urlencoded
Authorization: Bearer <session token>

socket_id=123.456&channel_name=private-noti-internal-<profile_id>
→ { "auth": "<app_key>:<hmac_sha256(app_secret, socket_id + ":" + channel_name)>" }
```

- Chỉ chấp nhận đúng kênh của profile trong token; kênh của user khác → `IDENTITY_AUTH_SOKETI_CHANNEL_FORBIDDEN`.
- Thiếu `SOKETI_APP_KEY`/`SOKETI_APP_SECRET` → `IDENTITY_AUTH_SOKETI_NOT_CONFIGURED` (fail closed).

## Cấu trúc

```
Dockerfile                       # build 2 binary grpc + http (build context = root repo)
cmd/grpc/                        # fx app: gRPC server (:9088) + health + reflection + chọn repo/publisher
cmd/http/                        # fx app: gateway (:8088): session JWT + route notice + /healthz
internal/config/config.go        # env: DATABASE_URL, SOKETI_*, SESSION_SECRET
internal/model/                  # CORE: Notice + Filter + hằng target/type (chỉ stdlib)
internal/service/                # NGHIỆP VỤ + interface contract (interfaces.go)
internal/repository/             # postgres + in-memory notice repo; soketi publisher (Pusher REST)
internal/dependency/             # validate + mapping (model <-> proto) + map lỗi -> mã lỗi chuẩn
internal/handler/                # transport gRPC mỏng
```

## Chạy

```bash
make run-notification     # cần soketi đang chạy
```

| Biến | Default | Việc |
|---|---|---|
| `SERVICE_NAME` | `notification` | |
| `GRPC_ADDR` | `:9088` | cmd/grpc listen |
| `HTTP_ADDR` | `:8088` | cmd/http listen |
| `GRPC_DIAL_ADDR` | rỗng → `127.0.0.1:9088` | gateway dial |
| `DATABASE_URL` | rỗng → in-memory | set thì dùng PostgreSQL |
| `SESSION_SECRET` | rỗng | khoá verify session JWT (phải trùng identity) |
| `SOKETI_BASE_URL` | `http://soketi:6001` | REST API nội bộ của soketi |
| `SOKETI_APP_ID` | `task-manager` | |
| `SOKETI_APP_KEY` | rỗng | app key (công khai) |
| `SOKETI_APP_SECRET` | rỗng | app secret (ký publish) |
| `SOKETI_CHANNEL_PREFIX` | `noti-internal-` | kênh = `private-<prefix><profile_id>` |
| `SOKETI_TIMEOUT` | `10s` | timeout gọi soketi |

## Thử nhanh

```bash
TOKEN=<session token>

curl -s localhost:8088/healthz

curl -s localhost:8088/v1/notices/unread-count -H "Authorization: Bearer $TOKEN"

curl -s localhost:8088/v1/notices -H "Authorization: Bearer $TOKEN"

curl -s -X POST localhost:8088/v1/notices/read \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"ids":["*"]}'
```

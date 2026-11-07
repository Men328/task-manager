# service/social

Service quản lý quan hệ bạn bè giữa các profile. Là **1 Go module riêng**
(`taskmanager/service/social`), dùng code chung qua module `taskmanager/common`.

- gRPC: `:9091` — HTTP gateway: `:8091`
- Proto: `common/proto/social/v1/friend.proto`
- Bảng DB: `social.friendships` (xem `deployments/migrations/`)
- Thiết kế: `docs/superpowers/specs/2026-11-04-friends-design.md`

## Xác thực

Proto **không** khai báo `google.api.http`, vì gateway sinh tự động không kiểm tra token.
HTTP API là route viết tay ở `cmd/http/friend_routes.go`: mọi route bắt buộc bearer token và
người gọi luôn lấy từ `sub` của session JWT, không nhận từ client.

## Phụ thuộc

- `identity-grpc`: `FindProfileByEmail`, `BatchGetProfiles` (RPC nội bộ).
- `notification-grpc`: `CreateNotice`, loại `FRIEND_REQUEST` và `FRIEND_ACCEPTED`. Lỗi gửi thông báo
  chỉ ghi log, không làm hỏng thao tác.

## Chạy

```bash
make run-social
```

| Biến | Default |
|---|---|
| `SERVICE_NAME` | `social` |
| `GRPC_ADDR` | `:9091` |
| `HTTP_ADDR` | `:8091` |
| `GRPC_DIAL_ADDR` | `127.0.0.1:9091` |
| `DATABASE_URL` | rỗng = repository in-memory |
| `SESSION_SECRET` | rỗng, gateway từ chối mọi token; phải khớp với identity |
| `IDENTITY_GRPC_DIAL_ADDR` | `localhost:9081` |
| `NOTIFICATION_GRPC_DIAL_ADDR` | `localhost:9088` |
| `UPSTREAM_TIMEOUT` | `10s` |

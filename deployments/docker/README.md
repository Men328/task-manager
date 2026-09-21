# deployments/docker

| File | Việc |
|---|---|
| `docker-compose.yml` | Stack đầy đủ: postgres + migrate + identity(-grpc) + task(-grpc) + frontend |

Dockerfile nằm **trong từng service** (`service/identity/Dockerfile`, `service/task/Dockerfile`,
`frontend/Dockerfile`) để mỗi service tự đóng gói. Compose trỏ tới chúng với build context là
**root repo**, vì mỗi service cần copy thêm module `common/`.

Mỗi service Go build ra **2 binary** trong cùng image: `grpc` (gRPC server) và `http`
(grpc-gateway). Compose tách thành 2 container: `<svc>-grpc` (nội bộ) và `<svc>` (HTTP public).
Container gateway nhận `GRPC_DIAL_ADDR=<svc>-grpc:<port>` để dial sang container gRPC.

## Chạy

```bash
make up        # docker compose up --build -d
make ps        # trạng thái
make logs      # log
make down      # dừng
```

Hoặc trực tiếp:

```bash
docker compose -f deployments/docker/docker-compose.yml up --build -d
```

## Port

| Service (compose) | Container | Host | Container |
|---|---|---|---|
| frontend | tm-frontend | http://localhost:3000 | 80 |
| identity (gateway) | tm-identity | http://localhost:8081 | 8081 |
| identity-grpc | tm-identity-grpc | localhost:9081 | 9081 |
| task (gateway) | tm-task | http://localhost:8082 | 8082 |
| task-grpc | tm-task-grpc | localhost:9082 | 9082 |
| postgres | tm-postgres | localhost:5432 | 5432 |

Tên service HTTP giữ nguyên (`identity`, `task`) nên nginx của frontend không phải đổi proxy.

## Thứ tự khởi động

```
postgres (healthy) -> migrate (chạy xong, exit 0)
  -> identity(-grpc)/task(-grpc)
  -> gateway identity/task (healthy, /healthz chỉ 200 khi gọi được gRPC health)
  -> frontend
```

## Migration

Service `migrate` chạy `deployments/migrations/` một lần rồi thoát (one-shot).

```bash
make migrate-up        # chạy toàn bộ
make migrate-down      # rollback 1 bước
```

## Ghi chú

- `identity-grpc` nối Postgres qua `DATABASE_URL` (đã bật sẵn trong compose). Bỏ trống biến này
  thì service tự quay về repository in-memory stub và log cảnh báo. `task-grpc` vẫn in-memory.
- Image `migrate/migrate:latest` nên pin version khi dùng thật.

## Đăng nhập Google

Gateway identity đọc cấu hình từ env. Tạo `deployments/docker/.env`:

```bash
cp deployments/docker/.env.example deployments/docker/.env
openssl rand -base64 48   # dán kết quả vào SESSION_SECRET
```

Rồi điền `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` lấy từ
[Google Cloud Console](https://console.cloud.google.com/apis/credentials) (OAuth client ID,
loại *Web application*).

| Biến | Ý nghĩa |
|---|---|
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` | Credentials của OAuth client |
| `GOOGLE_REDIRECT_URL` | Phải **trùng khít** Authorized redirect URI đã khai báo ở Google |
| `FRONTEND_BASE_URL` | Origin của SPA; backend redirect về đây kèm token ở fragment |
| `SESSION_SECRET` | Khoá ký JWT (HS256) và state OAuth, tối thiểu 16 ký tự |
| `SESSION_TTL` | Hạn của session token, mặc định `720h` (30 ngày) |
| `COOKIE_SECURE` | Đặt `true` khi chạy sau HTTPS |

`GOOGLE_REDIRECT_URL` mặc định là `http://localhost:3000/api/identity/v1/auth/google/callback`
— trỏ vào **origin của frontend**, không phải `:8081`, để callback đi qua nginx và giữ cùng
origin với SPA.

Nếu chưa cấu hình, gateway vẫn khởi động bình thường nhưng log cảnh báo, và
`/v1/auth/google/login` trả mã lỗi `IDENTITY_AUTH_NOT_CONFIGURED`.

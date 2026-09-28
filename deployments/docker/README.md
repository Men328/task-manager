# deployments/docker

| File | Việc |
|---|---|
| `docker-compose.yml` | Stack đầy đủ: postgres + migrate + identity(-grpc) + task(-grpc) + workspace(-grpc) + frontend + cloudflared (profile `tunnel`) |

Dockerfile nằm **trong từng service** (`service/identity/Dockerfile`, `service/task/Dockerfile`,
`service/workspace/Dockerfile`, `frontend/Dockerfile`) để mỗi service tự đóng gói. Compose trỏ tới
chúng với build context là **root repo**, vì mỗi service cần copy thêm module `common/`.

Mỗi service Go build ra **2 binary** trong cùng image: `grpc` (gRPC server) và `http`
(grpc-gateway). Compose tách thành 2 container: `<svc>-grpc` (nội bộ) và `<svc>` (HTTP, chỉ
nội bộ — nginx của frontend proxy tới). Container gateway nhận
`GRPC_DIAL_ADDR=<svc>-grpc:<port>` để dial sang container gRPC.

## Chạy

```bash
make up        # docker compose up --build -d
make up-tunnel # như trên, kèm cloudflared (cần CLOUDFLARE_TUNNEL_TOKEN trong .env)
make ps        # trạng thái
make logs      # log
make down      # dừng
```

Hoặc trực tiếp:

```bash
docker compose -f deployments/docker/docker-compose.yml up --build -d
docker compose -f deployments/docker/docker-compose.yml --profile tunnel up --build -d
```

## Port

`frontend` là **entrypoint duy nhất** ra host (bind `127.0.0.1`). Mọi port khác chỉ nằm
trong network Docker nội bộ và được các container gọi nhau bằng tên service.

| Service (compose) | Container | Truy cập | Ghi chú |
|---|---|---|---|
| frontend | tm-frontend | `http://127.0.0.1:3000` (`FRONTEND_PORT`) | nginx: serve SPA + proxy `/api/*` |
| identity (gateway) | tm-identity | nội bộ `identity:8081` | không publish ra host |
| identity-grpc | tm-identity-grpc | nội bộ `identity-grpc:9081` | không publish ra host |
| task (gateway) | tm-task | nội bộ `task:8082` | không publish ra host |
| task-grpc | tm-task-grpc | nội bộ `task-grpc:9082` | không publish ra host |
| workspace (gateway) | tm-workspace | nội bộ `workspace:8083` | không publish ra host |
| workspace-grpc | tm-workspace-grpc | nội bộ `workspace-grpc:9083` | không publish ra host |
| postgres | tm-postgres | nội bộ `postgres:5432` | không publish ra host |
| cloudflared | tm-cloudflared | — (outbound) | profile `tunnel`, đẩy `frontend:3000` ra Internet |

Tên service HTTP giữ nguyên (`identity`, `task`, `workspace`) nên nginx của frontend không phải đổi proxy.

## Cloudflare Tunnel

FE gọi API **same-origin** tại `/api/...` trên chính port frontend; nginx của frontend mới
forward nội bộ tới các service Go. Vì vậy chỉ cần đưa **một origin** ra Internet và nó khớp
thẳng với public hostname của Cloudflare — không lộ port `8081/8082/8083`.

1. Tạo tunnel tại [Cloudflare Zero Trust](https://one.dash.cloudflare.com/) →
   **Networks → Tunnels → Create a tunnel** (chọn *Cloudflared*).
2. Ở bước "Install connector", copy token và điền vào `deployments/docker/.env`:

   ```bash
   CLOUDFLARE_TUNNEL_TOKEN=<token>
   ```

3. Trong tab **Public Hostname** của tunnel, thêm hostname và trỏ service tới:

   ```
   http://frontend:3000
   ```

   (`frontend` là tên service trong compose; cloudflared chạy cùng network nên phân giải được.)

4. Bật connector (chọn một cách):

   ```bash
   make up-tunnel        # chỉ lần này
   make tunnel-logs      # xem log cloudflared
   ```

   Hoặc để **mọi** lệnh `docker compose up -d` / `make up` đều kèm tunnel: bỏ comment
   dòng `COMPOSE_PROFILES=tunnel` trong `deployments/docker/.env`. Khi đó lệnh quen thuộc

   ```bash
   docker compose up -d --build
   ```

   (chạy trong `deployments/docker/`) cũng tạo container `tm-cloudflared`.

Khi đã có hostname công khai, chỉ cần đổi **một** biến trong `deployments/docker/.env`:

```bash
PUBLIC_BASE_URL=https://<hostname>
COOKIE_SECURE=true
```

`GOOGLE_REDIRECT_URL` và `FRONTEND_BASE_URL` được suy ra từ `PUBLIC_BASE_URL`
(`<PUBLIC_BASE_URL>/api/identity/v1/auth/google/callback` và `<PUBLIC_BASE_URL>`).
Muốn tách riêng thì cứ set thẳng 2 biến đó, chúng sẽ override.

rồi thêm đúng redirect URI (`https://<hostname>/api/identity/v1/auth/google/callback`) vào
**Authorized redirect URIs** của OAuth client trên Google Cloud Console, và `make up-tunnel` lại.
Redirect vẫn trỏ vào frontend (`/api/identity/...`) nên callback đi qua chính nginx như lúc chạy local.

| Biến | Ý nghĩa |
|---|---|
| `CLOUDFLARE_TUNNEL_TOKEN` | Token connector của tunnel; bắt buộc để bật `cloudflared` |
| `CLOUDFLARED_VERSION` | Tag image `cloudflare/cloudflared`, mặc định `latest` |
| `FRONTEND_PORT` | Port host bind entrypoint, mặc định `3000` (luôn bind `127.0.0.1`) |
| `COMPOSE_PROFILES` | Biến của Compose: đặt `tunnel` để `docker compose up` mặc định bật cloudflared |
| `PUBLIC_BASE_URL` | Origin công khai duy nhất; nguồn suy ra OAuth redirect + frontend base |

> `cloudflared` nằm sau profile `tunnel` để máy nào chưa có token vẫn chạy stack bình thường
> (không crash-loop). Bật bằng `make up-tunnel`, `--profile tunnel`, hoặc
> `COMPOSE_PROFILES=tunnel` trong `.env`.

## Thứ tự khởi động

```
postgres (healthy) -> migrate (chạy xong, exit 0)
  -> identity(-grpc)/task(-grpc)/workspace(-grpc)
  -> gateway identity/task/workspace (healthy, /healthz chỉ 200 khi gọi được gRPC health)
  -> frontend (healthy, /healthz do nginx trả)
  -> cloudflared (profile tunnel)
```

## Migration

Service `migrate` chạy `deployments/migrations/` một lần rồi thoát (one-shot).

```bash
make migrate-up        # chạy toàn bộ
make migrate-down      # rollback 1 bước
```

## Ghi chú

- `identity-grpc` nối Postgres qua `DATABASE_URL` (đã bật sẵn trong compose). Bỏ trống biến này
  thì service tự quay về repository in-memory stub và log cảnh báo. `task-grpc` và `workspace-grpc`
  vẫn in-memory.
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
| `PUBLIC_BASE_URL` | Origin công khai; nguồn suy ra 2 biến dưới (local: `http://localhost:3000`) |
| `GOOGLE_REDIRECT_URL` | Mặc định `${PUBLIC_BASE_URL}/api/identity/v1/auth/google/callback`; phải **trùng khít** Authorized redirect URI đã khai báo ở Google |
| `FRONTEND_BASE_URL` | Mặc định `${PUBLIC_BASE_URL}`; origin của SPA, backend redirect về đây kèm token ở fragment |
| `SESSION_SECRET` | Khoá ký JWT (HS256) và state OAuth, tối thiểu 16 ký tự |
| `SESSION_TTL` | Hạn của session token, mặc định `720h` (30 ngày) |
| `COOKIE_SECURE` | Đặt `true` khi chạy sau HTTPS |

`GOOGLE_REDIRECT_URL` mặc định trỏ vào **origin của frontend**, không phải `:8081`, để callback
đi qua nginx và giữ cùng origin với SPA.

Nếu chưa cấu hình, gateway vẫn khởi động bình thường nhưng log cảnh báo, và
`/v1/auth/google/login` trả mã lỗi `IDENTITY_AUTH_NOT_CONFIGURED`.

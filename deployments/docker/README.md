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

- Repository của 2 service Go hiện vẫn là **in-memory stub**, biến `DATABASE_URL` đã chừa sẵn
  trong compose (đang comment, ở container `*-grpc`) để nối vào khi implement repository DB.
- Image `migrate/migrate:latest` nên pin version khi dùng thật.

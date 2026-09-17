# deployments

Hạ tầng của hệ thống.

```
deployments/
├── docker/
│   ├── docker-compose.yml      # stack: postgres + migrate + identity + task + frontend
│   └── README.md
└── migrations/
    ├── 000001_init_identity.up.sql / .down.sql
    ├── 000002_init_task.up.sql     / .down.sql
    └── README.md
```

## Chạy

```bash
make up            # docker compose up --build -d (từ root repo)
make ps / make logs / make down

make migrate-up    # chạy toàn bộ migration
make migrate-down  # rollback 1 bước
```

Hoặc gọi trực tiếp:

```bash
docker compose -f deployments/docker/docker-compose.yml up --build -d
```

## Ghi chú

- Dockerfile **không** nằm ở đây mà ở từng service: `service/identity/Dockerfile`,
  `service/task/Dockerfile`, `frontend/Dockerfile`. Compose trỏ tới chúng với build context là
  **root repo** (`../..`) vì mỗi service Go cần copy thêm module `common/`.
- Volume migration mount `../migrations` (tức `deployments/migrations`) vào `/migrations` trong container.
- Chi tiết: `docker/README.md` và `migrations/README.md`.

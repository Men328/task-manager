# deployments

Hạ tầng của hệ thống.

```
deployments/
├── docker/
│   ├── docker-compose.yml      # stack: postgres + migrate + identity + task + mail-provider + frontend + cloudflared
│   ├── .env.example            # mẫu env (OAuth, session, Cloudflare Tunnel) -> copy sang .env
│   └── README.md
└── migrations/
    ├── 000001_init_identity.up.sql / .down.sql
    ├── 000002_init_task.up.sql     / .down.sql
    ├── 000005_init_mail_provider.up.sql / .down.sql
    └── README.md
```

## Chạy

```bash
make up            # docker compose up --build -d (từ root repo)
make up-tunnel     # kèm cloudflared (Cloudflare Tunnel; cần token trong deployments/docker/.env)
make ps / make logs / make down
make migrate-up    # chạy toàn bộ migration
make migrate-down  # rollback 1 bước
```

Hoặc gọi trực tiếp:

```bash
docker compose -f deployments/docker/docker-compose.yml up --build -d
docker compose -f deployments/docker/docker-compose.yml --profile tunnel up --build -d
```

## Ghi chú

- Dockerfile **không** nằm ở đây mà ở từng service: `service/identity/Dockerfile`,
  `service/task/Dockerfile`, `frontend/Dockerfile`. Compose trỏ tới chúng với build context là
  **root repo** (`../..`) vì mỗi service Go cần copy thêm module `common/`.
- Entrypoint duy nhất là `frontend` (nginx serve SPA + proxy `/api/*`), bind `127.0.0.1`.
  Cloudflare Tunnel (`cloudflared`, profile `tunnel`) trỏ vào `http://frontend:3000` nên chỉ
  một origin/port ra Internet. Chi tiết: `docker/README.md`.
- Volume migration mount `../migrations` (tức `deployments/migrations`) vào `/migrations` trong container.
- Chi tiết: `docker/README.md` và `migrations/README.md`.

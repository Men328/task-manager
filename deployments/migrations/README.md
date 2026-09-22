# migrations

Migration PostgreSQL, chạy bằng [golang-migrate](https://github.com/golang-migrate/migrate).

```
deployments/migrations/
├── 000001_init_identity.up.sql   / .down.sql   # schema identity: PROFILES, AUTH_PROVIDERS
├── 000002_init_task.up.sql       / .down.sql   # schema task: TASK_STATUSES, STATUS_TRANSITIONS, TASKS, TASK_STATUS_LOGS
├── 000003_init_workspace.up.sql  / .down.sql   # schema workspace: WORKSPACES (namespace gốc)
└── 000004_task_workspace_id.up.sql / .down.sql # TASKS.workspace_id + backfill workspace mặc định
```

## Quy ước

- Tên file: `{version}_{ten}.up.sql` / `.down.sql`, version 6 chữ số tăng dần.
- Mỗi cặp up/down phải **đối xứng** (down phải đưa DB về đúng trạng thái trước đó).
- Migration là **append-only**: đã chạy ở môi trường chung thì tạo file mới, không sửa file cũ.
- Mặc định mỗi file chạy trong 1 transaction. Cần chạy ngoài transaction (vd: `CREATE INDEX CONCURRENTLY`)
  thì thêm dòng `-- migrate: no-transaction` ở đầu file.

## Tên object thực tế trong DB

`design/db_schema.dbml` quy ước tên bảng viết HOA (`PROFILES`, `TASKS`), nhưng SQL trong migration
**không quote** identifier, nên PostgreSQL đã fold về chữ thường:

| DBML | Tên thật trong Postgres |
|---|---|
| `identity.PROFILES` | `identity.profiles` |
| `identity.AUTH_PROVIDERS` | `identity.auth_providers` |
| `task.TASKS` | `task.tasks` |
| `task.TASK_STATUSES` | `task.task_statuses` |
| `task.STATUS_TRANSITIONS` | `task.status_transitions` |
| `task.TASK_STATUS_LOGS` | `task.task_status_logs` |
| `workspace.WORKSPACES` | `workspace.workspaces` |

Cột, index và constraint cũng vậy (`uq_profiles_email`, `uq_auth_providers_provider_uid`, ...).

→ SQL trong code phải dùng **tên chữ thường** (`identity.profiles`), hoặc quote đúng chữ thường
(`identity."profiles"`). Viết `identity."PROFILES"` sẽ lỗi `relation does not exist`.
Xem `service/identity/internal/repository/postgres_profile_repository.go`.

Đừng sửa migration cũ sang dạng quote chữ HOA: phải rename table kèm mọi FK đang trỏ tới.

## Chạy

```bash
# qua docker compose (tự start postgres)
make migrate-up
make migrate-down

# hoặc bằng CLI migrate
migrate -path ./deployments/migrations -database "postgres://task_manager:task_manager@localhost:5432/task_manager?sslmode=disable" up
```

## Ánh xạ với thiết kế

Nguồn thiết kế: `design/db_schema.dbml` (dbdiagram.io). Các ràng buộc DBML không diễn đạt được
đã bổ sung trong SQL:

| Ràng buộc | Cách làm |
|---|---|
| Mỗi profile chỉ 1 status default | partial unique index `uq_task_statuses_one_default ... WHERE is_default AND NOT is_archived` |
| `from_status_id <> to_status_id` | `CHECK ck_status_transitions_not_self` |
| Status/transition/task phải cùng profile | composite FK tới `(id, profile_id)` |
| Chặn vòng lặp cây task | trigger `trg_tasks_prevent_cycle` |
| `updated_at` tự cập nhật | trigger `fn_set_updated_at` cho từng bảng |
| Email không phân biệt hoa/thường | unique index trên `lower(email)` cho profile chưa xoá mềm |

Yêu cầu: PostgreSQL >= 13 (`gen_random_uuid()` có sẵn trong core).

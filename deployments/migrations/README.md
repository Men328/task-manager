# migrations

Migration PostgreSQL, chạy bằng [golang-migrate](https://github.com/golang-migrate/migrate).

```
deployments/migrations/
├── 000001_init_identity.up.sql   / .down.sql   # schema identity: PROFILES, AUTH_PROVIDERS
├── 000002_init_task.up.sql       / .down.sql   # schema task: TASK_STATUSES, STATUS_TRANSITIONS, TASKS, TASK_STATUS_LOGS
├── 000005_init_mail_provider.up.sql / .down.sql # schema mail_provider: SESSIONS, NOTI_INDEXES
├── 000006_init_calendar.up.sql   / .down.sql   # schema calendar: SCHEDULES
├── 000007_init_event.up.sql      / .down.sql   # schema event: EVENTS
├── 000008_init_backlog.up.sql    / .down.sql   # schema backlog: BACKLOGS
└── 000009_init_notification.up.sql / .down.sql # schema notification: NOTICES
```

> Migration `000003`/`000004` (workspace) đã bị xoá cùng tính năng workspace. Repo không còn
> `workspace` schema; version `000005` giữ nguyên số cũ nên dãy version có lỗ hổng — golang-migrate
> chấp nhận điều này và chỉ chạy các file đang có theo thứ tự.

> Từ migration này, `mail-provider` lưu subscription (refresh token + checkpoint `historyId`) vào
> 2 bảng trên khi `DATABASE_URL` được set; không set thì quay về store in-memory.

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
| `calendar.SCHEDULES` | `calendar.schedules` |
| `event.EVENTS` | `event.events` |
| `backlog.BACKLOGS` | `backlog.backlogs` |
| `notification.NOTICES` | `notification.notices` |
| `mail_provider.SESSIONS` | `mail_provider.sessions` |
| `mail_provider.NOTI_INDEXES` | `mail_provider.noti_indexes` |

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
| `calendar`: `title` không rỗng | `CHECK ck_calendar_schedules_title_not_blank (btrim(title) <> '')` |
| `calendar`: `end_at` không sớm hơn `start_at` | `CHECK ck_calendar_schedules_time_range (end_at IS NULL OR end_at >= start_at)` |
| `calendar`: `color` đúng định dạng hex | `CHECK ck_calendar_schedules_color_format (color IS NULL OR color ~* '^#[0-9a-f]{6}([0-9a-f]{2})?$')` |
| `event`: `title` không rỗng | `CHECK ck_event_events_title_not_blank (btrim(title) <> '')` |
| `event`: `end_at` không sớm hơn `start_at` | `CHECK ck_event_events_time_range (end_at IS NULL OR end_at >= start_at)` |
| `event`: `color` đúng định dạng hex | `CHECK ck_event_events_color_format (color IS NULL OR color ~* '^#[0-9a-f]{6}([0-9a-f]{2})?$')` |
| `event`: `status` chỉ nhận giá trị đã biết | `CHECK ck_event_events_status (status IN ('PLANNED','CONFIRMED','CANCELLED'))` |
| `backlog`: `title` không rỗng | `CHECK ck_backlog_backlogs_title_not_blank (btrim(title) <> '')` |
| `backlog`: `status` chỉ nhận giá trị đã biết | `CHECK ck_backlog_backlogs_status (status IN ('NEW','TRIAGED','ARCHIVED'))` |
| `notification`: `title` không rỗng | `CHECK ck_notification_notices_title_not_blank (btrim(title) <> '')` |
| `notification`: `target_type` chỉ nhận giá trị đã biết | `CHECK ck_notification_notices_target_type (target_type IN ('task','schedule','event','backlog'))` |
| `notification`: notice chưa đọc thì `read_at` phải NULL | `CHECK ck_notification_notices_read_at (is_read OR read_at IS NULL)` |
| `mail_provider`: hộp thư đã ngắt kết nối không chặn hộp thư mới | partial unique index `uq_mail_provider_sessions_email ... (lower(email)) WHERE revoked_at IS NULL` |
| `mail_provider`: `history_id` là uint64 opaque, không dùng `bigint` | `varchar(32)` + `CHECK ck_mail_provider_noti_indexes_history_id_digits (history_id IS NULL OR history_id ~ '^[0-9]+$')` |
| `mail_provider`: 1 profile 1 kết nối / 1 checkpoint | `UNIQUE (profile_id, provider)` và `UNIQUE (profile_id)` |
| `mail_provider`: không lưu email/provider rỗng | `CHECK btrim(...) <> ''` cho `email`, `provider` |
| `mail_provider`: `refresh_token` có thể chưa có (Google không trả) | cột nullable, `NULL` thay cho chuỗi rỗng |

Yêu cầu: PostgreSQL >= 13 (`gen_random_uuid()` có sẵn trong core).

# 05 — Chi tiết gọi service & truy cập dữ liệu

> Tài liệu **chi tiết nhất** của bộ docs: service nào gọi service nào, gọi bằng RPC/API nào, qua
> địa chỉ/biến môi trường nào, và service nào đọc/ghi kho dữ liệu nào (PostgreSQL, S3, Kafka,
> Soketi, API bên ngoài).
>
> Nguồn: `common/proto/**/*.proto`, `service/*/internal/repository/*_client.go`,
> `service/*/cmd/http/*.go`, câu SQL trong `service/*/internal/repository/postgres_*.go`,
> `frontend/nginx.conf`, `deployments/docker/docker-compose.yml`.

## 0. Tóm tắt nhanh

| Kho dữ liệu | Có dùng? | Ai dùng |
|---|---|---|
| **PostgreSQL** | ✅ | identity, project, task, calendar, event, backlog, attachment, notification, mail-provider, social, worker (ghi) · message (chỉ đọc) |
| **Redis** | ❌ **Không dùng** | — (không có dependency, config hay container Redis nào) |
| **S3 (RustFS)** | ✅ | attachment, mail-provider |
| **Kafka** | ✅ | message (producer), worker (consumer) |
| **Soketi (Pusher REST)** | ✅ | notification, message (publish); identity (ký auth kênh); browser (WebSocket) |
| **Cache in-memory** | ✅ | mail-provider (queue notice, cache token khi thiếu DB); mọi service có repo in-memory khi thiếu `DATABASE_URL` |
| **API bên ngoài** | ✅ | identity (Google OAuth/OIDC), mail-provider (Gmail, Google token, Pub/Sub, DeepSeek) |

**Quy ước giao tiếp**

- Browser → service: **HTTP/JSON** qua nginx `/api/<svc>/...` → gateway HTTP (`<svc>:80xx`).
- Gateway HTTP → gRPC server cùng service: **gRPC** qua `GRPC_DIAL_ADDR` (`<svc>-grpc:90xx`).
- Service ↔ service: **chỉ gRPC** (plaintext, nội bộ), địa chỉ cấu hình bằng `<TARGET>_GRPC_DIAL_ADDR`.
- Mọi gateway có `GET /healthz` → gọi `grpc.health.v1.Health/Check` của gRPC server cùng service.

## 1. Ma trận gọi service ↔ service (gRPC nội bộ)

| # | Caller (process) | Callee | RPC | Khi nào | Cấu hình | Lỗi xử lý |
|---|---|---|---|---|---|---|
| 1 | identity (HTTP) | identity-grpc | `ProfileService.LoginWithProvider` | Google callback | `GRPC_DIAL_ADDR` | redirect lỗi về FE |
| 2 | identity (HTTP) | identity-grpc | `ProfileService.GetProfile` | `GET /v1/auth/me` | `GRPC_DIAL_ADDR` | trả lỗi |
| 3 | identity (HTTP) | mail-provider-grpc | `MailService.Subscribe` | Callback khi user tích "đọc Gmail" | `MAIL_GRPC_DIAL_ADDR=mail-provider-grpc:9084` | chỉ log, vẫn đăng nhập |
| 4 | identity-grpc | task-grpc | `TaskStatusService.SeedDefaultStatuses` | Tạo profile mới (`CreateProfile` / `LoginWithProvider`) | `TASK_GRPC_DIAL_ADDR=task-grpc:9082`, `TASK_TIMEOUT=10s` | best-effort, log |
| 5 | identity-grpc | project-grpc | `ProjectService.SeedDefaultProject` | Tạo profile mới | `PROJECT_GRPC_DIAL_ADDR=project-grpc:9090` | best-effort, log |
| 6 | task-grpc | project-grpc | `ProjectService.GetProject` | Create/Update task (kiểm tra project tồn tại, đúng profile, chưa archive) | `PROJECT_GRPC_DIAL_ADDR` | trả lỗi nghiệp vụ |
| 7 | task-grpc (ReminderWorker) | notification-grpc | `NoticeService.CreateNotice` (type `TASK_REMINDER`, target `TASK`) | Ticker định kỳ, task đến `remind_at` | `NOTIFICATION_GRPC_DIAL_ADDR=notification-grpc:9088`, `REMINDER_TIMEZONE` | release để thử lại |
| 8 | calendar-grpc | project-grpc | `ProjectService.GetProject` | Create/Update lịch | `PROJECT_GRPC_DIAL_ADDR` | trả lỗi |
| 9 | event-grpc | project-grpc | `ProjectService.GetProject` | Create/Update sự kiện | `PROJECT_GRPC_DIAL_ADDR` | trả lỗi |
| 10 | report-grpc | task-grpc | `TaskService.ListTasks` | Mọi API báo cáo task | `TASK_GRPC_DIAL_ADDR`, `TASK_TIMEOUT=10s` | trả lỗi |
| 11 | report-grpc | task-grpc | `TaskStatusService.ListTaskStatuses` | Overview / breakdown | như trên | trả lỗi |
| 12 | report-grpc | event-grpc | `EventService.ListEvents` | `GET /v1/reports/activity` | `EVENT_GRPC_DIAL_ADDR=event-grpc:9085`, `ACTIVITY_TIMEOUT=10s` | trả lỗi |
| 13 | report-grpc | calendar-grpc | `CalendarService.ListSchedules` | `GET /v1/reports/activity` | `CALENDAR_GRPC_DIAL_ADDR=calendar-grpc:9083` | trả lỗi |
| 14 | report-grpc | backlog-grpc | `BacklogService.ListBacklogs` | `GET /v1/reports/activity` | `BACKLOG_GRPC_DIAL_ADDR=backlog-grpc:9086` | trả lỗi |
| 15 | mail-provider (HTTP) | mail-provider-grpc | `MailService.HandleNotification` | Webhook Pub/Sub push | `GRPC_DIAL_ADDR` | 200 / không ack |
| 16 | mail-provider-grpc | project-grpc | `ProjectService.ListProjects` | Trước phân loại (lấy project ứng viên, bỏ default & archived) | `PROJECT_GRPC_DIAL_ADDR` | đẩy email vào backlog |
| 17 | mail-provider-grpc | task-grpc | `TaskService.CreateTask` | Category `task` | `TASK_GRPC_DIAL_ADDR`, `MAIL_TASK_TIMEOUT=15s` | giữ email trong backlog |
| 18 | mail-provider-grpc | calendar-grpc | `CalendarService.CreateSchedule` | Category `schedule` | `CALENDAR_GRPC_DIAL_ADDR`, `MAIL_SCHEDULE_TIMEOUT=15s` | giữ email trong backlog |
| 19 | mail-provider-grpc | event-grpc | `EventService.CreateEvent` | Category `event` | `EVENT_GRPC_DIAL_ADDR`, `MAIL_EVENT_TIMEOUT=15s` | giữ email trong backlog |
| 20 | mail-provider-grpc | backlog-grpc | `BacklogService.CreateBacklog` | Category `other` hoặc thiếu dữ kiện | `BACKLOG_GRPC_DIAL_ADDR`, `MAIL_BACKLOG_TIMEOUT=15s` | chỉ log (email chỉ còn bản gốc trên S3) |
| 21 | mail-provider-grpc | notification-grpc | `NoticeService.CreateNotice` (`TASK_CREATED` / `SCHEDULE_CREATED` / `EVENT_CREATED` / `BACKLOG_CREATED`) | Sau khi tạo thành công | `NOTIFICATION_GRPC_DIAL_ADDR`, `MAIL_NOTIFICATION_TIMEOUT=15s` | chỉ log |
| 22 | social-grpc | identity-grpc | `ProfileService.FindProfileByEmail` | `SearchByEmail` | `IDENTITY_GRPC_DIAL_ADDR=identity-grpc:9081`, `UPSTREAM_TIMEOUT=5s` | trả lỗi |
| 23 | social-grpc | identity-grpc | `ProfileService.BatchGetProfiles` | ListFriends, ListFriendRequests, SendFriendRequest, AcceptFriendRequest (lấy tên/email hiển thị) | như trên | trả lỗi |
| 24 | social-grpc | notification-grpc | `NoticeService.CreateNotice` (`FRIEND_REQUEST` / `FRIEND_ACCEPTED`, target `FRIEND`) | Gửi / chấp nhận lời mời | `NOTIFICATION_GRPC_DIAL_ADDR` | chỉ log |
| 25 | message-grpc | social-grpc | `FriendService.AreFriends` | Mỗi `SendMessage` | `SOCIAL_GRPC_DIAL_ADDR=social-grpc:9091`, `UPSTREAM_TIMEOUT=5s` | `MESSAGE_NOT_FRIENDS` |

**Các gateway HTTP → gRPC cùng service** (grpc-gateway sinh từ proto hoặc route viết tay) được
liệt kê đầy đủ ở mục 3.

> Ghi chú: compose khai báo `PROJECT_GRPC_DIAL_ADDR` cho `report-grpc` nhưng code `report` hiện
> **không tạo client project** (chỉ chuyển `project_id` làm filter sang task/event/calendar).
> `MailService.Unsubscribe` / `GetSubscription` có trong proto nhưng chưa có caller trong code.

### 1.1 Sơ đồ gọi gRPC (nhãn = RPC)

```mermaid
flowchart LR
    classDef svc fill:#dcfce7,stroke:#16a34a,color:#14532d
    ID["identity"]:::svc
    PRJ["project"]:::svc
    TASK["task"]:::svc
    CAL["calendar"]:::svc
    EVT["event"]:::svc
    BKL["backlog"]:::svc
    RPT["report"]:::svc
    NOTI["notification"]:::svc
    MAIL["mail-provider"]:::svc
    SOC["social"]:::svc
    MSG["message"]:::svc

    ID -->|"SeedDefaultStatuses"| TASK
    ID -->|"SeedDefaultProject"| PRJ
    ID -->|"Subscribe"| MAIL

    TASK -->|"GetProject"| PRJ
    TASK -->|"CreateNotice TASK_REMINDER"| NOTI
    CAL -->|"GetProject"| PRJ
    EVT -->|"GetProject"| PRJ

    RPT -->|"ListTasks, ListTaskStatuses"| TASK
    RPT -->|"ListEvents"| EVT
    RPT -->|"ListSchedules"| CAL
    RPT -->|"ListBacklogs"| BKL

    MAIL -->|"ListProjects"| PRJ
    MAIL -->|"CreateTask"| TASK
    MAIL -->|"CreateSchedule"| CAL
    MAIL -->|"CreateEvent"| EVT
    MAIL -->|"CreateBacklog"| BKL
    MAIL -->|"CreateNotice *_CREATED"| NOTI

    SOC -->|"FindProfileByEmail, BatchGetProfiles"| ID
    SOC -->|"CreateNotice FRIEND_*"| NOTI
    MSG -->|"AreFriends"| SOC
```

### 1.2 Fan-in / fan-out

| Service | Gọi ra (fan-out) | Bị gọi bởi (fan-in, ngoài gateway của chính nó) |
|---|---|---|
| identity | task, project, mail-provider, Google | social |
| project | — | identity, task, calendar, event, mail-provider |
| task | project, notification | identity, report, mail-provider |
| calendar | project | report, mail-provider |
| event | project | report, mail-provider |
| backlog | — | report, mail-provider |
| attachment | — (chỉ S3) | — |
| report | task, event, calendar, backlog | — |
| notification | — (chỉ Soketi) | task, mail-provider, social |
| mail-provider | project, task, calendar, event, backlog, notification, Gmail, DeepSeek, Pub/Sub | identity |
| social | identity, notification | message |
| message | social (+ Kafka, Soketi) | — |
| worker | — (Kafka, Postgres) | — |

## 2. Gọi ra hệ thống bên ngoài / hạ tầng qua HTTP

| Caller | Đích | Method + URL | Mục đích | Cấu hình |
|---|---|---|---|---|
| identity (HTTP) | Google OAuth | `GET https://accounts.google.com/o/oauth2/v2/auth` (redirect browser) | Bắt đầu đăng nhập; scope `openid email profile` (+ `gmail.readonly`, `access_type=offline`) | `GOOGLE_CLIENT_ID`, `GOOGLE_REDIRECT_URL` |
| identity (HTTP) | Google OAuth | `POST https://oauth2.googleapis.com/token` | Đổi `code` + PKCE verifier lấy token | `GOOGLE_CLIENT_SECRET` |
| identity (HTTP) | Google OIDC | `GET https://openidconnect.googleapis.com/v1/userinfo` | Lấy email, tên, avatar | — |
| mail-provider-grpc | Gmail API | `POST /gmail/v1/users/me/watch` | Đăng ký watch (topic Pub/Sub, label `INBOX`) | `GMAIL_BASE_URL`, `MAIL_PUBSUB_TOPIC`, `MAIL_WATCH_LABEL_IDS` |
| mail-provider-grpc | Gmail API | `POST /gmail/v1/users/me/stop` | Dừng watch | — |
| mail-provider-grpc | Gmail API | `GET /gmail/v1/users/me/history?startHistoryId=...` | Lấy thay đổi từ checkpoint | `GMAIL_TIMEOUT=20s` |
| mail-provider-grpc | Gmail API | `GET /gmail/v1/users/me/messages/{id}` | Lấy nội dung email | `GMAIL_MAX_BODY_BYTES` |
| mail-provider-grpc | Gmail API | `GET /gmail/v1/users/me/messages/{id}/attachments/{attId}` | Lấy attachment | `MAIL_MAX_ATTACHMENTS`, `MAIL_MAX_ATTACHMENT_BYTES` |
| mail-provider-grpc | Gmail API | `GET /gmail/v1/users/me/profile` | Đặt lại checkpoint khi history 404 | — |
| mail-provider-grpc | Google OAuth | `POST https://oauth2.googleapis.com/token` (grant `refresh_token`) | Làm mới access token Gmail | `GOOGLE_TOKEN_URL`, `GOOGLE_CLIENT_ID/SECRET` |
| mail-provider-grpc | Google Pub/Sub | `POST {MAIL_PUBSUB_BASE_URL}/v1/{subscription}:pull` và `:acknowledge` | Pull mode (chỉ khi set `MAIL_PULL_SUBSCRIPTION`), auth ADC scope `pubsub` | `GOOGLE_APPLICATION_CREDENTIALS`, `MAIL_PULL_*` |
| Google Pub/Sub | mail-provider (HTTP) | `POST /api/mail/v1/notifications` (qua cloudflared + nginx) | Push mode, kèm OIDC JWT | `MAIL_PUBSUB_AUDIENCE`, `MAIL_PUBSUB_SERVICE_ACCOUNT` |
| mail-provider-grpc | DeepSeek | `POST {DEEPSEEK_BASE_URL}/chat/completions` | Phân loại email (model `deepseek-chat`, trả JSON) | `DEEPSEEK_API_KEY`, `DEEPSEEK_TIMEOUT=60s` |
| notification-grpc | Soketi | `POST {SOKETI_BASE_URL}/apps/{SOKETI_APP_ID}/events` | Publish `notice.created` vào `private-noti-internal-<profile_id>` (ký HMAC app secret) | `SOKETI_*`, `SOKETI_TIMEOUT=10s` |
| message-grpc | Soketi | `POST {SOKETI_BASE_URL}/apps/{SOKETI_APP_ID}/events` | Publish `message.created` vào `private-chat-<recipient>` + `private-chat-<sender>` | `SOKETI_*`, `SOKETI_TIMEOUT=3s` |
| Browser | Soketi | WebSocket qua `/api/soketi/` | Nhận realtime notice + chat | `SOKETI_APP_KEY` (public) |
| Browser | identity (HTTP) | `POST /api/identity/v1/auth/soketi` (form `socket_id`, `channel_name`) | Xin chữ ký subscribe kênh private | `SOKETI_APP_KEY/SECRET`, prefix kênh |

## 3. API công khai theo service (Browser → nginx → gateway → gRPC)

Prefix nginx: `/api/<svc>/` → `http://<svc>:<port>/` (cắt prefix). Ví dụ
`GET /api/task/v1/tasks` → `task:8082/v1/tasks` → gRPC `TaskService.ListTasks` tại `task-grpc:9082`.

Cột **Auth**: `JWT` = gateway verify session token (Bearer) và lấy `profile_id` từ `sub`;
`profile_id` = client tự truyền `profile_id` trong query/body (gateway không verify token).

### 3.1 identity — `/api/identity` → `identity:8081` → `identity-grpc:9081`

| Method | Path | RPC / handler | Auth | Ghi chú |
|---|---|---|---|---|
| POST | `/v1/profiles` | `ProfileService.CreateProfile` | — | Tạo profile ⇒ seed status + project |
| GET | `/v1/profiles/{id}` | `ProfileService.GetProfile` | — | |
| GET | `/v1/profiles` | `ProfileService.ListProfiles` | — | |
| PATCH | `/v1/profiles/{id}` | `ProfileService.UpdateProfile` | — | |
| DELETE | `/v1/profiles/{id}` | `ProfileService.DeleteProfile` | — | Cascade dữ liệu mọi schema |
| GET | `/v1/auth/google/login?gmail=0\|1` | route viết tay | — | Redirect Google |
| GET | `/v1/auth/google/callback` | route viết tay → `LoginWithProvider` (+ `mail.Subscribe`) | — | Redirect về FE kèm token |
| GET | `/v1/auth/me` | route viết tay → `GetProfile` | JWT | |
| POST | `/v1/auth/soketi` | route viết tay | JWT | Ký kênh private |
| — | `LoginWithProvider`, `FindProfileByEmail`, `BatchGetProfiles` | gRPC nội bộ | — | Không có HTTP binding |

### 3.2 project — `/api/project` → `project:8090` → `project-grpc:9090`

| Method | Path | RPC | Auth |
|---|---|---|---|
| POST | `/v1/projects` | `ProjectService.CreateProject` | profile_id |
| GET | `/v1/projects/{id}` | `GetProject` | — |
| GET | `/v1/projects?profile_id=` | `ListProjects` (lazy seed `Chung`) | profile_id |
| PATCH | `/v1/projects/{id}` | `UpdateProject` | — |
| POST | `/v1/projects/{id}/archive` | `ArchiveProject` | — |
| POST | `/v1/projects/{id}/unarchive` | `UnarchiveProject` | — |
| — | `SeedDefaultProject` | gRPC nội bộ (identity gọi) | — |

### 3.3 task — `/api/task` → `task:8082` → `task-grpc:9082`

| Method | Path | RPC | Gọi tiếp |
|---|---|---|---|
| POST | `/v1/tasks` | `TaskService.CreateTask` | project `GetProject` |
| GET | `/v1/tasks/{id}?include_subtasks=` | `GetTask` | — |
| GET | `/v1/tasks?profile_id=&project_id=&root_only=&status_id=&due_from=&due_to=&open_only=&query=&sort_field=&page_token=` | `ListTasks` | — |
| PATCH | `/v1/tasks/{id}` | `UpdateTask` | project `GetProject` (khi đổi project) |
| DELETE | `/v1/tasks/{id}` | `DeleteTask` | — |
| POST | `/v1/tasks/{id}/status` | `ChangeTaskStatus` | — (có thể sinh task lặp kế tiếp) |
| POST | `/v1/statuses` | `TaskStatusService.CreateTaskStatus` | — |
| GET | `/v1/statuses/{id}` | `GetTaskStatus` | — |
| GET | `/v1/statuses?profile_id=` | `ListTaskStatuses` | — |
| PATCH | `/v1/statuses/{id}` | `UpdateTaskStatus` | — |
| DELETE | `/v1/statuses/{id}` | `DeleteTaskStatus` | — |
| POST | `/v1/statuses/seed` | `SeedDefaultStatuses` | — |
| POST | `/v1/transitions` | `StatusTransitionService.CreateStatusTransition` | — |
| GET | `/v1/transitions/{id}` | `GetStatusTransition` | — |
| GET | `/v1/transitions?profile_id=&from_status_id=` | `ListStatusTransitions` | — |
| PATCH | `/v1/transitions/{id}` | `UpdateStatusTransition` | — |
| DELETE | `/v1/transitions/{id}` | `DeleteStatusTransition` | — |
| POST | `/v1/transitions/validate` | `ValidateStatusTransition` | — |

### 3.4 calendar — `/api/calendar` → `calendar:8083` → `calendar-grpc:9083`

| Method | Path | RPC | Gọi tiếp |
|---|---|---|---|
| POST | `/v1/schedules` | `CalendarService.CreateSchedule` | project `GetProject` |
| GET | `/v1/schedules/{id}` | `GetSchedule` | — |
| GET | `/v1/schedules?profile_id=&project_id=&from=&to=` | `ListSchedules` | — |
| PATCH | `/v1/schedules/{id}` | `UpdateSchedule` | project `GetProject` |
| DELETE | `/v1/schedules/{id}` | `DeleteSchedule` | — |

### 3.5 event — `/api/event` → `event:8085` → `event-grpc:9085`

| Method | Path | RPC | Gọi tiếp |
|---|---|---|---|
| POST | `/v1/events` | `EventService.CreateEvent` | project `GetProject` |
| GET | `/v1/events/{id}` | `GetEvent` | — |
| GET | `/v1/events?profile_id=&project_id=&from=&to=&status=` | `ListEvents` | — |
| PATCH | `/v1/events/{id}` | `UpdateEvent` | project `GetProject` |
| DELETE | `/v1/events/{id}` | `DeleteEvent` | — |

### 3.6 backlog — `/api/backlog` → `backlog:8086` → `backlog-grpc:9086`

| Method | Path | RPC |
|---|---|---|
| POST | `/v1/backlogs` | `BacklogService.CreateBacklog` |
| GET | `/v1/backlogs/{id}` | `GetBacklog` |
| GET | `/v1/backlogs?profile_id=&status=&category=` | `ListBacklogs` |
| PATCH | `/v1/backlogs/{id}` | `UpdateBacklog` |
| DELETE | `/v1/backlogs/{id}` | `DeleteBacklog` |

### 3.7 attachment — `/api/attachment` → `attachment:8089` → `attachment-grpc:9089`

| Method | Path | RPC / handler | Kho |
|---|---|---|---|
| POST | `/v1/attachments/upload` (multipart) | route viết tay → `UploadAttachment` (bytes) | S3 `PUT` + PG `INSERT` |
| GET | `/v1/attachments/{id}/download` | route viết tay → `GetAttachment` (kèm bytes) | PG `SELECT` + S3 `GET` |
| POST | `/v1/attachments` (JSON, base64) | `AttachmentService.UploadAttachment` | như upload |
| GET | `/v1/attachments?owner_type=&owner_id=` | `ListAttachments` | PG |
| GET | `/v1/attachments/{id}` | `GetAttachment` | PG + S3 |
| DELETE | `/v1/attachments/{id}` | `DeleteAttachment` | PG `DELETE` + S3 `REMOVE` |

### 3.8 report — `/api/report` → `report:8087` → `report-grpc:9087`

| Method | Path | RPC | Gọi tiếp (gRPC) |
|---|---|---|---|
| GET | `/v1/reports/overview` | `ReportService.GetReportOverview` | task `ListTasks`, `ListTaskStatuses` |
| GET | `/v1/reports/status-breakdown` | `GetStatusBreakdown` | task `ListTasks`, `ListTaskStatuses` |
| GET | `/v1/reports/timeseries?interval=` | `GetTaskTimeSeries` | task `ListTasks` |
| GET | `/v1/reports/activity` | `GetActivityReport` | event `ListEvents`, calendar `ListSchedules`, backlog `ListBacklogs` |

### 3.9 notification — `/api/notification` → `notification:8088` → `notification-grpc:9088`

| Method | Path | RPC | Auth |
|---|---|---|---|
| GET | `/v1/notices?limit=&unread_only=` | `NoticeService.ListNotices` | JWT |
| GET | `/v1/notices/unread-count` | `CountUnreadNotices` | JWT |
| POST | `/v1/notices/read` `{ids:[...] \| ["*"]}` | `MarkNoticesRead` | JWT |
| — | `CreateNotice` | gRPC nội bộ (task, mail-provider, social) | — |

### 3.10 social — `/api/social` → `social:8091` → `social-grpc:9091`

Proto không có HTTP binding; toàn bộ là route viết tay (`cmd/http/friend_routes.go`), bắt buộc JWT.

| Method | Path | RPC | Gọi tiếp |
|---|---|---|---|
| GET | `/v1/friends/search?email=` | `FriendService.SearchByEmail` | identity `FindProfileByEmail` |
| GET | `/v1/friends` | `ListFriends` | identity `BatchGetProfiles` |
| DELETE | `/v1/friends/{profile_id}` | `RemoveFriend` | — |
| GET | `/v1/friend-requests` | `ListFriendRequests` | identity `BatchGetProfiles` |
| POST | `/v1/friend-requests` | `SendFriendRequest` | identity `BatchGetProfiles`, notification `CreateNotice FRIEND_REQUEST` |
| POST | `/v1/friend-requests/{id}/accept` | `AcceptFriendRequest` | identity `BatchGetProfiles`, notification `CreateNotice FRIEND_ACCEPTED` |
| POST | `/v1/friend-requests/{id}/decline` | `DeclineFriendRequest` | — |
| DELETE | `/v1/friend-requests/{id}` | `CancelFriendRequest` | — |
| — | `AreFriends` | gRPC nội bộ (message gọi) | — |

### 3.11 message — `/api/message` → `message:8092` → `message-grpc:9092`

Route viết tay (`cmd/http/message_routes.go`), bắt buộc JWT.

| Method | Path | RPC | Gọi tiếp |
|---|---|---|---|
| GET | `/v1/conversations` | `MessageService.ListConversations` | PG read |
| GET | `/v1/conversations/{peer_id}/messages?before=&limit=50` | `ListMessages` | PG read |
| POST | `/v1/conversations/{peer_id}/messages` `{body, client_id}` | `SendMessage` | social `AreFriends` → Soketi publish → Kafka produce |

### 3.12 mail-provider — `/api/mail` → `mail-provider:8084` → `mail-provider-grpc:9084`

| Method | Path | RPC | Ai gọi |
|---|---|---|---|
| POST | `/v1/notifications` (`MAIL_NOTIFICATION_PATH`) | `MailService.HandleNotification` | Google Pub/Sub push |
| — | `Subscribe` | gRPC nội bộ | identity (HTTP) |
| — | `Unsubscribe`, `GetSubscription` | gRPC nội bộ | (chưa có caller) |

### 3.13 worker — không có API

Chỉ `GET :8094/healthz` (ping Postgres). Không publish qua nginx.

## 4. Truy cập dữ liệu theo service

Ký hiệu: **C** INSERT · **R** SELECT · **U** UPDATE · **D** DELETE.
Tất cả dùng chung 1 Postgres `postgres:5432/task_manager` qua `pgxpool` (biến `DATABASE_URL`);
mỗi service **chỉ chạm schema của mình** (đã kiểm tra câu SQL trong code, không tính file test).

### 4.1 Bảng tổng hợp

| Service | PostgreSQL (schema.bảng: thao tác) | S3 / RustFS | Kafka | Soketi | Redis |
|---|---|---|---|---|---|
| identity | `identity.profiles`: CRUD · `identity.auth_providers`: C R D | — | — | — (chỉ ký auth kênh, không gọi Soketi) | ❌ |
| project | `project.projects`: C R U | — | — | — | ❌ |
| task | `task.tasks`: C R U D · `task.task_statuses`: C R U D · `task.status_transitions`: C R U D · `task.task_status_logs`: C R | — | — | — | ❌ |
| calendar | `calendar.schedules`: C R U D | — | — | — | ❌ |
| event | `event.events`: C R U D | — | — | — | ❌ |
| backlog | `backlog.backlogs`: C R U D | — | — | — | ❌ |
| attachment | `attachment.attachments`: C R D | bucket `task-manager`: PUT / GET / REMOVE | — | — | ❌ |
| report | **không có DB** | — | — | — | ❌ |
| notification | `notification.notices`: C R U | — | — | publish `notice.created` | ❌ |
| mail-provider | `mail_provider.sessions`: C R U (UPSERT) · `mail_provider.noti_indexes`: C R U D (UPSERT) | bucket `task-manager`: PUT email JSON + attachment; `EnsureBucket` lúc khởi động | — | — | ❌ |
| social | `social.friendships`: C R U D | — | — | — | ❌ |
| message | `message.conversations`: R · `message.messages`: R (**chỉ đọc**) | — | **produce** `chat.messages` | publish `message.created` | ❌ |
| worker | `message.messages`: C (`ON CONFLICT DO NOTHING`) · `message.conversations`: C/U (UPSERT) | — | **consume** `chat.messages` (group `message-worker`) | — | ❌ |
| frontend (nginx) | — | — | — | WebSocket client | ❌ |

### 4.2 Sơ đồ service → kho dữ liệu

```mermaid
flowchart LR
    classDef svc fill:#dcfce7,stroke:#16a34a,color:#14532d
    classDef db fill:#dbeafe,stroke:#2563eb,color:#1e3a8a
    classDef other fill:#fef3c7,stroke:#d97706,color:#78350f

    subgraph PG["PostgreSQL task_manager"]
        s_id[("identity.profiles<br/>identity.auth_providers")]:::db
        s_prj[("project.projects")]:::db
        s_task[("task.tasks<br/>task.task_statuses<br/>task.status_transitions<br/>task.task_status_logs")]:::db
        s_cal[("calendar.schedules")]:::db
        s_evt[("event.events")]:::db
        s_bkl[("backlog.backlogs")]:::db
        s_att[("attachment.attachments")]:::db
        s_noti[("notification.notices")]:::db
        s_mail[("mail_provider.sessions<br/>mail_provider.noti_indexes")]:::db
        s_soc[("social.friendships")]:::db
        s_msg[("message.conversations<br/>message.messages")]:::db
    end

    S3[("RustFS S3<br/>bucket task-manager")]:::other
    K[["Kafka chat.messages"]]:::other
    SK{{"Soketi"}}:::other
    RD["Redis: không sử dụng"]:::other

    ID["identity"]:::svc --> s_id
    PRJ["project"]:::svc --> s_prj
    TASK["task"]:::svc --> s_task
    CAL["calendar"]:::svc --> s_cal
    EVT["event"]:::svc --> s_evt
    BKL["backlog"]:::svc --> s_bkl
    ATT["attachment"]:::svc --> s_att
    ATT --> S3
    NOTI["notification"]:::svc --> s_noti
    NOTI --> SK
    MAIL["mail-provider"]:::svc --> s_mail
    MAIL --> S3
    SOC["social"]:::svc --> s_soc
    MSG["message"]:::svc -. "read only" .-> s_msg
    MSG --> K
    MSG --> SK
    K --> WRK["worker"]:::svc
    WRK -->|"write"| s_msg
    RPT["report"]:::svc -. "không có DB, chỉ gRPC" .-> TASK
```

### 4.3 Chi tiết bảng (cột quan trọng, index, ràng buộc)

| Bảng | Owner | Cột chính | Ràng buộc / index đáng chú ý | Migration |
|---|---|---|---|---|
| `identity.profiles` | identity | id, email, password_hash, display_name, avatar_url, timezone, locale, is_active, last_login_at, deleted_at, timestamps | `uq_profiles_email`, `ix_profiles_active` | 000001 |
| `identity.auth_providers` | identity | profile_id, provider, provider_user_id | FK profile CASCADE, `ix_auth_providers_profile` | 000001 |
| `project.projects` | project | id, profile_id, name, is_default, is_archived | `uq_projects_one_default`, `uq_projects_profile_name` (lower), `ix_project_projects_profile_archived` | 000011 |
| `task.task_statuses` | task | id, profile_id, name, slug, color, category, is_default, is_terminal, position | `uq_task_statuses_one_default`, `ix_task_statuses_profile_position` | 000002 |
| `task.status_transitions` | task | profile_id, from_status_id, to_status_id, is_active | FK (status, profile) CASCADE, `ix_status_transitions_profile_from` | 000002 |
| `task.tasks` | task | id, profile_id, project_id, parent_task_id, status_id, title, description, priority, start_at, due_at, completed_at, remind_at, reminded_at, recurrence_rule/tz/series_id/next_id, is_archived | FK status RESTRICT, FK parent, FK project RESTRICT; index profile+status, parent+position, profile+due, profile+active, profile+project, reminder due, recurrence series, **trigram search** | 000002, 000011–000014 |
| `task.task_status_logs` | task | task_id, profile_id, from/to_status_id, note, changed_at | FK task CASCADE | 000002 |
| `calendar.schedules` | calendar | id, profile_id, project_id, title, description, location, start_at, end_at, all_day, color | CHECK end ≥ start, FK project RESTRICT, index profile+start, profile+project | 000006, 000011 |
| `event.events` | event | id, profile_id, project_id, title, start_at, end_at, all_day, location, status, source | CHECK end ≥ start, FK project RESTRICT, index profile+start | 000007, 000011 |
| `backlog.backlogs` | backlog | id, profile_id, title, description, sender, source, category, reason, object_key, status | index profile+created, source | 000008 |
| `attachment.attachments` | attachment | id, profile_id, owner_type, owner_id, file_name, content_type, size, object_key | index owner, profile | 000010 |
| `notification.notices` | notification | id, profile_id, type, title, body, target_type, target_id, source, is_read, read_at, timestamps | CHECK target_type (thêm FRIEND ở 000016), index profile+unread, profile+created, target | 000009, 000016 |
| `mail_provider.sessions` | mail-provider | profile_id, provider, email, access_token, refresh_token, access_token_expires_at, revoked_at | `uq_mail_provider_sessions_email`, index profile; UPSERT theo `(profile_id, provider)` | 000005 |
| `mail_provider.noti_indexes` | mail-provider | profile_id, history_id (checkpoint), watch_expires_at, last_notice_at | index watch_expires (cho renewer); UPSERT theo `profile_id` | 000005 |
| `social.friendships` | social | requester_id, addressee_id, status (`pending`/`accepted`), accepted_at | CHECK not self, `uq_friendships_pair` (cặp không thứ tự), index requester/addressee + status | 000015 |
| `message.conversations` | worker (ghi), message (đọc) | id (UUIDv5 cặp), member_a < member_b, last_message_id, last_message_at | `uq_conversations_pair`, index member + last_message_at | 000017 |
| `message.messages` | worker (ghi), message (đọc) | id (UUIDv7), conversation_id, sender_id, recipient_id, body, client_id, created_at, persisted_at | FK conversation CASCADE, index conversation + created_at desc | 000017 |

### 4.4 Object storage (RustFS / S3)

| Service | Bucket | Object key | Thao tác | SDK / cấu hình |
|---|---|---|---|---|
| mail-provider | `task-manager` (`S3_BUCKET`) | `mail/{profile_id}/{yyyymmdd}/{message_id}.json` | PUT (email gốc) | `minio-go`, `S3_ENDPOINT=rustfs:9000`, `S3_ACCESS_KEY/SECRET_KEY`, `S3_USE_SSL`, `S3_TIMEOUT=30s` |
| mail-provider | `task-manager` | `mail/{profile_id}/{yyyymmdd}/{message_id}/attachments/{filename}` | PUT (≤ 5 tệp, ≤ 10 MiB) | như trên; tắt bằng `MAIL_ARCHIVE_ENABLED=false` |
| attachment | `task-manager` | `attachments/{profile_id}/{owner_type}/{owner_id}/{uuid}{ext}` (lưu ở `attachment.attachments.object_key`) | PUT / GET / REMOVE (xoá metadata trước, xoá object best-effort) | `minio-go`; thiếu S3 ⇒ blob store in-memory |

`object_key` của email gốc được truyền sang `backlog.backlogs.object_key` để truy vết.

### 4.5 Kafka

| Thuộc tính | Giá trị |
|---|---|
| Broker | `kafka:9092` (nội bộ), `127.0.0.1:29092` (host, cho `make run-*`) |
| Topic | `chat.messages` (`KAFKA_MESSAGE_TOPIC`), tạo bởi `kafka-init`, `KAFKA_MESSAGE_PARTITIONS=6` |
| Producer | `message-grpc` — async, batch 10ms, `acks=all`, retry 10 lần; key = `conversation_id`; value = protojson `ChatMessage` |
| Consumer | `worker` (N replica, `WORKER_REPLICAS=2`) — group `message-worker`; commit offset **sau** khi ghi DB |
| Đảm bảo | At-least-once, idempotent theo `id`, thứ tự theo conversation (cùng partition) |
| Lỗi | Lỗi tạm thời ⇒ backoff `500ms → 30s`, không commit; payload hỏng / lỗi PG lớp 22, 23 ⇒ log ERROR + commit |

### 4.6 Soketi (realtime)

| Publisher | Kênh | Event | Payload dùng ở FE |
|---|---|---|---|
| notification-grpc | `private-noti-internal-<profile_id>` (`SOKETI_CHANNEL_PREFIX=noti-internal-`) | `notice.created` | Gọi lại `GET /v1/notices/unread-count` để cập nhật badge |
| message-grpc | `private-chat-<recipient_id>` và `private-chat-<sender_id>` (`SOKETI_CHAT_CHANNEL_PREFIX=chat-`) | `message.created` | Thêm tin vào khung chat, khớp `client_id` |

Uỷ quyền subscribe: browser → `POST /api/identity/v1/auth/soketi` → identity trả
`auth = app_key:HMAC_SHA256(app_secret, socket_id:channel_name)`, chỉ cho kênh của chính `sub`.

### 4.7 State in-memory (không bền)

| Service | Thành phần | Ghi chú |
|---|---|---|
| mail-provider-grpc | Queue notice (`MAIL_QUEUE_SIZE=256`), `MAIL_WORKER_COUNT=2` worker | Queue đầy ⇒ không ack Pub/Sub để gửi lại |
| mail-provider-grpc | Subscription/token store khi thiếu `DATABASE_URL` | Restart là mất watch |
| task-grpc | ReminderWorker (ticker) | Trạng thái claim lưu ở PG (`reminded_at`), không ở RAM |
| mọi service có DB | Repository in-memory khi thiếu `DATABASE_URL` | Chỉ dùng dev/test |
| attachment-grpc | Blob store in-memory khi thiếu S3 | Chỉ dùng dev/test |

## 5. Luồng end-to-end chi tiết (từng bước, từng kho)

### 5.1 Tạo task từ board

| Bước | Từ | Đến | Giao thức / API | Kho |
|---|---|---|---|---|
| 1 | Browser | nginx | `POST /api/task/v1/tasks` | — |
| 2 | nginx | task:8082 | HTTP `POST /v1/tasks` | — |
| 3 | task gateway | task-grpc:9082 | gRPC `TaskService.CreateTask` | — |
| 4 | task-grpc | project-grpc:9090 | gRPC `ProjectService.GetProject` | project-grpc: R `project.projects` |
| 5 | task-grpc | Postgres | — | R `task.task_statuses` (default), C `task.tasks`, C `task.task_status_logs` |

### 5.2 Email → task (mail pipeline)

| Bước | Từ | Đến | Giao thức / API | Kho |
|---|---|---|---|---|
| 1 | Pub/Sub | mail-provider:8084 | `POST /v1/notifications` (OIDC) | — |
| 2 | gateway | mail-provider-grpc | gRPC `HandleNotification` | queue RAM |
| 3 | worker | Postgres | — | R `mail_provider.sessions`, `noti_indexes` |
| 4 | worker | Google | `POST oauth2.googleapis.com/token` (nếu hết hạn) | U `mail_provider.sessions` |
| 5 | worker | Gmail | `GET history`, `GET messages/{id}`, `GET attachments/{id}` | — |
| 6 | worker | RustFS | S3 PUT | `mail/...json`, attachments |
| 7 | worker | project-grpc | gRPC `ListProjects` | R `project.projects` |
| 8 | worker | DeepSeek | `POST /chat/completions` | — |
| 9 | worker | task-grpc | gRPC `CreateTask` | task-grpc → project `GetProject`; C `task.tasks` |
| 10 | worker | notification-grpc | gRPC `CreateNotice TASK_CREATED` | C `notification.notices` |
| 11 | notification-grpc | Soketi | `POST /apps/{id}/events` | — |
| 12 | worker | Postgres | — | U checkpoint `mail_provider.noti_indexes` |

### 5.3 Gửi tin nhắn

| Bước | Từ | Đến | Giao thức / API | Kho |
|---|---|---|---|---|
| 1 | Browser | message:8092 | `POST /v1/conversations/{peer}/messages` (JWT) | — |
| 2 | gateway | message-grpc | gRPC `SendMessage` | — |
| 3 | message-grpc | social-grpc | gRPC `AreFriends` | R `social.friendships` |
| 4 | message-grpc | Soketi | `POST /apps/{id}/events` `message.created` ×2 kênh | — |
| 5 | message-grpc | Kafka | produce `chat.messages` | log Kafka |
| 6 | worker | Kafka | consume | — |
| 7 | worker | Postgres | — | C `message.messages`, UPSERT `message.conversations` |
| 8 | Browser | message:8092 | `GET /v1/conversations/...` | message-grpc: R `message.*` |

### 5.4 Báo cáo hoạt động

| Bước | Từ | Đến | Giao thức / API | Kho |
|---|---|---|---|---|
| 1 | Browser | report:8087 | `GET /v1/reports/activity?profile_id=&from=&to=` | — |
| 2 | gateway | report-grpc | gRPC `GetActivityReport` | — |
| 3 | report-grpc | event-grpc | gRPC `ListEvents` | R `event.events` |
| 4 | report-grpc | calendar-grpc | gRPC `ListSchedules` | R `calendar.schedules` |
| 5 | report-grpc | backlog-grpc | gRPC `ListBacklogs` | R `backlog.backlogs` |
| 6 | report-grpc | — | tổng hợp trong RAM | — |

## 6. Bảng địa chỉ & biến môi trường (Docker Compose)

| Service | Container HTTP | Container gRPC | Biến dial ra ngoài |
|---|---|---|---|
| identity | `identity:8081` | `identity-grpc:9081` | `TASK_GRPC_DIAL_ADDR`, `PROJECT_GRPC_DIAL_ADDR` (grpc) · `MAIL_GRPC_DIAL_ADDR` (http) |
| task | `task:8082` | `task-grpc:9082` | `PROJECT_GRPC_DIAL_ADDR`, `NOTIFICATION_GRPC_DIAL_ADDR` |
| calendar | `calendar:8083` | `calendar-grpc:9083` | `PROJECT_GRPC_DIAL_ADDR` |
| mail-provider | `mail-provider:8084` | `mail-provider-grpc:9084` | `PROJECT_`, `TASK_`, `CALENDAR_`, `EVENT_`, `BACKLOG_`, `NOTIFICATION_GRPC_DIAL_ADDR`, `S3_ENDPOINT` |
| event | `event:8085` | `event-grpc:9085` | `PROJECT_GRPC_DIAL_ADDR` |
| backlog | `backlog:8086` | `backlog-grpc:9086` | — |
| report | `report:8087` | `report-grpc:9087` | `TASK_`, `EVENT_`, `CALENDAR_`, `BACKLOG_GRPC_DIAL_ADDR` (+ `PROJECT_` khai báo nhưng không dùng) |
| notification | `notification:8088` | `notification-grpc:9088` | `SOKETI_BASE_URL` |
| attachment | `attachment:8089` | `attachment-grpc:9089` | `S3_ENDPOINT` |
| project | `project:8090` | `project-grpc:9090` | — |
| social | `social:8091` | `social-grpc:9091` | `IDENTITY_GRPC_DIAL_ADDR`, `NOTIFICATION_GRPC_DIAL_ADDR` |
| message | `message:8092` | `message-grpc:9092` | `SOCIAL_GRPC_DIAL_ADDR`, `SOKETI_BASE_URL`, `KAFKA_BROKERS` |
| worker | health `:8094` | — | `KAFKA_BROKERS`, `DATABASE_URL` |

Mọi service có DB dùng chung
`DATABASE_URL=postgres://task_manager:***@postgres:5432/task_manager?sslmode=disable`.
`SESSION_SECRET` phải giống nhau giữa identity, notification, social, message.

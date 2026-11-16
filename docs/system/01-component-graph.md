# 01 — Graph các thành phần dự án

> Tài liệu tổng hợp từ việc quét mã nguồn (`service/*`, `common/proto`, `frontend/src`,
> `deployments/`). Mọi sơ đồ dùng **Mermaid** — mở bằng VS Code (extension *Markdown Preview
> Mermaid Support*), GitHub, hoặc dán vào <https://mermaid.live>.
>
> Chi tiết "service nào gọi service nào, bằng API nào, dùng DB gì" xem
> [05-service-calls-and-data.md](05-service-calls-and-data.md).

## 1. Bức tranh tổng thể

Hệ thống **Task Manager** là một monorepo gồm:

| Nhóm | Thành phần | Ghi chú |
|---|---|---|
| Frontend | `frontend/` (React + TS + Mantine + Vite, chạy sau nginx) | Entry point duy nhất publish ra host (`:3000`) |
| Backend (Go, 13 service) | `identity`, `task`, `project`, `calendar`, `event`, `backlog`, `attachment`, `report`, `notification`, `mail-provider`, `social`, `message`, `worker` | Mỗi service = 1 Go module, thường có 2 binary: `cmd/grpc` + `cmd/http` (grpc-gateway) |
| Shared | `common/` (proto, code gen, bộ mã lỗi chuẩn) | `common/proto/<svc>/v1/*.proto` là interface duy nhất giữa các service |
| Hạ tầng | PostgreSQL, Kafka, Soketi, RustFS (S3), Cloudflared | Định nghĩa trong `deployments/docker/docker-compose.yml` |
| Bên ngoài | Google OAuth, Gmail API, Google Pub/Sub, DeepSeek LLM | Chỉ `identity` và `mail-provider` gọi ra ngoài |

> **Không có Redis** trong dự án. Mọi trạng thái bền vững nằm ở PostgreSQL (mỗi service 1 schema),
> nội dung file nằm ở RustFS (S3), luồng sự kiện nằm ở Kafka. Cache/queue tạm nằm trong RAM của
> process (vd queue notice của `mail-provider`).

## 2. Graph thành phần (toàn hệ thống)

```mermaid
graph LR
    classDef fe fill:#dbeafe,stroke:#2563eb,color:#1e3a8a
    classDef svc fill:#dcfce7,stroke:#16a34a,color:#14532d
    classDef infra fill:#fef3c7,stroke:#d97706,color:#78350f
    classDef ext fill:#fce7f3,stroke:#db2777,color:#831843

    User(("Người dùng<br/>(browser)"))
    FE["frontend<br/>React SPA + nginx :3000"]:::fe

    subgraph Core["Nghiệp vụ lõi"]
        ID["identity<br/>hồ sơ, đăng nhập, session JWT"]:::svc
        PRJ["project<br/>nhóm công việc"]:::svc
        TASK["task<br/>task, status, lifecycle,<br/>nhắc việc, lặp lại"]:::svc
        CAL["calendar<br/>lịch cá nhân"]:::svc
        EVT["event<br/>sự kiện"]:::svc
        BKL["backlog<br/>email chưa phân loại"]:::svc
        ATT["attachment<br/>tệp đính kèm"]:::svc
        RPT["report<br/>báo cáo (read-only)"]:::svc
    end

    subgraph Comm["Giao tiếp & thông báo"]
        NOTI["notification<br/>thông báo + realtime"]:::svc
        SOC["social<br/>bạn bè"]:::svc
        MSG["message<br/>nhắn tin 1-1"]:::svc
        WRK["worker<br/>Kafka consumer"]:::svc
    end

    MAIL["mail-provider<br/>Gmail -> phân loại -> dispatch"]:::svc

    PG[("PostgreSQL")]:::infra
    S3[("RustFS (S3)")]:::infra
    KAFKA[["Kafka<br/>topic chat.messages"]]:::infra
    SOK{{"Soketi<br/>(WebSocket / Pusher)"}}:::infra

    GOOG["Google OAuth"]:::ext
    GMAIL["Gmail API"]:::ext
    PUBSUB["Google Pub/Sub"]:::ext
    DS["DeepSeek LLM"]:::ext

    User --> FE
    User -. WebSocket .-> SOK
    FE -->|"/api/*"| ID & TASK & PRJ & CAL & EVT & BKL & ATT & RPT & NOTI & SOC & MSG
    FE -->|"/api/soketi"| SOK

    ID -->|seed status| TASK
    ID -->|seed project| PRJ
    ID -->|Subscribe| MAIL
    ID --> GOOG

    TASK -->|GetProject| PRJ
    CAL -->|GetProject| PRJ
    EVT -->|GetProject| PRJ
    TASK -->|CreateNotice nhắc việc| NOTI

    RPT --> TASK & CAL & EVT & BKL

    MAIL --> PRJ
    MAIL -->|CreateTask| TASK
    MAIL -->|CreateSchedule| CAL
    MAIL -->|CreateEvent| EVT
    MAIL -->|CreateBacklog| BKL
    MAIL -->|CreateNotice| NOTI
    MAIL --> GMAIL & DS & S3
    PUBSUB -->|push webhook| MAIL
    MAIL -. pull .-> PUBSUB
    GMAIL -. watch .-> PUBSUB

    SOC -->|FindProfileByEmail / BatchGetProfiles| ID
    SOC -->|CreateNotice| NOTI
    MSG -->|AreFriends| SOC
    MSG -->|produce| KAFKA
    MSG -->|broadcast| SOK
    KAFKA -->|consume| WRK
    NOTI -->|publish| SOK

    ATT --> S3
    ID & TASK & PRJ & CAL & EVT & BKL & ATT & NOTI & SOC & MAIL & WRK --> PG
    MSG -->|read only| PG
```

## 3. Graph phụ thuộc theo tầng (ai phụ thuộc ai)

Mũi tên `A --> B` = **A cần B** để hoạt động đầy đủ. Service càng ở dưới càng "gốc".

```mermaid
graph TD
    FE["frontend"]
    MAIL["mail-provider"]
    RPT["report"]
    MSG["message"]
    WRK["worker"]
    SOC["social"]
    ID["identity"]
    TASK["task"]
    CAL["calendar"]
    EVT["event"]
    BKL["backlog"]
    ATT["attachment"]
    NOTI["notification"]
    PRJ["project"]

    FE --> ID & TASK & PRJ & CAL & EVT & BKL & ATT & RPT & NOTI & SOC & MSG
    MAIL --> TASK & CAL & EVT & BKL & NOTI & PRJ
    RPT --> TASK & CAL & EVT & BKL
    MSG --> SOC
    MSG -. "Kafka (async)" .-> WRK
    SOC --> ID & NOTI
    ID --> TASK & PRJ & MAIL
    TASK --> PRJ & NOTI
    CAL --> PRJ
    EVT --> PRJ
```

Nhận xét:

- **`project`** và **`backlog`**, **`attachment`**, **`notification`** là service "lá" ở tầng gRPC
  (không gọi service nội bộ nào khác; `notification` chỉ gọi Soketi).
- **`identity` ↔ `mail-provider`** là phụ thuộc hai chiều về mặt luồng nghiệp vụ
  (identity gọi `Subscribe`; mail-provider không gọi lại identity mà chỉ dùng token được truyền sang).
- **`report`** không có DB — chỉ tổng hợp qua gRPC.
- **`worker`** không có API — chỉ consume Kafka và ghi Postgres.

## 4. Chức năng từng thành phần

### 4.1 `frontend`

| Trang (route) | Chức năng | Service gọi |
|---|---|---|
| `/login`, `/auth/callback` | Đăng nhập Google, tuỳ chọn "Cho phép đọc Gmail" | identity |
| `/` (Board) | Kanban/Table/List/Timeline task, kéo thả đổi status, chi tiết task, task con, đính kèm | task, project, attachment |
| `/today` (Agenda) | Việc cần làm hôm nay / quá hạn / sắp tới, đổi status nhanh | task, project |
| `/projects` | Quản lý project (tạo, đổi tên, archive/unarchive) | project |
| `/calendar` | Lịch FullCalendar, CRUD lịch | calendar, attachment |
| `/events` | Bảng CRUD sự kiện | event, attachment |
| `/backlog` | Email chưa phân loại, triage | backlog, attachment |
| `/reports` | Thẻ tổng quan, donut status, line chart, hoạt động | report |
| `/friends` | Tìm bạn theo email, lời mời kết bạn | social |
| `/messages`, `/messages/:peerId` | Chat 1-1 realtime | message, soketi |
| `/statuses` | Status & lifecycle (đã ẩn khỏi menu) | task |
| Topbar | Chuông thông báo, badge chưa đọc, realtime | notification, soketi, identity (`/v1/auth/soketi`) |

### 4.2 Backend services

| Service | Port gRPC / HTTP | Chức năng chính | Lưu trữ |
|---|---|---|---|
| **identity** | 9081 / 8081 | Hồ sơ (profile), liên kết provider, OAuth Google, phát **session JWT HS256**, ký uỷ quyền kênh Soketi, seed dữ liệu mặc định cho profile mới | PG `identity.profiles`, `identity.auth_providers` |
| **project** | 9090 / 8090 | Project theo profile; project mặc định `Chung` (không archive được); archive/unarchive | PG `project.projects` |
| **task** | 9082 / 8082 | Task cha/con, status theo profile, lifecycle allowlist, audit log, tìm kiếm (trigram), **nhắc việc** (worker nền), **task lặp lại** (RRULE) | PG `task.tasks`, `task.task_statuses`, `task.status_transitions`, `task.task_status_logs` |
| **calendar** | 9083 / 8083 | Lịch cá nhân (khối thời gian, all-day, màu) | PG `calendar.schedules` |
| **event** | 9085 / 8085 | Sự kiện có vòng đời `PLANNED → CONFIRMED → CANCELLED`, truy vết email nguồn | PG `event.events` |
| **backlog** | 9086 / 8086 | Email không phân loại được (`NEW/TRIAGED/ARCHIVED`) | PG `backlog.backlogs` |
| **attachment** | 9089 / 8089 | Tệp đính kèm đa hình `(owner_type, owner_id)`; upload multipart / download | PG `attachment.attachments` + S3 |
| **report** | 9087 / 8087 | Báo cáo tổng quan, status breakdown, time series, activity | Không có DB (gRPC) |
| **notification** | 9088 / 8088 | Lưu notice, đếm chưa đọc, đánh dấu đã đọc, push realtime | PG `notification.notices` + Soketi |
| **mail-provider** | 9084 / 8084 | Watch Gmail, nhận Pub/Sub, lưu email gốc S3, phân loại (DeepSeek / keyword), dispatch | PG `mail_provider.sessions`, `mail_provider.noti_indexes` + S3 |
| **social** | 9091 / 8091 | Kết bạn: tìm theo email, gửi/chấp nhận/từ chối/huỷ lời mời, huỷ kết bạn | PG `social.friendships` |
| **message** | 9092 / 8092 | Chat 1-1 giữa bạn bè: broadcast realtime + đẩy Kafka; đọc lịch sử | PG `message.*` (chỉ đọc) + Kafka + Soketi |
| **worker** | — / health 8094 | Consume `chat.messages`, ghi tin nhắn idempotent | PG `message.conversations`, `message.messages` |

### 4.3 Hạ tầng

| Thành phần | Dùng bởi | Vai trò |
|---|---|---|
| **PostgreSQL** `postgres:5432` (DB `task_manager`) | 11 service (trừ `report`; `message` chỉ đọc) | Mỗi service sở hữu 1 schema riêng |
| **migrate** | job khởi động | Chạy `deployments/migrations/*.sql` |
| **RustFS** `rustfs:9000` | `attachment`, `mail-provider` | Object storage S3-compatible, bucket `task-manager` |
| **Kafka** `kafka:9092` (`kafka-init` tạo topic) | `message` (producer), `worker` (consumer) | Topic `chat.messages`, 6 partition, key = `conversation_id` |
| **Soketi** `soketi:6001` | `notification`, `message` (publish REST); browser (WS) | Realtime theo giao thức Pusher |
| **Cloudflared** (profile `tunnel`) | — | Public HTTPS cho webhook Pub/Sub |

## 5. Liên hệ dữ liệu (khoá logic giữa các schema)

`profile_id` (uuid từ `identity.profiles`) là khoá xuyên suốt. Vì mọi schema nằm chung 1
database, migration **có khai báo FK chéo schema**:

- Hầu hết bảng có `profile_id REFERENCES identity.profiles(id) ON DELETE CASCADE` (xoá profile ⇒ xoá
  dữ liệu của profile đó ở mọi schema). Riêng `social.friendships` (`requester_id`, `addressee_id`)
  và `message.conversations` (`member_a`, `member_b`) tham chiếu 2 profile.
- `task.tasks`, `calendar.schedules`, `event.events` có FK `(project_id, profile_id) REFERENCES
  project.projects(id, profile_id) ON DELETE RESTRICT`.
- Ngoài FK, các service vẫn kiểm tra nghiệp vụ qua gRPC trước khi ghi (vd `task`/`calendar`/`event`
  gọi `project.GetProject` để chặn project đã archive hoặc của profile khác).
- `attachment.attachments (owner_type, owner_id)` là liên kết **đa hình**, không có FK tới đối tượng sở hữu.

```mermaid
erDiagram
    IDENTITY_PROFILES ||--o{ IDENTITY_AUTH_PROVIDERS : "liên kết"
    IDENTITY_PROFILES ||--o{ PROJECT_PROJECTS : "sở hữu"
    IDENTITY_PROFILES ||--o{ TASK_TASK_STATUSES : "bộ status riêng"
    IDENTITY_PROFILES ||--o{ TASK_STATUS_TRANSITIONS : "rule lifecycle"
    PROJECT_PROJECTS ||--o{ TASK_TASKS : "project_id"
    PROJECT_PROJECTS ||--o{ CALENDAR_SCHEDULES : "project_id"
    PROJECT_PROJECTS ||--o{ EVENT_EVENTS : "project_id"
    TASK_TASK_STATUSES ||--o{ TASK_TASKS : "status_id"
    TASK_TASKS ||--o{ TASK_TASKS : "parent_task_id"
    TASK_TASKS ||--o{ TASK_TASK_STATUS_LOGS : "audit"
    IDENTITY_PROFILES ||--o{ BACKLOG_BACKLOGS : "sở hữu"
    IDENTITY_PROFILES ||--o{ NOTIFICATION_NOTICES : "nhận"
    IDENTITY_PROFILES ||--o{ MAIL_PROVIDER_SESSIONS : "Gmail token"
    MAIL_PROVIDER_SESSIONS ||--|| MAIL_PROVIDER_NOTI_INDEXES : "checkpoint historyId"
    IDENTITY_PROFILES ||--o{ SOCIAL_FRIENDSHIPS : "requester / addressee"
    IDENTITY_PROFILES ||--o{ MESSAGE_CONVERSATIONS : "member_a / member_b"
    MESSAGE_CONVERSATIONS ||--o{ MESSAGE_MESSAGES : "chứa"
    ATTACHMENT_ATTACHMENTS }o--|| TASK_TASKS : "owner_type=task"
    ATTACHMENT_ATTACHMENTS }o--|| CALENDAR_SCHEDULES : "owner_type=schedule"
    ATTACHMENT_ATTACHMENTS }o--|| EVENT_EVENTS : "owner_type=event"
    ATTACHMENT_ATTACHMENTS }o--|| BACKLOG_BACKLOGS : "owner_type=backlog"
```

## 6. Cấu trúc bên trong 1 service (giống nhau cho mọi service)

```mermaid
graph LR
    subgraph HTTP["cmd/http (grpc-gateway)"]
        GW["REST /v1/* -> gRPC<br/>+ route viết tay (auth, upload, webhook)"]
    end
    subgraph GRPC["cmd/grpc (fx app)"]
        H["internal/handler<br/>transport gRPC mỏng"]
        D["internal/dependency<br/>validate, mapping, map lỗi"]
        S["internal/service<br/>nghiệp vụ + interfaces.go (port)"]
        M["internal/model<br/>core, chỉ stdlib"]
        R["internal/repository<br/>Postgres / in-memory / gRPC client / S3 / HTTP"]
    end
    GW -->|gRPC| H
    H --> D
    H --> S
    S --> M
    R -. "thoả mãn port (structural typing)" .-> S
    R --> M
    D --> M
```

Quy tắc kiến trúc được kiểm tra tự động bằng `make quality-arch` (`quality.json`):
hướng phụ thuộc, `model` là core, tầng phẳng, bắt buộc `service/interfaces.go`, `cmd` dùng `fx`.

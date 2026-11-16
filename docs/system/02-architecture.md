# 02 — Sơ đồ kiến trúc hệ thống

> Toàn bộ sơ đồ là **Mermaid**. Xem trực tiếp trên GitHub / VS Code (extension *Markdown Preview
> Mermaid Support*) / <https://mermaid.live>.

## 1. Kiến trúc triển khai (Docker Compose)

Chỉ container `frontend` (nginx) được publish ra host (`127.0.0.1:3000`). Mọi API đi qua
`/api/<service>/...` và được nginx forward nội bộ tới gateway HTTP của service. Mỗi service
chạy **2 container**: `<svc>` (grpc-gateway, HTTP) và `<svc>-grpc` (gRPC server, nghiệp vụ).

```mermaid
flowchart TB
    classDef edge fill:#e0e7ff,stroke:#4f46e5,color:#1e1b4b
    classDef gw fill:#ecfeff,stroke:#0891b2,color:#164e63
    classDef grpc fill:#dcfce7,stroke:#16a34a,color:#14532d
    classDef infra fill:#fef3c7,stroke:#d97706,color:#78350f
    classDef ext fill:#fce7f3,stroke:#db2777,color:#831843

    Browser["Browser (SPA)"]
    CF["cloudflared<br/>(profile tunnel)"]:::edge
    NGINX["frontend: nginx :80<br/>host 127.0.0.1:3000<br/>static SPA + reverse proxy /api/*"]:::edge

    Browser -->|HTTPS| CF --> NGINX
    Browser -->|"http://localhost:3000"| NGINX

    subgraph GW["HTTP gateways (grpc-gateway + route viết tay)"]
        idh["identity :8081"]:::gw
        tah["task :8082"]:::gw
        cah["calendar :8083"]:::gw
        mah["mail-provider :8084"]:::gw
        evh["event :8085"]:::gw
        bah["backlog :8086"]:::gw
        reh["report :8087"]:::gw
        noh["notification :8088"]:::gw
        ath["attachment :8089"]:::gw
        prh["project :8090"]:::gw
        soh["social :8091"]:::gw
        meh["message :8092"]:::gw
    end

    NGINX -->|/api/identity/| idh
    NGINX -->|/api/task/| tah
    NGINX -->|/api/calendar/| cah
    NGINX -->|/api/mail/| mah
    NGINX -->|/api/event/| evh
    NGINX -->|/api/backlog/| bah
    NGINX -->|/api/report/| reh
    NGINX -->|/api/notification/| noh
    NGINX -->|/api/attachment/| ath
    NGINX -->|/api/project/| prh
    NGINX -->|/api/social/| soh
    NGINX -->|/api/message/| meh
    NGINX -->|"/api/soketi/ (WebSocket)"| SOK

    subgraph GR["gRPC servers (nghiệp vụ)"]
        idg["identity-grpc :9081"]:::grpc
        tag["task-grpc :9082"]:::grpc
        cag["calendar-grpc :9083"]:::grpc
        mag["mail-provider-grpc :9084"]:::grpc
        evg["event-grpc :9085"]:::grpc
        bag["backlog-grpc :9086"]:::grpc
        reg["report-grpc :9087"]:::grpc
        nog["notification-grpc :9088"]:::grpc
        atg["attachment-grpc :9089"]:::grpc
        prg["project-grpc :9090"]:::grpc
        sog["social-grpc :9091"]:::grpc
        meg["message-grpc :9092"]:::grpc
        wrk["worker xN<br/>(health :8094)"]:::grpc
    end

    idh --> idg
    tah --> tag
    cah --> cag
    mah --> mag
    evh --> evg
    bah --> bag
    reh --> reg
    noh --> nog
    ath --> atg
    prh --> prg
    soh --> sog
    meh --> meg

    PG[("postgres :5432<br/>DB task_manager")]:::infra
    MIG["migrate (job)"]:::infra
    S3[("rustfs :9000<br/>bucket task-manager")]:::infra
    KAFKA[["kafka :9092<br/>chat.messages x6 partition"]]:::infra
    SOK{{"soketi :6001"}}:::infra

    MIG --> PG
    idg & tag & cag & mag & evg & bag & nog & atg & prg & sog & meg & wrk --> PG
    atg & mag --> S3
    meg --> KAFKA --> wrk
    nog & meg --> SOK

    GOOG["Google OAuth / OIDC"]:::ext
    GMAIL["Gmail API"]:::ext
    PUBSUB["Google Cloud Pub/Sub"]:::ext
    DS["DeepSeek API"]:::ext

    idh --> GOOG
    mag --> GMAIL & DS & PUBSUB
    mag --> GOOG
    PUBSUB -->|"push: POST /api/mail/v1/notifications"| CF
```

## 2. Kiến trúc logic (các lớp)

```mermaid
flowchart LR
    subgraph Client["Client"]
        SPA["React SPA<br/>Redux store, i18n vi/en,<br/>FullCalendar, Mantine"]
    end

    subgraph Edge["Edge"]
        NG["nginx reverse proxy<br/>same-origin /api/*"]
    end

    subgraph Gateway["API Gateway layer (cmd/http)"]
        G1["grpc-gateway<br/>REST JSON <-> gRPC"]
        G2["Route viết tay:<br/>OAuth, session JWT, Soketi auth,<br/>multipart upload/download,<br/>Pub/Sub webhook"]
    end

    subgraph Domain["Domain layer (cmd/grpc)"]
        D1["identity / project / task<br/>calendar / event / backlog"]
        D2["attachment / notification<br/>social / message"]
        D3["report (aggregator)"]
        D4["mail-provider<br/>(ingest + classify + dispatch)"]
    end

    subgraph Async["Async layer"]
        K["Kafka"]
        W["worker (consumer group)"]
        RW["task reminder worker<br/>(ticker trong task-grpc)"]
        MW["mail workers / watch renewer<br/>/ pubsub puller (trong mail-provider-grpc)"]
    end

    subgraph Data["Data layer"]
        PG[("PostgreSQL<br/>schema-per-service")]
        S3[("RustFS S3")]
        RT{{"Soketi realtime"}}
    end

    SPA --> NG --> G1 & G2
    G1 & G2 -->|gRPC| D1 & D2 & D3 & D4
    D3 -->|gRPC| D1
    D4 -->|gRPC| D1 & D2
    D2 --> K --> W --> PG
    D1 & D2 & D4 --> PG
    D2 & D4 --> S3
    D2 --> RT
    RT -. WebSocket .-> SPA
    RW --> D2
    MW --> D4
```

## 3. Xác thực & phân quyền

```mermaid
sequenceDiagram
    autonumber
    participant B as Browser
    participant N as nginx
    participant IH as identity (HTTP)
    participant G as Google OAuth
    participant IG as identity-grpc
    participant TG as task-grpc
    participant PG2 as project-grpc
    participant MG as mail-provider-grpc

    B->>N: GET /api/identity/v1/auth/google/login?gmail=1
    N->>IH: forward
    IH-->>B: 302 accounts.google.com (state ký HMAC, PKCE verifier)
    B->>G: đăng nhập + consent (openid email profile [+ gmail.readonly])
    G-->>B: 302 /v1/auth/google/callback?code&state
    B->>IH: GET /v1/auth/google/callback
    IH->>G: POST oauth2.googleapis.com/token (code + verifier)
    IH->>G: GET openidconnect.googleapis.com/v1/userinfo
    IH->>IG: LoginWithProvider(provider, subject, email, ...)
    alt profile mới
        IG->>TG: SeedDefaultStatuses(profile_id)
        IG->>PG2: SeedDefaultProject(profile_id)
    end
    IG-->>IH: profile
    opt người dùng tích "Cho phép đọc Gmail"
        IH->>MG: Subscribe(profile_id, email, access_token, refresh_token)
    end
    IH-->>B: 302 FRONTEND/auth/callback#token=<session JWT HS256>
    B->>IH: GET /v1/auth/me (Bearer token)
```

- Session token: **JWT HS256**, khoá `SESSION_SECRET` dùng chung giữa `identity`, `notification`,
  `social`, `message` (các gateway này tự verify token và lấy `profile_id` từ claim `sub`).
- Các gateway `task`, `project`, `calendar`, `event`, `backlog`, `attachment`, `report` hiện nhận
  `profile_id` từ query/body của client (chưa verify token ở gateway).
- Kênh realtime private: `private-noti-internal-<profile_id>` và `private-chat-<profile_id>`;
  chữ ký subscribe do `identity` cấp tại `POST /v1/auth/soketi`, chỉ cho đúng chủ kênh.

## 4. Luồng chính (sequence)

### 4.1 Kéo thả task đổi trạng thái (lifecycle)

```mermaid
sequenceDiagram
    autonumber
    participant B as Browser
    participant TH as task (HTTP)
    participant TG as task-grpc
    participant DB as Postgres (task.*)
    B->>TH: POST /api/task/v1/tasks/{id}/status {status_id, note}
    TH->>TG: ChangeTaskStatus
    TG->>DB: SELECT task + status đích
    TG->>DB: IsAllowed? SELECT task.status_transitions (from -> to, active)
    alt không có rule
        TG-->>B: lỗi TASK_TRANSITION_NOT_ALLOWED (UI hiện notification đỏ)
    else hợp lệ
        TG->>DB: UPDATE task.tasks (status_id, completed_at nếu category DONE)
        TG->>DB: INSERT task.task_status_logs
        opt task lặp lại và vừa DONE
            TG->>DB: SpawnNextOccurrence -> INSERT task.tasks (lần kế tiếp)
        end
        TG-->>B: {task, next_task}
    end
```

### 4.2 Email Gmail → task / lịch / sự kiện / backlog

```mermaid
sequenceDiagram
    autonumber
    participant GM as Gmail
    participant PS as Pub/Sub
    participant MH as mail-provider (HTTP)
    participant MG as mail-provider-grpc
    participant DB as Postgres (mail_provider.*)
    participant S3 as RustFS
    participant DS as DeepSeek
    participant PR as project-grpc
    participant T as task/calendar/event/backlog-grpc
    participant NO as notification-grpc
    participant SK as Soketi
    participant B as Browser

    GM->>PS: thay đổi hộp thư (users.watch)
    alt push mode
        PS->>MH: POST /v1/notifications (OIDC JWT)
        MH->>MG: HandleNotification
    else pull mode
        MG->>PS: subscriptions.pull / acknowledge
    end
    MG->>MG: enqueue (queue RAM), trả 200 ngay
    MG->>DB: đọc checkpoint historyId + token
    MG->>GM: users.history.list -> users.messages.get (+ attachments)
    MG->>S3: PUT mail/{profile}/{yyyymmdd}/{msg}.json + attachments
    MG->>PR: ListProjects (ứng viên gán project)
    MG->>DS: POST /chat/completions (phân loại)
    Note over MG: lỗi/thiếu key -> rule keyword dự phòng
    MG->>T: CreateTask / CreateSchedule / CreateEvent / CreateBacklog
    MG->>NO: CreateNotice (TASK_CREATED, ...)
    NO->>SK: POST /apps/{id}/events (notice.created)
    SK-->>B: WebSocket -> cập nhật badge
    MG->>DB: tiến checkpoint historyId
```

### 4.3 Nhắn tin realtime + lưu bất đồng bộ

```mermaid
sequenceDiagram
    autonumber
    participant A as Browser (người gửi)
    participant MH as message (HTTP)
    participant MG as message-grpc
    participant SO as social-grpc
    participant SK as Soketi
    participant K as Kafka chat.messages
    participant W as worker xN
    participant DB as Postgres (message.*)
    participant R as Browser (người nhận)

    A->>MH: POST /v1/conversations/{peer}/messages (Bearer)
    MH->>MG: SendMessage(sender = sub)
    MG->>SO: AreFriends(sender, peer)
    MG->>SK: message.created -> private-chat-peer & private-chat-sender
    SK-->>R: WebSocket
    MG-)K: produce async (key = conversation_id)
    MG-->>A: tin nhắn (khớp client_id)
    K->>W: consume (group message-worker)
    W->>DB: INSERT message.messages ON CONFLICT DO NOTHING
    W->>DB: UPSERT message.conversations (last_message)
    W->>K: commit offset
```

### 4.4 Nhắc việc (reminder) và thông báo bạn bè

```mermaid
sequenceDiagram
    autonumber
    participant RW as task-grpc ReminderWorker
    participant DB as Postgres task.tasks
    participant NO as notification-grpc
    participant SC as social-grpc
    participant SK as Soketi
    loop mỗi interval
        RW->>DB: ClaimDueReminders (remind_at <= now, chưa nhắc)
        RW->>NO: CreateNotice(TASK_REMINDER, target TASK)
        alt gửi lỗi
            RW->>DB: ReleaseReminder (thử lại lần sau)
        end
    end
    SC->>NO: CreateNotice(FRIEND_REQUEST / FRIEND_ACCEPTED, target FRIEND)
    NO->>SK: publish private-noti-internal-<profile_id>
```

## 5. Quyết định kiến trúc chính

| Quyết định | Lý do / hệ quả |
|---|---|
| Microservice, **1 service = 1 Go module** | Build/deploy độc lập; dùng `go.work` khi dev |
| **gRPC nội bộ + grpc-gateway** ra ngoài | Interface khai báo 1 lần trong proto; REST tự sinh |
| **Schema-per-service** trên 1 Postgres | Ranh giới dữ liệu rõ, vận hành đơn giản; ràng buộc chéo qua gRPC |
| Fallback **in-memory** khi thiếu `DATABASE_URL` | Chạy local/test không cần DB |
| **Kafka + worker** cho chat | Realtime trước (Soketi), lưu sau; scale ngang worker theo partition |
| **Soketi** (Pusher protocol) | Push realtime notice + chat; kênh private ký bởi identity |
| **RustFS** thay MinIO | MinIO OSS bị archive; vẫn dùng SDK `minio-go` (S3 API) |
| Bộ **mã lỗi chuẩn** (`common/errorcode`) | FE dịch `reason` thành message vi/en, không lộ lỗi kỹ thuật |
| Best-effort cho side effect (seed, notice, soketi) | Lỗi phụ chỉ log, không làm hỏng thao tác chính |
| **Không dùng Redis** | Không có cache phân tán; state tạm nằm trong RAM của từng process |

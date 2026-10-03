# Task Manager

Hệ thống quản lý task cá nhân — monorepo: backend Go (gRPC + grpc-gateway/HTTP-JSON) và frontend React + Mantine.

## Cấu trúc

```
.
├── go.work                     # go.work gom các Go module
├── common/                     # Go module `taskmanager/common`
│   ├── proto/                  # INTERFACE: toàn bộ file .proto
│   ├── gen/                    # CODE GEN (go + openapi swagger)
│   ├── errorcode/              # BỘ MÃ LỖI CHUẨN: error_codes.json + Go helper
│   ├── go.mod / go.sum
│   └── README.md
├── service/                    # mỗi thư mục là 1 Go module + 1 service riêng
│   ├── identity/               # hồ sơ identity   (gRPC :9081 / HTTP :8081)
│   │   ├── go.mod              # require taskmanager/common (replace ../../common)
│   │   ├── Dockerfile          # image chứa 2 binary: grpc + http
│   │   ├── cmd/grpc, cmd/http  # entrypoint gRPC server / grpc-gateway (fx app)
│   │   └── internal/{config,dependency,handler,model,repository,service}
│   ├── task/                   # task, status, lifecycle (gRPC :9082 / HTTP :8082)
│   ├── calendar/               # lịch cá nhân (schedule)   (gRPC :9083 / HTTP :8083)
│   └── mail-provider/          # Gmail notice -> DeepSeek -> task (gRPC :9084 / HTTP :8084)
├── frontend/                   # React + TSX + Mantine + Vite (+ Dockerfile, nginx.conf)
├── deployments/                # hạ tầng: docker/ (compose) + migrations/ (SQL)
├── scripts/                    # gen grpc, cài tool, dev, smoke test
├── design/                     # thiết kế DB (dbdiagram DBML)
└── docs/
```

## Go module layout

| Module | Thư mục |
|---|---|
| `taskmanager/common` | `common/` |
| `taskmanager/service/identity` | `service/identity/` |
| `taskmanager/service/task` | `service/task/` |
| `taskmanager/service/calendar` | `service/calendar/` |
| `taskmanager/service/mail-provider` | `service/mail-provider/` |

- Root `go.work` gom 5 module lại để phát triển local.
- Service dùng code chung qua module `common`:
  `require taskmanager/common v0.0.0-...` + `replace taskmanager/common => ../../common`.
  Nhờ `replace`, service build được **cả khi không có `go.work`** (đúng cách Dockerfile đang build).
- Root không phải là module, nên `go build ./...` ở root **không dùng được** — `make build`
  sẽ loop từng module. Muốn build tay: `cd service/task && go build ./...`.

> **Về `go.sum`**: `go.sum` chỉ chứa checksum của dependency tải từ registry. Vì `common` là
> module nội bộ thay bằng đường dẫn filesystem (`replace`), nó **không có dòng nào trong `go.sum`**
> — đây là hành vi đúng của Go. Muốn `common` được version hoá qua `go.sum` thì phải publish nó
> lên module proxy/repo riêng (private), lúc đó bỏ `replace` và `go get taskmanager/common@vX.Y.Z`.

## Yêu cầu

- Go >= 1.25, Node >= 20, npm, Docker (nếu chạy compose)
- Không cần cài `protoc` — dùng `buf` (script tự cài)

## Quickstart (local)

```bash
make tools      # cài buf + 4 protoc plugin (1 lần)
make gen        # sinh code vào common/gen
make smoke      # verify end-to-end (build + start + gọi API thật)

make run-identity   # terminal 1  -> :8081 / :9081
make run-task       # terminal 2  -> :8082 / :9082
make run-calendar   # terminal 3  -> :8083 / :9083
make run-mail-provider  # terminal 4 -> :8084 / :9084 (cần MAIL_PUBSUB_TOPIC + DEEPSEEK_API_KEY)
make web-install && make web-dev    # terminal 5 -> :5173
make seed-demo      # (tuỳ chọn) seed dữ liệu demo giống design/ui.png
```

## Frontend (UI theo `design/ui.png`)

Kanban dashboard tối giản cho **người dùng cá nhân**: sidebar (brand + menu), topbar
(search/ngôn ngữ/thông báo/avatar), board header (tabs Kanban/Table/List/Timeline, search, filter,
New Task) và board 4 cột. Không có nhân sự/project/sprint.

- Cột lấy từ **status của profile** (`position` + `color`), card lấy từ **task gốc**.
- Task thuộc **profile**: `GET /v1/tasks?profile_id=...` trả task của profile đó; board fetch
  ngay khi đã có profile.
- Progress trên card suy ra từ task con (`done/total` theo `category = DONE`).
- **Kéo thả card** sang cột khác gọi `POST /v1/tasks/{id}/status`; rule allowlist ở backend chặn thì UI
  hiện notification đỏ — đúng nghiệp vụ lifecycle.
- Lần đầu chạy (chưa có status): **bộ status + lifecycle mặc định** (4 status + 5 rule) do task
  service tự tạo khi identity tạo profile mới; nếu vì lý do nào đó chưa có, board/trang Statuses có
  nút **Tạo bộ status mặc định** gọi lại `POST /v1/statuses/seed` (idempotent).
- **Lịch (Calendar)**: mục **Lịch** trong nhóm *Kế hoạch* của sidebar (`/calendar`) hiển thị lịch
  của profile bằng [FullCalendar](https://github.com/fullcalendar/fullcalendar), CRUD cơ bản qua
  `calendar` service (`/v1/schedules`).

Chi tiết: `frontend/README.md`.

## Tự động tạo task từ Gmail

Tính năng tuỳ chọn: user tích **"Cho phép đọc Gmail"** ngay ở màn đăng nhập, sau đó mỗi email
mới có thể được chuyển thành task.

- Luồng: `identity` xin thêm scope `gmail.readonly` (+ `access_type=offline`) → gọi
  `mail-provider.Subscribe` → `users.watch(topic)`; khi Gmail có thay đổi, **Pub/Sub** báo về
  `mail-provider` (push qua webhook `POST /api/mail/v1/notifications`, hoặc **pull** nếu set
  `MAIL_PULL_SUBSCRIPTION`) → gọi Gmail API lấy email gốc → gọi **DeepSeek** để bóc thành JSON →
  gọi `task service` tạo task cho đúng profile.
- Chạy local không cần tunnel nếu dùng pull mode (`MAIL_PULL_SUBSCRIPTION` + credential GCP);
  push mode bắt buộc endpoint HTTPS công khai nên cần `make up-tunnel`.
- Access token + refresh token của user được lưu ở **Postgres** (`mail_provider.sessions`, xem
  `service/mail-provider/README.md`); set `DATABASE_URL` cho `mail-provider-grpc` để bật. Không set
  thì service quay về cache trong RAM và restart là mất, user phải đăng nhập lại.
- Cấu hình: `MAIL_PUBSUB_TOPIC`, `MAIL_PUBSUB_AUDIENCE`, `MAIL_PUBSUB_SERVICE_ACCOUNT`,
  `DEEPSEEK_API_KEY`… trong `deployments/docker/.env` (xem `.env.example`).
- Topic Pub/Sub phải nằm **cùng GCP project với OAuth client**, nếu không `users.watch` báo
  `Invalid topicName`.

Chi tiết đầy đủ: `service/mail-provider/README.md`.

## Bộ mã lỗi chuẩn

Mọi lỗi backend đều kèm **mã lỗi chuẩn** để frontend dịch thành message đẹp, không hiển thị
HTTP status / URL / text kỹ thuật lên notification.

- Nguồn sự thật: `common/errorcode/error_codes.json` (mã + grpc code + http status + message vi/en).
  Go embed file này; build lỗi chỉ bằng mã qua `errorcode.Error(code)`, message được tra từ chính
  mã lỗi với locale mặc định `en` (`errorcode.DefaultLanguage`). Khi cần ngôn ngữ khác thì dùng
  `errorcode.Message(code, lang)`.
- Trên dây: gRPC status kèm `google.rpc.ErrorInfo`, grpc-gateway render thành
  `details[].reason`. Ví dụ lỗi thiếu title:

  ```json
  {
    "code": 3,
    "message": "Task title is required.",
    "details": [
      { "@type": "type.googleapis.com/google.rpc.ErrorInfo", "reason": "TASK_TITLE_REQUIRED", "domain": "taskmanager" }
    ]
  }
  ```

- Frontend (`frontend/src/lib/errorCatalog.ts` + `getErrorMessage` trong `api/client.ts`) map
  `reason` → message theo ngôn ngữ; mã lạ thì fallback theo HTTP status, tuyệt đối không show mã thô.
- Đổi bộ mã: sửa JSON canonical rồi `make error-codes` để sinh bản mirror cho FE
  (`frontend/src/config/error_codes.json`, có commit sẵn để Docker build chỉ với context `frontend/`).
  `make check-error-codes` fail nếu mirror lệch.

## Quickstart (Docker)

```bash
make up        # postgres + migrate + identity + task + calendar + mail-provider + frontend
make up-core   # như trên nhưng KHÔNG kèm cloudflared (tắt tunnel)
make up-tunnel # kèm cloudflared (Cloudflare Tunnel; cần token trong .env)
make ps
make down
```

> Đặt `COMPOSE_PROFILES=tunnel` trong `deployments/docker/.env` thì `make up` (và cả
> `docker compose up -d` trong `deployments/docker/`) mặc định chạy luôn cloudflared.

FE gọi API **same-origin** tại `/api/...` ngay trên port `frontend`; nginx trong container
frontend forward nội bộ tới các service Go. Chỉ `frontend` được publish ra host (bind
`127.0.0.1`), nên khi đưa ra Internet chỉ có **một origin/port** khớp với public hostname
của Cloudflare.

| Service | Truy cập |
|---|---|
| frontend (entrypoint) | http://localhost:3000 |
| identity | nội bộ `identity:8081` (`/healthz`) |
| task | nội bộ `task:8082` (`/healthz`) |
| calendar | nội bộ `calendar:8083` (`/healthz`) |
| mail-provider | nội bộ `mail-provider:8084` (`/healthz`) + webhook `/api/mail/v1/notifications` |
| postgres | nội bộ `postgres:5432` |
| cloudflared | profile `tunnel` — `make tunnel-logs` |

Chi tiết Cloudflare Tunnel + biến env: `deployments/docker/README.md`.

## Migration

```bash
make migrate-up      # chạy toàn bộ migration
make migrate-down    # rollback 1 bước
```

Chi tiết: `deployments/migrations/README.md`.

## Kiểm tra chất lượng

```bash
make quality-all
```

Chạy 4 nhóm kiểm tra:

| Target | Việc |
|---|---|
| `quality-fmt` | `gofmt -l` — fail nếu file chưa format |
| `quality-vet` | `go vet` từng module |
| `quality-arch` | Đọc `quality.json`, kiểm tra rules kiến trúc bằng `tools/quality` |
| `check-error-codes` | Fail nếu mirror bộ mã lỗi của FE lệch bản canonical |

Rules khai báo trong `quality.json` (nguồn duy nhất): hướng phụ thuộc giữa các tầng, `model` là core,
tầng phải flat, `service/interfaces.go` bắt buộc, không comment trong code Go, `cmd` dùng `fx`
(không tự quản lifecycle). Thêm/sửa rule = sửa `quality.json`, không cần sửa code checker.

## Quy ước

- **1 service = 1 thư mục = 1 Go module** trong `service/`. Mỗi service có 2 binary dưới `cmd/`
  (`cmd/grpc` = gRPC server, `cmd/http` = grpc-gateway) + `internal/` theo layering:
  `handler` (transport) → `service` (nghiệp vụ, chỉ contract interface + DI) →
  `repository` (I/O: db/cache/gRPC khác), `dependency` (validate/mapping dùng chung) + `Dockerfile`.
  Mỗi tầng là **1 package flat** (không có thư mục con).
- **Dependency inversion**: port (interface) do tầng dùng định nghĩa — `service/interfaces.go`
  khai báo interface I/O mà service cần + contract cho handler. `service` chỉ import `model`,
  không import `repository`/`dependency`; adapter ở `repository` thoả mãn port nhờ structural
  typing của Go (không import `service`). `cmd` là nơi duy nhất wiring concrete vào interface.
- **`model` là core**: chỉ import stdlib, không import tầng nào (kể cả proto). Việc convert
  (vd enum core <-> enum proto) đặt ở `dependency`.
- **`cmd` dùng `go.uber.org/fx`**: DI graph + lifecycle (OnStart/OnStop) do fx quản lý, không tự
  bắt signal/graceful shutdown. Adapter repository được inject vào port qua `fx.As`.
- **Proto tập trung** ở `common/proto/<service>/v1/*.proto`, code gen đổ về `common/gen`.
- HTTP path chuẩn hoá `/v1/<resource>`, khai báo bằng `option (google.api.http)` trong proto.
  `GET /v1/tasks` lọc theo `profile_id` (query bắt buộc trên thực tế để board có dữ liệu).
- JSON trả về dùng **lowerCamelCase** (mặc định protojson) — khớp type của frontend.
  Riêng **query param** phải dùng tên field proto: `GET /v1/tasks?profile_id=...&root_only=true`.
- Tên bảng DB theo `<service>.<TABLES>` — xem `design/db_schema.dbml`.
- **Chất lượng code**: `make quality-all` = `gofmt` + `go vet` + rules kiến trúc trong `quality.json`
  (checker ở `tools/quality`).

## Troubleshooting

**`go: go.work requires go >= X (running go Y)`** — `go.work` bị ghi version của toolchain đã tạo nó
(ví dụ do chạy `go work init` bằng Go mới hơn). Sửa:

```bash
go work edit -go=1.25.0    # go.work + 5 go.mod đang ở 1.25.0 -> Go >= 1.25 là chạy được
```

rồi reload IDE để gopls load lại. **`make vet`** dùng `scripts/vet.sh` để bỏ qua cảnh báo vet phát sinh
trong source của dependency (Go 1.26 hay gặp với protobuf).

## Trạng thái hiện tại

Cả 4 service Go đều có **repository PostgreSQL** (`postgres_*_repository.go`) + fallback **in-memory
stub** khi không set `DATABASE_URL`. Business logic nằm ở `internal/service` (DI qua interface),
transport gRPC ở `internal/handler`, validate/mapping ở `internal/dependency`; `cmd/grpc/infra.go`
chọn Postgres/in-memory và quản lý `pgxpool` theo fx lifecycle.

`calendar` giữ **lịch cá nhân** ở `calendar.schedules` (CRUD cơ bản: title/mô tả/địa điểm/thời gian/
all-day/màu) và phục vụ UI FullCalendar.

`mail-provider` dùng Postgres cho **subscription** (`mail_provider.sessions` +
`mail_provider.noti_indexes`: refresh/access token, checkpoint `historyId`, hạn watch) và gọi ra
ngoài (Gmail, DeepSeek) cũng như sang `task` qua gRPC.

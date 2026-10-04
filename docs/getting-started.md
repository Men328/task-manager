# Getting started

## 1. Cài toolchain

```bash
make tools
```

Script `scripts/install-tools.sh` sẽ:
- tải `buf` (binary) vào `$(go env GOPATH)/bin` nếu chưa có;
- `go install` 4 plugin: `protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-grpc-gateway`, `protoc-gen-openapiv2`.

> Đảm bảo `$(go env GOPATH)/bin` có trong `PATH`.

## 2. Generate code

```bash
make gen
```

Luồng: `buf dep update` (tải `google/api/annotations.proto`) → `buf lint` → `buf generate` →
`go mod tidy` cho **từng module** (`common`, `service/backlog`, `service/calendar`, `service/event`,
`service/identity`, `service/mail-provider`, `service/report`, `service/task`).

Output:

```
common/gen/go/backlog/v1/backlog*.go
common/gen/go/calendar/v1/schedule*.go
common/gen/go/event/v1/event*.go
common/gen/go/identity/v1/profile.pb.go
common/gen/go/identity/v1/profile.pb.gw.go
common/gen/go/identity/v1/profile_grpc.pb.go
common/gen/go/mail/v1/mail.pb.go
common/gen/go/mail/v1/mail_grpc.pb.go
common/gen/go/report/v1/report*.go
common/gen/go/task/v1/{task,status,transition}*.go
common/gen/openapi/task_manager.swagger.json
```

`sửa proto` = sửa file trong `common/proto/` rồi chạy lại `make gen`. **Không sửa** file trong `common/gen/`.

## 3. Chạy backend (local)

Mỗi service gồm 2 process: `cmd/grpc` (gRPC server) + `cmd/http` (grpc-gateway).

```bash
bash scripts/dev.sh        # chạy cả service (grpc + gateway), Ctrl+C để dừng
# hoặc chạy 1 service (cả grpc + gateway)
make run-identity
make run-task
make run-calendar
make run-event
make run-backlog
make run-report           # cần task service đang chạy (:9082)
make run-mail-provider   # cần MAIL_PUBSUB_TOPIC + DEEPSEEK_API_KEY để chạy đủ luồng

# hoặc chạy riêng từng process
make run-identity-grpc     # :9081
make run-identity-http     # :8081 (dial 127.0.0.1:9081)
```

Kiểm tra:

```bash
curl -s localhost:8081/healthz   # {"status":"ok","service":"identity"}
curl -s localhost:8082/healthz   # {"status":"ok","service":"task"}
curl -s localhost:8083/healthz   # {"status":"ok","service":"calendar"}
curl -s localhost:8085/healthz   # {"status":"ok","service":"event"}
curl -s localhost:8086/healthz   # {"status":"ok","service":"backlog"}
curl -s localhost:8087/healthz   # {"status":"ok","service":"report"}
curl -s localhost:8084/healthz   # {"status":"ok","service":"mail-provider"}
```

`/healthz` chỉ trả 200 khi gateway gọi được gRPC health service của process gRPC tương ứng.

## 4. Chạy frontend

```bash
make web-install
make web-dev
```

Vite proxy `/api/identity/*` → `localhost:8081`, `/api/task/*` → `localhost:8082`,
`/api/calendar/*` → `localhost:8083`, `/api/event/*` → `localhost:8085`,
`/api/backlog/*` → `localhost:8086`, `/api/report/*` → `localhost:8087`,
`/api/mail/*` → `localhost:8084`
(xem `frontend/vite.config.ts`).

## 5. Chạy bằng Docker

```bash
make up        # build + start: postgres, migrate, rustfs, identity(-grpc), task(-grpc), calendar(-grpc), event(-grpc), backlog(-grpc), report(-grpc), mail-provider(-grpc), frontend
make ps
make logs
make down
```

Hoặc trực tiếp:

```bash
docker compose -f deployments/docker/docker-compose.yml up --build -d
```

Thứ tự khởi động: `postgres (healthy)` → `migrate (exit 0)` → `rustfs (healthy)` →
`*-grpc` → gateway `identity`/`task`/`calendar`/`event`/`backlog`/`report`/`mail-provider` (healthy) →
`frontend`. Mỗi service Go chạy 2 container: `<svc>-grpc` và `<svc>` (gateway).

Dockerfile nằm trong từng service (`service/*/Dockerfile`, `frontend/Dockerfile`); build context là
**root repo** vì mỗi service Go cần copy thêm module `common/`. Chi tiết: `deployments/docker/README.md`.

## 6. Migration DB

```bash
make migrate-up       # chạy toàn bộ (qua docker compose, tự start postgres)
make migrate-down     # rollback 1 bước
```

Hoặc bằng CLI:

```bash
migrate -path ./deployments/migrations \
  -database "postgres://task_manager:task_manager@localhost:5432/task_manager?sslmode=disable" up
```

Chi tiết + bảng ánh xạ với `design/db_schema.dbml`: `deployments/migrations/README.md`.

## 7. Smoke test end-to-end

```bash
make smoke
```

Build 14 binary (gRPC + gateway của 7 service), start ở port mặc định, gọi thật toàn bộ luồng
(profile → status → transition rule → task cha/con → đổi status → ràng buộc xoá → CRUD lịch →
sự kiện → backlog → báo cáo) rồi tắt.
Yêu cầu port `8081-8087` và `9081-9087` đang trống.

Khi service đã chạy sẵn thì chỉ chạy phần test:

```bash
node scripts/smoke-test.mjs
```

### Mapping mã lỗi (grpc-gateway mặc định)

| gRPC code | HTTP |
|---|---|
| `InvalidArgument` | 400 |
| `FailedPrecondition` | 400 |
| `Unauthenticated` | 401 |
| `PermissionDenied` | 403 |
| `NotFound` | 404 |
| `AlreadyExists` | 409 |
| `Internal` | 500 |

## 8. Thêm 1 service mới

Mỗi service là 1 Go module riêng:

```bash
mkdir -p service/myservice/{cmd/{grpc,http},internal/{config,dependency,handler,model,repository,service}}
printf 'module taskmanager/service/myservice\n\ngo 1.25.0\n' > service/myservice/go.mod
( cd service/myservice && go mod edit -replace taskmanager/common=../../common )
go work use ./service/myservice
```

1. Thêm proto `common/proto/myservice/v1/foo.proto` với
   `option go_package = "taskmanager/common/gen/go/myservice/v1;myservicev1";`
2. `make gen`
3. Copy `service/identity` làm template (đổi port trong `internal/config`, đổi import, đổi register
   trong `cmd/grpc/main.go` + `cmd/http/main.go`; giữ layering `handler -> service -> repository`,
   `dependency` cho validate/mapping). Port I/O khai báo ở `internal/service/interfaces.go`
   (service **không** import `repository`); adapter ở `internal/repository` chỉ cần đúng method
   set là thoả mãn port nhờ structural typing của Go. `cmd` dùng `go.uber.org/fx`: khai báo
   provider trong `fx.Provide` (repository inject vào port qua `fx.As(new(<port>))`) và lifecycle
   (OnStart/OnStop) trong `fx.Hook`.
4. Thêm `service/myservice/Dockerfile` (build 2 binary `grpc` + `http`), service vào
   `deployments/docker/docker-compose.yml` (1 container grpc + 1 gateway), target vào `Makefile`
   + `scripts/dev.sh` + proxy trong `frontend/vite.config.ts` / `frontend/nginx.conf`.

## 9. Kiểm tra chất lượng

```bash
make quality-all
```

- `quality-fmt`: `gofmt -l` — fail nếu file chưa format.
- `quality-vet`: `go vet` từng module.
- `quality-arch`: đọc `quality.json`, kiểm tra rules kiến trúc (hướng phụ thuộc giữa các tầng,
  `model` là core, tầng flat, `service/interfaces.go` bắt buộc, không comment, `cmd` dùng `fx`).

Rules nằm trong `quality.json`; checker ở `tools/quality` (Go, chỉ dùng stdlib).

## Troubleshooting

### `go: go.work requires go >= 1.27.1 (running go 1.26.4)`

IDE/gopls báo `packages.Load error: ... go.work requires go >= 1.27.1`.

`go.work` bị ghi đúng version của toolchain đã tạo nó — thường là do chạy `go work init` bằng Go mới hơn
toolchain của bạn. Go từ chối chạy nếu toolchain < version ghi trong `go.work`.

Sửa (chạy được ngay cả khi `go.work` đang cao hơn toolchain hiện tại):

```bash
go work edit -go=1.25.0    # khớp `go` directive trong 6 go.mod
go build all               # kiểm tra
```

Rồi reload cửa sổ IDE để gopls load lại workspace.

`go.work` và cả 9 module đang ở `go 1.25.0` — đây là mức tối thiểu mà dependency yêu cầu, nên
**Go >= 1.25 là chạy được**. Đừng chạy lại `go work init` (nó ghi version toolchain hiện tại vào `go.work`).

### `make vet` báo lỗi trong `.../pkg/mod/google.golang.org/protobuf@...`

Go 1.26 chạy analyzer `unreachable` lên **cả package dependency**, nên `go vet ./...` in ra cảnh báo
trong module cache (không phải code của mình). `scripts/vet.sh` lọc các dòng trỏ vào module cache nên
`make vet` vẫn sạch và vẫn giữ đầy đủ analyzer cho code của mình. Go 1.27 không còn hiện tượng này.

## Biến môi trường

Mỗi service đọc env với default như sau:

| Biến | identity | task | calendar | event | backlog | report |
|---|---|---|---|---|---|---|
| `SERVICE_NAME` | `identity` | `task` | `calendar` | `event` | `backlog` | `report` |
| `GRPC_ADDR` | `:9081` | `:9082` | `:9083` | `:9085` | `:9086` | `:9087` |
| `HTTP_ADDR` | `:8081` | `:8082` | `:8083` | `:8085` | `:8086` | `:8087` |
| `GRPC_DIAL_ADDR` | (rỗng) | (rỗng) | (rỗng) | (rỗng) | (rỗng) | (rỗng) |
| `LOG_LEVEL` | `info` | `info` | `info` | `info` | `info` | `info` |
| `DATABASE_URL` | (rỗng → in-memory) | (rỗng → in-memory) | (rỗng → in-memory) | (rỗng → in-memory) | (rỗng → in-memory) | — (không có DB) |
| `TASK_GRPC_DIAL_ADDR` | `127.0.0.1:9082` (seed lifecycle khi tạo profile) | — | — | — | — | `127.0.0.1:9082` (nguồn dữ liệu báo cáo) |
| `GOOGLE_CLIENT_ID` | (rỗng) | — |
| `GOOGLE_CLIENT_SECRET` | (rỗng) | — |
| `GOOGLE_REDIRECT_URL` | `http://localhost:8081/v1/auth/google/callback` | — |
| `FRONTEND_BASE_URL` | `http://localhost:5173` | — |
| `SESSION_SECRET` | (rỗng) | — |
| `SESSION_TTL` | `720h` | — |
| `COOKIE_SECURE` | `false` | — |

`GRPC_ADDR` là địa chỉ `cmd/grpc` listen; `GRPC_DIAL_ADDR` là target `cmd/http` dial tới
(rỗng → tự suy ra `127.0.0.1:<port>`; đặt trong Docker, ví dụ `identity-grpc:9081`).

`report` không có `DATABASE_URL`: nó gọi `task`/`event`/`calendar`/`backlog` qua
`TASK_GRPC_DIAL_ADDR` (mặc định `127.0.0.1:9082`), `EVENT_GRPC_DIAL_ADDR` (`:9085`),
`CALENDAR_GRPC_DIAL_ADDR` (`:9083`), `BACKLOG_GRPC_DIAL_ADDR` (`:9086`) rồi tổng hợp.
`TASK_TIMEOUT` và `ACTIVITY_TIMEOUT` (mặc định `10s`) là timeout mỗi lần gọi nguồn dữ liệu.

`DATABASE_URL` dùng cho `cmd/grpc` của identity, task, calendar, event, backlog **và** của
mail-provider: có giá trị thì chạy repository Postgres (mail-provider lưu subscription vào
`mail_provider.sessions` + `mail_provider.noti_indexes`; calendar lưu lịch vào `calendar.schedules`;
event lưu sự kiện vào `event.events`; backlog lưu email chưa phân loại vào `backlog.backlogs`),
rỗng thì quay về in-memory stub (log cảnh báo). Các biến `GOOGLE_*`, `FRONTEND_BASE_URL`,
`SESSION_*` chỉ gateway identity (`cmd/http`) dùng cho luồng đăng nhập Google.

`identity` còn đọc `MAIL_GRPC_DIAL_ADDR` (mặc định `127.0.0.1:9084`) để gọi mail-provider khi user
tích quyền đọc mail. `mail-provider` có bộ biến riêng khá dài — `MAIL_*` (topic Pub/Sub, label, gia
hạn watch, queue/worker), `DEEPSEEK_*` (phân loại email), `GMAIL_*`, `GOOGLE_CLIENT_ID/SECRET`
(refresh token), `S3_*` (lưu email gốc + attachment lên object storage S3-compatible) và các
`*_GRPC_DIAL_ADDR` tới task/calendar/event/backlog. Bảng đầy đủ: `service/mail-provider/README.md`.
## Đăng nhập Google (local)

```bash
export SESSION_SECRET="$(openssl rand -base64 48)"

# OAuth client loại "Web application", redirect URI:
#   http://localhost:5173/api/identity/v1/auth/google/callback
export GOOGLE_CLIENT_ID="....apps.googleusercontent.com"
export GOOGLE_CLIENT_SECRET="...."
export GOOGLE_REDIRECT_URL="http://localhost:5173/api/identity/v1/auth/google/callback"
export FRONTEND_BASE_URL="http://localhost:5173"

make run-identity   # gRPC :9081 + gateway :8081
make web-dev        # http://localhost:5173
```

Mở http://localhost:5173 → tự chuyển tới `/login` → **Đăng nhập với Google**. Backend đổi
authorization code, tạo profile (kèm dòng `identity.auth_providers`) nếu là user mới, rồi redirect
về `/auth/callback#token=<JWT>`. Token nằm ở fragment nên không lọt vào access log; frontend lưu vào
`localStorage` và xoá fragment khỏi address bar.

Callback trỏ về origin của Vite (`:5173`) chứ không phải `:8081` để request đi qua proxy và giữ
cùng origin với SPA — nhớ khai báo đúng URI này ở Google Cloud Console.

Màn login có thêm ô **"Cho phép đọc Gmail để tự động tạo công việc"**. Khi user tích, URL login thành
`/v1/auth/google/login?gmail=1`, backend xin thêm scope `gmail.readonly` + `access_type=offline`
(buộc màn consent hiện lại để có refresh token), rồi gọi `mail-provider.Subscribe` để đăng ký
`users.watch`. Muốn chạy đủ luồng này cần thêm `MAIL_PUBSUB_TOPIC` (đúng project với OAuth client)
và `DEEPSEEK_API_KEY`; chi tiết: `service/mail-provider/README.md`.

Notice có thể nhận theo 2 cách: **push** (cần URL HTTPS công khai — dùng `make up-tunnel`) hoặc
**pull** (set `MAIL_PULL_SUBSCRIPTION` + credential GCP, chạy local không cần tunnel).

Muốn dữ liệu tồn tại qua restart thì trỏ `DATABASE_URL` vào Postgres đã migrate:

```bash
export DATABASE_URL="postgres://task_manager:task_manager@localhost:5432/task_manager?sslmode=disable"
```

## Lệnh hữu ích

| Lệnh | Việc |
|---|---|
| `make build` / `make vet` / `make test` | loop qua từng Go module |
| `make tidy` | `go mod tidy` từng module |
| `make work` | `go work sync` |
| `make fmt` | gofmt `common` + `service` + `tools` |
| `make quality-all` | gofmt + vet + kiểm tra rules kiến trúc (`quality.json`) |
| `make help` | liệt kê toàn bộ target |

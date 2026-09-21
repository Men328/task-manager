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
`go mod tidy` cho **từng module** (`common`, `service/identity`, `service/task`).

Output:

```
common/gen/go/identity/v1/profile.pb.go
common/gen/go/identity/v1/profile.pb.gw.go
common/gen/go/identity/v1/profile_grpc.pb.go
common/gen/go/task/v1/{task,status,transition}*.go
common/gen/openapi/task_manager.swagger.json
```

`sửa proto` = sửa file trong `common/proto/` rồi chạy lại `make gen`. **Không sửa** file trong `common/gen/`.

## 3. Chạy backend (local)

Mỗi service gồm 2 process: `cmd/grpc` (gRPC server) + `cmd/http` (grpc-gateway).

```bash
bash scripts/dev.sh        # chạy cả 2 service (4 process), Ctrl+C để dừng
# hoặc chạy 1 service (cả grpc + gateway)
make run-identity
make run-task

# hoặc chạy riêng từng process
make run-identity-grpc     # :9081
make run-identity-http     # :8081 (dial 127.0.0.1:9081)
```

Kiểm tra:

```bash
curl -s localhost:8081/healthz   # {"status":"ok","service":"identity"}
curl -s localhost:8082/healthz   # {"status":"ok","service":"task"}
```

`/healthz` chỉ trả 200 khi gateway gọi được gRPC health service của process gRPC tương ứng.

## 4. Chạy frontend

```bash
make web-install
make web-dev
```

Vite proxy `/api/identity/*` → `localhost:8081`, `/api/task/*` → `localhost:8082` (xem `frontend/vite.config.ts`).

## 5. Chạy bằng Docker

```bash
make up        # build + start: postgres, migrate, identity(-grpc), task(-grpc), frontend
make ps
make logs
make down
```

Hoặc trực tiếp:

```bash
docker compose -f deployments/docker/docker-compose.yml up --build -d
```

Thứ tự khởi động: `postgres (healthy)` → `migrate (exit 0)` → `*-grpc` → gateway `identity`/`task`
(healthy) → `frontend`. Mỗi service Go chạy 2 container: `<svc>-grpc` và `<svc>` (gateway).

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

Build 4 binary (gRPC + gateway của mỗi service), start ở port mặc định, gọi thật toàn bộ luồng
(profile → status → transition rule → task cha/con → đổi status → ràng buộc xoá) rồi tắt.
Yêu cầu port `8081/8082/9081/9082` đang trống.

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
go work edit -go=1.25.0    # khớp `go` directive trong 3 go.mod
go build all               # kiểm tra
```

Rồi reload cửa sổ IDE để gopls load lại workspace.

`go.work` và cả 3 module đang ở `go 1.25.0` — đây là mức tối thiểu mà dependency yêu cầu, nên
**Go >= 1.25 là chạy được**. Đừng chạy lại `go work init` (nó ghi version toolchain hiện tại vào `go.work`).

### `make vet` báo lỗi trong `.../pkg/mod/google.golang.org/protobuf@...`

Go 1.26 chạy analyzer `unreachable` lên **cả package dependency**, nên `go vet ./...` in ra cảnh báo
trong module cache (không phải code của mình). `scripts/vet.sh` lọc các dòng trỏ vào module cache nên
`make vet` vẫn sạch và vẫn giữ đầy đủ analyzer cho code của mình. Go 1.27 không còn hiện tượng này.

## Biến môi trường

Mỗi service đọc env với default như sau:

| Biến | identity | task |
|---|---|---|
| `SERVICE_NAME` | `identity` | `task` |
| `GRPC_ADDR` | `:9081` | `:9082` |
| `HTTP_ADDR` | `:8081` | `:8082` |
| `GRPC_DIAL_ADDR` | (rỗng) | (rỗng) |
| `LOG_LEVEL` | `info` | `info` |
| `DATABASE_URL` | (rỗng → in-memory) | (chưa dùng) |
| `GOOGLE_CLIENT_ID` | (rỗng) | — |
| `GOOGLE_CLIENT_SECRET` | (rỗng) | — |
| `GOOGLE_REDIRECT_URL` | `http://localhost:8081/v1/auth/google/callback` | — |
| `FRONTEND_BASE_URL` | `http://localhost:5173` | — |
| `SESSION_SECRET` | (rỗng) | — |
| `SESSION_TTL` | `720h` | — |
| `COOKIE_SECURE` | `false` | — |

`GRPC_ADDR` là địa chỉ `cmd/grpc` listen; `GRPC_DIAL_ADDR` là target `cmd/http` dial tới
(rỗng → tự suy ra `127.0.0.1:<port>`; đặt trong Docker, ví dụ `identity-grpc:9081`).

`DATABASE_URL` chỉ `cmd/grpc` của identity dùng: có giá trị thì chạy repository Postgres, rỗng thì
quay về in-memory stub (log cảnh báo). Các biến `GOOGLE_*`, `FRONTEND_BASE_URL`, `SESSION_*` chỉ
gateway identity (`cmd/http`) dùng cho luồng đăng nhập Google.

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

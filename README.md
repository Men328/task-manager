# Task Manager

Hệ thống quản lý task cá nhân — monorepo: backend Go (gRPC + grpc-gateway/HTTP-JSON) và frontend React + Mantine.

## Cấu trúc

```
.
├── go.work                     # workspace gom các Go module
├── common/                     # Go module `taskmanager/common`
│   ├── proto/                  # INTERFACE: toàn bộ file .proto
│   ├── gen/                    # CODE GEN (go + openapi swagger)
│   ├── go.mod / go.sum
│   └── README.md
├── service/                    # mỗi thư mục là 1 Go module + 1 service riêng
│   ├── identity/               # hồ sơ identity   (gRPC :9081 / HTTP :8081)
│   │   ├── go.mod              # require taskmanager/common (replace ../../common)
│   │   ├── Dockerfile          # image chứa 2 binary: grpc + http
│   │   ├── cmd/grpc, cmd/http  # entrypoint gRPC server / grpc-gateway (fx app)
│   │   └── internal/{config,dependency,handler,model,repository,service}
│   └── task/                   # task, status, lifecycle (gRPC :9082 / HTTP :8082)
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

- Root `go.work` gom 3 module lại để phát triển local.
- Service dùng code chung qua module `common`:
  `require taskmanager/common v0.0.0-...` + `replace taskmanager/common => ../../common`.
  Nhờ `replace`, service build được **cả khi không có `go.work`** (đúng cách Dockerfile đang build).
- Root không phải là module, nên `go build ./...` ở root **không dùng được** — `make build`
  sẽ loop từng module. Muốn build tay: `go build all` (workspace) hoặc `cd service/task && go build ./...`.

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
make web-install && make web-dev    # terminal 3 -> :5173
make seed-demo      # (tuỳ chọn) seed dữ liệu demo giống design/ui.png
```

## Frontend (UI theo `design/ui.png`)

Kanban dashboard tối giản cho **người dùng cá nhân**: sidebar (brand + không gian việc + menu), topbar
(search/ngôn ngữ/thông báo/avatar), board header (tabs Kanban/Table/List/Timeline, search, filter,
New Task) và board 4 cột. Không có nhân sự/project/sprint.

- Cột lấy từ **status của profile** (`position` + `color`), card lấy từ **task gốc**.
- Progress trên card suy ra từ task con (`done/total` theo `category = DONE`).
- **Kéo thả card** sang cột khác gọi `POST /v1/tasks/{id}/status`; rule allowlist ở backend chặn thì UI
  hiện notification đỏ — đúng nghiệp vụ lifecycle.
- Lần đầu chạy (chưa có status) có nút **Tạo bộ status mặc định** (4 status + rule) để board có cột ngay.

Chi tiết: `frontend/README.md`.

## Quickstart (Docker)

```bash
make up     # postgres + migrate + identity + task + frontend
make ps
make down
```

| Service | URL |
|---|---|
| frontend | http://localhost:3000 |
| identity | http://localhost:8081 (`/healthz`) |
| task | http://localhost:8082 (`/healthz`) |
| postgres | localhost:5432 |

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

Chạy 3 nhóm kiểm tra:

| Target | Việc |
|---|---|
| `quality-fmt` | `gofmt -l` — fail nếu file chưa format |
| `quality-vet` | `go vet` từng module |
| `quality-arch` | Đọc `quality.json`, kiểm tra rules kiến trúc bằng `tools/quality` |

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
- JSON trả về dùng **lowerCamelCase** (mặc định protojson) — khớp type của frontend.
  Riêng **query param** phải dùng tên field proto: `GET /v1/tasks?profile_id=...`.
- Tên bảng DB theo `<service>.<TABLES>` — xem `design/db_schema.dbml`.
- **Chất lượng code**: `make quality-all` = `gofmt` + `go vet` + rules kiến trúc trong `quality.json`
  (checker ở `tools/quality`).

## Troubleshooting

**`go: go.work requires go >= X (running go Y)`** — `go.work` bị ghi version của toolchain đã tạo nó
(ví dụ do chạy `go work init` bằng Go mới hơn). Sửa:

```bash
go work edit -go=1.25.0    # go.work + 3 go.mod đang ở 1.25.0 -> Go >= 1.25 là chạy được
```

rồi reload IDE để gopls load lại. **`make vet`** dùng `scripts/vet.sh` để bỏ qua cảnh báo vet phát sinh
trong source của dependency (Go 1.26 hay gặp với protobuf).

## Trạng thái hiện tại

Base scaffold: repository của 2 service Go là **in-memory stub**, chưa nối DB (migration + schema
đã sẵn). Business logic nằm ở `internal/service` (DI qua interface), transport gRPC ở
`internal/handler`, validate/mapping ở `internal/dependency`.

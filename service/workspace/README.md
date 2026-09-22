# service/workspace

Service quản lý workspace — **namespace gốc** chứa mọi tài nguyên (task, và các loại khác thêm
sau). Là **1 Go module riêng** (`taskmanager/service/workspace`), dùng code chung qua module
`taskmanager/common` (xem `replace` trong `go.mod`).

- gRPC: `:9083` — HTTP/JSON gateway: `:8083`
- Proto: `common/proto/workspace/v1/workspace.proto`
- Bảng DB: `workspace.WORKSPACES` (xem `deployments/migrations/000003_init_workspace.up.sql`)

## API

| RPC | HTTP | Việc |
|---|---|---|
| `CreateWorkspace` | `POST /v1/workspaces` | Tạo workspace |
| `GetWorkspace` | `GET /v1/workspaces/{id}` | Chi tiết workspace |
| `ListWorkspaces` | `GET /v1/workspaces?owner_profile_id=...` | Danh sách workspace của owner |
| `UpdateWorkspace` | `PATCH /v1/workspaces/{id}` | Cập nhật một phần |

## Cấu trúc

```
Dockerfile                       # build 2 binary grpc + http (build context = root repo)
cmd/grpc/                        # fx app: gRPC server (:9083) + health/reflection + lifecycle + chọn repo
cmd/http/                        # fx app: grpc-gateway (:8083) + /healthz, dial qua GRPC_DIAL_ADDR
internal/config/config.go        # có DATABASE_URL
internal/model/                  # CORE: Workspace, WorkspaceFilter, WorkspaceUpdate + lỗi domain (chỉ stdlib)
internal/service/                # NGHIỆP VỤ + interface contract (interfaces.go): 1 port + 1 service
internal/repository/             # adapter I/O: postgres_workspace_repository.go + in-memory fallback
internal/dependency/             # validate + mapping (model <-> proto) + map lỗi -> gRPC
internal/handler/                # transport gRPC mỏng
```

**Dependency inversion**: `service/interfaces.go` khai báo port `WorkspaceRepository` và contract
`WorkspaceService`. `service` chỉ import `model` — không import `repository`/`dependency`. Adapter ở
`repository` thoả mãn port nhờ structural typing. `cmd` là nơi duy nhất wiring concrete vào interface.

## Nghiệp vụ

- `slug` unique theo `owner_profile_id` (tạo trùng -> `WORKSPACE_SLUG_ALREADY_EXISTS`).
- Mỗi owner chỉ có 1 workspace `is_default` đang hoạt động: set default sẽ tự bỏ default của workspace khác.
- `is_archived` ẩn khỏi list mặc định; list hỗ trợ `include_archived`.
- Không hard-delete workspace (dùng `is_archived`/`deleted_at`).

## Chạy

```bash
make run-workspace
```

| Biến | Default |
|---|---|
| `SERVICE_NAME` | `workspace` |
| `GRPC_ADDR` | `:9083` (cmd/grpc listen) |
| `HTTP_ADDR` | `:8083` (cmd/http listen) |
| `GRPC_DIAL_ADDR` | rỗng -> suy ra `127.0.0.1:9083` (cmd/http dial target) |
| `LOG_LEVEL` | `info` |
| `DATABASE_URL` | rỗng -> in-memory stub; set thì dùng PostgreSQL |

## Thử API

```bash
P=<profile_id>

# tạo workspace mặc định
curl -s -X POST localhost:8083/v1/workspaces -H 'Content-Type: application/json' \
  -d "{\"owner_profile_id\":\"$P\",\"name\":\"Cá nhân\",\"slug\":\"personal\",\"is_default\":true}"

# danh sách
curl -s "localhost:8083/v1/workspaces?owner_profile_id=$P"

# chi tiết
curl -s localhost:8083/v1/workspaces/<workspace_id>

# cập nhật
curl -s -X PATCH localhost:8083/v1/workspaces/<workspace_id> -H 'Content-Type: application/json' \
  -d '{"name":"Việc riêng","position":1}'
```

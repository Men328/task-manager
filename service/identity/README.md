# service/identity

Service quản lý hồ sơ identity. Là **1 Go module riêng** (`taskmanager/service/identity`),
dùng code chung qua module `taskmanager/common` (xem `replace` trong `go.mod`).

- gRPC: `:9081` — HTTP/JSON gateway: `:8081`
- Proto: `common/proto/identity/v1/profile.proto`
- Bảng DB: `identity.PROFILES`, `identity.AUTH_PROVIDERS` (xem `deployments/migrations/`)

## Cấu trúc

```
Dockerfile                       # build 2 binary grpc + http (build context = root repo)
cmd/grpc/                        # fx app: gRPC server (:9081) + lifecycle
cmd/http/                        # fx app: grpc-gateway (:8081) + lifecycle, dial qua GRPC_DIAL_ADDR
internal/config/config.go        # đọc env
internal/model/                  # CORE: domain model + lỗi domain (chỉ stdlib, không import tầng nào)
internal/service/                # NGHIỆP VỤ + interface contract (interfaces.go), DI qua constructor
internal/repository/             # adapter I/O (in-memory stub), KHÔNG import service
internal/dependency/             # validate + mapping (model <-> proto, convert) + map lỗi -> gRPC
internal/handler/                # transport gRPC mỏng: validate -> service -> map response
```

**Dependency inversion**: `service/interfaces.go` khai báo `ProfileRepository` (port I/O mà
service cần) và `ProfileService` (contract cho handler). `service` chỉ import `model` — không
import `repository`/`dependency`. Adapter ở `repository` thoả mãn port nhờ structural typing
(không import `service`). `cmd` là nơi duy nhất wiring concrete vào interface.

## Chạy

```bash
make run-identity
```

Biến môi trường (có default):

| Biến | Default |
|---|---|
| `SERVICE_NAME` | `identity` |
| `GRPC_ADDR` | `:9081` (cmd/grpc listen) |
| `HTTP_ADDR` | `:8081` (cmd/http listen) |
| `GRPC_DIAL_ADDR` | rỗng -> suy ra `127.0.0.1:9081` (cmd/http dial target) |
| `LOG_LEVEL` | `info` |

## Thử API

```bash
curl -s localhost:8081/healthz

curl -s -X POST localhost:8081/v1/profiles \
  -H 'Content-Type: application/json' \
  -d '{"email":"a@example.com","display_name":"Nguyen Van A"}'

curl -s localhost:8081/v1/profiles
```

gRPC trực tiếp (reflection đã bật):

```bash
grpcurl -plaintext localhost:9081 list
grpcurl -plaintext -d '{"id":"<uuid>"}' localhost:9081 identity.v1.ProfileService/GetProfile
```

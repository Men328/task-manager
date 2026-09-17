# common

Go module **`taskmanager/common`** — chứa interface (proto) và code dùng chung cho mọi service.

| Thư mục | Vai trò | Được sửa tay? |
|---|---|---|
| `proto/` | **Interface** — toàn bộ file `.proto` | ✅ có |
| `gen/` | **Code gen** — output của `buf generate` | ❌ không (bị ghi đè) |
| `errorcode/` | **Bộ mã lỗi chuẩn** — `error_codes.json` + helper Go | ✅ có (xem `errorcode/README.md`) |

## Service dùng `common` thế nào

Trong `service/<name>/go.mod`:

```go
module taskmanager/service/identity

require taskmanager/common v0.0.0-00010101000000-000000000000

replace taskmanager/common => ../../common
```

Rồi import bình thường:

```go
import (
    identityv1 "taskmanager/common/gen/go/identity/v1"
    taskv1     "taskmanager/common/gen/go/task/v1"
)
```

- Root `go.work` gom `common` + các service để dev local.
- `replace` giúp module build được **kể cả khi không có `go.work`** — đây là cách Dockerfile build.
- `go.sum` **không** có entry cho `taskmanager/common`: module thay bằng đường dẫn filesystem thì
  không cần checksum. `go.sum` chỉ chứa checksum của dependency tải từ registry.

## Thêm proto mới

1. Tạo `common/proto/<service>/v1/<resource>.proto`:

```proto
syntax = "proto3";
package myservice.v1;

import "google/api/annotations.proto";

option go_package = "taskmanager/common/gen/go/myservice/v1;myservicev1";

service FooService {
  rpc GetFoo(GetFooRequest) returns (GetFooResponse) {
    option (google.api.http) = {get: "/v1/foos/{id}"};
  }
}
```

2. Chạy `make gen` → code vào `common/gen/go/myservice/v1/` + `common/gen/openapi/`.
3. Service mới cần 1 dòng `go work use ./service/<name>` để vào workspace.

## Ghi chú

- `buf.yaml` depend vào `buf.build/googleapis/googleapis` để có `google/api/annotations.proto`.
- `buf.gen.yaml` dùng `paths=source_relative` nên vị trí file gen khớp với `go_package`.
- Plugin `protoc-gen-openapiv2` dùng `strategy: all` — nếu không, file swagger merge sẽ bị ghi đè
  và mất service của directory khác.
- `buf.lock` do `buf dep update` sinh ra — nên commit.

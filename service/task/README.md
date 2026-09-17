# service/task

Service quản lý task, status và lifecycle (rule chuyển trạng thái). Là **1 Go module riêng**
(`taskmanager/service/task`), dùng code chung qua module `taskmanager/common`
(xem `replace` trong `go.mod`).

- gRPC: `:9082` — HTTP/JSON gateway: `:8082`
- Proto: `common/proto/task/v1/{task,status,transition}.proto`
- Bảng DB: `task.TASKS`, `task.TASK_STATUSES`, `task.STATUS_TRANSITIONS`, `task.TASK_STATUS_LOGS`
  (xem `deployments/migrations/`)

## Cấu trúc

```
Dockerfile                       # build 2 binary grpc + http (build context = root repo)
cmd/grpc/                        # fx app: gRPC server (:9082), register 3 service + lifecycle
cmd/http/                        # fx app: grpc-gateway (:8082) + lifecycle, dial qua GRPC_DIAL_ADDR
internal/config/config.go
internal/model/                  # CORE: Task, Status, Transition, StatusLog + enum + lỗi domain (chỉ stdlib)
internal/service/                # NGHIỆP VỤ + interface contract (interfaces.go): 3 port + 3 service
internal/repository/             # adapter I/O (in-memory stub) cho 3 port, KHÔNG import service
internal/dependency/             # validate + mapping (model <-> proto, convert enum) + map lỗi -> gRPC
internal/handler/                # transport gRPC mỏng cho 3 service
```

**Dependency inversion**: `service/interfaces.go` khai báo 3 port I/O
(`TaskRepository`, `StatusRepository`, `TransitionRepository`) và 3 contract service.
`service` chỉ import `model` — không import `repository`/`dependency`. Adapter ở
`repository` thoả mãn port nhờ structural typing (không import `service`). `cmd` là nơi duy
nhất wiring concrete vào interface.

## Nghiệp vụ đã có trong base

- **Task cha/con**: `parent_task_id`, chặn tạo vòng lặp (`ErrCycle`), không xoá task còn task con.
- **Status theo profile**: mỗi profile có bộ status riêng; set `is_default` sẽ tự bỏ default của status khác; không xoá status đang được task dùng (giống `ON DELETE RESTRICT`).
- **Lifecycle allowlist** (`STATUS_TRANSITIONS`): `ChangeTaskStatus` chỉ cho phép khi tồn tại rule `from -> to` đang active. Không có rule = cấm. `ValidateStatusTransition` để check trước.
- **Audit**: mỗi lần tạo task / đổi status ghi 1 dòng `TaskStatusLog` (in-memory).
- Chuyển sang status `category = DONE` tự set `completed_at`; rời DONE thì xoá.

## Chạy

```bash
make run-task
```

| Biến | Default |
|---|---|
| `SERVICE_NAME` | `task` |
| `GRPC_ADDR` | `:9082` (cmd/grpc listen) |
| `HTTP_ADDR` | `:8082` (cmd/http listen) |
| `GRPC_DIAL_ADDR` | rỗng -> suy ra `127.0.0.1:9082` (cmd/http dial target) |
| `LOG_LEVEL` | `info` |

## Thử luồng lifecycle

```bash
P=<profile_id>

# 1. tạo 3 status
curl -s -X POST localhost:8082/v1/statuses -H 'Content-Type: application/json' \
  -d "{\"profile_id\":\"$P\",\"name\":\"Todo\",\"slug\":\"todo\",\"is_default\":true,\"category\":\"TASK_STATUS_CATEGORY_TODO\"}"
curl -s -X POST localhost:8082/v1/statuses -H 'Content-Type: application/json' \
  -d "{\"profile_id\":\"$P\",\"name\":\"Doing\",\"slug\":\"doing\",\"category\":\"TASK_STATUS_CATEGORY_IN_PROGRESS\"}"
curl -s -X POST localhost:8082/v1/statuses -H 'Content-Type: application/json' \
  -d "{\"profile_id\":\"$P\",\"name\":\"Done\",\"slug\":\"done\",\"is_terminal\":true,\"category\":\"TASK_STATUS_CATEGORY_DONE\"}"

# 2. khai báo rule: Todo -> Doing được phép; Todo -> Done KHÔNG khai báo => bị cấm
curl -s -X POST localhost:8082/v1/transitions -H 'Content-Type: application/json' \
  -d "{\"profile_id\":\"$P\",\"from_status_id\":\"<todo_id>\",\"to_status_id\":\"<doing_id>\"}"

# 3. tạo task (không truyền status_id -> dùng status default)
curl -s -X POST localhost:8082/v1/tasks -H 'Content-Type: application/json' \
  -d "{\"profile_id\":\"$P\",\"title\":\"Viết báo cáo\"}"

# 4. đổi status
curl -s -X POST localhost:8082/v1/tasks/<task_id>/status -H 'Content-Type: application/json' \
  -d '{"status_id":"<doing_id>","note":"bắt đầu làm"}'
```

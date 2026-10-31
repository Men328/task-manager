# service/project

Service quản lý **project** (nhóm công việc) của từng profile. Mỗi profile có đúng 1 project mặc định
`Chung` (không thể archive), được tạo lazily qua `SeedDefaultProject` hoặc khi `ListProjects`/`CreateProject`.
Là 1 Go module riêng (`taskmanager/service/project`), dùng `taskmanager/common` qua `replace`.

- gRPC: `:9090` — HTTP/JSON gateway: `:8090`
- Proto: `common/proto/project/v1/project.proto`
- Bảng DB: `project.projects` (`deployments/migrations/000011_init_project.up.sql`)
- Env: `SERVICE_NAME`, `GRPC_ADDR`, `HTTP_ADDR`, `GRPC_DIAL_ADDR`, `LOG_LEVEL`, `DATABASE_URL` (trống = in-memory)

Cấu trúc giống `service/calendar`: `cmd/{grpc,http}`, `internal/{config,model,service,repository,dependency,handler}`.

## Quy tắc
- Tên trim, bắt buộc, không trùng trong profile (không phân biệt hoa thường).
- `Create` không bao giờ tạo project mặc định; project archived không sửa được; project mặc định không archive được.

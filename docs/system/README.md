# Tài liệu hệ thống — Task Manager

Bộ tài liệu tổng hợp từ việc quét toàn bộ mã nguồn (13 service Go, `common/proto`, `frontend`,
`deployments`). Sơ đồ viết bằng **Mermaid** — xem trực tiếp trên GitHub, VS Code (extension
*Markdown Preview Mermaid Support*) hoặc dán vào <https://mermaid.live>.

| # | File | Nội dung |
|---|---|---|
| 01 | [01-component-graph.md](01-component-graph.md) | Graph các thành phần, chức năng từng service/trang, liên hệ dữ liệu (ERD), cấu trúc bên trong 1 service |
| 02 | [02-architecture.md](02-architecture.md) | Sơ đồ kiến trúc: triển khai Docker, các lớp logic, xác thực, sequence các luồng chính, quyết định kiến trúc |
| 03 | [03-business.md](03-business.md) | Nghiệp vụ: thuật ngữ, quy tắc từng phân hệ, quy tắc xuyên suốt |
| 04 | [04-user-stories.md](04-user-stories.md) | User story theo epic, kèm tiêu chí chấp nhận |
| 05 | [05-service-calls-and-data.md](05-service-calls-and-data.md) | **Chi tiết**: service nào gọi service nào, bằng RPC/API nào, cấu hình nào; service nào đọc/ghi Postgres / S3 / Kafka / Soketi; bảng & cột; luồng end-to-end từng bước |
| 06 | [06-mail-flow.md](06-mail-flow.md) | Sơ đồ đơn giản luồng xử lý mail: Gmail → phân loại → task / lịch / sự kiện / backlog → thông báo |

Tóm tắt một dòng: SPA React → nginx `/api/*` → grpc-gateway → gRPC server của từng service;
service gọi nhau bằng gRPC; dữ liệu ở PostgreSQL (schema riêng mỗi service), tệp ở RustFS (S3),
chat qua Kafka + worker, realtime qua Soketi; **không dùng Redis**.

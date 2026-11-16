# 06 — Luồng xử lý mail (đơn giản)

Luồng tự động biến email Gmail thành công việc / lịch / sự kiện / backlog.
Chi tiết từng bước gọi API: [02-architecture.md §4.2](02-architecture.md),
[05-service-calls-and-data.md §5.2](05-service-calls-and-data.md),
`service/mail-provider/README.md`.

```mermaid
flowchart TD
    A["Gmail có thư mới"] --> B["Google Pub/Sub báo cho mail-provider"]
    B --> C["Lấy thư mới từ Gmail<br/>(tính từ mốc historyId đã lưu)"]
    C --> D["Lưu email gốc + file đính kèm<br/>lên RustFS (S3)"]
    D --> E{"DeepSeek dùng được?"}
    E -- Có --> F["AI phân loại"]
    E -- Không --> G["Phân loại bằng từ khoá"]
    F --> H{"Loại email?"}
    G --> H
    H -- task --> T["Tạo công việc<br/>(task service)"]
    H -- schedule --> S["Tạo lịch<br/>(calendar service)"]
    H -- event --> V["Tạo sự kiện<br/>(event service)"]
    H -- "other / thiếu thông tin" --> BL["Đưa vào backlog<br/>(backlog service)"]
    T --> N["Tạo thông báo + đẩy realtime<br/>(notification → Soketi)"]
    S --> N
    V --> N
    BL --> N
    N --> K["Lưu mốc historyId mới"]
```

**Ghi nhớ nhanh**

- Không email nào bị mất: phân loại không chắc hoặc thiếu dữ kiện (ví dụ lịch không có giờ) đều vào **backlog** kèm lý do.
- AI lỗi hoặc không có `DEEPSEEK_API_KEY` thì tự chuyển sang **luật từ khoá**.
- Chỉ lưu mốc `historyId` mới **sau khi xử lý xong**, nên thông báo trùng hoặc đến sai thứ tự vẫn không làm mất thư.

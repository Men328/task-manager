# report service

Service báo cáo (report): tổng hợp dữ liệu từ các service khác rồi trả các đầu API phục vụ UI
báo cáo. Service **không có DB riêng** — nguồn dữ liệu qua gRPC:

- `task` (`TaskService.ListTasks` + `TaskStatusService.ListTaskStatuses`) — task, status, lifecycle.
- `event` (`EventService.ListEvents`) — sự kiện.
- `calendar` (`CalendarService.ListSchedules`) — lịch.
- `backlog` (`BacklogService.ListBacklogs`) — email không phân loại được.

- gRPC: `:9087` — HTTP gateway: `:8087`
- Interface (proto): `common/proto/report/v1/report.proto`
- Code gen: `common/gen/go/report/v1`

## API

Tất cả endpoint nhận `profile_id` (bắt buộc), `from`/`to` (RFC3339, tuỳ chọn — bỏ trống =
30 ngày gần nhất) và `include_archived` (mặc định false). Khoảng thời gian lọc task theo
`created_at`; riêng chuỗi thời gian đếm thêm task `completed_at` trong kỳ.

| Method | Path | Mô tả |
|---|---|---|
| GET | `/v1/reports/overview` | Tổng quan task: tổng task, root/subtask, đếm theo nhóm status, quá hạn, sắp đến hạn, tạo/hoàn thành trong kỳ, tỉ lệ hoàn thành & quá hạn (%) |
| GET | `/v1/reports/status-breakdown` | Tỉ lệ % task theo từng status (kèm màu, category, count) |
| GET | `/v1/reports/timeseries` | Chuỗi thời gian theo bucket `day`/`week`/`month`: số task tạo mới & hoàn thành mỗi bucket (zero-fill đủ bucket) |
| GET | `/v1/reports/activity` | Thống kê **sự kiện + lịch + backlog** trong kỳ: sự kiện theo status/sắp tới/cả ngày, lịch trong kỳ/hôm nay/cả ngày/sắp tới, backlog theo status + theo category |

Quy ước khoảng thời gian:
- Task: lọc theo `created_at` trong `[from, to]` (task tạo trong kỳ).
- Sự kiện / lịch: tính theo **giao với** `[from, to]` (`COALESCE(end_at, start_at) >= from` và `start_at <= to`);
  các chỉ số `upcoming`/`today` không giới hạn khoảng.
- Backlog: lọc theo `created_at` trong `[from, to]`.

Ví dụ:

```bash
curl 'http://localhost:8087/v1/reports/overview?profile_id=<uuid>&from=2025-01-01T00:00:00Z&to=2025-02-01T00:00:00Z'
curl 'http://localhost:8087/v1/reports/status-breakdown?profile_id=<uuid>'
curl 'http://localhost:8087/v1/reports/timeseries?profile_id=<uuid>&interval=REPORT_INTERVAL_WEEK'
curl 'http://localhost:8087/v1/reports/activity?profile_id=<uuid>&from=2025-01-01T00:00:00Z&to=2025-02-01T00:00:00Z'
```

## Chạy local

```bash
# cần task/calendar/event/backlog service đang chạy (:9082/:9083/:9085/:9086)
make run-report           # gRPC :9087 + gateway :8087
make run-report-grpc
make run-report-http
```

Biến môi trường:

| Biến | Mặc định | Ý nghĩa |
|---|---|---|
| `GRPC_ADDR` | `:9087` | địa chỉ gRPC server |
| `HTTP_ADDR` | `:8087` | địa chỉ HTTP gateway |
| `GRPC_DIAL_ADDR` | — | gateway dial gRPC server (mặc định suy ra từ `GRPC_ADDR`) |
| `TASK_GRPC_DIAL_ADDR` | `127.0.0.1:9082` | task service (nguồn task/status) |
| `EVENT_GRPC_DIAL_ADDR` | `127.0.0.1:9085` | event service |
| `CALENDAR_GRPC_DIAL_ADDR` | `127.0.0.1:9083` | calendar service |
| `BACKLOG_GRPC_DIAL_ADDR` | `127.0.0.1:9086` | backlog service |
| `TASK_TIMEOUT` | `10s` | timeout mỗi lần gọi task service |
| `ACTIVITY_TIMEOUT` | `10s` | timeout mỗi lần gọi event/calendar/backlog |
| `LOG_LEVEL` | `info` | mức log |

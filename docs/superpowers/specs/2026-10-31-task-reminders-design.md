# Nhắc việc cho task

Ngày: 2026-10-31 · Nhánh: `feature/task-reminders` · Trạng thái: chờ duyệt
Roadmap: [personal-essentials-roadmap](2026-10-31-personal-essentials-roadmap.md)

## 1. Ý định

App task cá nhân mà không nhắc thì người dùng quên mở app. Hiện notice chỉ sinh
khi mail tạo item (`*_CREATED`). Thêm **một thời điểm nhắc** cho mỗi task; đến
giờ, người dùng nhận notice realtime (chuông + toast) như notice hiện có.

## 2. Mục tiêu / Không phải mục tiêu / Phản mục tiêu

**Mục tiêu**
- Task có `remind_at` (tuỳ chọn). Đến giờ → notice `TASK_REMINDER`, bấm mở task.
- Đặt nhắc trong form task với preset + tuỳ chọn giờ.
- Không gửi trùng, không mất nhắc khi worker restart.

**Không phải mục tiêu**
- Nhiều lần nhắc / task (bảng `task.reminders` — nâng cấp sau nếu cần).
- Notice **quá hạn** tự động: `due_at` hiện không phân biệt "chỉ ngày" (0h) với
  "có giờ", nên không biết lúc nào thực sự quá hạn; quá hạn đã hiển thị ở agenda.
  *Lựa chọn của Claude — cần người dùng xác nhận.*
- Web Push / email / thông báo khi không mở tab (roadmap sau: PWA + Web Push).
- Nhắc cho schedule/event.

**Phản mục tiêu**
- Một nhắc bắn 2 lần (nhiều replica `task-grpc`, restart giữa chừng).
- Deploy lần đầu bắn hàng loạt notice cũ.
- Đổi giờ nhắc mà vẫn không nhắc lại (vì đã đánh dấu đã nhắc).

## 3. Thiết kế

### 3.1 Migration `000013_task_reminders`

```sql
ALTER TABLE task.tasks
  ADD COLUMN remind_at   timestamptz,
  ADD COLUMN reminded_at timestamptz;

CREATE INDEX IF NOT EXISTS ix_tasks_reminder_due
  ON task.tasks (remind_at)
  WHERE remind_at IS NOT NULL AND reminded_at IS NULL
    AND deleted_at IS NULL AND NOT is_archived;
```

`notification.notices.type` là `varchar(64)` không ràng buộc, `target_type='task'`
đã hợp lệ → **không** cần migration phía notification.

### 3.2 Proto

- `Task`: `google.protobuf.Timestamp remind_at = 17;`
- `CreateTaskRequest`: `google.protobuf.Timestamp remind_at = 11;`
- `UpdateTaskRequest`: `google.protobuf.Timestamp remind_at = 11;` +
  `bool clear_remind_at = 12;` (Timestamp không có cách nói "xoá").
  `remind_at` có giá trị + `clear_remind_at=true` → `TASK_REMIND_AT_CONFLICT`.

### 3.3 Quy tắc

- Đặt/đổi `remind_at` → `reminded_at = NULL` (nhắc lại theo giờ mới).
- `remind_at` trong quá khứ khi tạo/sửa: chấp nhận nhưng **không** nhắc
  (set `reminded_at = now()` luôn). *Lựa chọn của Claude*, tránh notice bất ngờ.
- Task ở status nhóm done/cancelled, lưu trữ, xoá: không nhắc (điều kiện của
  worker). Mở lại task (status rời nhóm done/cancelled) mà `remind_at` còn ở tương lai → vẫn nhắc.
- Không ràng buộc `remind_at <= due_at`.

### 3.4 Worker (`task-grpc`)

- Gói `internal/service`: `ReminderWorker` với interface `ReminderStore`
  (claim/release) và `NoticeSender` (khai báo trong `interfaces.go` theo rule).
- Mỗi `REMINDER_INTERVAL` (mặc định `30s`) claim theo lô `REMINDER_BATCH` (100):

```sql
UPDATE task.tasks t SET reminded_at = now()
WHERE t.id IN (
  SELECT id FROM task.tasks
  WHERE remind_at <= now() AND reminded_at IS NULL
    AND deleted_at IS NULL AND NOT is_archived
    AND status_id IN (SELECT id FROM task.task_statuses
                      WHERE category NOT IN ('done','cancelled'))
  ORDER BY remind_at
  LIMIT $1
  FOR UPDATE SKIP LOCKED)
RETURNING <taskColumns>;
```

- Với mỗi task đã claim → `CreateNotice{ type: "TASK_REMINDER", target_type: TASK,
  target_id, title: "Nhắc việc: <title>", body: "Hạn: <due theo Asia/Ho_Chi_Minh>"
  hoặc rỗng }`. Gửi lỗi → `UPDATE ... SET reminded_at = NULL WHERE id = $1 AND
  reminded_at = <giá trị đã claim>` để lần sau thử lại (at-least-once; trùng chỉ
  xảy ra khi notification nhận nhưng phản hồi lỗi — chấp nhận).
- Bỏ qua task có `remind_at < now() - REMINDER_MAX_LATE` (mặc định `6h`): đánh dấu
  đã nhắc, không gửi — chặn phản mục tiêu "deploy bắn hàng loạt" và trường hợp
  worker chết lâu.
- Không có `DATABASE_URL` (repo in-memory) → worker không chạy, log cảnh báo.
- Gắn vào fx lifecycle (OnStart chạy goroutine, OnStop huỷ context, chờ vòng hiện
  tại xong). `REMINDER_ENABLED=false` để tắt.
- Client gRPC notification trong `internal/repository` (theo mẫu
  `mail-provider/internal/repository/notification_client.go`), wiring ở
  `cmd/grpc/infra.go`.

### 3.5 Hạ tầng

- `config.go`: `NOTIFICATION_GRPC_DIAL_ADDR`, `REMINDER_ENABLED`,
  `REMINDER_INTERVAL`, `REMINDER_BATCH`, `REMINDER_MAX_LATE`.
- `docker-compose.yml` (`task-grpc`): `NOTIFICATION_GRPC_DIAL_ADDR: "notification-grpc:9088"`,
  `depends_on: notification-grpc (service_started)`. Kiểm tra không tạo vòng
  depends_on (notification không phụ thuộc task).
- `scripts/run-service.sh` / README: biến môi trường khi chạy local.

### 3.6 Frontend

- `TaskFormModal` + `TaskDetailModal`: mục "Nhắc tôi" — Không nhắc · Lúc 9h ngày
  đến hạn · Trước 1 ngày (9h) · Tuỳ chọn (ngày + giờ). Preset dựa trên hạn chỉ hiện
  khi task có hạn. Hiển thị "🔔 9:00 31/10" trên `TaskCard`, dòng agenda.
- `NotificationMenu`: icon + nhãn cho `TASK_REMINDER`; bấm → mở task (dùng luồng
  `useNoticeFocus` có sẵn cho `target_type=task`).
- Toast realtime đã có qua soketi — kiểm tra `TASK_REMINDER` hiển thị nổi bật
  (không tự tắt quá nhanh).
- `api/task.ts`: `remindAt`, `clearRemindAt`.

## 4. Kiểm thử

- Service: đặt/đổi/xoá `remind_at` reset `reminded_at`; quá khứ không nhắc;
  xung đột clear.
- Worker (với store/sender giả): claim → gửi; gửi lỗi → release; quá trễ → bỏ qua;
  dừng sạch khi OnStop.
- Postgres: 2 worker chạy song song trên cùng dữ liệu không claim trùng
  (`SKIP LOCKED`).
- E2E tay: đặt nhắc 1 phút sau → chuông + toast trong ≤ `interval`.

## 5. Để lại cho builder

- Câu chữ notice, định dạng giờ trong `body` (dùng `profile.timezone` nếu muốn —
  cần gọi identity, mặc định cứ `Asia/Ho_Chi_Minh`).
- Có thêm metric/log đếm notice gửi không.

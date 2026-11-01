# Task lặp lại

Ngày: 2026-10-31 · Nhánh: `feature/recurring-tasks` · Trạng thái: chờ duyệt
Phụ thuộc: `feature/task-reminders` (task mới mang theo lịch nhắc);
nên sau `feature/quick-add` (mở rộng parser)
Roadmap: [personal-essentials-roadmap](2026-10-31-personal-essentials-roadmap.md)

## 1. Ý định

Việc định kỳ (nộp báo cáo thứ Sáu, đóng tiền điện ngày 5, tập thể dục mỗi sáng)
hiện phải tạo tay lại mỗi lần. Đánh dấu xong một lần → app tự tạo lần kế tiếp.

## 2. Mục tiêu / Không phải mục tiêu / Phản mục tiêu

**Mục tiêu**
- Task có quy tắc lặp (RRULE). Khi chuyển sang status nhóm **done** → tạo task kế
  tiếp với hạn mới (mô hình Todoist).
- UI chọn kiểu lặp phổ biến + tuỳ chỉnh "mỗi N ngày/tuần/tháng".
- Quick-add hiểu "mỗi ngày", "hằng tuần thứ 2", "every monday"…
- Dừng lặp: sửa task hiện tại bỏ quy tắc.

**Không phải mục tiêu**
- Lặp cho schedule/event. Sinh trước các lần lặp tương lai.
- Nhân bản task con của task lặp. *Lựa chọn của Claude* — có thể thêm sau.
- Kiểu "lặp tính từ ngày hoàn thành" (Todoist `every!`) — chỉ tính từ hạn.

**Phản mục tiêu**
- Mở lại rồi hoàn thành lại một task → sinh 2 task kế tiếp.
- Task lặp hằng ngày bị bỏ 10 ngày, hoàn thành 1 lần → sinh task hạn trong quá
  khứ (vẫn quá hạn ngay).
- Lần lặp bị transition guard chặn nên không tạo được, người dùng không biết.

## 3. Thiết kế

### 3.1 Migration `000014_task_recurrence`

```sql
ALTER TABLE task.tasks
  ADD COLUMN recurrence_rule      text,          -- RRULE không DTSTART, VD 'FREQ=WEEKLY;BYDAY=FR'
  ADD COLUMN recurrence_tz        varchar(64),   -- IANA, VD 'Asia/Ho_Chi_Minh'
  ADD COLUMN recurrence_series_id uuid,          -- id task đầu chuỗi
  ADD COLUMN recurrence_next_id   uuid;          -- task kế tiếp đã sinh (chống sinh trùng)

CREATE INDEX IF NOT EXISTS ix_tasks_recurrence_series
  ON task.tasks (recurrence_series_id) WHERE recurrence_series_id IS NOT NULL;
```

### 3.2 Proto

- `Task`: `string recurrence_rule = 18; string recurrence_tz = 19; string recurrence_series_id = 20;`
- `CreateTaskRequest`: `string recurrence_rule = 12; string recurrence_tz = 13;`
- `UpdateTaskRequest`: `optional string recurrence_rule = 13; optional string recurrence_tz = 14;`
  (`recurrence_rule = ""` = bỏ lặp).
- Mã lỗi mới: `TASK_RECURRENCE_INVALID` (InvalidArgument/400).

### 3.3 Quy tắc

- Validate bằng `github.com/teambition/rrule-go`: chỉ nhận `FREQ` ∈
  DAILY/WEEKLY/MONTHLY/YEARLY; `INTERVAL` 1–365; cho phép `BYDAY`, `BYMONTHDAY`,
  `BYMONTH`, `COUNT`, `UNTIL`; cấm `BYHOUR/BYMINUTE/BYSECOND/BYSETPOS` (giữ đơn giản).
  `recurrence_tz` phải load được bằng `time.LoadLocation`; rỗng → `Asia/Ho_Chi_Minh`.
- Task lặp **phải có `due_at`** (mốc để tính). Đặt quy tắc mà không có hạn →
  `TASK_RECURRENCE_INVALID`. *Lựa chọn của Claude*: đơn giản hơn so với lấy ngày
  hoàn thành làm mốc.
- Task con được lặp (giữ cùng `parent_task_id`).
- Tạo task lặp mới: `recurrence_series_id = id` của chính nó.

### 3.4 Sinh lần kế tiếp (trong `ChangeTaskStatus`)

Sau khi đổi status thành công và status đích có `category = done`, nếu task có
`recurrence_rule` và `recurrence_next_id IS NULL`:

1. `DTSTART = due_at` quy về `recurrence_tz`. Lấy lần xuất hiện đầu tiên
   `> max(due_at, thời điểm hoàn thành)` — bỏ qua các chu kỳ đã lỡ; hoàn thành được xem là xong
   chu kỳ hiện tại (người dùng đã duyệt 2026-11-01, thay cho mốc "đầu ngày hôm nay" ban đầu vốn sinh
   ngay một bản cùng hạn hôm nay khi hoàn thành task hằng ngày đúng hạn)
   (chặn phản mục tiêu 2). Giữ giờ-phút của `due_at` (0h nếu là task chỉ ngày).
2. Hết lần (COUNT/UNTIL) → không sinh, chuỗi kết thúc.
3. Tạo task mới, **cùng transaction** với việc ghi `recurrence_next_id` vào task cũ:
   - copy `title, description, priority, project_id, parent_task_id,
     recurrence_rule, recurrence_tz, recurrence_series_id`;
   - `due_at` = lần mới; `start_at`, `remind_at` (nếu có) **dịch cùng độ lệch**
     so với `due_at` cũ; `reminded_at = NULL`;
   - `status_id` = status mặc định (`resolveStatusID(ctx, profile, "")`) — tạo mới
     không qua transition nên guard không chặn (phản mục tiêu 3);
   - `position` = 0, như tạo task thường (không đẩy xuống cuối cột).
4. Task cũ giữ `recurrence_rule` (lịch sử), có `recurrence_next_id` → mở lại rồi
   xong lại không sinh nữa (phản mục tiêu 1).
5. Đổi status **commit trước**, sinh lần kế tiếp **sau** (bước 3 là một
   transaction riêng: insert task mới + ghi `recurrence_next_id`). Sinh lỗi (VD dự
   án đã lưu trữ, status mặc định không còn) → **không** làm hỏng việc hoàn thành:
   log lỗi, response không có `next_task` mà có `next_task_error` (mã lỗi), FE hiện
   cảnh báo "Không tạo được lần lặp tiếp theo: <lý do>". *Lựa chọn của Claude*:
   người dùng không bao giờ bị chặn đánh dấu xong chỉ vì phần lặp hỏng.

Repo cần thêm thao tác giao dịch `SpawnNextOccurrence(ctx, prevID, next Task)`
(insert + `UPDATE ... SET recurrence_next_id = $new WHERE id = $prev AND
recurrence_next_id IS NULL`; cập nhật 0 dòng → rollback, coi như đã có người sinh
trước); in-memory repo làm tương đương.

`ChangeTaskStatusResponse` thêm `Task next_task = 2; string next_task_error = 3;`
để FE hiện toast "Đã tạo lần tiếp theo: Thứ Sáu 7/11" hoặc cảnh báo.

### 3.5 Báo cáo

Mỗi lần lặp là một task riêng → báo cáo đếm số task hoàn thành tăng đúng theo số
lần làm. Đây là hành vi mong muốn; ghi chú trong README report.

### 3.6 Frontend

- Form task: mục "Lặp lại" — Không · Hằng ngày · Ngày làm việc (T2–T6) · Hằng tuần
  vào [thứ của hạn] · Hằng tháng ngày [ngày của hạn] · Hằng năm · Tuỳ chỉnh (mỗi N
  ngày/tuần/tháng, chọn thứ). Mục bị khoá khi chưa có hạn, kèm gợi ý.
  `recurrence_tz` = múi giờ trình duyệt.
- Hiển thị: icon lặp + mô tả ngắn ("Mỗi thứ Sáu") trên `TaskCard`, dòng agenda,
  `TaskDetailModal`.
- Toast sau khi hoàn thành task lặp, có nút mở task mới.
- Quick-add: thêm mẫu `mỗi ngày|hằng ngày|daily|every day`, `mỗi tuần|hằng tuần|weekly`,
  `mỗi thứ 2|every monday`, `mỗi tháng|monthly`, `mỗi ngày 5` (BYMONTHDAY=5),
  `mỗi N ngày|every N days`; ngày đầu tiên = lần xuất hiện gần nhất nếu không có
  ngày riêng. Thêm case test vào bộ test parser.

## 4. Kiểm thử

- Tính lần kế: hằng ngày lỡ 10 ngày → mai/hôm nay đúng; hằng tháng ngày 31 →
  tháng 30 ngày bỏ qua (theo RFC 5545) — ghi rõ trong UI "Hằng tháng ngày 31" có
  thể bỏ tháng; COUNT/UNTIL kết thúc chuỗi; giữ giờ.
- Hoàn thành → mở lại → hoàn thành: chỉ 1 task kế tiếp.
- Task có `remind_at` trước hạn 1 ngày → task mới cũng trước hạn mới 1 ngày.
- Task con lặp giữ cha; dự án lưu trữ → tạo lần kế tiếp lỗi thế nào (theo quy tắc
  project hiện có: không tạo task trong project archived) — chọn: không sinh + log.

## 5. Để lại cho builder

- Câu chữ mô tả quy tắc ở FE (i18n vi/en).

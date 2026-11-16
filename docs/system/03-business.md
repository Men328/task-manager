# 03 — Nghiệp vụ dự án

## 1. Mục tiêu sản phẩm

**Task Manager** là ứng dụng quản lý công việc **cá nhân** (mỗi người dùng = 1 *profile*), gom
về một chỗ:

1. **Công việc (task)** có vòng đời do chính người dùng định nghĩa (status + rule chuyển trạng thái).
2. **Kế hoạch thời gian**: lịch (schedule), sự kiện (event), việc hôm nay (agenda).
3. **Hộp thư thông minh**: tự đọc Gmail, phân loại email thành task / lịch / sự kiện / backlog.
4. **Tổng quan**: báo cáo tiến độ, tỉ lệ hoàn thành, hoạt động.
5. **Kết nối**: kết bạn và nhắn tin 1-1 realtime, thông báo realtime.

Đối tượng là người dùng cá nhân — không có khái niệm tổ chức, phân quyền nhóm hay sprint.

## 2. Thuật ngữ (domain glossary)

| Thuật ngữ | Ý nghĩa | Service |
|---|---|---|
| Profile | Tài khoản người dùng (email, tên hiển thị) | identity |
| Auth provider | Liên kết profile với tài khoản Google (`provider`, `subject`) | identity |
| Session token | JWT HS256 phát sau đăng nhập, mặc định sống 30 ngày (`SESSION_TTL=720h`) | identity |
| Project | Nhóm công việc; mỗi profile có đúng 1 project mặc định `Chung` | project |
| Task | Việc cần làm, có thể có task con (`parent_task_id`) | task |
| Status | Trạng thái task theo profile, có `category` TODO / IN_PROGRESS / DONE | task |
| Transition (lifecycle rule) | Rule cho phép chuyển `from_status → to_status` (allowlist) | task |
| Status log | Lịch sử đổi trạng thái (audit) | task |
| Schedule (lịch) | Khối thời gian trên lịch, không có vòng đời | calendar |
| Event (sự kiện) | Sự kiện có vòng đời PLANNED / CONFIRMED / CANCELLED | event |
| Backlog | Email không phân loại được hoặc thiếu dữ kiện, chờ người dùng xử lý | backlog |
| Attachment | Tệp đính kèm cho task / lịch / sự kiện / backlog | attachment |
| Notice | Thông báo trong app (chuông) | notification |
| Friendship | Quan hệ bạn bè: `pending` → `accepted` | social |
| Conversation / Message | Cuộc trò chuyện 1-1 và tin nhắn | message, worker |

## 3. Các phân hệ nghiệp vụ

### 3.1 Tài khoản & đăng nhập (identity)

- Đăng nhập **chỉ bằng Google OAuth** (PKCE + `state` ký HMAC, có hạn).
- Lần đầu đăng nhập ⇒ tạo profile mới (`LoginWithProvider`) và **seed dữ liệu mặc định**:
  - 4 status: `To Do` (mặc định) → `On Progress` → `In Review` → `Completed` (terminal, DONE);
  - 5 rule: todo→on-progress, on-progress→in-review, in-review→completed, in-review→on-progress,
    completed→on-progress;
  - project mặc định `Chung`.
  Seed là **best-effort**: lỗi chỉ log, profile vẫn được tạo; người dùng có thể seed lại (idempotent).
- Tuỳ chọn **"Cho phép đọc Gmail"** ngay ở màn đăng nhập: xin thêm scope `gmail.readonly` +
  `access_type=offline`, rồi gọi `mail-provider.Subscribe`.
- Cấp chữ ký subscribe kênh realtime private (Soketi) — chỉ cho kênh của chính mình
  (`IDENTITY_AUTH_SOKETI_CHANNEL_FORBIDDEN` nếu xin kênh người khác).

### 3.2 Project

- Tên bắt buộc, trim, **không trùng trong cùng profile** (không phân biệt hoa thường) —
  `PROJECT_NAME_DUPLICATE`.
- Project mặc định `Chung`: tạo lazily, **không archive được**; `CreateProject` không bao giờ tạo
  project mặc định.
- Project đã archive **không sửa được** và **không gán thêm** task/lịch/sự kiện (`PROJECT_ARCHIVED`).
- Task, lịch, sự kiện **bắt buộc** thuộc 1 project (DB `NOT NULL` + FK `ON DELETE RESTRICT`).

### 3.3 Task, status & lifecycle

**Task**

- Thuộc 1 profile + 1 project; trường: tiêu đề (bắt buộc), mô tả, ưu tiên
  (`LOW / MEDIUM / HIGH / URGENT`), `start_at`, `due_at`, `remind_at`, `recurrence_rule`,
  `is_archived`.
- **Task cha/con**: task con luôn **cùng project với cha** (project client gửi lên bị bỏ qua); đổi
  project của task gốc ⇒ đổi project cho **cả cây con** (`SetProjectForSubtree`); chặn tạo vòng
  lặp cha-con (`TASK_CYCLE_DETECTED`); không xoá task còn task con.
- Tạo task không truyền `status_id` ⇒ dùng status mặc định của profile.
- Tiến độ trên card = số task con có status `category = DONE` / tổng task con.
- Danh sách task: lọc theo profile, project, status, khoảng hạn (`due_from/due_to`), `open_only`,
  `root_only`, tìm kiếm full-text (`query`, index trigram), sắp xếp, phân trang (`page_token`).

**Status**

- Mỗi profile có bộ status riêng, có `position` (thứ tự cột Kanban) và `color`.
- Chỉ 1 status `is_default` mỗi profile (set default mới ⇒ bỏ default cũ).
- Không xoá status đang được task dùng (`ON DELETE RESTRICT`).

**Lifecycle (allowlist)**

- `ChangeTaskStatus` chỉ hợp lệ khi tồn tại rule `from → to` **đang active**. Không có rule = **cấm**
  (`TASK_TRANSITION_NOT_ALLOWED`); UI hiện thông báo đỏ khi kéo thả sai.
- `ValidateStatusTransition` để kiểm tra trước; không cho rule trùng
  (`TASK_TRANSITION_ALREADY_EXISTS`) hay `from = to` (`TASK_TRANSITION_SAME_STATUS`).
- Chuyển sang status `DONE` ⇒ tự set `completed_at`; rời DONE ⇒ xoá `completed_at`.
- Mỗi lần tạo task / đổi status ⇒ ghi 1 dòng `task.task_status_logs` (kèm `note`).

**Nhắc việc (reminder)**

- Task có `remind_at`; worker nền trong `task-grpc` định kỳ *claim* các task đến hạn nhắc, gửi
  notice `TASK_REMINDER` sang `notification`. Gửi lỗi ⇒ *release* để thử lại lần sau. Múi giờ mặc
  định `Asia/Ho_Chi_Minh`.

**Task lặp lại (recurrence)**

- `recurrence_rule` dạng RRULE + `recurrence_tz`. Khi hoàn thành (DONE) một lần lặp ⇒ tự sinh task
  kế tiếp với hạn mới, cùng `recurrence_series_id`; `recurrence_next_id` chống sinh trùng khi mở
  lại rồi hoàn thành lại.
- Báo cáo đếm **mỗi lần lặp là 1 task** (task "hằng ngày" làm đủ 7 ngày = 7 task hoàn thành).

**Thêm nhanh (quick add, FE)**

- Một dòng như `Nộp báo cáo mai 9h !cao #đồ-án` ⇒ tách tiêu đề + hạn + ưu tiên + project.

### 3.4 Lịch (calendar) & sự kiện (event)

| | Lịch (`calendar.schedules`) | Sự kiện (`event.events`) |
|---|---|---|
| Bản chất | Khối thời gian trên lịch | Sự kiện cần xác nhận |
| Vòng đời | Không | `PLANNED` → `CONFIRMED` → `CANCELLED` |
| Trường riêng | `all_day`, `color` (`#RRGGBB[AA]`), `location` | `status`, `source` (id email nguồn), `location`, `all_day` |
| Bắt buộc | `title`, `profile_id`, `start_at`, project | như lịch |
| Ràng buộc | `end_at ≥ start_at` (`CALENDAR_TIME_RANGE_INVALID`) | `end_at ≥ start_at` (`EVENT_TIME_RANGE_INVALID`) |
| Truy vấn | lọc theo khoảng `[from, to)` (giao nhau) | khoảng + `status` |

Chưa hỗ trợ: lặp/RRULE cho lịch & sự kiện, người tham gia, đồng bộ Google Calendar.

### 3.5 Hộp thư thông minh (mail-provider + backlog)

1. **Đăng ký watch**: `users.watch(topic)` với hạn tối đa 7 ngày; worker nền **gia hạn** mỗi
   `12h` nếu còn < `24h`. Token Gmail (access + refresh) lưu ở `mail_provider.sessions`.
2. **Nhận thông báo**: Pub/Sub push (webhook, xác thực OIDC JWT) hoặc pull (long-poll).
   Ack ngay; chỉ **không ack** khi lỗi tạm thời (queue đầy) để Pub/Sub gửi lại.
3. **Không tin `historyId` trong notice**: đọc checkpoint đã lưu, `users.history.list` từ checkpoint,
   xử lý xong mới tiến checkpoint ⇒ notice trùng/lệch thứ tự không làm mất email. Checkpoint quá cũ
   (404) ⇒ đặt lại theo `users.getProfile`, bỏ qua khoảng đã mất.
4. **Lưu email gốc** (JSON) + tối đa `MAIL_MAX_ATTACHMENTS=5` tệp, mỗi tệp ≤ 10 MiB lên S3 tại
   `mail/{profile_id}/{yyyymmdd}/{message_id}...`.
5. **Phân loại** đúng 1 category `task | schedule | event | other`:
   - Chính: DeepSeek trả JSON (category, title, description, priority, các mốc thời gian, location,
     reason, project gợi ý từ danh sách project của user).
   - Dự phòng: rule keyword (tiếng Việt + Anh) trên subject (×2) + snippet + body; hoà điểm ưu tiên
     `event > schedule > task > other`.
6. **Dispatch**:

   | Category | Điều kiện | Đích | Không đủ điều kiện ⇒ backlog với `reason` |
   |---|---|---|---|
   | task | actionable + có title | `task.CreateTask` | `not_actionable` |
   | schedule | title + `start_at` (hoặc `due_at`) | `calendar.CreateSchedule` | `missing_schedule_time` |
   | event | title + `start_at` (hoặc `due_at`) | `event.CreateEvent` | `missing_event_time` |
   | other | luôn | `backlog.CreateBacklog` | reason từ LLM / `unclassified` |

   Tắt cả LLM lẫn rule ⇒ mọi email vào backlog với `classifier_disabled`. **Không email nào bị mất.**
7. **Thông báo**: tạo thành công ⇒ `notification.CreateNotice` (`TASK_CREATED`,
   `SCHEDULE_CREATED`, `EVENT_CREATED`, `BACKLOG_CREATED`); lỗi chỉ log.

**Backlog**: trạng thái `NEW` (mặc định) → `TRIAGED` → `ARCHIVED`; category mặc định `other`; lưu
`sender`, `source` (message id), `reason`, `object_key` (bản gốc trên S3) để người dùng xử lý lại.

Giới hạn đã biết: không dedupe theo `messageId` Gmail ⇒ có thể tạo trùng nếu checkpoint bị đặt lại
hoặc Pub/Sub gửi lại (at-least-once).

### 3.6 Tệp đính kèm (attachment)

- Gắn với đối tượng bằng `(owner_type, owner_id)`, `owner_type ∈ task | schedule | event | backlog`.
- Upload multipart (`POST /v1/attachments/upload`), tải xuống (`GET /v1/attachments/{id}/download`,
  có `Content-Disposition`, `nosniff`), liệt kê, xoá.
- Giới hạn kích thước `ATTACHMENT_MAX_BYTES` (mặc định 10 MiB).
- Metadata ở Postgres, nội dung ở RustFS (S3) theo `object_key`.

### 3.7 Báo cáo (report)

- Đầu vào: `profile_id` (bắt buộc), `from/to` (mặc định 30 ngày gần nhất), `include_archived`,
  `project_id` (tuỳ chọn).
- **Overview**: tổng task, root/subtask, theo nhóm status, quá hạn, sắp đến hạn, tạo/hoàn thành
  trong kỳ, tỉ lệ hoàn thành & quá hạn (%).
- **Status breakdown**: % task theo từng status (màu, category, count).
- **Time series**: bucket `day / week / month`, số task tạo & hoàn thành mỗi bucket (zero-fill).
- **Activity**: sự kiện theo status / sắp tới / cả ngày; lịch trong kỳ / hôm nay / sắp tới;
  backlog theo status + category.
- Quy ước kỳ: task & backlog theo `created_at`; lịch & sự kiện theo **giao** với `[from, to]`.

### 3.8 Thông báo (notification)

- Notice có `type`, `title`, `target_type` (TASK / SCHEDULE / EVENT / BACKLOG / FRIEND), `target_id`,
  `is_read`.
- Nguồn tạo: mail-provider (đối tượng mới từ email), task (nhắc việc), social (lời mời / chấp nhận
  kết bạn). `CreateNotice` là RPC **nội bộ**, không lộ ra Internet.
- Lưu trước, **push realtime sau** (Soketi lỗi vẫn giữ notice). Browser nhận `notice.created` ⇒ gọi
  API đếm chưa đọc để cập nhật badge.
- Người dùng chỉ đọc / đánh dấu notice của chính mình (scope theo `sub` trong token, cả gateway lẫn SQL);
  đánh dấu tất cả bằng `ids: ["*"]`. Bấm notice ⇒ mở đúng đối tượng (`?focus=<id>`).

### 3.9 Bạn bè (social)

- Tìm người theo **email** (qua `identity.FindProfileByEmail`).
- Gửi lời mời ⇒ `pending`; người nhận **chấp nhận** (`accepted`) hoặc **từ chối**; người gửi có thể
  **huỷ**; bạn bè có thể **huỷ kết bạn**. Mỗi cặp chỉ có 1 dòng bất kể ai gửi; không tự kết bạn
  (`SOCIAL_CANNOT_FRIEND_SELF`).
- Gửi / chấp nhận lời mời ⇒ notice `FRIEND_REQUEST` / `FRIEND_ACCEPTED` cho bên kia.
- Người gọi luôn lấy từ token (`sub`), không nhận từ client.

### 3.10 Nhắn tin (message + worker)

- Chỉ **bạn bè** mới nhắn được (`MESSAGE_NOT_FRIENDS`); nội dung 1..2000 ký tự; không tự nhắn mình.
- `conversation_id = UUIDv5(cặp profile)` (ổn định), `id = UUIDv7`.
- **Realtime trước, lưu sau**: broadcast `message.created` tới `private-chat-<người nhận>` và
  `private-chat-<người gửi>` (đồng bộ nhiều tab), rồi đẩy async vào Kafka; worker ghi DB.
- FE gửi lạc quan với `client_id`, gộp bản realtime/bản lưu theo `id`/`client_id`.
- Lịch sử: 100 cuộc trò chuyện gần nhất (kèm tin cuối); tin nhắn phân trang `before` + `limit=50`.
- Đảm bảo: at-least-once, idempotent theo `id`, thứ tự trong 1 cuộc trò chuyện (cùng partition).

## 4. Quy tắc xuyên suốt

| Quy tắc | Mô tả |
|---|---|
| Phạm vi dữ liệu | Mọi dữ liệu thuộc 1 `profile_id`; xoá profile ⇒ cascade toàn bộ |
| Mã lỗi chuẩn | Lỗi trả `google.rpc.ErrorInfo.reason`; FE dịch vi/en, không hiện lỗi kỹ thuật |
| Side effect best-effort | Seed, gửi notice, broadcast Soketi lỗi ⇒ chỉ log, không rollback thao tác chính |
| Không mất dữ liệu email | Mọi email đều kết thúc ở task/lịch/sự kiện hoặc backlog |
| Chạy không cần DB | Thiếu `DATABASE_URL` ⇒ repository in-memory (dev/test) |
| Đa ngôn ngữ | UI vi/en; bộ mã lỗi có message vi/en |

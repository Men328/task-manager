<!-- Sinh bởi _tools/gen_phu_luc.py. Sửa script rồi chạy lại, không sửa tay. -->
## Phụ lục A. Hướng dẫn cài đặt và chạy hệ thống

**Yêu cầu:** Go 1.25 trở lên, Node.js 20 trở lên, Docker và Docker Compose. Các bước:

- Lấy mã nguồn và tạo tệp cấu hình: sao chép `deployments/docker/.env.example` thành `deployments/docker/.env`, điền `SESSION_SECRET`, `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, thông tin RustFS và Soketi.
- Tuỳ chọn tự động phân loại email: điền `MAIL_PUBSUB_TOPIC` (topic Pub/Sub cùng dự án Google Cloud với OAuth client), `DEEPSEEK_API_KEY`; cấp quyền publish cho `gmail-api-push@system.gserviceaccount.com` trên topic.
- Cài công cụ và sinh mã (lần đầu): `make tools`, `make gen`.
- Khởi động toàn bộ hệ thống: `make up` (hoặc `make up-tunnel` để bật Cloudflare Tunnel). Lệnh này chạy PostgreSQL, migration, RustFS, Soketi, Kafka, 13 dịch vụ và giao diện.
- Truy cập `http://localhost:3000`, đăng nhập bằng tài khoản Google, tích "Cho phép đọc Gmail" nếu muốn bật phân loại email.
- Kiểm tra: `make ps` xem trạng thái container; `make test` chạy kiểm thử đơn vị; `make quality-all` kiểm tra chất lượng mã; `make down` dừng hệ thống.

## Phụ lục B. Danh sách đầy đủ API

Đường dẫn dưới đây được gọi qua cổng vào với tiền tố `/api/<dịch vụ>`, ví dụ `GET /api/task/v1/tasks`. Cột *Xác thực*: JWT là session token trong tiêu đề Authorization; “Không” là gateway chưa kiểm tra token mà nhận `profile_id` trong yêu cầu (xem mục 4.3.4).

[BANG: Bảng B.1: Danh sách đầy đủ API của hệ thống]
| Dịch vụ | Phương thức | Đường dẫn | Chức năng / RPC | Xác thực |
|---|---|---|---|---|
| identity | GET | `/v1/auth/google/callback` | Nhận mã uỷ quyền từ Google | state + PKCE |
| identity | GET | `/v1/auth/google/login` | Bắt đầu đăng nhập Google | Không (trang công khai) |
| identity | GET | `/v1/auth/me` | Hồ sơ người dùng hiện tại | JWT |
| identity | POST | `/v1/auth/soketi` | Ký uỷ quyền kênh realtime riêng tư | JWT |
| identity | GET | `/v1/profiles` | ListProfiles | Không |
| identity | POST | `/v1/profiles` | CreateProfile | Không |
| identity | DELETE | `/v1/profiles/{id}` | DeleteProfile | Không |
| identity | GET | `/v1/profiles/{id}` | GetProfile | Không |
| identity | PATCH | `/v1/profiles/{id}` | UpdateProfile | Không |
| project | GET | `/v1/projects` | ListProjects | Không |
| project | POST | `/v1/projects` | CreateProject | Không |
| project | GET | `/v1/projects/{id}` | GetProject | Không |
| project | PATCH | `/v1/projects/{id}` | UpdateProject | Không |
| project | POST | `/v1/projects/{id}/archive` | ArchiveProject | Không |
| project | POST | `/v1/projects/{id}/unarchive` | UnarchiveProject | Không |
| task | GET | `/v1/statuses` | ListTaskStatuses | Không |
| task | POST | `/v1/statuses` | CreateTaskStatus | Không |
| task | POST | `/v1/statuses/seed` | SeedDefaultStatuses | Không |
| task | DELETE | `/v1/statuses/{id}` | DeleteTaskStatus | Không |
| task | GET | `/v1/statuses/{id}` | GetTaskStatus | Không |
| task | PATCH | `/v1/statuses/{id}` | UpdateTaskStatus | Không |
| task | GET | `/v1/tasks` | ListTasks | Không |
| task | POST | `/v1/tasks` | CreateTask | Không |
| task | DELETE | `/v1/tasks/{id}` | DeleteTask | Không |
| task | GET | `/v1/tasks/{id}` | GetTask | Không |
| task | PATCH | `/v1/tasks/{id}` | UpdateTask | Không |
| task | POST | `/v1/tasks/{id}/status` | ChangeTaskStatus | Không |
| task | GET | `/v1/transitions` | ListStatusTransitions | Không |
| task | POST | `/v1/transitions` | CreateStatusTransition | Không |
| task | POST | `/v1/transitions/validate` | ValidateStatusTransition | Không |
| task | DELETE | `/v1/transitions/{id}` | DeleteStatusTransition | Không |
| task | GET | `/v1/transitions/{id}` | GetStatusTransition | Không |
| task | PATCH | `/v1/transitions/{id}` | UpdateStatusTransition | Không |
| calendar | GET | `/v1/schedules` | ListSchedules | Không |
| calendar | POST | `/v1/schedules` | CreateSchedule | Không |
| calendar | DELETE | `/v1/schedules/{id}` | DeleteSchedule | Không |
| calendar | GET | `/v1/schedules/{id}` | GetSchedule | Không |
| calendar | PATCH | `/v1/schedules/{id}` | UpdateSchedule | Không |
| event | GET | `/v1/events` | ListEvents | Không |
| event | POST | `/v1/events` | CreateEvent | Không |
| event | DELETE | `/v1/events/{id}` | DeleteEvent | Không |
| event | GET | `/v1/events/{id}` | GetEvent | Không |
| event | PATCH | `/v1/events/{id}` | UpdateEvent | Không |
| backlog | GET | `/v1/backlogs` | ListBacklogs | Không |
| backlog | POST | `/v1/backlogs` | CreateBacklog | Không |
| backlog | DELETE | `/v1/backlogs/{id}` | DeleteBacklog | Không |
| backlog | GET | `/v1/backlogs/{id}` | GetBacklog | Không |
| backlog | PATCH | `/v1/backlogs/{id}` | UpdateBacklog | Không |
| attachment | GET | `/v1/attachments` | ListAttachments | Không |
| attachment | POST | `/v1/attachments` | UploadAttachment | Không |
| attachment | POST | `/v1/attachments/upload` | Tải tệp lên (multipart) | Không |
| attachment | DELETE | `/v1/attachments/{id}` | DeleteAttachment | Không |
| attachment | GET | `/v1/attachments/{id}` | GetAttachment | Không |
| attachment | GET | `/v1/attachments/{id}/download` | Tải tệp xuống | Không |
| report | GET | `/v1/reports/activity` | GetActivityReport | Không |
| report | GET | `/v1/reports/overview` | GetReportOverview | Không |
| report | GET | `/v1/reports/status-breakdown` | GetStatusBreakdown | Không |
| report | GET | `/v1/reports/timeseries` | GetTaskTimeSeries | Không |
| notification | GET | `/v1/notices` | Danh sách thông báo | JWT |
| notification | POST | `/v1/notices/read` | Đánh dấu đã đọc (ids hoặc "*") | JWT |
| notification | GET | `/v1/notices/unread-count` | Số thông báo chưa đọc | JWT |
| social | GET | `/v1/friend-requests` | Lời mời đến và đã gửi | JWT |
| social | POST | `/v1/friend-requests` | Gửi lời mời kết bạn | JWT |
| social | DELETE | `/v1/friend-requests/{id}` | Huỷ lời mời đã gửi | JWT |
| social | POST | `/v1/friend-requests/{id}/accept` | Chấp nhận lời mời | JWT |
| social | POST | `/v1/friend-requests/{id}/decline` | Từ chối lời mời | JWT |
| social | GET | `/v1/friends` | Danh sách bạn bè | JWT |
| social | GET | `/v1/friends/search` | Tìm người dùng theo email | JWT |
| social | DELETE | `/v1/friends/{profile_id}` | Huỷ kết bạn | JWT |
| message | GET | `/v1/conversations` | Danh sách cuộc trò chuyện | JWT |
| message | GET | `/v1/conversations/{peer_id}/messages` | Lịch sử tin nhắn (phân trang) | JWT |
| message | POST | `/v1/conversations/{peer_id}/messages` | Gửi tin nhắn | JWT |
| message | POST | `/v1/conversations/{peer_id}/tasks` | Giao task qua chat (/task) | JWT |
| message | GET | `/v1/messages/{message_id}` | Đọc một tin nhắn | JWT |
| mail-provider | POST | `/v1/notifications` | Webhook nhận thông báo Gmail từ Pub/Sub | OIDC Pub/Sub |

## Phụ lục C. Biến môi trường chính

[BANG: Bảng C.1: Các biến môi trường chính]
| Biến | Dùng bởi | Ý nghĩa | Mặc định |
|---|---|---|---|
| `DATABASE_URL` | Các dịch vụ có CSDL, worker | Chuỗi kết nối PostgreSQL; bỏ trống thì dùng bộ nhớ (trừ worker) | rỗng |
| `GRPC_ADDR / HTTP_ADDR` | Mọi dịch vụ | Cổng gRPC server và gateway HTTP | theo dịch vụ |
| `<DỊCH VỤ>_GRPC_DIAL_ADDR` | Dịch vụ gọi dịch vụ khác | Địa chỉ gRPC của dịch vụ đích | 127.0.0.1:<cổng> |
| `SESSION_SECRET` | identity, notification, social, message | Khoá ký session JWT (phải giống nhau) | bắt buộc |
| `SESSION_TTL` | identity | Thời hạn phiên đăng nhập | 720h |
| `GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET` | identity, mail-provider | OAuth client của Google | bắt buộc khi đăng nhập |
| `SOKETI_BASE_URL / SOKETI_APP_ID / SOKETI_APP_KEY / SOKETI_APP_SECRET` | identity, notification, message, worker | Kết nối và ký sự kiện Soketi | soketi:6001 |
| `KAFKA_BROKERS` | message, task, worker | Địa chỉ Kafka | localhost:29092 |
| `KAFKA_MESSAGE_TOPIC / KAFKA_TASK_EVENTS_TOPIC` | message, task, worker | Tên topic | chat.messages / task.events |
| `S3_ENDPOINT / S3_BUCKET / S3_ACCESS_KEY / S3_SECRET_KEY` | attachment, mail-provider | Kho đối tượng S3 (RustFS) | bucket task-manager |
| `ATTACHMENT_MAX_BYTES` | attachment | Kích thước tệp tối đa | 10 MiB |
| `MAIL_PUBSUB_TOPIC` | mail-provider | Topic Pub/Sub nhận thông báo Gmail | rỗng (tắt) |
| `MAIL_PULL_SUBSCRIPTION` | mail-provider | Bật chế độ pull thay cho push | rỗng |
| `DEEPSEEK_API_KEY / DEEPSEEK_MODEL` | mail-provider | Khoá và model phân loại email | rỗng / deepseek-chat |
| `PROJECT_MATCH_THRESHOLD` | mail-provider | Ngưỡng tin cậy khớp email vào dự án | 0,6 |
| `MAIL_RULE_FALLBACK_ENABLED` | mail-provider | Dùng luật từ khoá khi DeepSeek lỗi | true |
| `REMINDER_INTERVAL / REMINDER_TIMEZONE` | task | Chu kỳ quét nhắc việc, múi giờ | 30s / Asia/Ho_Chi_Minh |
| `WORKER_REPLICAS` | docker compose | Số bản chạy của worker | 2 |

## Phụ lục D. Bộ mã lỗi chuẩn

Bộ mã gồm 135 mã lỗi, định nghĩa trong `common/errorcode/error_codes.json`. Mã HTTP 0 là lỗi phía trình duyệt (không tới được máy chủ).

[BANG: Bảng D.1: Bộ mã lỗi chuẩn của hệ thống]
| Mã lỗi | HTTP | Thông điệp (tiếng Việt) |
|---|---|---|
| `CLIENT_NETWORK_ERROR` | 0 | Không kết nối được máy chủ. Vui lòng kiểm tra kết nối và thử lại. |
| `CLIENT_UNKNOWN_ERROR` | 0 | Đã xảy ra lỗi không xác định. Vui lòng thử lại. |
| `CLIENT_BAD_REQUEST` | 400 | Yêu cầu không hợp lệ. Vui lòng kiểm tra lại dữ liệu. |
| `CLIENT_UNAUTHORIZED` | 401 | Bạn cần đăng nhập để thực hiện thao tác này. |
| `CLIENT_FORBIDDEN` | 403 | Bạn không có quyền thực hiện thao tác này. |
| `CLIENT_NOT_FOUND` | 404 | Không tìm thấy dữ liệu yêu cầu. |
| `CLIENT_CONFLICT` | 409 | Dữ liệu đã tồn tại hoặc xung đột với trạng thái hiện tại. |
| `CLIENT_TOO_MANY_REQUESTS` | 429 | Bạn thao tác quá nhanh. Vui lòng thử lại sau ít giây. |
| `CLIENT_SERVER_ERROR` | 500 | Máy chủ gặp lỗi. Vui lòng thử lại sau. |
| `CLIENT_UNAVAILABLE` | 503 | Dịch vụ tạm thời không khả dụng. Vui lòng thử lại sau. |
| `COMMON_INVALID_ARGUMENT` | 400 | Dữ liệu gửi lên không hợp lệ. |
| `COMMON_NOT_FOUND` | 404 | Không tìm thấy dữ liệu yêu cầu. |
| `COMMON_ALREADY_EXISTS` | 409 | Dữ liệu đã tồn tại. |
| `COMMON_FAILED_PRECONDITION` | 400 | Thao tác không thực hiện được ở trạng thái hiện tại. |
| `COMMON_INTERNAL` | 500 | Máy chủ gặp lỗi. Vui lòng thử lại sau. |
| `COMMON_UNAVAILABLE` | 503 | Dịch vụ tạm thời không khả dụng. Vui lòng thử lại sau. |
| `IDENTITY_EMAIL_REQUIRED` | 400 | Email là bắt buộc. |
| `IDENTITY_DISPLAY_NAME_REQUIRED` | 400 | Tên hiển thị là bắt buộc. |
| `IDENTITY_PROFILE_ID_REQUIRED` | 400 | Thiếu mã hồ sơ. |
| `IDENTITY_PROFILE_NOT_FOUND` | 404 | Không tìm thấy hồ sơ. |
| `IDENTITY_EMAIL_ALREADY_EXISTS` | 409 | Email này đã được sử dụng. |
| `IDENTITY_AUTH_PROVIDER_REQUIRED` | 400 | Thiếu nhà cung cấp đăng nhập. |
| `IDENTITY_AUTH_PROVIDER_USER_ID_REQUIRED` | 400 | Thiếu mã người dùng từ nhà cung cấp. |
| `IDENTITY_AUTH_PROVIDER_UNSUPPORTED` | 400 | Nhà cung cấp đăng nhập không được hỗ trợ. |
| `IDENTITY_AUTH_PROFILE_INACTIVE` | 403 | Tài khoản đã bị khoá. |
| `IDENTITY_AUTH_STATE_INVALID` | 400 | Phiên đăng nhập không hợp lệ hoặc đã hết hạn. Vui lòng thử lại. |
| `IDENTITY_AUTH_CODE_REQUIRED` | 400 | Thiếu mã uỷ quyền từ nhà cung cấp. |
| `IDENTITY_AUTH_EXCHANGE_FAILED` | 401 | Không xác thực được với nhà cung cấp đăng nhập. |
| `IDENTITY_AUTH_TOKEN_MISSING` | 401 | Bạn cần đăng nhập để thực hiện thao tác này. |
| `IDENTITY_AUTH_TOKEN_INVALID` | 401 | Phiên đăng nhập đã hết hạn. Vui lòng đăng nhập lại. |
| `IDENTITY_AUTH_NOT_CONFIGURED` | 500 | Đăng nhập hiện không khả dụng. Vui lòng thử lại sau. |
| `IDENTITY_AUTH_SOKETI_NOT_CONFIGURED` | 500 | Kênh thông báo thời gian thực chưa được cấu hình. |
| `IDENTITY_AUTH_SOKETI_REQUEST_INVALID` | 400 | Yêu cầu uỷ quyền kênh thông báo không hợp lệ. |
| `IDENTITY_AUTH_SOKETI_CHANNEL_FORBIDDEN` | 403 | Bạn không có quyền nhận thông báo của người dùng khác. |
| `TASK_PROFILE_ID_REQUIRED` | 400 | Thiếu mã hồ sơ. |
| `TASK_TITLE_REQUIRED` | 400 | Tiêu đề công việc là bắt buộc. |
| `TASK_ID_REQUIRED` | 400 | Thiếu mã công việc. |
| `TASK_ID_AND_STATUS_ID_REQUIRED` | 400 | Thiếu mã công việc hoặc trạng thái. |
| `TASK_NOT_FOUND` | 404 | Không tìm thấy công việc. |
| `TASK_PARENT_NOT_IN_PROFILE` | 400 | Công việc cha không thuộc hồ sơ này. |
| `TASK_HAS_SUBTASKS` | 400 | Công việc vẫn còn công việc con. Hãy xoá hoặc chuyển chúng trước. |
| `TASK_STATUS_SAME` | 400 | Công việc đang ở trạng thái này rồi. |
| `TASK_STATUS_NOT_IN_PROFILE` | 400 | Trạng thái không thuộc hồ sơ này. |
| `TASK_STATUS_PROFILE_MISMATCH` | 400 | Trạng thái không thuộc hồ sơ của công việc. |
| `TASK_NO_STATUS_AVAILABLE` | 400 | Hồ sơ chưa có trạng thái nào. Hãy tạo trạng thái trước. |
| `TASK_TRANSITION_NOT_ALLOWED` | 400 | Không được phép chuyển công việc sang trạng thái này. |
| `TASK_CYCLE_DETECTED` | 400 | Không thể gán vì sẽ tạo vòng lặp công việc. |
| `TASK_DUE_RANGE_INVALID` | 400 | Khoảng hạn không hợp lệ: due_from phải trước due_to và không dùng kèm has_due_at=false. |
| `TASK_PAGE_SIZE_INVALID` | 400 | Kích thước trang không hợp lệ. |
| `TASK_PAGE_TOKEN_INVALID` | 400 | Mã trang không hợp lệ. |
| `TASK_SEARCH_QUERY_TOO_SHORT` | 400 | Từ khoá tìm kiếm cần ít nhất 2 ký tự. |
| `TASK_REMIND_AT_CONFLICT` | 400 | Không thể vừa đặt vừa bỏ nhắc việc trong cùng một yêu cầu. |
| `TASK_RECURRENCE_INVALID` | 400 | Quy tắc lặp không hợp lệ hoặc task chưa có hạn. |
| `TASK_STATUS_NAME_REQUIRED` | 400 | Tên trạng thái là bắt buộc. |
| `TASK_STATUS_SLUG_REQUIRED` | 400 | Slug trạng thái là bắt buộc. |
| `TASK_STATUS_SLUG_ALREADY_EXISTS` | 409 | Slug trạng thái đã tồn tại trong hồ sơ này. |
| `TASK_STATUS_ID_REQUIRED` | 400 | Thiếu mã trạng thái. |
| `TASK_STATUS_NOT_FOUND` | 404 | Không tìm thấy trạng thái. |
| `TASK_STATUS_IN_USE` | 400 | Trạng thái đang được công việc sử dụng. Hãy lưu trữ thay vì xoá. |
| `TASK_TRANSITION_FIELDS_REQUIRED` | 400 | Thiếu thông tin quy tắc chuyển trạng thái. |
| `TASK_TRANSITION_SAME_STATUS` | 400 | Trạng thái nguồn và trạng thái đích không được trùng nhau. |
| `TASK_TRANSITION_ID_REQUIRED` | 400 | Thiếu mã quy tắc chuyển trạng thái. |
| `TASK_TRANSITION_NOT_FOUND` | 404 | Không tìm thấy quy tắc chuyển trạng thái. |
| `TASK_TRANSITION_ALREADY_EXISTS` | 409 | Quy tắc chuyển trạng thái này đã tồn tại. |
| `MAIL_PROFILE_ID_REQUIRED` | 400 | Thiếu mã hồ sơ. |
| `MAIL_EMAIL_REQUIRED` | 400 | Thiếu địa chỉ email. |
| `MAIL_ACCESS_TOKEN_REQUIRED` | 400 | Thiếu access token của Google. |
| `MAIL_SUBSCRIPTION_NOT_FOUND` | 404 | Hộp thư này chưa đăng ký nhận thông báo. |
| `MAIL_NOT_CONFIGURED` | 500 | Dịch vụ mail chưa được cấu hình. Vui lòng thử lại sau. |
| `MAIL_GMAIL_FAILED` | 503 | Không gọi được Gmail. Vui lòng thử lại sau. |
| `MAIL_TOKEN_REFRESH_FAILED` | 401 | Quyền truy cập Gmail đã hết hiệu lực. Vui lòng đăng nhập lại. |
| `MAIL_NOTIFICATION_INVALID` | 400 | Thông báo từ Gmail không hợp lệ. |
| `MAIL_UNAUTHORIZED` | 401 | Không xác thực được thông báo từ Pub/Sub. |
| `MAIL_ANALYZE_FAILED` | 500 | Không phân tích được nội dung email. |
| `MAIL_TASK_CREATE_FAILED` | 500 | Không tạo được công việc từ email. |
| `MAIL_SCHEDULE_CREATE_FAILED` | 500 | Không tạo được lịch từ email. |
| `MAIL_EVENT_CREATE_FAILED` | 500 | Không tạo được sự kiện từ email. |
| `MAIL_BACKLOG_CREATE_FAILED` | 500 | Không lưu được email vào backlog. |
| `MAIL_STORAGE_FAILED` | 503 | Không lưu được tệp vào kho lưu trữ. |
| `MAIL_NOTICE_CREATE_FAILED` | 500 | Không đẩy được thông báo sang notification service. |
| `NOTIFICATION_PROFILE_ID_REQUIRED` | 400 | Thiếu mã hồ sơ. |
| `NOTIFICATION_TITLE_REQUIRED` | 400 | Tiêu đề thông báo là bắt buộc. |
| `NOTIFICATION_TARGET_TYPE_INVALID` | 400 | Loại đối tượng của thông báo không hợp lệ. |
| `NOTIFICATION_TARGET_ID_REQUIRED` | 400 | Thiếu mã đối tượng mà thông báo trỏ tới. |
| `NOTIFICATION_NOT_FOUND` | 404 | Không tìm thấy thông báo. |
| `CALENDAR_PROFILE_ID_REQUIRED` | 400 | Thiếu mã hồ sơ. |
| `CALENDAR_TITLE_REQUIRED` | 400 | Tiêu đề lịch là bắt buộc. |
| `CALENDAR_ID_REQUIRED` | 400 | Thiếu mã lịch. |
| `CALENDAR_START_AT_REQUIRED` | 400 | Thời gian bắt đầu là bắt buộc. |
| `CALENDAR_TIME_RANGE_INVALID` | 400 | Thời gian kết thúc phải sau thời gian bắt đầu. |
| `CALENDAR_NOT_FOUND` | 404 | Không tìm thấy lịch. |
| `PROJECT_PROFILE_ID_REQUIRED` | 400 | Thiếu mã hồ sơ. |
| `PROJECT_ID_REQUIRED` | 400 | Thiếu mã dự án. |
| `PROJECT_NAME_REQUIRED` | 400 | Tên dự án là bắt buộc. |
| `PROJECT_NAME_DUPLICATE` | 409 | Tên dự án đã tồn tại. |
| `PROJECT_NOT_FOUND` | 404 | Không tìm thấy dự án. |
| `PROJECT_ARCHIVED` | 400 | Dự án đã được lưu trữ. |
| `PROJECT_DEFAULT_NOT_ARCHIVABLE` | 400 | Không thể lưu trữ dự án mặc định. |
| `PROJECT_NAME_TOO_LONG` | 400 | Tên dự án tối đa 100 ký tự. |
| `PROJECT_COLOR_INVALID` | 400 | Màu dự án không hợp lệ (dạng #RRGGBB hoặc #RRGGBBAA). |
| `TASK_SUBTASK_PROJECT_LOCKED` | 400 | Công việc con luôn thuộc cùng dự án với công việc cha. |
| `EVENT_PROFILE_ID_REQUIRED` | 400 | Thiếu mã hồ sơ. |
| `EVENT_TITLE_REQUIRED` | 400 | Tiêu đề sự kiện là bắt buộc. |
| `EVENT_ID_REQUIRED` | 400 | Thiếu mã sự kiện. |
| `EVENT_START_AT_REQUIRED` | 400 | Thời gian bắt đầu là bắt buộc. |
| `EVENT_TIME_RANGE_INVALID` | 400 | Thời gian kết thúc phải sau thời gian bắt đầu. |
| `EVENT_NOT_FOUND` | 404 | Không tìm thấy sự kiện. |
| `BACKLOG_PROFILE_ID_REQUIRED` | 400 | Thiếu mã hồ sơ. |
| `BACKLOG_TITLE_REQUIRED` | 400 | Tiêu đề mục backlog là bắt buộc. |
| `BACKLOG_ID_REQUIRED` | 400 | Thiếu mã mục backlog. |
| `BACKLOG_NOT_FOUND` | 404 | Không tìm thấy mục backlog. |
| `REPORT_PROFILE_ID_REQUIRED` | 400 | Thiếu mã hồ sơ. |
| `REPORT_RANGE_INVALID` | 400 | Khoảng thời gian không hợp lệ: thời gian bắt đầu phải trước thời gian kết thúc. |
| `REPORT_INTERVAL_INVALID` | 400 | Độ phân giải thời gian không hợp lệ (chỉ hỗ trợ ngày/tuần/tháng). |
| `REPORT_TASK_SOURCE_FAILED` | 503 | Không lấy được dữ liệu task để tổng hợp báo cáo. |
| `ATTACHMENT_PROFILE_ID_REQUIRED` | 400 | Thiếu mã hồ sơ. |
| `ATTACHMENT_OWNER_TYPE_REQUIRED` | 400 | Thiếu loại đối tượng đính kèm. |
| `ATTACHMENT_OWNER_TYPE_INVALID` | 400 | Loại đối tượng đính kèm không hợp lệ (chỉ hỗ trợ task/lịch/sự kiện/backlog). |
| `ATTACHMENT_OWNER_ID_REQUIRED` | 400 | Thiếu mã đối tượng đính kèm. |
| `ATTACHMENT_FILE_REQUIRED` | 400 | Vui lòng chọn tệp để tải lên. |
| `ATTACHMENT_FILE_TOO_LARGE` | 413 | Tệp vượt quá giới hạn kích thước cho phép. |
| `ATTACHMENT_STORAGE_FAILED` | 503 | Không lưu được tệp lên kho lưu trữ. Vui lòng thử lại. |
| `ATTACHMENT_ID_REQUIRED` | 400 | Thiếu mã tệp đính kèm. |
| `ATTACHMENT_NOT_FOUND` | 404 | Không tìm thấy tệp đính kèm. |
| `SOCIAL_CANNOT_FRIEND_SELF` | 400 | Bạn không thể kết bạn với chính mình. |
| `SOCIAL_PROFILE_NOT_FOUND` | 404 | Không tìm thấy người dùng. |
| `SOCIAL_FRIEND_REQUEST_EXISTS` | 409 | Bạn đã gửi lời mời kết bạn cho người này. |
| `SOCIAL_ALREADY_FRIENDS` | 409 | Hai bạn đã là bạn bè. |
| `SOCIAL_FRIEND_REQUEST_NOT_FOUND` | 404 | Không tìm thấy lời mời kết bạn. |
| `SOCIAL_FRIEND_NOT_FOUND` | 404 | Người này không có trong danh sách bạn bè. |
| `SOCIAL_EMAIL_INVALID` | 400 | Email không hợp lệ. |
| `MESSAGE_BODY_REQUIRED` | 400 | Nội dung tin nhắn không được để trống. |
| `MESSAGE_BODY_TOO_LONG` | 400 | Tin nhắn quá dài. |
| `MESSAGE_CANNOT_MESSAGE_SELF` | 400 | Bạn không thể nhắn tin cho chính mình. |
| `MESSAGE_NOT_FRIENDS` | 403 | Chỉ có thể nhắn tin với bạn bè. |

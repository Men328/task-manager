<!-- Nguồn: docs/system/01,02,05,06; deployments/migrations/*.up.sql (000001–000018); common/proto; frontend/nginx.conf; ảnh chụp từ hệ thống chạy thật (docs/bao-cao/_tools/evidence/shots.cjs) -->
# CHƯƠNG 4: THIẾT KẾ HỆ THỐNG

## 4.1. Thiết kế kiến trúc tổng thể

### 4.1.1. Kiến trúc triển khai

Hệ thống được triển khai bằng Docker Compose trên một máy chủ (Hình 4.1). Chỉ container giao diện (nginx) được mở cổng ra ngoài; nginx vừa phục vụ ứng dụng React, vừa là **cổng vào duy nhất** chuyển các yêu cầu `/api/<dịch vụ>/…` tới gateway HTTP của dịch vụ tương ứng và chuyển kết nối WebSocket `/api/soketi/` tới Soketi. Khi cần truy cập từ Internet (ví dụ để Google Pub/Sub gửi thông báo đẩy), Cloudflare Tunnel cung cấp địa chỉ HTTPS công khai mà không phải mở cổng mạng.

Mỗi dịch vụ chạy thành **hai container**: gateway HTTP (grpc-gateway, cổng 8081–8092) nhận REST/JSON và gRPC server (cổng 9081–9092) chứa nghiệp vụ. Các gRPC server gọi nhau trực tiếp qua mạng nội bộ Docker. Dịch vụ worker không có API, chạy nhiều bản song song để tiêu thụ Kafka. Container `migrate` chạy một lần khi khởi động để cập nhật lược đồ cơ sở dữ liệu.

[HINH: h4-01-trien-khai.png | Hình 4.1: Sơ đồ kiến trúc triển khai hệ thống]

### 4.1.2. Kiến trúc phân lớp logic

Về mặt logic, hệ thống gồm sáu lớp (Hình 4.2): giao diện, cổng vào, gateway, nghiệp vụ, xử lý bất đồng bộ và dữ liệu. Lớp nghiệp vụ được chia thành bốn nhóm: dịch vụ lõi (identity, project, task, calendar, event, backlog), dịch vụ hỗ trợ (attachment, notification, social, message), dịch vụ tổng hợp (report) và dịch vụ tích hợp bên ngoài (mail-provider).

[HINH: h4-02-phan-lop.png | Hình 4.2: Sơ đồ kiến trúc phân lớp logic]

### 4.1.3. Danh sách dịch vụ và trách nhiệm

[BANG: Bảng 4.1: Danh sách các dịch vụ trong hệ thống]
| STT | Dịch vụ | Cổng gRPC / HTTP | Trách nhiệm chính | Lưu trữ |
|---|---|---|---|---|
| 1 | identity | 9081 / 8081 | Hồ sơ, đăng nhập Google, phát session JWT, ký uỷ quyền kênh realtime | PostgreSQL |
| 2 | task | 9082 / 8082 | Công việc, trạng thái, luật chuyển, nhắc việc, lặp lại, phát sự kiện task.events | PostgreSQL, Kafka |
| 3 | calendar | 9083 / 8083 | Lịch cá nhân | PostgreSQL |
| 4 | mail-provider | 9084 / 8084 | Theo dõi Gmail, phân loại email, điều phối | PostgreSQL, S3 |
| 5 | event | 9085 / 8085 | Sự kiện có trạng thái | PostgreSQL |
| 6 | backlog | 9086 / 8086 | Email chưa phân loại | PostgreSQL |
| 7 | report | 9087 / 8087 | Tổng hợp báo cáo (không có CSDL riêng) | Không (tổng hợp qua gRPC) |
| 8 | notification | 9088 / 8088 | Lưu thông báo, đẩy realtime | PostgreSQL, Soketi |
| 9 | attachment | 9089 / 8089 | Tệp đính kèm | PostgreSQL, S3 |
| 10 | project | 9090 / 8090 | Dự án | PostgreSQL |
| 11 | social | 9091 / 8091 | Quan hệ bạn bè | PostgreSQL |
| 12 | message | 9092 / 8092 | Nhắn tin, giao task qua chat | PostgreSQL (đọc), Kafka, Soketi |
| 13 | worker | — / 8094 (health) | Tiêu thụ chat.messages, task.events | PostgreSQL, Soketi |

### 4.1.4. Cấu trúc bên trong một dịch vụ

Mọi dịch vụ có cùng một cấu trúc thư mục đã trình bày ở mục 2.3.3 (Hình 2.4): `cmd/grpc` và `cmd/http` là hai chương trình khởi chạy (dùng Uber Fx), `internal` gồm các tầng `handler`, `service`, `model`, `repository`, `dependency` và `config`. Sự đồng nhất này giúp một lập trình viên đọc hiểu được mọi dịch vụ sau khi nắm một dịch vụ, và cho phép công cụ kiểm tra kiến trúc (`tools/quality`) áp dụng cùng một bộ quy tắc cho tất cả.

### 4.1.5. Các quyết định kiến trúc

[BANG: Bảng 4.2: Các quyết định kiến trúc chính]
| STT | Quyết định | Lý do / hệ quả |
|---|---|---|
| 1 | Microservices, mỗi dịch vụ một Go module | Phát triển, triển khai độc lập; đổi lại vận hành phức tạp hơn |
| 2 | gRPC nội bộ, grpc-gateway cho giao diện | Hợp đồng khai báo một lần trong proto; REST được sinh tự động |
| 3 | Một PostgreSQL, mỗi dịch vụ một schema | Ranh giới dữ liệu rõ ràng mà vận hành đơn giản |
| 4 | Kafka + worker cho tin nhắn và sự kiện task | Hiển thị tức thì, lưu bất đồng bộ; mở rộng bằng cách thêm worker |
| 5 | Soketi (giao thức Pusher) cho realtime | Đẩy thông báo và tin nhắn; kênh riêng tư được ký uỷ quyền |
| 6 | Kho đối tượng S3 (RustFS) cho tệp | Không làm phình cơ sở dữ liệu; dễ chuyển sang dịch vụ đám mây |
| 7 | Bộ mã lỗi chuẩn dùng chung | Giao diện dịch lỗi theo ngôn ngữ, không lộ thông điệp kỹ thuật |
| 8 | Tác vụ phụ là nỗ lực tối đa | Lỗi gửi thông báo, khởi tạo dữ liệu mặc định không làm hỏng thao tác chính |
| 9 | Repository in-memory khi thiếu DATABASE_URL | Phát triển và kiểm thử không cần cơ sở dữ liệu |

## 4.2. Thiết kế giao tiếp giữa các dịch vụ

### 4.2.1. Giao tiếp đồng bộ qua gRPC

Hình 4.3 là đồ thị phụ thuộc giữa các dịch vụ: mũi tên từ A tới B nghĩa là A gọi B. Các dịch vụ điều phối (message, mail-provider, report) nằm ở trên; hai dịch vụ lá (project, notification) không gọi dịch vụ nội bộ nào và nằm ở dưới cùng. Đồ thị không có chu trình, nên không xảy ra tình trạng hai dịch vụ chờ lẫn nhau. Bảng 4.3 liệt kê chi tiết từng lời gọi.

[HINH: h4-03-phu-thuoc-grpc.png | Hình 4.3: Sơ đồ phụ thuộc giữa các dịch vụ]

[BANG: Bảng 4.3: Ma trận gọi gRPC giữa các dịch vụ]
| Bên gọi | Bên được gọi | RPC | Thời điểm | Khi lỗi |
|---|---|---|---|---|
| identity | task | SeedDefaultStatuses | Tạo hồ sơ mới | Ghi log, vẫn tạo hồ sơ |
| identity | project | SeedDefaultProject | Tạo hồ sơ mới | Ghi log, vẫn tạo hồ sơ |
| identity | mail-provider | Subscribe | Đăng nhập có tích đọc Gmail | Ghi log, vẫn đăng nhập |
| task, calendar, event | project | GetProject | Tạo, sửa bản ghi | Trả lỗi nghiệp vụ |
| task | notification | CreateNotice | Đến giờ nhắc việc | Trả lại, nhắc ở chu kỳ sau |
| report | task | ListTasks, ListTaskStatuses | Báo cáo công việc | Trả lỗi |
| report | event, calendar, backlog | ListEvents, ListSchedules, ListBacklogs | Báo cáo hoạt động | Trả lỗi |
| mail-provider | project | ListProjects | Trước khi phân loại email | Đưa email vào backlog |
| mail-provider | task, calendar, event | CreateTask, CreateSchedule, CreateEvent | Sau khi phân loại | Giữ email trong backlog |
| mail-provider | backlog | CreateBacklog | Email nhóm other / thiếu dữ kiện | Ghi log |
| mail-provider | notification | CreateNotice | Sau khi tạo bản ghi | Ghi log |
| social | identity | FindProfileByEmail, BatchGetProfiles | Tìm bạn, danh sách bạn | Trả lỗi |
| social | notification | CreateNotice | Gửi, chấp nhận lời mời | Ghi log |
| message | social | AreFriends | Mỗi tin nhắn | Từ chối gửi |
| message | task | CreateTaskFromMessage | Lệnh /task | Trả lỗi |
| worker | notification | CreateNotice | Task giao qua chat đổi trạng thái | Thử lại |

### 4.2.2. Giao tiếp bất đồng bộ và thời gian thực

Hệ thống dùng hai topic Kafka:

- **chat.messages**: dịch vụ nhắn tin ghi mỗi tin nhắn với khoá là mã cuộc trò chuyện; consumer group `message-worker` lưu tin vào bảng `message.messages` và cập nhật `message.conversations`. Ghi luỹ đẳng theo mã tin nhắn (`ON CONFLICT DO NOTHING`), commit offset sau khi ghi thành công.
- **task.events**: dịch vụ công việc phát sự kiện khi một task được giao qua chat đổi trạng thái, khoá là mã task; consumer group `task-event-worker` cập nhật thẻ task trong tin nhắn, đẩy sự kiện `message.updated` và tạo thông báo cho người giao. Sự kiện cũ hơn trạng thái đang lưu sẽ bị bỏ qua để tránh ghi đè sai thứ tự.

Thời gian thực đi qua Soketi với ba loại sự kiện: `notice.created` trên kênh `private-noti-internal-<mã người dùng>`, `message.created` và `message.updated` trên kênh `private-chat-<mã người dùng>`. Trình duyệt chỉ nghe được kênh của chính mình nhờ chữ ký uỷ quyền do dịch vụ định danh cấp.

### 4.2.3. Tích hợp hệ thống bên ngoài

[BANG: Bảng 4.4: Danh sách tích hợp hệ thống bên ngoài]
| Bên gọi | Hệ thống | API | Mục đích |
|---|---|---|---|
| identity | Google OAuth | accounts.google.com/o/oauth2/v2/auth, oauth2.googleapis.com/token | Đăng nhập, cấp token (PKCE) |
| identity | Google OIDC | openidconnect.googleapis.com/v1/userinfo | Lấy email, tên, ảnh đại diện |
| mail-provider | Gmail API | users.watch, users.history.list, users.messages.get, attachments.get | Theo dõi và đọc thư mới |
| mail-provider | Google Pub/Sub | Push về /api/mail/v1/notifications hoặc subscriptions.pull | Nhận thông báo hộp thư thay đổi |
| mail-provider | DeepSeek | POST /chat/completions (model deepseek-chat) | Phân loại email |
| notification, message, worker | Soketi | POST /apps/{app_id}/events | Đẩy sự kiện realtime |

## 4.3. Thiết kế API

### 4.3.1. Quy ước thiết kế API

- Giao diện gọi API cùng nguồn (same-origin) qua đường dẫn `/api/<dịch vụ>/v1/<tài nguyên>`; nginx bỏ tiền tố `/api/<dịch vụ>` trước khi chuyển tiếp.
- Tài nguyên dùng danh từ số nhiều, có phiên bản: `/v1/tasks`, `/v1/tasks/{id}`, `/v1/tasks/{id}/status`.
- Phương thức: GET đọc, POST tạo hoặc hành động, PATCH cập nhật một phần, DELETE xoá.
- Dữ liệu JSON, tên trường dạng *lowerCamelCase* khi trả về; tham số truy vấn dùng tên trường proto (`profile_id`, `root_only`).
- Danh sách lớn hỗ trợ phân trang bằng `page_token`.

### 4.3.2. Danh sách API tiêu biểu

Bảng 4.5 liệt kê các API tiêu biểu của từng dịch vụ; danh sách đầy đủ ở Phụ lục B.

[BANG: Bảng 4.5: Các API tiêu biểu của hệ thống]
| Dịch vụ | Phương thức và đường dẫn | Chức năng |
|---|---|---|
| identity | GET /v1/auth/google/login, GET /v1/auth/me, POST /v1/auth/soketi | Đăng nhập, hồ sơ, uỷ quyền kênh |
| project | GET, POST /v1/projects; POST /v1/projects/{id}/archive | Danh sách, tạo, lưu trữ dự án |
| task | GET, POST /v1/tasks; PATCH, DELETE /v1/tasks/{id} | Danh sách (lọc, tìm kiếm), tạo, sửa, xoá task |
| task | POST /v1/tasks/{id}/status | Đổi trạng thái theo luật chuyển |
| task | GET, POST /v1/statuses; POST /v1/transitions; POST /v1/statuses/seed | Trạng thái, luật chuyển, bộ mặc định |
| calendar / event | GET, POST /v1/schedules; GET, POST, PATCH /v1/events | Lịch, sự kiện |
| backlog | GET /v1/backlogs; PATCH /v1/backlogs/{id} | Xem, xử lý backlog |
| attachment | POST /v1/attachments/upload; GET /v1/attachments/{id}/download | Tải lên, tải xuống tệp |
| report | GET /v1/reports/overview, status-breakdown, timeseries, activity | Báo cáo |
| notification | GET /v1/notices, /v1/notices/unread-count; POST /v1/notices/read | Thông báo |
| social | GET /v1/friends/search; POST /v1/friend-requests; POST /v1/friend-requests/{id}/accept | Kết bạn |
| message | GET, POST /v1/conversations/{peer_id}/messages; POST /v1/conversations/{peer_id}/tasks | Nhắn tin, giao task |
| mail-provider | POST /v1/notifications | Webhook nhận Pub/Sub |

### 4.3.3. Bộ mã lỗi chuẩn

Mọi lỗi nghiệp vụ được định nghĩa tập trung trong tệp `common/errorcode/error_codes.json`, gồm mã lỗi, mã gRPC, mã HTTP và thông điệp tiếng Việt, tiếng Anh. Dịch vụ trả lỗi theo chuẩn `google.rpc.ErrorInfo`; grpc-gateway chuyển thành JSON có trường `details[].reason`. Giao diện tra mã lỗi để hiển thị thông điệp thân thiện theo ngôn ngữ người dùng, không hiển thị mã HTTP hay thông điệp kỹ thuật. Bảng 4.6 là một số mã lỗi thu được khi kiểm thử hệ thống thật (Chương 5).

[BANG: Bảng 4.6: Một số mã lỗi chuẩn]
| Mã lỗi | HTTP | Ý nghĩa |
|---|---|---|
| IDENTITY_AUTH_TOKEN_MISSING | 401 | Chưa đăng nhập |
| IDENTITY_AUTH_SOKETI_CHANNEL_FORBIDDEN | 403 | Xin nghe kênh realtime của người khác |
| TASK_TITLE_REQUIRED | 400 | Thiếu tiêu đề công việc |
| TASK_TRANSITION_NOT_ALLOWED | 400 | Không có luật chuyển trạng thái |
| PROJECT_NAME_DUPLICATE | 409 | Trùng tên dự án |
| PROJECT_ARCHIVED | 400 | Dự án đã lưu trữ |
| MESSAGE_NOT_FRIENDS | 403 | Nhắn tin cho người chưa kết bạn |
| ATTACHMENT_FILE_TOO_LARGE | 429 | Tệp vượt giới hạn kích thước |

### 4.3.4. Thiết kế xác thực và phân quyền

Sau khi đăng nhập Google, dịch vụ định danh phát **session JWT** (HS256, hạn 30 ngày, claim `sub` là mã người dùng, `iss` là `taskmanager-identity`). Giao diện lưu token và gửi kèm mỗi yêu cầu trong tiêu đề `Authorization: Bearer`. Các gateway thông báo, bạn bè, nhắn tin và định danh tự kiểm tra chữ ký token và lấy mã người dùng từ `sub`, không nhận mã người dùng do trình duyệt gửi. Kênh realtime riêng tư chỉ được ký uỷ quyền cho đúng chủ kênh. Các gateway còn lại (công việc, lịch, sự kiện, backlog, tệp, báo cáo) hiện nhận `profile_id` từ yêu cầu; việc chuyển sang kiểm tra token tập trung ở cổng vào là một hướng phát triển (xem phần Kết luận).

## 4.4. Thiết kế cơ sở dữ liệu

### 4.4.1. Sơ đồ thực thể liên kết

Cơ sở dữ liệu `task_manager` gồm 11 schema, mỗi schema thuộc một dịch vụ. Do các schema nằm chung một cơ sở dữ liệu, hệ thống khai báo khoá ngoại chéo schema: hầu hết bảng tham chiếu `identity.profiles` với `ON DELETE CASCADE` (xoá người dùng thì xoá toàn bộ dữ liệu liên quan); công việc, lịch và sự kiện tham chiếu `project.projects` với `ON DELETE RESTRICT`. Riêng tệp đính kèm dùng liên kết **đa hình** `(owner_type, owner_id)` nên không có khoá ngoại tới đối tượng sở hữu. Hình 4.4 và Hình 4.5 là sơ đồ thực thể liên kết của hai nhóm dữ liệu.

[HINH: h4-04-erd.png | Hình 4.4: Sơ đồ thực thể liên kết nhóm công việc và kế hoạch]

[HINH: h4-05-erd-giao-tiep.png | Hình 4.5: Sơ đồ thực thể liên kết nhóm tài khoản, email và giao tiếp]

### 4.4.2. Danh sách các bảng

[BANG: Bảng 4.7: Danh sách các bảng trong hệ thống]
| STT | Schema.Bảng | Dịch vụ | Mô tả |
|---|---|---|---|
| 1 | identity.profiles | identity | Hồ sơ người dùng |
| 2 | identity.auth_providers | identity | Liên kết tài khoản Google |
| 3 | project.projects | project | Dự án |
| 4 | task.task_statuses | task | Trạng thái công việc theo người dùng |
| 5 | task.status_transitions | task | Luật chuyển trạng thái |
| 6 | task.tasks | task | Công việc |
| 7 | task.task_status_logs | task | Lịch sử chuyển trạng thái |
| 8 | calendar.schedules | calendar | Lịch |
| 9 | event.events | event | Sự kiện |
| 10 | backlog.backlogs | backlog | Email chưa phân loại |
| 11 | attachment.attachments | attachment | Thông tin tệp đính kèm |
| 12 | notification.notices | notification | Thông báo |
| 13 | mail_provider.sessions | mail-provider | Token Gmail của người dùng |
| 14 | mail_provider.noti_indexes | mail-provider | Mốc historyId, hạn theo dõi Gmail |
| 15 | social.friendships | social | Quan hệ bạn bè |
| 16 | message.conversations | worker (ghi), message (đọc) | Cuộc trò chuyện |
| 17 | message.messages | worker (ghi), message (đọc) | Tin nhắn, thẻ task giao qua chat |

### 4.4.3. Bảng identity.profiles

Lưu hồ sơ người dùng; email là duy nhất. Hồ sơ được tạo khi đăng nhập Google lần đầu.

[BANG: Bảng 4.8: Cấu trúc bảng identity.profiles]
| STT | Tên cột | Kiểu dữ liệu | Ràng buộc | Mô tả |
|---|---|---|---|---|
| 1 | id | uuid | Khoá chính | Mã người dùng |
| 2 | email | varchar(255) | Not null, duy nhất | Email Google |
| 3 | display_name | varchar(100) | Not null | Tên hiển thị |
| 4 | avatar_url | text | Cho phép null | Ảnh đại diện |
| 5 | timezone | varchar(64) | Mặc định Asia/Ho_Chi_Minh | Múi giờ |
| 6 | locale | varchar(10) | Mặc định vi | Ngôn ngữ |
| 7 | is_active | boolean | Mặc định true | Còn hoạt động |
| 8 | last_login_at | timestamptz | Cho phép null | Lần đăng nhập gần nhất |
| 9 | created_at, updated_at, deleted_at | timestamptz | Not null, mặc định now(); deleted_at cho phép null | Thời điểm tạo, cập nhật, xoá mềm |

### 4.4.4. Bảng project.projects

[BANG: Bảng 4.9: Cấu trúc bảng project.projects]
| STT | Tên cột | Kiểu dữ liệu | Ràng buộc | Mô tả |
|---|---|---|---|---|
| 1 | id | uuid | Khoá chính | Mã dự án |
| 2 | profile_id | uuid | Khoá ngoại profiles, cascade | Chủ sở hữu |
| 3 | name | varchar(100) | Not null; duy nhất theo người dùng (không phân biệt hoa thường) | Tên dự án |
| 4 | description | text | Mặc định rỗng | Mô tả (dùng để khớp email vào dự án) |
| 5 | color | varchar(9) | Cho phép null | Màu hiển thị |
| 6 | is_default | boolean | Mỗi người dùng đúng một dự án mặc định | Dự án "Chung" |
| 7 | is_archived | boolean | Mặc định false | Đã lưu trữ |
| 8 | created_at, updated_at | timestamptz | Not null, mặc định now() | Thời điểm tạo, cập nhật |

### 4.4.5. Bảng task.task_statuses

[BANG: Bảng 4.10: Cấu trúc bảng task.task_statuses]
| STT | Tên cột | Kiểu dữ liệu | Ràng buộc | Mô tả |
|---|---|---|---|---|
| 1 | id | uuid | Khoá chính | Mã trạng thái |
| 2 | profile_id | uuid | Khoá ngoại profiles, cascade | Chủ sở hữu |
| 3 | name, slug | varchar(50) | Not null | Tên, định danh ngắn |
| 4 | color | varchar(9) | Cho phép null | Màu cột Kanban |
| 5 | category | enum (todo, in_progress, done) | Mặc định todo | Nhóm trạng thái |
| 6 | is_default | boolean | Mỗi người dùng tối đa một | Trạng thái của task mới |
| 7 | is_terminal | boolean | Not null, mặc định false | Trạng thái kết thúc |
| 8 | position | integer | Not null, mặc định 0 | Thứ tự cột |
| 9 | is_archived | boolean | Not null, mặc định false | Đã lưu trữ |

### 4.4.6. Bảng task.status_transitions

Mỗi dòng là một luật cho phép chuyển từ `from_status_id` sang `to_status_id`; không có luật thì bước chuyển bị cấm.

[BANG: Bảng 4.11: Cấu trúc bảng task.status_transitions]
| STT | Tên cột | Kiểu dữ liệu | Ràng buộc | Mô tả |
|---|---|---|---|---|
| 1 | id | uuid | Khoá chính | Mã luật |
| 2 | profile_id | uuid | Khoá ngoại profiles | Chủ sở hữu |
| 3 | from_status_id | uuid | Khoá ngoại (task_statuses.id, profile_id) | Trạng thái nguồn |
| 4 | to_status_id | uuid | Khoá ngoại (task_statuses.id, profile_id) | Trạng thái đích |
| 5 | is_active | boolean | Mặc định true | Luật đang áp dụng |
| 6 | requires_note | boolean | Mặc định false | Bắt buộc ghi chú khi chuyển |
| 7 | guard | jsonb | Cho phép null | Điều kiện bổ sung (dự phòng mở rộng) |

### 4.4.7. Bảng task.tasks

Bảng trung tâm của hệ thống, được mở rộng dần qua các migration (dự án, nhắc việc, lặp lại, giao qua chat).

[BANG: Bảng 4.12: Cấu trúc bảng task.tasks]
| STT | Tên cột | Kiểu dữ liệu | Ràng buộc | Mô tả |
|---|---|---|---|---|
| 1 | id | uuid | Khoá chính | Mã công việc |
| 2 | profile_id | uuid | Khoá ngoại profiles, cascade | Chủ sở hữu |
| 3 | project_id | uuid | Not null; khoá ngoại projects, restrict | Dự án |
| 4 | parent_task_id | uuid | Khoá ngoại tasks (cùng người dùng) | Task cha |
| 5 | status_id | uuid | Not null; khoá ngoại task_statuses, restrict | Trạng thái hiện tại |
| 6 | title | varchar(500) | Not null | Tiêu đề |
| 7 | description | text | Cho phép null | Mô tả |
| 8 | priority | enum (low, medium, high, urgent) | Mặc định medium | Độ ưu tiên |
| 9 | start_at, due_at | timestamptz | Cho phép null; chỉ mục (profile_id, due_at) | Ngày bắt đầu, hạn |
| 10 | completed_at | timestamptz | Cho phép null; tự ghi khi vào nhóm hoàn thành | Thời điểm hoàn thành |
| 11 | remind_at, reminded_at | timestamptz | Cho phép null; chỉ mục nhắc việc chưa gửi | Giờ nhắc, thời điểm đã nhắc |
| 12 | recurrence_rule, recurrence_tz | text, varchar(64) | Cho phép null | Quy tắc lặp (RRULE), múi giờ |
| 13 | recurrence_series_id, recurrence_next_id | uuid | Cho phép null; chỉ mục theo chuỗi lặp | Chuỗi lặp, lần lặp kế tiếp (chống sinh trùng) |
| 14 | requested_by | uuid | Khoá ngoại profiles, set null | Người giao qua chat |
| 15 | source_message_id | uuid | Cho phép null | Tin nhắn nguồn |
| 16 | is_archived, deleted_at | boolean, timestamptz | is_archived not null, mặc định false; deleted_at cho phép null | Lưu trữ, xoá mềm |

Các chỉ mục quan trọng: theo (người dùng, trạng thái), (người dùng, hạn), (người dùng, dự án), theo giờ nhắc chưa gửi, theo chuỗi lặp và chỉ mục trigram trên tiêu đề phục vụ tìm kiếm.

### 4.4.8. Bảng calendar.schedules và event.events

Hai bảng có cấu trúc gần giống nhau; `event.events` có thêm `status` (PLANNED, CONFIRMED, CANCELLED) và `source` (mã email nguồn).

[BANG: Bảng 4.13: Cấu trúc bảng calendar.schedules]
| STT | Tên cột | Kiểu dữ liệu | Ràng buộc | Mô tả |
|---|---|---|---|---|
| 1 | id | uuid | Khoá chính | Mã lịch |
| 2 | profile_id, project_id | uuid | Khoá ngoại profiles, projects | Chủ sở hữu, dự án |
| 3 | title | varchar(500) | Not null | Tiêu đề |
| 4 | description | text | Cho phép null | Mô tả |
| 5 | location | varchar(255) | Cho phép null | Địa điểm |
| 6 | start_at | timestamptz | Not null | Bắt đầu |
| 7 | end_at | timestamptz | CHECK end_at ≥ start_at | Kết thúc |
| 8 | all_day | boolean | Mặc định false | Cả ngày |
| 9 | color | varchar(9) | Cho phép null | Màu hiển thị |

### 4.4.9. Bảng notification.notices

[BANG: Bảng 4.14: Cấu trúc bảng notification.notices]
| STT | Tên cột | Kiểu dữ liệu | Ràng buộc | Mô tả |
|---|---|---|---|---|
| 1 | id | uuid | Khoá chính | Mã thông báo |
| 2 | profile_id | uuid | Khoá ngoại profiles, cascade | Người nhận |
| 3 | type | varchar(64) | Not null | Loại: TASK_CREATED, TASK_REMINDER, FRIEND_REQUEST, TASK_STATUS_CHANGED… |
| 4 | title, body | varchar(500), text | title not null; body cho phép null | Nội dung |
| 5 | target_type | varchar(32) | CHECK task, schedule, event, backlog, friend, message | Loại đối tượng đích |
| 6 | target_id | uuid | Not null | Mã đối tượng đích (mở đúng đối tượng khi bấm) |
| 7 | is_read, read_at | boolean, timestamptz | is_read not null, mặc định false; read_at cho phép null | Trạng thái đã đọc |

### 4.4.10. Bảng message.messages

[BANG: Bảng 4.15: Cấu trúc bảng message.messages]
| STT | Tên cột | Kiểu dữ liệu | Ràng buộc | Mô tả |
|---|---|---|---|---|
| 1 | id | uuid (UUIDv7) | Khoá chính | Mã tin nhắn, sinh trước khi gửi Kafka |
| 2 | conversation_id | uuid (UUIDv5 của cặp người dùng) | Khoá ngoại conversations, cascade | Cuộc trò chuyện |
| 3 | sender_id, recipient_id | uuid | Not null | Người gửi, người nhận |
| 4 | body | text | Not null | Nội dung (1–2000 ký tự) |
| 5 | client_id | varchar(64) | Not null, mặc định chuỗi rỗng | Mã phía trình duyệt để khớp tin gửi lạc quan |
| 6 | created_at, persisted_at | timestamptz | Not null; persisted_at mặc định now() | Thời điểm gửi, thời điểm worker lưu |
| 7 | task_id, task_title | uuid, text | Cho phép null | Thẻ task giao qua chat |
| 8 | task_status_name, task_status_category, task_status_color, task_status_changed_at | text, varchar, timestamptz | Cho phép null | Ảnh chụp trạng thái task trên thẻ |

### 4.4.11. Lưu trữ đối tượng và hàng đợi

Nội dung tệp không lưu trong PostgreSQL mà ở bucket `task-manager` của RustFS. Tệp đính kèm có khoá `attachments/{người dùng}/{loại đối tượng}/{mã đối tượng}/{uuid}{phần mở rộng}`; email gốc có khoá `mail/{người dùng}/{yyyymmdd}/{mã thư}.json` và tệp đính kèm của thư nằm trong thư mục con `attachments/`. Thông điệp Kafka được mã hoá dạng protojson theo message `ChatMessage` (topic chat.messages) và sự kiện task (topic task.events) khai báo trong proto.

## 4.5. Thiết kế xử lý nghiệp vụ trọng tâm

### 4.5.1. Vòng đời công việc theo luật chuyển trạng thái

Mỗi người dùng có bộ trạng thái riêng, mỗi trạng thái thuộc một trong ba nhóm *cần làm*, *đang làm*, *hoàn thành*. Khi đổi trạng thái, dịch vụ công việc kiểm tra trong bảng `status_transitions` có luật đang hoạt động từ trạng thái hiện tại tới trạng thái đích hay không (cơ chế *allowlist*); nếu không có, yêu cầu bị từ chối với mã `TASK_TRANSITION_NOT_ALLOWED`. Khi chuyển vào nhóm hoàn thành, hệ thống ghi `completed_at`; khi rời nhóm này, `completed_at` được xoá. Mọi lần chuyển đều được ghi vào `task_status_logs`. Thiết kế này cho phép người dùng tự định nghĩa quy trình (ví dụ bắt buộc qua bước *In Review*) mà không phải sửa mã nguồn.

### 4.5.2. Phân loại email và điều phối

Sau khi DeepSeek (hoặc luật từ khoá) phân loại, email còn phải **khớp được một dự án** của người dùng thì mới tạo bản ghi tự động. Dịch vụ xử lý thư lấy danh sách dự án (trừ dự án mặc định và dự án đã lưu trữ) và chấm điểm theo số từ khoá trùng giữa tên, mô tả dự án và nội dung email (tối thiểu 2 từ trùng, bỏ qua hư từ); nếu độ tin cậy dưới ngưỡng `PROJECT_MATCH_THRESHOLD` (mặc định 0,6), email được đưa vào backlog với lý do `no_project_matched`. Nhờ đó công việc tự động luôn nằm đúng dự án, còn trường hợp không chắc chắn để người dùng quyết định.

[BANG: Bảng 4.16: Bảng quyết định điều phối email sau phân loại]
| Nhóm | Điều kiện tạo bản ghi | Đích | Lý do khi vào backlog |
|---|---|---|---|
| task | Cần hành động, có tiêu đề, khớp dự án | Tạo công việc | not_actionable, no_project_matched |
| schedule | Có thời gian bắt đầu, khớp dự án | Tạo lịch | missing_schedule_time, no_project_matched |
| event | Có thời gian bắt đầu, khớp dự án | Tạo sự kiện | missing_event_time, no_project_matched |
| other | Không tạo bản ghi tự động | Backlog | Lý do do AI trả về hoặc unclassified |
| (bất kỳ) | Tạo bản ghi thất bại | Backlog | create_failed |

### 4.5.3. Nhắc việc và sinh công việc lặp lại

Bộ nhắc việc chạy trong dịch vụ công việc với chu kỳ mặc định 30 giây. Mỗi chu kỳ, nó *nhận* các task có `remind_at` đã tới mà chưa nhắc bằng cách ghi `reminded_at` trong cùng câu lệnh cập nhật, nên nhiều bản chạy song song không gửi trùng; nếu gửi thông báo lỗi, `reminded_at` được xoá để nhắc lại ở chu kỳ sau.

Công việc lặp lại lưu quy tắc theo chuẩn RRULE (ví dụ `FREQ=DAILY`) và múi giờ. Khi một lần lặp hoàn thành, hệ thống tính hạn kế tiếp bằng thư viện rrule-go và tạo task mới cùng `recurrence_series_id`; trường `recurrence_next_id` của task cũ ghi lại task vừa sinh, để việc mở lại rồi hoàn thành lại không sinh trùng.

### 4.5.4. Nhắn tin và giao task qua chat

Tin nhắn được thiết kế theo nguyên tắc *hiển thị trước, lưu sau*: dịch vụ nhắn tin sinh sẵn mã tin nhắn (UUIDv7, có thứ tự theo thời gian) và mã cuộc trò chuyện (UUIDv5 của cặp người dùng, luôn cố định cho một cặp), đẩy tin qua Soketi rồi ghi bất đồng bộ vào Kafka. Worker lưu tin luỹ đẳng theo mã nên xử lý lại không tạo bản trùng; khoá Kafka là mã cuộc trò chuyện nên tin của một cuộc trò chuyện luôn được xử lý đúng thứ tự.

Với lệnh `/task`, dịch vụ nhắn tin gọi `CreateTaskFromMessage` để tạo task trong dự án mặc định của người nhận, ghi `requested_by` và `source_message_id`, rồi gửi tin nhắn kèm thẻ task. Khi người nhận đổi trạng thái, sự kiện trên topic `task.events` giúp thẻ trong tin nhắn và thông báo cho người giao được cập nhật mà dịch vụ công việc không cần biết tới dịch vụ nhắn tin.

## 4.6. Thiết kế giao diện ứng dụng

Giao diện được xây dựng với bố cục thống nhất: thanh bên trái chia nhóm *Tổng quan*, *Kế hoạch*, *Kết nối*, *Cấu hình*; thanh trên cùng có bộ chọn dự án, ô tìm kiếm toàn cục, chọn ngôn ngữ, chuông thông báo và thông tin tài khoản. Các ảnh dưới đây được chụp từ hệ thống đang chạy với tài khoản menc1lhp@gmail.com.

### 4.6.1. Trang đăng nhập

Trang đăng nhập (Hình 4.6) có một nút *Đăng nhập với Google* và ô tích *Cho phép đọc Gmail* để bật chức năng tự động phân loại email.

[HINH: h4-ui-01-dang-nhap.png | Hình 4.6: Trang đăng nhập]

### 4.6.2. Trang bảng công việc (Kanban)

Trang chính (Hình 4.7) hiển thị bảng Kanban của dự án đang chọn; mỗi cột là một trạng thái với màu và số lượng task. Thẻ task hiển thị độ ưu tiên, hạn, tiến độ task con và biểu tượng lặp lại. Phía trên có ô *Thêm nhanh* nhận một dòng văn bản, các chế độ xem Kanban, Bảng, Danh sách, Dòng thời gian, ô tìm kiếm và bộ lọc.

[HINH: h4-ui-02-kanban.png | Hình 4.7: Trang bảng công việc dạng Kanban]

### 4.6.3. Cửa sổ chi tiết công việc

Bấm vào thẻ mở cửa sổ chi tiết (Hình 4.8): dự án, mô tả, trạng thái, các mốc thời gian, thiết lập lặp lại và nhắc việc, tiến độ, danh sách task con và khối tệp đính kèm có nút tải lên, tải xuống, xoá.

[HINH: h4-ui-03-chi-tiet-task.png | Hình 4.8: Cửa sổ chi tiết công việc]

### 4.6.4. Trang việc hôm nay

Trang *Hôm nay* (Hình 4.9) gom các task quá hạn, đến hạn hôm nay và sắp tới để người dùng tập trung xử lý.

[HINH: h4-ui-04-hom-nay.png | Hình 4.9: Trang việc hôm nay]

### 4.6.5. Trang quản lý dự án

[HINH: h4-ui-05-du-an.png | Hình 4.10: Trang quản lý dự án]

Trang dự án (Hình 4.10) liệt kê dự án với màu, mô tả, nhãn dự án mặc định; hỗ trợ lọc đang hoạt động hoặc đã lưu trữ, tạo, sửa và lưu trữ dự án.

### 4.6.6. Trang lịch

[HINH: h4-ui-06-lich.png | Hình 4.11: Trang lịch]

Trang lịch (Hình 4.11) dùng FullCalendar với các chế độ tháng, tuần, ngày; người dùng bấm vào ô ngày để tạo lịch hoặc bấm vào lịch để sửa.

### 4.6.7. Trang sự kiện

[HINH: h4-ui-07-su-kien.png | Hình 4.12: Trang sự kiện]

Trang sự kiện (Hình 4.12) là bảng các sự kiện với thời gian, địa điểm và nhãn trạng thái (dự kiến, đã xác nhận, đã huỷ), có bộ lọc theo trạng thái.

### 4.6.8. Trang backlog

[HINH: h4-ui-08-backlog.png | Hình 4.13: Trang backlog]

Trang backlog (Hình 4.13) hiển thị các email chưa phân loại được, kèm người gửi, nhóm phân loại, lý do và trạng thái xử lý. Các mục trong hình là email thật gửi tới hộp thư của tài khoản thử nghiệm.

### 4.6.9. Trang báo cáo

[HINH: h4-ui-09-bao-cao.png | Hình 4.14: Trang báo cáo]

Trang báo cáo (Hình 4.14) gồm bộ chọn khoảng thời gian và mức gộp (ngày, tuần, tháng), các thẻ số liệu tổng quan, biểu đồ tròn tỉ lệ theo trạng thái và biểu đồ đường số task tạo mới, hoàn thành theo thời gian.

### 4.6.10. Thông báo

[HINH: h4-ui-10-thong-bao.png | Hình 4.15: Danh sách thông báo]

Chuông thông báo (Hình 4.15) hiển thị số thông báo chưa đọc và cập nhật tức thời qua WebSocket. Danh sách gồm thông báo nhắc việc, task tạo từ email, lời mời kết bạn, trạng thái task được giao; bấm vào một thông báo mở đúng đối tượng.

### 4.6.11. Trang bạn bè

[HINH: h4-ui-11-ban-be.png | Hình 4.16: Trang bạn bè]

Trang bạn bè (Hình 4.16) cho phép tìm người dùng theo email, xem danh sách bạn bè, lời mời đến và lời mời đã gửi.

### 4.6.12. Trang tin nhắn

[HINH: h4-ui-12-tin-nhan.png | Hình 4.17: Trang tin nhắn và thẻ task giao qua chat]

Trang tin nhắn (Hình 4.17) gồm danh sách cuộc trò chuyện bên trái và khung chat bên phải. Ngoài tin nhắn văn bản, khung chat hiển thị **thẻ task** được giao bằng lệnh `/task`, với tiêu đề và nhãn trạng thái được cập nhật tự động khi người nhận đổi trạng thái.

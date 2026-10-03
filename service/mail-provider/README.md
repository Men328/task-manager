# service/mail-provider

Service nhận **thông báo Gmail (Pub/Sub push)** → gọi **Gmail API** lấy email gốc →
gọi **DeepSeek** (có rule keyword dự phòng) để **phân loại email** thành
`task | schedule | event | other` → gọi service tương ứng (`task`, `calendar`, `event`) hoặc
**đẩy vào `backlog`** nếu là `other`. Email gốc + attachment được lưu trên **object storage
S3-compatible (RustFS)**.

- gRPC: `:9084` — HTTP gateway + webhook Pub/Sub: `:8084`
- Proto: `common/proto/mail/v1/mail.proto`
- Lưu trữ quan hệ: **PostgreSQL** — `mail_provider.sessions` (refresh/access token) +
  `mail_provider.noti_indexes` (checkpoint `historyId`, hạn watch), migration `000005`.
  Set `DATABASE_URL` để dùng; không set thì quay về **in-memory** và restart là mất hết watch
  (user phải đăng nhập lại).
- Lưu trữ đối tượng: **S3-compatible** (`S3_ENDPOINT`, bucket `S3_BUCKET`) — email gốc dạng JSON +
  attachment. Stack Docker dùng **RustFS** vì MinIO OSS đã bị archive (dl.min.io ngừng phát hành
  binary và repo Docker `minio/minio`/`minio/mc` bị xoá); client vẫn là SDK `minio-go` (S3 API).
  Rỗng ⇒ tắt archive, mọi thứ vẫn chạy (chỉ mất bản lưu email gốc). Vì tương thích ngược, các biến
  `MINIO_*` cũ vẫn được đọc nếu `S3_*` chưa đặt.

## Chức năng

| Việc | Ai gọi | RPC |
|---|---|---|
| Đăng ký watch Gmail sau khi user đăng nhập | gateway identity | `Subscribe` |
| Dừng watch khi user ngắt kết nối | nội bộ | `Unsubscribe` |
| Xem trạng thái watch | nội bộ | `GetSubscription` |
| Nhận notice từ Pub/Sub push | webhook `POST /v1/notifications` | `HandleNotification` |

`Subscribe` gọi `users.watch` với `MAIL_PUBSUB_TOPIC`, lưu `historyId` làm checkpoint và
`expiration` (tối đa 7 ngày). Một worker nền định kỳ gia hạn watch trước khi hết hạn.

Khi có notice, service **không tin** `historyId` trong notice: nó đọc checkpoint đã lưu rồi gọi
`users.history.list(startHistoryId=<checkpoint>)` để lấy đủ các message mới, xử lý xong mới
tiến checkpoint. Nhờ vậy notice trùng hoặc đến lệch thứ tự không làm mất email.
Nếu checkpoint quá cũ (Gmail trả 404), service đặt lại checkpoint theo `users.getProfile` và
**bỏ qua** các email trong khoảng đã mất (có log cảnh báo) — giới hạn đã biết của Gmail, không
phụ thuộc việc checkpoint lưu ở đâu.

## Nhận notice: push hay pull

Có 2 cách, **chọn một** (hoặc chạy cả hai, xem ghi chú cuối):

| | Push (mặc định) | Pull |
|---|---|---|
| Bật bằng | push subscription trên topic | set `MAIL_PULL_SUBSCRIPTION` |
| Endpoint công khai | **bắt buộc** HTTPS (tunnel/Cloudflare) | không cần |
| Xác thực | OIDC JWT (`MAIL_PUBSUB_AUDIENCE`) | ADC service account |
| Chạy local thuần | không được | **được** |
| Chiều gọi | Pub/Sub gọi vào webhook | mail-provider gọi ra Pub/Sub |

Pull worker dùng REST `subscriptions.pull` (long-poll) rồi `acknowledge`. Nó **chỉ chạy khi**
`MAIL_PULL_SUBSCRIPTION` được set, nên không ảnh hưởng luồng push.

Ack policy: notice xử lý xong, notice không hợp lệ, và hộp thư chưa đăng ký đều được ack;
riêng lỗi tạm thời (queue đầy) thì **không** ack để Pub/Sub gửi lại sau.

Chạy đồng thời push subscription và pull subscription trên cùng topic vẫn an toàn: mỗi bên nhận
một bản copy, nhưng checkpoint `historyId` khiến lần xử lý thứ hai không tạo task trùng.

## Luồng xử lý

```
user tích "đọc Gmail" ở màn login
  → identity OAuth (scope gmail.readonly + access_type=offline)
  → identity gọi mail-provider.Subscribe(profile_id, email, access_token, refresh_token)
      → users.watch(topic)  → lưu checkpoint historyId

Gmail có mail mới
  → Pub/Sub push  →  POST /api/mail/v1/notifications   (nginx → mail-provider:8084)
      → xác thực OIDC JWT (MAIL_PUBSUB_AUDIENCE)
      → giải mã envelope, gọi HandleNotification (gRPC, nội bộ)
      → enqueue (trả 200 ngay)
  → worker:
      access token (refresh nếu hết hạn) → users.history.list → users.messages.get
      → object storage S3: lưu email gốc (JSON) + attachment (nếu có)
      → phân loại: DeepSeek /chat/completions, lỗi thì rule keyword dự phòng
      → dispatch theo category (xem bảng bên dưới)
      → tiến checkpoint
```

## Phân loại email: rule & dispatch

Mỗi email được gán **đúng một** category. Thứ tự ưu tiên khi điểm bằng nhau:
`event` > `schedule` > `task` > `other`.

### 1. Rule chính — DeepSeek (LLM)

`internal/repository/deepseek_client.go` yêu cầu model trả về duy nhất một JSON object:

```json
{
  "category": "task | schedule | event | other",
  "is_actionable": true,
  "title": "tiêu đề ngắn gọn",
  "description": "mô tả chi tiết",
  "priority": "low | medium | high | urgent",
  "due_at": "RFC3339 hoặc rỗng",
  "start_at": "RFC3339 hoặc rỗng",
  "end_at": "RFC3339 hoặc rỗng",
  "all_day": false,
  "location": "địa điểm hoặc rỗng",
  "reason": "lý do ngắn nếu là other"
}
```

Định nghĩa category (đưa nguyên văn vào system prompt):

| Category | Nghĩa | Ví dụ |
|---|---|---|
| `task` | việc cần làm, có hành động | yêu cầu nộp báo cáo, hoá đơn, deadline |
| `schedule` | lịch hẹn/lịch trình có thời gian | cuộc họp, appointment, ca làm |
| `event` | sự kiện/thiệp mời/hội thảo/tiệc | thư mời hội thảo, sinh nhật, khai trương |
| `other` | thông báo, quảng cáo, newsletter, spam, không đủ dữ kiện | bản tin nội bộ, khuyến mãi |

### 2. Rule dự phòng — keyword (khi thiếu `DEEPSEEK_API_KEY` hoặc DeepSeek lỗi/trả category lạ)

`internal/service/classifier.go` chấm điểm từ khoá trên `subject + snippet + body`
(từ khoá xuất hiện trong subject được nhân đôi), lấy category điểm cao nhất:

- **event**: `sự kiện`, `hội thảo`, `hội nghị`, `conference`, `webinar`, `workshop`, `seminar`,
  `sinh nhật`, `birthday`, `đám cưới`, `wedding`, `tiệc`, `party`, `khai trương`, `ra mắt`,
  `lễ`, `ceremony`, `thiệp mời`, `invitation`, `mời bạn`, `mời tham dự`, `mời tham gia`, `event`
- **schedule**: `lịch`, `lịch hẹn`, `appointment`, `schedule`, `calendar`, `cuộc họp`, `meeting`,
  `họp`, `đặt phòng`, `booking`, `reminder`, `nhắc lịch`, `ca làm`, `shift`, `call`, `zoom`,
  `google meet`
- **task**: `cần`, `hoàn thành`, `nộp`, `thanh toán`, `hóa đơn`/`hoá đơn`, `invoice`, `deadline`,
  `hạn chót`, `yêu cầu`, `action required`, `todo`/`to-do`, `task`, `công việc`, `xử lý`,
  `kiểm tra`, `duyệt`, `review`, `báo cáo`, `xác nhận`
- Không khớp từ khoá nào ⇒ `other` (reason `no_rule_matched`).

Với `schedule`/`event`, rule cố bóc thời gian bắt đầu từ text (`2026-10-05 09:00`,
`12/11/2026 14:00`) để tạo bản ghi; không bóc được thì đẩy về backlog.

### 3. Dispatch sau phân loại

| Category | Điều kiện | Đích | Fallback → backlog (`reason`) |
|---|---|---|---|
| `task` | `is_actionable` + có `title` | `task service` (`CreateTask`) | `not_actionable` |
| `schedule` | có `title` + `start_at` (hoặc `due_at`) | `calendar service` (`CreateSchedule`) | `missing_schedule_time` |
| `event` | có `title` + `start_at` (hoặc `due_at`) | `event service` (`CreateEvent`) | `missing_event_time` |
| `other` | luôn | **`backlog service`** (`CreateBacklog`) | — (reason từ LLM hoặc `unclassified`) |

Nhờ bảng này **không email nào bị mất**: khi không phân loại được hoặc thiếu dữ kiện để tạo bản ghi,
dữ liệu vẫn nằm ở backlog kèm `reason`, `source` (message id) và `object_key` (email gốc trên object storage).
Bỏ `MAIL_RULE_FALLBACK_ENABLED` đi kèm `DEEPSEEK_API_KEY` rỗng thì email rơi hết vào backlog với
reason `classifier_disabled`.

### 4. Lưu trữ object storage (S3)

Email gốc được lưu trước khi dispatch, object key:

```
mail/{profile_id}/{yyyymmdd}/{message_id}.json
mail/{profile_id}/{yyyymmdd}/{message_id}/attachments/{filename}
```

Giới hạn: `MAIL_MAX_ATTACHMENTS` tệp/email và `MAIL_MAX_ATTACHMENT_BYTES` mỗi tệp.

## Biến môi trường

| Biến | Default | Việc |
|---|---|---|
| `SERVICE_NAME` | `mail-provider` | tên log |
| `GRPC_ADDR` | `:9084` | cmd/grpc listen |
| `HTTP_ADDR` | `:8084` | cmd/http listen |
| `GRPC_DIAL_ADDR` | rỗng → `127.0.0.1:9084` | cmd/http dial gRPC |
| `LOG_LEVEL` | `info` | mức log |
| `DATABASE_URL` | rỗng | Postgres cho subscription; rỗng ⇒ in-memory (mất khi restart) |
| `MAIL_PUBSUB_TOPIC` | rỗng | `projects/<gcp>/topics/<topic>`; rỗng ⇒ `Subscribe` trả `MAIL_NOT_CONFIGURED` |
| `MAIL_PUBSUB_AUDIENCE` | rỗng | audience của OIDC token Pub/Sub; rỗng ⇒ **bỏ qua** xác thực webhook |
| `MAIL_PUBSUB_SERVICE_ACCOUNT` | rỗng | email service account phải khớp claim `email` |
| `MAIL_NOTIFICATION_PATH` | `/v1/notifications` | path webhook |
| `MAIL_PUBSUB_BASE_URL` | `https://pubsub.googleapis.com` | base URL Pub/Sub (đổi được để trỏ emulator) |
| `MAIL_PULL_SUBSCRIPTION` | rỗng | `projects/<gcp>/subscriptions/<sub>`; **rỗng ⇒ tắt pull mode** |
| `MAIL_PULL_MAX_MESSAGES` | `10` | số message mỗi lần pull |
| `MAIL_PULL_TIMEOUT` | `2m` | timeout HTTP cho pull (long-poll) |
| `MAIL_PULL_RETRY_DELAY` | `5s` | nghỉ trước khi thử lại khi pull lỗi |
| `MAIL_PULL_HEARTBEAT` | `5m` | log nhịp tim khi pull rỗng kéo dài (`0s` = tắt) |
| `GOOGLE_APPLICATION_CREDENTIALS` | rỗng | file key service account cho ADC khi dùng pull |
| `MAIL_WATCH_LABEL_IDS` | `INBOX` | danh sách label, phân tách bởi dấu phẩy |
| `MAIL_WATCH_RENEW_INTERVAL` | `12h` | chu kỳ quét gia hạn watch |
| `MAIL_WATCH_RENEW_THRESHOLD` | `24h` | gia hạn khi watch còn ít hơn mức này |
| `MAIL_DEFAULT_PRIORITY` | `medium` | priority khi DeepSeek không xác định |
| `MAIL_QUEUE_SIZE` | `256` | sức chứa queue notice trong RAM |
| `MAIL_WORKER_COUNT` | `2` | số worker xử lý song song |
| `MAIL_TASK_TIMEOUT` | `15s` | timeout gọi task service |
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` | rỗng | dùng để refresh access token |
| `GOOGLE_TOKEN_URL` | `https://oauth2.googleapis.com/token` | token endpoint |
| `GMAIL_BASE_URL` | `https://gmail.googleapis.com` | Gmail API base |
| `GMAIL_TIMEOUT` | `20s` | timeout gọi Gmail |
| `GMAIL_MAX_BODY_BYTES` | `32768` | cắt bớt body trước khi gửi DeepSeek |
| `DEEPSEEK_API_KEY` | rỗng | rỗng ⇒ không gọi LLM, dùng rule keyword dự phòng |
| `DEEPSEEK_BASE_URL` | `https://api.deepseek.com` | base URL |
| `DEEPSEEK_MODEL` | `deepseek-chat` | model |
| `DEEPSEEK_TIMEOUT` | `60s` | timeout gọi DeepSeek |
| `MAIL_RULE_FALLBACK_ENABLED` | `true` | bật rule keyword khi LLM thiếu/lỗi |
| `TASK_GRPC_DIAL_ADDR` | rỗng → `127.0.0.1:9082` | task service |
| `MAIL_TASK_TIMEOUT` | `15s` | timeout gọi task service |
| `CALENDAR_GRPC_DIAL_ADDR` | rỗng → `127.0.0.1:9083` | calendar service (schedule) |
| `MAIL_SCHEDULE_TIMEOUT` | `15s` | timeout gọi calendar service |
| `EVENT_GRPC_DIAL_ADDR` | rỗng → `127.0.0.1:9085` | event service |
| `MAIL_EVENT_TIMEOUT` | `15s` | timeout gọi event service |
| `BACKLOG_GRPC_DIAL_ADDR` | rỗng → `127.0.0.1:9086` | backlog service |
| `MAIL_BACKLOG_TIMEOUT` | `15s` | timeout gọi backlog service |
| `S3_ENDPOINT` | rỗng | `host:port` server S3; rỗng ⇒ tắt archive |
| `S3_ACCESS_KEY` / `S3_SECRET_KEY` | rỗng | credential S3 |
| `S3_BUCKET` | `task-manager` | bucket chứa email gốc |
| `S3_REGION` | `us-east-1` | region ký S3 |
| `S3_USE_SSL` | `false` | dùng HTTPS tới server S3 |
| `S3_TIMEOUT` | `30s` | timeout gọi S3 |
| `MAIL_ARCHIVE_ENABLED` | `true` | bật/tắt lưu email gốc |
| `MAIL_MAX_ATTACHMENTS` | `5` | số attachment tối đa lưu mỗi email |
| `MAIL_MAX_ATTACHMENT_BYTES` | `10485760` | kích thước tối đa mỗi attachment |

## Cấu trúc

```
Dockerfile                       # build 2 binary grpc + http (build context = root repo)
cmd/grpc/                        # fx app: gRPC server (:9084), store Postgres/in-memory, worker nền
cmd/http/                        # fx app: grpc-gateway (:8084) + webhook Pub/Sub + /healthz
internal/config/config.go        # đọc env
internal/model/                  # CORE: domain model + lỗi domain (chỉ stdlib)
internal/service/                # nghiệp vụ: subscribe, token cache, queue, worker push + worker pull, rule phân loại
internal/repository/             # adapter I/O: Postgres/in-memory store, Gmail, Pub/Sub pull, OAuth token, DeepSeek, task/calendar/event/backlog client, S3
internal/dependency/             # validate + mapping model <-> proto + map lỗi -> gRPC
internal/handler/                # transport gRPC mỏng
```

`service/interfaces.go` khai báo các port (`SubscriptionRepository`, `GmailClient`,
`TokenRefresher`, `MailAnalyzer`, `TaskCreator`, `ScheduleCreator`, `EventCreator`,
`BacklogCreator`, `BlobStore`, `MailService`, `Worker`);
`service` chỉ import `model`, adapter ở `repository` thoả mãn port nhờ structural typing, `cmd`
là nơi wiring concrete.

## Chạy local

```bash
make run-mail-provider
```

Cần task service đang chạy để tạo task:

```bash
make run-task
```

Chạy local **không cần tunnel** nếu dùng pull mode:

```bash
gcloud pubsub subscriptions create gmail-pull --topic=<TOPIC_ID>
gcloud pubsub subscriptions add-iam-policy-binding gmail-pull \
  --member=serviceAccount:<sa>@<project>.iam.gserviceaccount.com \
  --role=roles/pubsub.subscriber

export MAIL_PULL_SUBSCRIPTION=projects/<project>/subscriptions/gmail-pull
export GOOGLE_APPLICATION_CREDENTIALS=/duong/dan/key.json
make run-mail-provider
```

Thử webhook bằng tay (khi `MAIL_PUBSUB_AUDIENCE` trống):

```bash
DATA=$(printf '{"emailAddress":"user@gmail.com","historyId":"12345"}' | base64 -w0)
curl -s -X POST localhost:8084/v1/notifications \
  -H 'Content-Type: application/json' \
  -d "{\"message\":{\"data\":\"$DATA\",\"messageId\":\"m1\"}}"
```

## Cấu hình Google Cloud

1. Topic Pub/Sub phải nằm **cùng project** với OAuth client, nếu không `users.watch` trả
   `Invalid topicName`.
2. Cấp quyền publish cho Gmail:
   `gcloud pubsub topics add-iam-policy-binding <topic> --member=serviceAccount:gmail-api-push@system.gserviceaccount.com --role=roles/pubsub.publisher`
3. Tạo push subscription trỏ tới `<PUBLIC_BASE_URL>/api/mail/v1/notifications`, kèm OIDC token:
   `--push-auth-service-account=<sa>@<project>.iam.gserviceaccount.com`
   `--push-auth-token-audience=<MAIL_PUBSUB_AUDIENCE>`
4. Khai báo scope `gmail.readonly` cho OAuth consent screen và thêm chính user vào **Test users**
   nếu app còn ở trạng thái Testing (khi đó refresh token chỉ sống 7 ngày).

## Giới hạn đã biết

- Có `DATABASE_URL` thì subscription nằm ở Postgres nên restart container **không** mất watch
  (không cần user đăng nhập lại); không set thì store in-memory và restart là mất subscription + token.
- Không dedupe theo `messageId` của Gmail: nếu checkpoint bị đặt lại, có thể tạo task trùng.
- Pub/Sub push là at-least-once ⇒ một email có thể được xử lý hơn một lần trong tình huống retry.

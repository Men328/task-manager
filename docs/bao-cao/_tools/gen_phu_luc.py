"""Sinh noi-dung/phu-luc.md: A cài đặt, B toàn bộ API (đọc từ proto + route viết tay),
C biến môi trường chính, D bộ mã lỗi chuẩn (đọc từ common/errorcode/error_codes.json).

python gen_phu_luc.py
"""
import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
OUT = Path(__file__).resolve().parent.parent / "noi-dung" / "phu-luc.md"

SERVICE_OF_PROTO = {
    "identity": "identity", "task": "task", "calendar": "calendar", "event": "event",
    "backlog": "backlog", "attachment": "attachment", "report": "report", "project": "project",
}

# Route HTTP viết tay (không sinh từ proto) — đối chiếu cmd/http/*_routes.go, multipart.go, pubsub.go
HANDWRITTEN = [
    ("identity", "GET", "/v1/auth/google/login", "Bắt đầu đăng nhập Google", "Không (trang công khai)"),
    ("identity", "GET", "/v1/auth/google/callback", "Nhận mã uỷ quyền từ Google", "state + PKCE"),
    ("identity", "GET", "/v1/auth/me", "Hồ sơ người dùng hiện tại", "JWT"),
    ("identity", "POST", "/v1/auth/soketi", "Ký uỷ quyền kênh realtime riêng tư", "JWT"),
    ("attachment", "POST", "/v1/attachments/upload", "Tải tệp lên (multipart)", "Không"),
    ("attachment", "GET", "/v1/attachments/{id}/download", "Tải tệp xuống", "Không"),
    ("notification", "GET", "/v1/notices", "Danh sách thông báo", "JWT"),
    ("notification", "GET", "/v1/notices/unread-count", "Số thông báo chưa đọc", "JWT"),
    ("notification", "POST", "/v1/notices/read", "Đánh dấu đã đọc (ids hoặc \"*\")", "JWT"),
    ("social", "GET", "/v1/friends/search", "Tìm người dùng theo email", "JWT"),
    ("social", "GET", "/v1/friends", "Danh sách bạn bè", "JWT"),
    ("social", "DELETE", "/v1/friends/{profile_id}", "Huỷ kết bạn", "JWT"),
    ("social", "GET", "/v1/friend-requests", "Lời mời đến và đã gửi", "JWT"),
    ("social", "POST", "/v1/friend-requests", "Gửi lời mời kết bạn", "JWT"),
    ("social", "POST", "/v1/friend-requests/{id}/accept", "Chấp nhận lời mời", "JWT"),
    ("social", "POST", "/v1/friend-requests/{id}/decline", "Từ chối lời mời", "JWT"),
    ("social", "DELETE", "/v1/friend-requests/{id}", "Huỷ lời mời đã gửi", "JWT"),
    ("message", "GET", "/v1/conversations", "Danh sách cuộc trò chuyện", "JWT"),
    ("message", "GET", "/v1/conversations/{peer_id}/messages", "Lịch sử tin nhắn (phân trang)", "JWT"),
    ("message", "POST", "/v1/conversations/{peer_id}/messages", "Gửi tin nhắn", "JWT"),
    ("message", "POST", "/v1/conversations/{peer_id}/tasks", "Giao task qua chat (/task)", "JWT"),
    ("message", "GET", "/v1/messages/{message_id}", "Đọc một tin nhắn", "JWT"),
    ("mail-provider", "POST", "/v1/notifications", "Webhook nhận thông báo Gmail từ Pub/Sub", "OIDC Pub/Sub"),
]


def proto_routes():
    rows = []
    for f in sorted((ROOT / "common/proto").glob("*/v1/*.proto")):
        svc = SERVICE_OF_PROTO.get(f.parts[-3])
        if not svc:
            continue
        text = f.read_text()
        for m in re.finditer(r"rpc (\w+)\([^)]*\) returns \([^)]*\)\s*\{(.*?)\n  \}", text, re.S):
            hm = re.search(r"(get|post|patch|delete|put):\s*\"([^\"]+)\"", m.group(2))
            if hm:
                rows.append((svc, hm.group(1).upper(), hm.group(2), m.group(1), "Không"))
    return rows


def main():
    L = ["<!-- Sinh bởi _tools/gen_phu_luc.py. Sửa script rồi chạy lại, không sửa tay. -->"]
    w = L.append
    w("## Phụ lục A. Hướng dẫn cài đặt và chạy hệ thống")
    w("")
    w("**Yêu cầu:** Go 1.25 trở lên, Node.js 20 trở lên, Docker và Docker Compose. Các bước:")
    w("")
    for s in [
        "Lấy mã nguồn và tạo tệp cấu hình: sao chép `deployments/docker/.env.example` thành `deployments/docker/.env`, điền `SESSION_SECRET`, `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, thông tin RustFS và Soketi.",
        "Tuỳ chọn tự động phân loại email: điền `MAIL_PUBSUB_TOPIC` (topic Pub/Sub cùng dự án Google Cloud với OAuth client), `DEEPSEEK_API_KEY`; cấp quyền publish cho `gmail-api-push@system.gserviceaccount.com` trên topic.",
        "Cài công cụ và sinh mã (lần đầu): `make tools`, `make gen`.",
        "Khởi động toàn bộ hệ thống: `make up` (hoặc `make up-tunnel` để bật Cloudflare Tunnel). Lệnh này chạy PostgreSQL, migration, RustFS, Soketi, Kafka, 13 dịch vụ và giao diện.",
        "Truy cập `http://localhost:3000`, đăng nhập bằng tài khoản Google, tích \"Cho phép đọc Gmail\" nếu muốn bật phân loại email.",
        "Kiểm tra: `make ps` xem trạng thái container; `make test` chạy kiểm thử đơn vị; `make quality-all` kiểm tra chất lượng mã; `make down` dừng hệ thống.",
    ]:
        w("- " + s)
    w("")
    w("## Phụ lục B. Danh sách đầy đủ API")
    w("")
    w("Đường dẫn dưới đây được gọi qua cổng vào với tiền tố `/api/<dịch vụ>`, ví dụ `GET /api/task/v1/tasks`. Cột *Xác thực*: JWT là session token trong tiêu đề Authorization; “Không” là gateway chưa kiểm tra token mà nhận `profile_id` trong yêu cầu (xem mục 4.3.4).")
    w("")
    w("[BANG: Bảng B.1: Danh sách đầy đủ API của hệ thống]")
    w("| Dịch vụ | Phương thức | Đường dẫn | Chức năng / RPC | Xác thực |")
    w("|---|---|---|---|---|")
    rows = proto_routes() + [r for r in HANDWRITTEN]
    order = ["identity", "project", "task", "calendar", "event", "backlog", "attachment", "report",
             "notification", "social", "message", "mail-provider"]
    rows.sort(key=lambda r: (order.index(r[0]), r[2], r[1]))
    for r in rows:
        w(f"| {r[0]} | {r[1]} | `{r[2]}` | {r[3]} | {r[4]} |")
    w("")
    w("## Phụ lục C. Biến môi trường chính")
    w("")
    w("[BANG: Bảng C.1: Các biến môi trường chính]")
    w("| Biến | Dùng bởi | Ý nghĩa | Mặc định |")
    w("|---|---|---|---|")
    for r in [
        ("DATABASE_URL", "Các dịch vụ có CSDL, worker", "Chuỗi kết nối PostgreSQL; bỏ trống thì dùng bộ nhớ (trừ worker)", "rỗng"),
        ("GRPC_ADDR / HTTP_ADDR", "Mọi dịch vụ", "Cổng gRPC server và gateway HTTP", "theo dịch vụ"),
        ("<DỊCH VỤ>_GRPC_DIAL_ADDR", "Dịch vụ gọi dịch vụ khác", "Địa chỉ gRPC của dịch vụ đích", "127.0.0.1:<cổng>"),
        ("SESSION_SECRET", "identity, notification, social, message", "Khoá ký session JWT (phải giống nhau)", "bắt buộc"),
        ("SESSION_TTL", "identity", "Thời hạn phiên đăng nhập", "720h"),
        ("GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET", "identity, mail-provider", "OAuth client của Google", "bắt buộc khi đăng nhập"),
        ("SOKETI_BASE_URL / SOKETI_APP_ID / SOKETI_APP_KEY / SOKETI_APP_SECRET", "identity, notification, message, worker", "Kết nối và ký sự kiện Soketi", "soketi:6001"),
        ("KAFKA_BROKERS", "message, task, worker", "Địa chỉ Kafka", "localhost:29092"),
        ("KAFKA_MESSAGE_TOPIC / KAFKA_TASK_EVENTS_TOPIC", "message, task, worker", "Tên topic", "chat.messages / task.events"),
        ("S3_ENDPOINT / S3_BUCKET / S3_ACCESS_KEY / S3_SECRET_KEY", "attachment, mail-provider", "Kho đối tượng S3 (RustFS)", "bucket task-manager"),
        ("ATTACHMENT_MAX_BYTES", "attachment", "Kích thước tệp tối đa", "10 MiB"),
        ("MAIL_PUBSUB_TOPIC", "mail-provider", "Topic Pub/Sub nhận thông báo Gmail", "rỗng (tắt)"),
        ("MAIL_PULL_SUBSCRIPTION", "mail-provider", "Bật chế độ pull thay cho push", "rỗng"),
        ("DEEPSEEK_API_KEY / DEEPSEEK_MODEL", "mail-provider", "Khoá và model phân loại email", "rỗng / deepseek-chat"),
        ("PROJECT_MATCH_THRESHOLD", "mail-provider", "Ngưỡng tin cậy khớp email vào dự án", "0,6"),
        ("MAIL_RULE_FALLBACK_ENABLED", "mail-provider", "Dùng luật từ khoá khi DeepSeek lỗi", "true"),
        ("REMINDER_INTERVAL / REMINDER_TIMEZONE", "task", "Chu kỳ quét nhắc việc, múi giờ", "30s / Asia/Ho_Chi_Minh"),
        ("WORKER_REPLICAS", "docker compose", "Số bản chạy của worker", "2"),
    ]:
        w("| " + " | ".join(f"`{r[0]}`" if i == 0 else r[i] for i in range(4)) + " |")
    w("")
    w("## Phụ lục D. Bộ mã lỗi chuẩn")
    w("")
    codes = json.loads((ROOT / "common/errorcode/error_codes.json").read_text())["codes"]
    w(f"Bộ mã gồm {len(codes)} mã lỗi, định nghĩa trong `common/errorcode/error_codes.json`. Mã HTTP 0 là lỗi phía trình duyệt (không tới được máy chủ).")
    w("")
    w("[BANG: Bảng D.1: Bộ mã lỗi chuẩn của hệ thống]")
    w("| Mã lỗi | HTTP | Thông điệp (tiếng Việt) |")
    w("|---|---|---|")
    for code, info in codes.items():
        w(f"| `{code}` | {info.get('httpStatus')} | {info.get('message', {}).get('vi', '')} |")
    OUT.write_text("\n".join(L) + "\n", encoding="utf-8")
    print("written", OUT, len(rows), "API,", len(codes), "mã lỗi")


main()

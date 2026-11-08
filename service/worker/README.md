# service/worker

Service chuyên **consume Kafka**. Không có gRPC/HTTP API, chỉ `GET /healthz` (`:8094`, ping Postgres).
Scale ngang bằng số replica — đây là thành phần dự kiến scale khi tải tăng.

Hiện có 1 consumer:

| Topic | Group | Việc |
|---|---|---|
| `chat.messages` | `message-worker` | lưu tin nhắn vào `message.messages` + cập nhật `message.conversations` |

## Đảm bảo

- **At-least-once**: chỉ commit offset sau khi ghi DB thành công. Ghi idempotent theo `id`
  (`ON CONFLICT DO NOTHING`), nên xử lý lại không tạo bản trùng.
- **Thứ tự**: message-service dùng `conversation_id` làm key, nên 1 cuộc trò chuyện luôn vào cùng
  partition và được 1 worker xử lý tuần tự.
- **Lỗi tạm thời** (DB down, timeout): thử lại với backoff nhân đôi (`RETRY_INITIAL_BACKOFF` ->
  `RETRY_MAX_BACKOFF`), không commit, không bỏ tin.
- **Sự kiện hỏng** (JSON lỗi, thiếu field, vi phạm ràng buộc DB — mã lỗi Postgres lớp 22/23): ghi log
  ERROR kèm payload rồi commit để không kẹt partition.

## Scale

```bash
docker compose -f deployments/docker/docker-compose.yml up -d --scale worker=4
```

Số worker xử lý song song tối đa = số partition của topic (`KAFKA_MESSAGE_PARTITIONS`, mặc định 6).
Worker vượt số partition sẽ đứng chờ. Muốn tăng thêm:

```bash
docker exec tm-kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 \
  --alter --topic chat.messages --partitions 12
```

## Thêm consumer mới

1. Model + interface trong `internal/model`, `internal/service/interfaces.go`.
2. Logic xử lý trong `internal/service`, decode payload trong `internal/dependency`.
3. Vòng consume trong `internal/handler` (theo mẫu `MessageConsumer`).
4. Wire reader + `fx.Invoke` trong `cmd/worker`.

## Chạy

```bash
make run-worker   # cần postgres + kafka (docker compose up -d kafka-init postgres migrate)
```

| Biến | Default |
|---|---|
| `HEALTH_ADDR` | `:8094` |
| `DATABASE_URL` | bắt buộc |
| `KAFKA_BROKERS` | `localhost:29092` |
| `KAFKA_MESSAGE_TOPIC` | `chat.messages` |
| `KAFKA_MESSAGE_GROUP_ID` | `message-worker` |
| `RETRY_INITIAL_BACKOFF` / `RETRY_MAX_BACKOFF` | `500ms` / `30s` |
| `PERSIST_TIMEOUT` | `10s` |

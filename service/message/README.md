# service/message

Nhắn tin 1-1 giữa bạn bè. Là **1 Go module riêng** (`taskmanager/service/message`).

- gRPC: `:9092` — HTTP gateway: `:8092`
- Proto: `common/proto/message/v1/message.proto`
- Bảng DB (chỉ ĐỌC): `message.conversations`, `message.messages` — ghi bởi `service/worker`

## Luồng gửi tin

```
POST /v1/conversations/{peer_id}/messages
  -> gateway: xác thực session JWT, người gửi = `sub`
  -> message-grpc:
       1. validate (1..2000 ký tự, peer là UUID, không tự nhắn mình)
       2. social-grpc AreFriends — chỉ bạn bè mới nhắn được
       3. sinh id (UUIDv7) + created_at, conversation_id = UUIDv5(cặp profile)
       4. broadcast qua soketi: event `message.created` vào 2 kênh private
          `private-chat-<recipient>` và `private-chat-<sender>` (đồng bộ nhiều tab)
       5. đẩy ASYNC vào Kafka topic `chat.messages` (key = conversation_id, value = protojson ChatMessage)
  -> trả tin nhắn cho client (client khớp bản gửi lạc quan bằng `client_id`)
```

- Soketi lỗi: chỉ ghi log, tin vẫn vào Kafka để lưu.
- Kafka writer chạy async (batch 10ms, `acks=all`, thử lại 10 lần). Lỗi giao nhận sau khi thử lại
  được ghi log kèm payload ở mức ERROR — tin đó đã được broadcast nhưng không được lưu.
- Lưu DB là bất đồng bộ nên lịch sử có thể trễ vài ms so với realtime; FE gộp theo `id`/`client_id`.

## API (qua nginx `/api/message`)

| Method | Path | Ghi chú |
|---|---|---|
| GET | `/v1/conversations` | 100 cuộc trò chuyện gần nhất + tin cuối |
| GET | `/v1/conversations/{peer_id}/messages?before=<RFC3339>&limit=50` | trả cũ -> mới, `hasMore` để phân trang |
| POST | `/v1/conversations/{peer_id}/messages` | body `{"body": "...", "client_id": "..."}` |

Kênh `private-chat-<profile_id>` được identity ký uỷ quyền (`POST /v1/auth/soketi`) chỉ cho chính chủ.

## Chạy

```bash
make run-message   # cần social-grpc, soketi, kafka (docker compose up -d kafka-init soketi)
```

| Biến | Default |
|---|---|
| `GRPC_ADDR` / `HTTP_ADDR` | `:9092` / `:8092` |
| `DATABASE_URL` | rỗng = API đọc trả danh sách rỗng |
| `SESSION_SECRET` | phải khớp identity |
| `SOCIAL_GRPC_DIAL_ADDR` | `localhost:9091` |
| `UPSTREAM_TIMEOUT` | `5s` |
| `SOKETI_BASE_URL` / `SOKETI_APP_ID` / `SOKETI_APP_KEY` / `SOKETI_APP_SECRET` | thiếu = không broadcast |
| `SOKETI_CHAT_CHANNEL_PREFIX` | `chat-` (phải khớp identity) |
| `SOKETI_TIMEOUT` | `3s` |
| `KAFKA_BROKERS` | `localhost:29092` |
| `KAFKA_MESSAGE_TOPIC` | `chat.messages` |

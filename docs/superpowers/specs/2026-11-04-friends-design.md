# Thiết kế: kết bạn

Ngày: 2026-11-04 · Trạng thái: chờ duyệt · Nhánh: `feature/friends`

## 1. Ý định

Ứng dụng chưa có khái niệm "bạn bè". Người dùng muốn tìm được nhau bằng email, gửi
lời mời kết bạn, xem danh sách bạn bè và hủy kết bạn.

Người dùng đã chốt: giai đoạn này bạn bè **chỉ là một danh sách**. Dữ liệu (project,
task, lịch) vẫn riêng theo từng profile, chưa chia sẻ gì.

## 2. Mục tiêu và phạm vi

**Mục tiêu**

- Tìm người dùng bằng email và gửi lời mời.
- Người nhận được thông báo realtime, rồi chấp nhận hoặc từ chối.
- Người gửi rút lại được lời mời đang chờ.
- Xem danh sách bạn bè, lời mời đến, lời mời đã gửi.
- Hủy kết bạn.

**Không làm**

- Chia sẻ project/task, giao việc, nhắn tin, nhóm.
- Chặn người dùng, gợi ý bạn bè, mời người chưa có tài khoản qua email.
- Giới hạn tần suất gửi lời mời (xem mục 8).

**Không được xảy ra**

- Lấy được danh sách email, hoặc dò email bằng tìm kiếm theo từng phần.
- Thao tác thay người khác bằng cách gửi `profile_id` tùy ý.

## 3. Ràng buộc

- Gateway mặc định của các service **không kiểm tra token**: `/v1/profiles` hiện trả
  toàn bộ profile cho bất kỳ ai. Không dùng lại kiểu này cho social.
- Proto chỉ thêm, không đổi số field. Sửa proto xong chạy `make gen`, commit `common/gen`.
- Mã lỗi mới: sửa `error_codes.json` + `errorcode.go`, chạy `make error-codes`.
- `make quality-all` phải pass: service không import repository/dependency, `cmd` dùng fx.
- Sửa repository Postgres thì sửa song song bản in-memory, có test.
- Profile bị xóa mềm (`deleted_at`), nên `ON DELETE CASCADE` không chạy ở luồng thường.

## 4. Các cách đã cân nhắc

| Cách | Ưu điểm | Đánh đổi |
|---|---|---|
| Đặt trong `identity` | Truy vấn profile tại chỗ, không thêm container | `identity` phình ra, trộn tài khoản với quan hệ xã hội |
| **Service `social` riêng (người dùng chọn)** | Tách biệt rõ, có chỗ cho tính năng xã hội sau này | Thêm 2 container, mỗi lần tìm email hay hiển thị danh sách phải gọi sang `identity` |

Khai báo API qua `google.api.http` bị loại ở cả hai cách vì gateway không kiểm tra token.

## 5. Thiết kế

### 5.1 Service `service/social`

Cấu trúc giống `service/project`: `cmd/grpc`, `cmd/http`, `internal/{config,model,service,repository,dependency,handler}`.

- gRPC `:9091`, HTTP `:8091`.
- Nginx: `/api/social/` → `http://social:8091/`.
- Gọi `identity-grpc:9081` (tra profile) và `notification-grpc:9088` (gửi thông báo).
- Thêm vào: `go.work`, `MODULES` trong `Makefile`, `scripts/gen.sh`, `scripts/dev.sh`,
  `scripts/build-images.sh`, `docker-compose.yml`, `frontend/nginx.conf`.

### 5.2 Dữ liệu: migration `000015_init_social`

```sql
CREATE SCHEMA IF NOT EXISTS social;

CREATE TABLE social.friendships (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    requester_id uuid        NOT NULL REFERENCES identity.profiles (id) ON DELETE CASCADE,
    addressee_id uuid        NOT NULL REFERENCES identity.profiles (id) ON DELETE CASCADE,
    status       varchar(16) NOT NULL DEFAULT 'pending',
    created_at   timestamptz NOT NULL DEFAULT now(),
    accepted_at  timestamptz,
    CONSTRAINT ck_friendships_not_self CHECK (requester_id <> addressee_id),
    CONSTRAINT ck_friendships_status   CHECK (status IN ('pending', 'accepted'))
);

-- Mỗi cặp chỉ có 1 dòng, bất kể ai là người gửi.
CREATE UNIQUE INDEX uq_friendships_pair ON social.friendships
    (LEAST(requester_id, addressee_id), GREATEST(requester_id, addressee_id));
CREATE INDEX ix_friendships_requester ON social.friendships (requester_id, status);
CREATE INDEX ix_friendships_addressee ON social.friendships (addressee_id, status);
```

Từ chối, rút lời mời và hủy kết bạn đều **xóa dòng**, nên không cần trạng thái khác
và hai người có thể kết bạn lại từ đầu.

### 5.3 Luồng trạng thái

| Hành động | Ai làm được | Kết quả |
|---|---|---|
| Gửi lời mời | Bất kỳ ai, trừ chính mình | Tạo dòng `pending`, báo `FRIEND_REQUEST` cho người nhận |
| Gửi khi đối phương đã mời mình | Người nhận cũ | Chuyển thẳng sang `accepted`, báo `FRIEND_ACCEPTED` |
| Gửi khi đã mời, hoặc đã là bạn | — | Lỗi `SOCIAL_FRIEND_REQUEST_EXISTS` / `SOCIAL_ALREADY_FRIENDS` |
| Chấp nhận | Chỉ người nhận | `accepted`, báo `FRIEND_ACCEPTED` cho người gửi |
| Từ chối | Chỉ người nhận | Xóa dòng, không báo |
| Rút lời mời | Chỉ người gửi | Xóa dòng, không báo |
| Hủy kết bạn | Một trong hai bên | Xóa dòng, không báo |

Hai người gửi cùng lúc: unique index chặn dòng thứ hai; service bắt lỗi trùng rồi đọc
lại và chuyển sang trường hợp "đối phương đã mời mình".

Thông báo là best-effort: lỗi khi gọi `notification` chỉ ghi log, không làm hỏng thao tác. Thông báo
chạy trên context tách khỏi request với hạn 2 giây, để service phía sau bị treo không làm lỗi một
thao tác đã xong.

### 5.4 HTTP API (đều bắt buộc bearer token)

Route viết tay trong `service/social/cmd/http`, theo mẫu
`service/notification/cmd/http/notice_routes.go` và `session.go`. `profile_id` luôn lấy
từ `sub` của token, không nhận từ client. Cần env `SESSION_SECRET`.

| Method | Path | Việc làm |
|---|---|---|
| GET | `/v1/friends/search?email=` | Tìm, trả profile rút gọn + `relation` |
| GET | `/v1/friends` | Danh sách bạn bè |
| DELETE | `/v1/friends/{profile_id}` | Hủy kết bạn |
| GET | `/v1/friend-requests` | Lời mời đến và đã gửi (`incoming`, `outgoing`) |
| POST | `/v1/friend-requests` | Gửi lời mời, body `{ "profile_id": "..." }` |
| POST | `/v1/friend-requests/{id}/accept` | Chấp nhận |
| POST | `/v1/friend-requests/{id}/decline` | Từ chối |
| DELETE | `/v1/friend-requests/{id}` | Rút lời mời |

**Tìm kiếm:** chuẩn hóa `trim` + lowercase rồi **khớp đúng toàn bộ email**. Không thấy
thì trả `200` với `profile: null`, không có lỗi riêng. `relation` là một trong `none`,
`self`, `friend`, `outgoing`, `incoming`.

Profile rút gọn gồm `id`, `email`, `displayName`, `avatarUrl`. Không trả `timezone`,
`locale`, `lastLoginAt`.

### 5.5 Thay đổi ở service khác

**identity** — 2 RPC nội bộ, **không** khai báo `google.api.http`:

- `FindProfileByEmail(email)`: dùng `GetByEmail` có sẵn.
- `BatchGetProfiles(ids)`: thêm `ListByIDs` vào repository (cả Postgres và in-memory),
  bỏ qua profile đã xóa mềm. Giới hạn 500 id mỗi lần gọi.

**notification** — thêm `NOTICE_TARGET_TYPE_FRIEND = 5`, nhận loại đích `friend` trong code và
trong ràng buộc DB (migration `000016_notice_friend_target`).

**Profile đã xóa mềm:** `BatchGetProfiles` không trả về nên social bỏ các dòng đó khỏi
danh sách. Không dọn bảng `friendships`.

### 5.6 Mã lỗi (owner `social`)

- `SOCIAL_CANNOT_FRIEND_SELF` (InvalidArgument)
- `SOCIAL_PROFILE_NOT_FOUND` (NotFound)
- `SOCIAL_FRIEND_REQUEST_EXISTS` (AlreadyExists)
- `SOCIAL_ALREADY_FRIENDS` (AlreadyExists)
- `SOCIAL_FRIEND_REQUEST_NOT_FOUND` (NotFound): dùng cả khi lời mời không thuộc về người gọi,
  để không lộ sự tồn tại của nó.
- `SOCIAL_FRIEND_NOT_FOUND` (NotFound)
- `SOCIAL_EMAIL_INVALID` (InvalidArgument)
- Token: dùng các mã tương ứng mà notification đang dùng.

### 5.7 Frontend

- Trang `/friends`, mục "Bạn bè" trong `navigation.ts`.
- Trang gồm ô tìm theo email ở trên, dưới là 3 tab: **Bạn bè**, **Lời mời đến** (kèm số
  đếm), **Đã gửi**.
- Kết quả tìm hiển thị avatar, tên, email và nút theo `relation`: Kết bạn / Đã gửi /
  Chấp nhận / Đã là bạn. `self` không có nút.
- Hủy kết bạn cần hộp thoại xác nhận.
- `frontend/src/api/social.ts` + `socialApi` trong `client.ts`, slice trong `context/`.
- `normalizeTargetType` nhận `friend`; `targetPath` trong `NotificationMenu.tsx` trả `/friends`.
- Khi nhận notice `FRIEND_REQUEST`/`FRIEND_ACCEPTED` trong lúc mở trang thì tải lại danh sách.
- Chuỗi hiển thị có đủ `vi.json` và `en.json`.

### 5.8 Kiểm thử

- Service: bảng trạng thái ở 5.3 trên repository in-memory, gồm quyền của từng bên và
  trường hợp hai người mời nhau.
- Repository Postgres: unique index theo cặp, xóa, liệt kê.
- HTTP: thiếu/sai token bị từ chối, `profile_id` trong body không thay được người gọi.
- identity: `ListByIDs` bỏ profile đã xóa mềm.
- Frontend: hành vi nút theo `relation` nếu tách được thành hàm thuần.
- `scripts/smoke-test`: thêm một lượt kết bạn đầy đủ giữa 2 profile.

## 6. Quyết định của người dùng

- Bạn bè chỉ là danh sách, chưa chia sẻ.
- Tách service `social` riêng, không đặt trong `identity`.

## 7. Quyết định của Claude (cần người dùng soát lại)

1. Thêm bước chấp nhận/từ chối và rút lời mời, dù yêu cầu gốc không nêu.
2. Tìm kiếm chỉ khớp đúng toàn bộ email.
3. Từ chối và hủy kết bạn **không báo** cho đối phương.
4. Hai người mời nhau thì tự thành bạn.
5. Từ chối xóa dòng, nên người gửi có thể mời lại ngay (xem mục 8).
6. Bạn bè thấy email của nhau.
7. Danh sách không phân trang.

## 8. Rủi ro đã biết

- **Spam lời mời:** không có giới hạn tần suất và không có chặn, nên một người có thể gửi
  lại sau mỗi lần bị từ chối. Tìm kiếm chính xác vẫn cho biết một email có tài khoản hay
  không; người dùng đăng nhập có thể thử từng email.
- **`/v1/profiles` vẫn hở:** lỗ hổng có sẵn, nằm ngoài nhánh này, và làm giảm tác dụng của
  việc giới hạn tìm kiếm. Nên sửa ở một nhánh riêng.

## 9. Phần để người làm tự quyết

Tên message/RPC trong proto, tên slice frontend, cấu trúc bên trong component, bố cục chi
tiết, miễn giữ đúng hành vi ở trên.

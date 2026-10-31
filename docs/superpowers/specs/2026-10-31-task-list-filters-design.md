# Task list: lọc theo hạn, ưu tiên, sắp xếp, phân trang

Ngày: 2026-10-31 · Nhánh: `feature/task-list-filters` · Trạng thái: chờ duyệt
Roadmap: [personal-essentials-roadmap](2026-10-31-personal-essentials-roadmap.md)

## 1. Ý định

`ListTasks` hiện chỉ lọc theo profile, project, status, task cha, archive và luôn
sắp theo `position, created_at`, trả toàn bộ kết quả. Màn Hôm nay/Sắp tới/Quá hạn,
thanh lọc Board và tìm kiếm đều cần lọc theo hạn và ưu tiên. Nhánh này chỉ làm
**nền backend + hàm API ở FE**, không có UI.

## 2. Mục tiêu / Không phải mục tiêu

**Mục tiêu**
- Lọc theo khoảng `due_at`, có/không có hạn, chưa hoàn thành, danh sách ưu tiên.
- Sắp xếp theo vị trí (mặc định), hạn, ưu tiên, ngày tạo, ngày sửa; tăng/giảm.
- Phân trang **tuỳ chọn**: không gửi `page_size` thì hành vi y như cũ.

**Không phải mục tiêu**
- UI (nhánh 2, 3). Tìm kiếm theo chữ (nhánh 4).
- Sửa lỗ hổng `profile_id` rỗng của API cũ (xem roadmap §6).

**Phản mục tiêu**
- Board/Report hiện tại đổi hành vi hoặc bị cắt dữ liệu vì phân trang ngầm.

## 3. Thiết kế

### 3.1 Proto (`common/proto/task/v1/task.proto`)

```proto
enum TaskSortField {
  TASK_SORT_FIELD_UNSPECIFIED = 0; // = POSITION
  TASK_SORT_FIELD_POSITION = 1;
  TASK_SORT_FIELD_DUE_AT = 2;
  TASK_SORT_FIELD_PRIORITY = 3;
  TASK_SORT_FIELD_CREATED_AT = 4;
  TASK_SORT_FIELD_UPDATED_AT = 5;
}

message ListTasksRequest {
  // ... field 1–7 giữ nguyên
  google.protobuf.Timestamp due_from = 8;   // due_at >= due_from
  google.protobuf.Timestamp due_to = 9;     // due_at <  due_to
  optional bool has_due_at = 10;            // true: due_at NOT NULL, false: due_at IS NULL
  bool open_only = 11;                      // status không thuộc nhóm done/cancelled
  repeated TaskPriority priorities = 12;    // rỗng = mọi mức
  TaskSortField sort_field = 13;
  bool sort_desc = 14;
  int32 page_size = 15;                     // 0 = không phân trang
  string page_token = 16;
}

message ListTasksResponse {
  // ... field cũ giữ nguyên
  string next_page_token = N;               // rỗng = hết
}
```

grpc-gateway map tự động sang query: `?due_from=2026-10-31T00:00:00Z&priorities=TASK_PRIORITY_HIGH&priorities=TASK_PRIORITY_URGENT&sort_field=TASK_SORT_FIELD_DUE_AT`.

### 3.2 Quy tắc

- Nếu dùng **bất kỳ** field mới nào (8–16) mà `profile_id` rỗng → lỗi
  `TASK_PROFILE_ID_REQUIRED` (mã có sẵn). Request chỉ dùng field cũ giữ hành vi cũ.
- `due_from`/`due_to` là nửa mở `[from, to)`; chỉ một trong hai cũng được.
  `due_from >= due_to` → `TASK_DUE_RANGE_INVALID` (mã mới, InvalidArgument/400).
- `open_only`: `JOIN task.task_statuses s ON s.id = t.status_id` và
  `s.category NOT IN ('done','cancelled')`. Không dùng `completed_at` vì task
  chuyển sang status `cancelled` không được gán `completed_at`.
- Lọc theo hạn ngầm định loại task không có hạn; `has_due_at=false` kết hợp với
  `due_from/due_to` → `TASK_DUE_RANGE_INVALID`.
- Sắp xếp:
  - `DUE_AT`: task không có hạn luôn **cuối** (cả tăng lẫn giảm) — `NULLS LAST`.
  - `PRIORITY`: theo thứ tự enum (URGENT > HIGH > MEDIUM > LOW); `sort_desc=true`
    nghĩa là URGENT trước (đây là cách dùng thường gặp; FE mặc định gửi desc).
    Cột `priority` là enum Postgres `task.task_priority ('low','medium','high','urgent')`
    nên `ORDER BY priority` so sánh đúng thứ tự này, không cần `CASE`.
  - Luôn thêm `id` làm khoá phụ cuối để thứ tự ổn định giữa các trang.
- Phân trang: `page_size` 1–500 (lớn hơn → kẹp về 500, âm → `TASK_PAGE_SIZE_INVALID`).
  `page_token` là offset mã hoá base64url (`"o:<offset>"`), opaque với client;
  token sai định dạng → `TASK_PAGE_TOKEN_INVALID`. Server lấy `page_size + 1` dòng
  để biết còn trang sau.
  *Lựa chọn của Claude:* offset thay vì keyset vì có 5 kiểu sắp xếp; dữ liệu một
  người dùng nhỏ, sai lệch khi có task mới chèn giữa các trang chấp nhận được.
- `include_subtasks=true` vẫn hoạt động như cũ trên từng phần tử trang.

### 3.3 Backend

- `model.TaskFilter` thêm: `DueFrom, DueTo *time.Time`, `HasDueAt *bool`,
  `OpenOnly bool`, `Priorities []TaskPriority`, `SortField`, `SortDesc`,
  `Limit, Offset int`. Kết quả trả thêm `HasMore` (hoặc repo trả `[]Task` dài
  `Limit+1`, service cắt).
- `PostgresTaskRepository.List`: chuyển câu SQL tĩnh sang dựng có điều kiện
  (giữ placeholder, **không nối chuỗi giá trị**); `ORDER BY` lấy từ whitelist map
  `SortField → "due_at"…`.
- In-memory repo lọc/sắp/cắt tương đương.
- Handler validate + mã hoá/giải mã token; service giữ logic nghiệp vụ.
- Index: dùng `ix_tasks_profile_due (profile_id, due_at)` có sẵn. Không migration.

### 3.4 Frontend

- `frontend/src/api/task.ts`: mở rộng `ListTasksParams` (`dueFrom`, `dueTo`,
  `hasDueAt`, `openOnly`, `priorities`, `sortField`, `sortDesc`,
  `pageSize`, `pageToken`); `buildQuery` cần hỗ trợ mảng (lặp key).
- Thêm `listTasksPage()` trả `{ tasks, nextPageToken }` và helper
  `listAllTasks(params)` đi hết các trang (dùng cho agenda).
- Thêm type + bản dịch cho mã lỗi mới.

## 4. Kiểm thử

- Test service + repo in-memory: từng bộ lọc, kết hợp, `NULLS LAST`, phân trang
  qua 3 trang không trùng/không sót, token hỏng, thiếu `profile_id`.
- Test repo Postgres (theo mẫu `postgres_project_repository_test.go` nếu có DB
  test) cho câu SQL dựng động.
- Gọi thử qua gateway: `curl '/v1/tasks?profile_id=..&due_to=..&open_only=true'`.

## 5. Để lại cho builder

- Tên chính xác của các mã lỗi mới, số field `next_page_token` trong response.
- Có tách hàm dựng SQL thành helper riêng để test không.

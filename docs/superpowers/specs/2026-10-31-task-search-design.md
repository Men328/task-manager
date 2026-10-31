# Tìm kiếm task (ô search TopBar + ⌘K)

Ngày: 2026-10-31 · Nhánh: `feature/task-search` · Trạng thái: chờ duyệt
Phụ thuộc: `feature/task-list-filters`
Roadmap: [personal-essentials-roadmap](2026-10-31-personal-essentials-roadmap.md)

## 1. Ý định

Ô search trên TopBar ([TopBar.tsx:42](../../../frontend/src/components/layout/TopBar.tsx))
chỉ giữ chữ trong state, không tìm gì; gợi ý ⌘K cũng chưa gắn. Người dùng gõ là
phải ra task, kể cả gõ không dấu.

## 2. Mục tiêu / Không phải mục tiêu / Phản mục tiêu

**Mục tiêu**
- Tìm theo tiêu đề + mô tả task, **mọi dự án**, không phân biệt hoa thường và dấu
  ("bao cao" ra "Báo cáo"), khớp một phần ("baocao" không cần, "bao ca" thì cần).
- Dropdown kết quả ngay dưới ô search, tối đa 20 kết quả, có chip dự án.
- ⌘K (macOS) / Ctrl+K focus ô search; Esc đóng; ↑/↓/Enter chọn.

**Không phải mục tiêu**
- Tìm trong schedule, event, backlog (roadmap sau).
- Trang kết quả đầy đủ, cú pháp tìm nâng cao.

**Phản mục tiêu**
- Mỗi phím gõ bắn một request; kết quả cũ đè kết quả mới.
- Gõ không dấu không ra gì (người Việt gõ nhanh hay bỏ dấu).

## 3. Thiết kế

### 3.1 Migration `000012_task_search`

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS unaccent;

-- unaccent() không IMMUTABLE nên không dùng trực tiếp trong index được.
CREATE OR REPLACE FUNCTION task.f_search_text(title text, description text)
RETURNS text LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
  SELECT lower(public.unaccent('public.unaccent'::regdictionary,
         coalesce(title, '') || ' ' || coalesce(description, '')))
$$;

CREATE INDEX IF NOT EXISTS ix_tasks_search_trgm
  ON task.tasks USING gin (task.f_search_text(title, description) gin_trgm_ops)
  WHERE deleted_at IS NULL;
```

Down: drop index, function (giữ extension — có thể đã dùng ở nơi khác).

- `unaccent` mặc định đổi `đ/Đ → d/D`; builder kiểm chứng trên `postgres:17-alpine`.
- `pg_trgm`, `unaccent` là extension *trusted* (PG13+), user `task_manager` sở hữu
  DB tạo được; nếu `migrate` báo thiếu quyền, ghi lại cách xử lý trong README.

### 3.2 API

`ListTasksRequest` thêm `string query = 17;`

- `query` sau khi trim: rỗng → bỏ qua; < 2 ký tự → `TASK_SEARCH_QUERY_TOO_SHORT`;
  > 200 ký tự → cắt về 200. Bắt buộc `profile_id` (như mọi field mới).
- Điều kiện: `task.f_search_text(title, description) LIKE '%' || <q_norm> || '%'`
  trong đó `q_norm = lower(unaccent($q))` (tính trong SQL), escape `%`, `_`, `\`.
- Khi có `query` và `sort_field` không chỉ định: sắp theo
  `similarity(task.f_search_text(...), q_norm) DESC`, rồi `updated_at DESC`.
- Mặc định FE gửi `include_archived=false`, **không** `open_only`
  (tìm cả task đã xong, hiển thị mờ).

### 3.3 Frontend

- `TopBar`: debounce 250ms; huỷ request cũ bằng `AbortController` (hoặc bỏ qua
  response có `requestId` cũ — giống cách "ignore stale fetch" đã làm ở commit
  `fix(frontend): ... ignore stale fetch responses`).
- Dropdown (Mantine `Popover`/`Combobox`): mỗi dòng = chấm màu + tên dự án, tiêu
  đề (tô đậm đoạn khớp, so không dấu), status, hạn. Task đã xong: mờ + gạch ngang.
- Chọn kết quả: mở `TaskDetailModal` và chuyển switcher sang dự án của task
  (cùng hành vi với agenda).
- Phím tắt: `keydown` toàn cục `(metaKey || ctrlKey) && key === 'k'` →
  `preventDefault` + focus; hiển thị "⌘K" trên macOS, "Ctrl K" nơi khác.
- Ô search hiện ẩn dưới breakpoint `xs` (`visibleFrom="xs"`): thêm nút icon kính
  lúp trên mobile mở cùng dropdown ở dạng modal toàn màn.
- Trạng thái: đang tải (skeleton), không có kết quả, lỗi.

## 4. Kiểm thử

- Repo Postgres: "bao cao" khớp "Báo cáo tuần"; "Đồ án" khớp "do an"; ký tự `%`
  trong query không thành wildcard; task đã xoá không ra.
- In-memory repo: so khớp tương đương (bỏ dấu bằng `golang.org/x/text/unicode/norm`
  + bảng `đ→d`) để test service.
- Kiểm tay: gõ nhanh không bị kết quả nhảy lung tung; ⌘K/Ctrl+K; mobile.

## 5. Để lại cho builder

- Có trả `similarity` về FE hay không; ngưỡng similarity tối thiểu (mặc định không
  đặt ngưỡng, chỉ dùng LIKE).

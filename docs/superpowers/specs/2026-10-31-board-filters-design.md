# Thanh lọc và sắp xếp trên Board

Ngày: 2026-10-31 · Nhánh: `feature/board-filters` · Trạng thái: chờ duyệt
Phụ thuộc: `feature/task-list-filters`
Roadmap: [personal-essentials-roadmap](2026-10-31-personal-essentials-roadmap.md)

## 1. Ý định

Board (trong 1 dự án) hiện hiển thị mọi task, không lọc được. Khi dự án nhiều task,
người dùng cần thu hẹp nhanh: việc quá hạn, việc gấp, việc tuần này.

## 2. Mục tiêu / Không phải mục tiêu

**Mục tiêu**
- Bộ lọc ở `BoardHeader`: **Hạn** (Tất cả · Quá hạn · Hôm nay · 7 ngày tới ·
  Không có hạn), **Ưu tiên** (chọn nhiều), **Ẩn đã hoàn thành** (bật/tắt).
- Sắp xếp (chỉ ở chế độ bảng `TaskTable`): Vị trí · Hạn · Ưu tiên · Ngày tạo ·
  Ngày sửa, tăng/giảm.
- Trạng thái lọc nằm trên URL (`?due=overdue&prio=high,urgent&hideDone=1&sort=due_at:asc`)
  để reload/chia sẻ giữ nguyên.
- Nút "Xoá lọc" + số bộ lọc đang bật.

**Không phải mục tiêu**
- Sắp xếp trong Kanban (Kanban luôn theo `position` để kéo-thả còn đúng).
- Lưu bộ lọc thành "view" có tên.

**Phản mục tiêu**
- Đang lọc mà kéo-thả trong Kanban làm hỏng `position` của task bị ẩn.

## 3. Thiết kế

- Mapping bộ lọc → `ListTasks` (luôn kèm `profile_id`, `project_id` hiện tại):
  - Quá hạn: `due_to=today0`, `open_only=true`.
  - Hôm nay: `due_from=today0`, `due_to=tomorrow0`.
  - 7 ngày tới: `due_from=today0`, `due_to=today0+8d`.
  - Không có hạn: `has_due_at=false`.
  - Ưu tiên: `priorities=[...]`. Ẩn đã hoàn thành: `open_only=true`.
  - Mốc ngày dùng chung hàm `agendaRanges` (nhánh agenda); nếu board-filters merge
    trước thì tạo hàm ở `lib/dateRanges.ts` và agenda dùng lại.
- Lọc phía **server** (gọi lại `ListTasks` khi bộ lọc đổi), giữ cấu trúc
  `root_only` + `include_subtasks` như Board hiện tại.
  - Hệ quả: lọc áp lên **task gốc**; task con hiện theo cha. *Lựa chọn của Claude*,
    vì Board hiện hiển thị cây; người cần thấy task con theo hạn dùng màn agenda.
- **Kéo-thả khi đang lọc**: tắt kéo-thả trong Kanban khi có bộ lọc bật, hiện gợi ý
  "Xoá lọc để sắp xếp lại" (tránh phản mục tiêu). Đổi status bằng menu vẫn được.
- Trống: "Không có task nào khớp bộ lọc" + nút Xoá lọc.
- Code: `components/board/BoardFilters.tsx`, đọc/ghi URL bằng `useSearchParams`,
  `boardSlice` nhận tham số lọc khi fetch; i18n vi/en.

## 4. Kiểm thử

- Kiểm tay từng bộ lọc + kết hợp; reload giữ lọc; chuyển dự án giữ lọc;
  kéo-thả bị tắt khi lọc; TaskTable sắp xếp đúng (không hạn ở cuối).

## 5. Để lại cho builder

- Dạng control (SegmentedControl, MultiSelect, Chip…), vị trí trong header.

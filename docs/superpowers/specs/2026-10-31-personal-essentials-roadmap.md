# Roadmap: các tính năng "quan trọng nhất" cho quản lý task cá nhân

Ngày: 2026-10-31 · Trạng thái: chờ duyệt

## 1. Ý định

So với các app task cá nhân thực tế (Todoist, TickTick, Things, Microsoft To Do),
hệ thống còn thiếu những thứ người dùng thấy ngay: không có màn "hôm nay làm gì",
không lọc theo hạn, ô tìm kiếm chỉ là trang trí, thêm task phải mở modal, không
nhắc việc, không có task lặp. Roadmap này tách 6 tính năng đó thành 7 nhánh, mỗi
nhánh 1 spec riêng, triển khai và merge độc lập.

## 2. Quyết định chung (người dùng đã chốt)

- Màn Hôm nay / Sắp tới / Quá hạn và kết quả tìm kiếm **gộp mọi dự án**, ngược với
  quy tắc "luôn đứng trong 1 project" của spec project (2026-10-28). Quy tắc đó
  vẫn giữ cho Board, Calendar, Events, Backlog, Report.
- Khi gộp, mỗi task phải **nhìn là biết thuộc dự án nào**: dải màu dự án ở mép trái,
  chip (màu + tên dự án) cạnh tiêu đề, hàng chip lọc theo dự án ở đầu màn; bấm
  task thì mở task và switcher chuyển sang dự án đó.
- Nhắc việc: cột `remind_at` trên task (1 lần nhắc / task), worker chạy trong
  `task-grpc`.
- Lặp lại: sinh task kế tiếp **khi hoàn thành**, quy tắc dạng RRULE, **chỉ task**
  (không làm schedule/event lặp).
- Tìm kiếm: `ILIKE` + `pg_trgm` + `unaccent`.
- Thêm nhanh: parse ở frontend, gọi `CreateTask` có sẵn.

## 3. Danh sách nhánh

| # | Nhánh | Spec | Phụ thuộc | Đụng tới | Rủi ro |
|---|---|---|---|---|---|
| 1 | `feature/task-list-filters` | [task-list-filters](2026-10-31-task-list-filters-design.md) | — | task (proto, repo, service), FE api | Thấp |
| 2 | `feature/agenda-view` | [agenda-view](2026-10-31-agenda-view-design.md) | 1 | FE | Thấp |
| 3 | `feature/board-filters` | [board-filters](2026-10-31-board-filters-design.md) | 1 | FE | Thấp |
| 4 | `feature/task-search` | [task-search](2026-10-31-task-search-design.md) | 1 | task (migration, repo), FE | Thấp–Vừa |
| 5 | `feature/quick-add` | [quick-add](2026-10-31-quick-add-design.md) | — (nên sau 2 để có chỗ đặt) | FE | Vừa |
| 6 | `feature/task-reminders` | [task-reminders](2026-10-31-task-reminders-design.md) | — (UI đẹp hơn nếu sau 2) | task (migration, worker, client notification), compose, FE | Cao |
| 7 | `feature/recurring-tasks` | [recurring-tasks](2026-10-31-recurring-tasks-design.md) | 6 | task (migration, service), FE, parser của 5 | Cao |

Thứ tự đề xuất: 1 → 2 → 3 → 4 → 5 → 6 → 7. Nhánh 2, 3, 4 có thể làm song song
sau khi 1 merge.

## 4. Ràng buộc chung

- Proto chỉ **thêm** field/RPC (không đổi số field, không đổi kiểu) để client cũ
  không vỡ; sau khi sửa proto chạy `make gen`, commit cả `common/gen`.
- Mã lỗi mới thêm vào `common/errorcode/error_codes.json` + `errorcode.go`, rồi
  `make error-codes` để đồng bộ sang FE (`make check-error-codes` phải pass).
- Rule kiến trúc trong `quality.json` (service không import repository/dependency,
  cmd dùng fx, không tự bắt signal…) — `make quality-all` phải pass.
- Mỗi thay đổi repository Postgres phải sửa song song repository in-memory
  (`task_repository.go`) và có test.
- **Số migration** gợi ý: 000012 (search), 000013 (reminders), 000014 (recurrence).
  Nếu merge khác thứ tự thì đổi số lúc rebase — migration đã merge không đổi số.
- **Quy ước `due_at` hiện tại:** form task chỉ chọn ngày và gửi
  `new Date('YYYY-MM-DDT00:00:00')` → `due_at` = **0h giờ địa phương** của ngày đó
  ([TaskFormModal.tsx:94](../../../frontend/src/components/task/TaskFormModal.tsx)).
  Không có cờ "có giờ hay không". Mọi phép so "hôm nay / quá hạn" phải dựa trên
  **mốc đầu ngày** theo múi giờ người dùng, không so với `now()`.
- Múi giờ: FE dùng múi giờ trình duyệt (`Intl.DateTimeFormat().resolvedOptions().timeZone`)
  để tính mốc ngày. `identity.profiles.timezone` (mặc định `Asia/Ho_Chi_Minh`)
  hiện chưa được FE dùng — không bắt buộc dùng trong roadmap này.

## 5. Ngoài phạm vi toàn roadmap

- Tag/nhãn, checklist trong task, bình luận, thùng rác, export/import, PWA/Web Push,
  phím tắt ngoài ⌘K — xem báo cáo đánh giá, làm ở roadmap sau.
- Lịch biểu / sự kiện lặp lại, tìm kiếm trong schedule/event/backlog.

## 6. Tồn đọng phát hiện khi viết spec (không thuộc roadmap, cần biết)

- **Gateway task không xác thực**: `profile_id` do client gửi lên (query/body),
  gateway `task` không kiểm tra session. `ListTasks` với `profile_id` rỗng trả
  task của **mọi** profile (`$1::uuid IS NULL`). Các API mới trong roadmap bắt
  buộc `profile_id` để không làm lỗ hổng này rộng thêm, nhưng lỗ hổng gốc cần một
  nhánh riêng (đọc session ở gateway như `identity`/`notification` đang làm).

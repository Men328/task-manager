# Màn Hôm nay / Sắp tới / Quá hạn (agenda)

Ngày: 2026-10-31 · Nhánh: `feature/agenda-view` · Trạng thái: chờ duyệt
Phụ thuộc: `feature/task-list-filters`
Roadmap: [personal-essentials-roadmap](2026-10-31-personal-essentials-roadmap.md)

## 1. Ý định

Người dùng mở app để biết **hôm nay phải làm gì**. Hiện chỉ có Board theo từng dự
án. Màn mới gom task có hạn của **mọi dự án** theo mốc thời gian, nhìn là biết
task nào thuộc dự án nào.

## 2. Mục tiêu / Không phải mục tiêu / Phản mục tiêu

**Mục tiêu**
- Route `/today`, mục nav "Hôm nay" đứng **đầu** nhóm Planning.
- 3 tab: **Hôm nay** · **Sắp tới** (7 ngày tới) · **Quá hạn**.
- Gộp mọi dự án chưa lưu trữ; phân biệt dự án rõ ràng; lọc theo dự án.
- Đánh dấu hoàn thành ngay trên danh sách.

**Không phải mục tiêu**
- Kéo-thả sắp xếp, đổi hạn bằng kéo-thả. Hiện schedule/event trong agenda.
- Đổi trang mặc định `/` (Board vẫn là trang chủ). *Có thể đổi sau khi dùng thử.*

**Phản mục tiêu**
- Task hạn hôm nay bị xếp vào "Quá hạn" từ 0h01 (do `due_at` = 0h địa phương).
- Danh sách nhiều dự án mà không phân biệt được task của dự án nào.

## 3. Thiết kế

### 3.1 Mốc thời gian (tính ở FE, múi giờ trình duyệt)

`today0` = 0h hôm nay, `tomorrow0` = `today0 + 1 ngày`, `week0` = `today0 + 8 ngày`.

| Tab | Query `ListTasks` (luôn kèm `profile_id`, `open_only=true`, `include_archived=false`, `sort_field=DUE_AT`) |
|---|---|
| Hôm nay | `due_from=today0`, `due_to=tomorrow0`; phía trên có khối **"Quá hạn (n)"** thu gọn được, dữ liệu như tab Quá hạn |
| Sắp tới | `due_from=tomorrow0`, `due_to=week0`; nhóm theo ngày ("Ngày mai", "Thứ Hai 3/11"…) |
| Quá hạn | `due_to=today0`, `sort_desc=false` (cũ nhất trước) |

Không truyền `project_id` (gộp). Không `root_only` → task con có hạn cũng hiện.
Dùng `listAllTasks` (đi hết trang, `page_size=200`).

### 3.2 Dòng task (phân biệt dự án)

```
▌ ☐  Nộp báo cáo tuần     [● Đồ án]  [Cao]   Hôm nay
▌    ↳ trong: Chuẩn bị bảo vệ
```

- `▌` dải màu 4px = `project.color` (rỗng → màu mặc định của UI).
- Chip dự án: chấm màu + tên, cắt `…` khi dài; tooltip tên đầy đủ.
- Badge ưu tiên dùng `PriorityBadge` có sẵn; ẩn khi MEDIUM để đỡ rối
  (*lựa chọn của Claude*).
- Cột thời gian: Quá hạn → "Trễ 3 ngày" (màu đỏ); Hôm nay → giờ nếu `due_at` khác 0h,
  không thì để trống; Sắp tới → không lặp lại vì đã nhóm theo ngày.
- Task con: dòng phụ "↳ trong: <tiêu đề task cha>" (lấy từ `parent_task_id`; tải
  task cha bằng `getTask` theo lô, cache trong state của trang).
- Task thuộc dự án đã lưu trữ: **ẩn** (FE lọc theo danh sách project), kèm dòng
  ghi chú nhỏ "n task trong dự án đã lưu trữ bị ẩn" nếu n > 0.

### 3.3 Lọc theo dự án

- Hàng chip đầu màn: "Tất cả" + mỗi dự án (màu + tên + số task trong tab hiện tại).
- Bấm chip = bật/tắt dự án đó; lưu lựa chọn vào `localStorage`
  (`agenda.hiddenProjects`, bọc try/catch).
- Lọc ở FE trên dữ liệu đã tải (không gọi lại API).

### 3.4 Hành động

- **Checkbox hoàn thành**: gọi `changeTaskStatus` sang status đầu tiên (theo
  `position`) có `category = done`. Lỗi transition (`TASK_TRANSITION_NOT_ALLOWED`)
  → toast dùng errorCatalog, giữ nguyên dòng. Thành công → dòng mờ đi rồi biến mất
  sau ~1,5 giây; toast có nút "Hoàn tác" (đổi lại status cũ — cũng đi qua
  transition; nếu luật transition không cho quay lại thì báo lỗi, task giữ "xong").
- **Bấm dòng**: mở `TaskDetailModal` ngay tại agenda **và** đặt dự án hiện tại của
  switcher = dự án của task (để khi người dùng sang Board là đúng chỗ).
- Sau khi sửa/xoá trong modal: tải lại tab hiện tại. Khi cửa sổ được focus lại
  hoặc có notice realtime liên quan task: tải lại.

### 3.5 Trạng thái trống / lỗi

- Hôm nay trống + không quá hạn: "Hôm nay không có việc nào đến hạn 🎉" và nút
  "Xem sắp tới".
- Lỗi tải: thông báo + nút thử lại.

### 3.6 Code

- `frontend/src/pages/AgendaPage/` (trang), `components/agenda/` (`AgendaList`,
  `AgendaRow`, `ProjectFilterChips`).
- State: slice mới `agenda` trong Redux (theo mẫu `board`), hoặc state cục bộ
  của trang — builder chọn; không trộn vào `boardSlice` vì board scope theo project.
- Thêm route trong `App.tsx`, mục nav trong `navigation.ts`, i18n vi/en.

## 4. Kiểm thử

- Tính mốc ngày: hàm thuần `agendaRanges(now, tz)` có test (khi FE có runner —
  nhánh quick-add thêm vitest; nếu nhánh này làm trước thì thêm vitest ở đây).
- Kiểm tay: task hạn hôm nay (0h) nằm ở Hôm nay chứ không ở Quá hạn; task con
  hiện breadcrumb; ẩn/hiện dự án; hoàn tác; dự án lưu trữ bị ẩn.

## 5. Để lại cho builder

- Bố cục cụ thể (Mantine `Tabs`, `Stack`…), icon nav.
- Redux slice hay state cục bộ.

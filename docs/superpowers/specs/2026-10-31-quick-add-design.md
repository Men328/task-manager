# Thêm task nhanh bằng một dòng

Ngày: 2026-10-31 · Nhánh: `feature/quick-add` · Trạng thái: chờ duyệt
Roadmap: [personal-essentials-roadmap](2026-10-31-personal-essentials-roadmap.md)

## 1. Ý định

Thêm task hiện phải mở `TaskFormModal` và điền từng ô. App task cá nhân thực tế
cho gõ một dòng như *"Nộp báo cáo mai 9h !cao #đồ-án"* rồi Enter. Ý tưởng nảy ra
thì ghi lại được trong 3 giây.

## 2. Mục tiêu / Không phải mục tiêu / Phản mục tiêu

**Mục tiêu**
- Ô `QuickAddInput` đặt ở đầu màn agenda và trên `BoardHeader`.
- Hiểu **ngày/giờ**, **ưu tiên**, **dự án** bằng tiếng Việt và tiếng Anh.
- Xem trước kết quả parse (chip) **trước khi** Enter; chip bấm vào để bỏ (giữ chữ
  đó trong tiêu đề).
- Backend không đổi: gọi `createTask` có sẵn.

**Không phải mục tiêu**
- Lặp lại ("mỗi ngày") — nhánh recurring-tasks mở rộng parser.
- Nhắc việc trong cú pháp. Gợi ý tự hoàn thành tên dự án ngoài `#`.

**Phản mục tiêu**
- Chữ bình thường trong tiêu đề bị hiểu nhầm thành ngày (VD "Họp thứ 2 bên đối
  tác" → OK hiểu là thứ Hai; nhưng "Đọc chương 2" không được thành ngày 2).
- Task tạo ra với ngày sai mà người dùng không nhận ra.

## 3. Cú pháp

| Loại | Tiếng Việt | Tiếng Anh | Ghi chú |
|---|---|---|---|
| Ngày tương đối | `hôm nay`, `nay`, `mai`, `ngày mai`, `mốt`, `ngày kia` | `today`, `tomorrow`, `tmr` | |
| Thứ | `thứ 2`…`thứ 7`, `t2`…`t7`, `chủ nhật`, `cn` | `mon`…`sun`, `monday`… | Thứ **sắp tới**; trùng hôm nay → hôm nay |
| Tuần | `tuần sau` (= thứ Hai tuần sau) | `next week` | `thứ 6 tuần sau` = thứ Sáu tuần sau |
| Ngày tuyệt đối | `31/10`, `31/10/2026`, `31-10` | (cùng dạng d/m) | Năm vắng → lần tới gần nhất (đã qua → năm sau) |
| Giờ | `9h`, `9h30`, `21h`, `9:30`, `9 giờ`, `9g` | `9am`, `9:30pm`, `21:00` | Giờ không ngày → hôm nay, đã qua → mai |
| Ưu tiên | `!gấp`, `!cao`, `!tb`, `!thấp`; `!1`(gấp)…`!4`(thấp) | `!urgent`, `!high`, `!medium`, `!low` | |
| Dự án | `#tên-dự-án` hoặc `#"Tên có dấu cách"` | | So không dấu, không hoa thường; `-`/`_` = dấu cách |

- **Chỉ ngày (không giờ)** → `due_at` = 0h giờ địa phương, khớp quy ước form hiện tại.
- Có giờ → `due_at` = đúng giờ đó.
- Chống hiểu nhầm: số trần (`2`, `chương 2`) **không** là ngày; chỉ nhận các mẫu
  trong bảng, ở **ranh giới từ**. Token ngày/giờ chỉ nhận lần xuất hiện **cuối cùng**
  của mỗi loại; các lần trước giữ trong tiêu đề.
- Thoát: bọc trong dấu nháy kép `"..."` (ngoài `#"..."`) thì giữ nguyên văn.
- `#` không khớp dự án nào → không gán, giữ nguyên chữ, chip cảnh báo "Không có dự
  án «x»". Khớp nhiều (tiền tố) → chọn khớp chính xác; nếu không có → chip cảnh báo
  "Nhiều dự án khớp" + không gán.
- Không có `#` → dự án hiện tại của switcher (ở agenda: dự án mặc định).
- Không có `!` → MEDIUM. Status → default (backend tự chọn khi `status_id` rỗng).
- Tiêu đề = chuỗi còn lại sau khi bỏ token, gộp khoảng trắng; rỗng → không cho tạo.

## 4. Thiết kế

- `frontend/src/lib/quickAdd/parse.ts`: hàm **thuần**
  `parseQuickAdd(input, { now, timeZone, projects, locale }) → { title, dueAt?, priority?, projectId?, tokens[], warnings[] }`;
  `tokens` có vị trí `[start, end)` để tô màu trong input và cho bấm bỏ.
- `components/task/QuickAddInput.tsx`: input + hàng chip xem trước; Enter tạo,
  Shift+Enter / nút "Chi tiết" mở `TaskFormModal` điền sẵn kết quả parse; sau khi
  tạo: xoá ô, toast "Đã thêm vào <dự án>" có nút "Mở".
- Thêm **vitest** (devDependency) + script `npm test`; test parser bằng bảng
  case cố định `now`/`timeZone` (`Asia/Ho_Chi_Minh`). *Lựa chọn của Claude*: FE hiện
  chưa có test runner, parser là chỗ dễ sai nhất của roadmap nên cần test.
- Không thêm thư viện parse ngày (chrono-node không hỗ trợ tiếng Việt); dùng `Date`
  + `Intl` có sẵn.

## 5. Kiểm thử

- ≥ 40 case parser: từng mẫu bảng §3, kết hợp, chống nhầm ("Đọc chương 2",
  "Mua 2 hộp sữa", "Gọi lúc 9h" khi đã 10h → mai), cuối năm (31/12 → 1/1),
  thứ trùng hôm nay, dự án có dấu/không dấu, `#` không khớp.
- Kiểm tay: chip xem trước khớp kết quả tạo; bỏ chip; Shift+Enter.

## 6. Để lại cho builder

- Kiểu tô màu token trong input (overlay hay chỉ chip bên dưới).

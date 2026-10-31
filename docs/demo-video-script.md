# Kịch bản video demo — "Hộp thư tự biết việc gì ra việc nấy"

Tài liệu này là **kịch bản nói** (lời thoại + thao tác + thời lượng) để demo Task Manager
trước giáo viên, tập trung vào tính năng cốt lõi: **nhận email → agent phân tích → tự tạo
công việc / lịch / sự kiện / backlog**.

- **Thời lượng bản chính:** ~5 phút (mục 2, timecode cộng lại 5:05).
- **Bản ngắn 60–90 giây:** xem [mục 5](#5-bản-ngắn-60–90-giây).
- **Phân đoạn kỹ thuật thay thế:** xem [mục 6](#6-phân-đoạn-kỹ-thuật-tuỳ-chọn) — **nên dùng khi demo trước giáo viên.**
- **Tông giọng:** trình bày, rõ ràng, không "bán hàng" quá đà. Điểm mạnh lớn nhất là *không email nào bị mất*.
- **Tỉ lệ khung hình:** 16:9, quay 1080p trở lên. Zoom trình duyệt ~110% cho dễ đọc.

### Xưng hô (đọc kỹ trước khi quay)

- Người trình bày xưng **"em"**; gọi giáo viên là **"thầy/cô"**.
- Trong bản viết dùng "thầy/cô" cho trung tính. **Khi nói trực tiếp, đọc thành "thầy" hoặc "cô"**
  tuỳ người ngồi trước mặt. Nếu muốn cố định, find-replace toàn bộ `thầy/cô` thành `thầy` hoặc `cô`.
- Khi nói về người sử dụng ứng dụng (không phải em, không phải giáo viên), gọi là **"người dùng"**.
- Nói về sản phẩm thì dùng **"hệ thống"** hoặc **"agent"**, tránh nhập nhằng với "em".

> **Một câu định vị dùng xuyên suốt:** "Điểm khác biệt không phải là thêm một hộp thư nữa —
> mà là người dùng không còn phải làm người phân loại. Có một agent làm việc đó thay họ."

---

## 1. Chuẩn bị trước khi bấm ghi (checklist)

### 1.1. Hệ thống

- [ ] Chạy full stack (Docker compose hoặc `bash scripts/dev.sh`) và đảm bảo web mở được.
- [ ] `mail-provider` có đủ: `DEEPSEEK_API_KEY`, `MAIL_PUBSUB_TOPIC`, `MAIL_PUBSUB_AUDIENCE`,
      `S3_ENDPOINT`/RustFS. Nếu thiếu một trong số này, luồng email sẽ không chạy hoặc
      rơi hết vào backlog — **demo sẽ hỏng**.
- [ ] Service `notification` + soketi đang chạy (để chuông thông báo nảy realtime).
- [ ] (Tuỳ chọn) `node scripts/seed-demo.mjs` để có sẵn dữ liệu nền cho board/lịch/sự kiện/báo cáo.
- [ ] **Đặt ngôn ngữ UI = Tiếng Việt** (nút ngôn ngữ ở góc phải topbar).
- [ ] Nếu muốn cột Kanban hiện tiếng Việt ("Cần làm / Đang làm / Chờ duyệt / Hoàn thành"),
      sửa tên status ở **Trạng thái & vòng đời** trước khi demo. Bản seed mặc định đang là
      `To Do / On Progress / In Review / Completed` — **đừng đọc sai tên cột trên hình**.

### 1.2. Tài khoản & email mẫu

- [ ] Đăng nhập sẵn tài khoản demo, **đã tích "Cho phép đọc Gmail"** từ trước (bước này chỉ quay
      lại một lần ở Cảnh 2; nếu tài khoản đã kết nối rồi thì thao tác sẽ không lặp lại được).
- [ ] Chuẩn bị **một tài khoản Gmail khác** (điện thoại hoặc cửa sổ ẩn danh) và **soạn nháp sẵn 4 email**
      dưới đây, lúc demo chỉ cần bấm Gửi. Đừng gõ nội dung trước mặt giáo viên.
- [ ] Canh **thời gian tương đối** khi soạn: "9h sáng mai", "trước thứ Sáu"… để lịch/hạn rơi vào
      ngày gần ngày demo, nhìn trên UI mới tự nhiên.

### 1.3. Bốn email mẫu (dùng đúng thứ tự này)

| # | Tiêu đề email | Nội dung | Kết quả mong đợi |
|---|---|---|---|
| 1 | `Nhờ nộp báo cáo quý 3` | "Chào anh, nhờ anh nộp báo cáo quý 3 trước thứ Sáu tuần này giúp em. Sếp cần gấp nên anh ưu tiên nhé. Cảm ơn anh." | **Công việc** trong cột *Cần làm*, tiêu đề gọn, mô tả tóm tắt, mức ưu tiên cao, hạn = thứ Sáu |
| 2 | `Họp kickoff dự án CRM` | "Mời anh 9 giờ sáng mai họp kickoff dự án CRM tại phòng họp tầng 3, khoảng một tiếng." | **Lịch** đúng 9h sáng mai, địa điểm "phòng họp tầng 3" |
| 3 | `Thư mời tham dự AI Summit 2025` | "Trân trọng kính mời anh tham dự hội thảo AI Summit 2025 vào ngày 20/11 lúc 8h30 tại Trung tâm Hội nghị Quốc gia." | **Sự kiện** có thời gian + địa điểm; mở chi tiết thấy trường *Nguồn email* |
| 4 | `Bản tin nội bộ tuần này + ưu đãi 50%` | "Tổng hợp tin tức nội bộ tuần này và chương trình ưu đãi 50% cho khách hàng thân thiết…" | **Backlog** nhãn *Khác*, kèm người gửi, lý do, bản lưu email gốc |

> **Mẹo:** email #4 là "cú chốt" của phần demo — nó chứng minh hệ thống **không ném email đi**.
> Đừng bỏ cảnh này.

### 1.4. Khung hình

- [ ] Ẩn bookmark bar, đóng tab thừa, tắt thông báo hệ thống (Slack/Zalo/Telegram) để không có
      notification lạ nảy lên.
- [ ] Mở sẵn 2 cửa sổ cạnh nhau: trái = Gmail (tài khoản gửi), phải = app. Hoặc dùng 2 màn hình.
- [ ] Che/mờ mọi email, tên người thật không muốn lộ. Dùng tài khoản demo cho app.

### 1.5. Rủi ro cần biết trước

- Gmail → Pub/Sub → xử lý thường mất **5–30 giây**. **Luôn quay dư**, và dùng
  [câu thoại chờ ở mục 4](#4-câu-thoại-chờ-lấp-khoảng-trễ) để lấp khoảng trễ, hậu kỳ tua nhanh.
- Gửi cùng lúc 4 email có thể khiến chúng về không đúng thứ tự. **Gửi lần lượt, chờ cái trước
  hiện lên rồi mới gửi cái sau** — vừa dễ quay vừa dễ kể.
- Nếu agent "không chắc", email vào **Backlog** — hãy nói đó là **chủ đích thiết kế**, không phải lỗi:
  "khi không chắc, hệ thống để người dùng quyết, chứ không tự đoán bừa và làm mất email."

---

## 2. Kịch bản chính (~5:05)

### Cảnh 0 — Mở đầu & đặt vấn đề (0:00 – 0:20)

**Hình:** Cận cảnh hộp thư Gmail cuộn nhanh, unread đỏ chi chít → cắt sang màn hình trống.

**Lời thoại:**
> "Em xin phép bắt đầu ạ.
> Mỗi ngày, một người dùng bình thường nhận hàng chục email: hoá đơn, deadline, thư mời họp,
> thư mời hội thảo, rồi cả newsletter, quảng cáo — tất cả nằm lẫn trong một hộp thư.
> Họ phải tự đọc, tự nhớ, tự phân loại — và thường là… quên.
> Vấn đề không nằm ở sự lười biếng, mà ở chỗ con người đang làm một việc mà máy móc làm tốt hơn:
> phân loại. Đó là lý do em xây dựng Task Manager, và hôm nay em xin trình bày thầy/cô
> tính năng cốt lõi của hệ thống."

**Text overlay:** `47 email mới · 0 việc được ghi lại`

### Cảnh 1 — Tổng quan trong 30 giây (0:20 – 0:55)

**Thao tác:** Lia chuột qua sidebar theo nhóm, dừng ở mỗi mục ~4–5 giây:
**Báo cáo** (Tổng quan) → **Công việc của tôi**, **Lịch**, **Sự kiện**, **Backlog** (Kế hoạch) →
**Trạng thái & vòng đời** (Cấu hình). Rồi mở nhanh board Kanban.

**Lời thoại:**
> "Task Manager là một không gian làm việc cá nhân, gồm bốn phần.
> **Công việc** — bảng Kanban với các cột trạng thái.
> **Lịch** — nơi các cuộc hẹn có giờ.
> **Sự kiện** — hội thảo, tiệc, thư mời.
> Và **Backlog** — chỗ chứa những email chưa phân loại được.
> Cộng thêm **Báo cáo** để nhìn lại, và **Trạng thái & vòng đời** để tuỳ biến quy trình.
> Bốn nơi đầu chính là bốn ngả mà một email có thể rơi vào. Và việc quyết định nó rơi vào đâu —
> là phần em xin trình bày ngay sau đây."

**Text overlay:** `Công việc · Lịch · Sự kiện · Backlog` → `4 ngả của một email`

### Cảnh 2 — Kết nối Gmail bằng một cái tích (0:55 – 1:25)

**Thao tác:** Màn đăng nhập → trỏ chuột vào ô tích → bấm **Đăng nhập với Google** → qua màn hình
cấp quyền của Google (che email) → quay lại app.

**Lời thoại:**
> "Đăng nhập bằng Google. Chỉ một ô tích duy nhất:
> **'Cho phép đọc Gmail để tự động tạo công việc'**.
> Đây là điểm em muốn nói rõ ngay: hệ thống **chỉ đọc** mail để tạo việc.
> Nó không gửi thư thay người dùng, và không xoá thư của họ.
> Đây là quyền chỉ-đọc, và người dùng có thể ngắt kết nối bất cứ lúc nào.
> Sau khi kết nối, người dùng không cần mở app này thường xuyên nữa.
> Cứ có email mới là hệ thống tự chạy."

**Text overlay:** `chỉ đọc — không gửi, không xoá`

### Cảnh 3 — Trạng thái "trước": bàn làm việc đang trống (1:25 – 1:55)

**Thao tác:** Vào **Công việc của tôi**, lia qua 4 cột Kanban. Mở một card bất kỳ, chỉ vào tiến độ
task con, rồi kéo thả một card sang cột khác để cho thấy board sống. Sau đó đóng lại.

**Lời thoại:**
> "Đây là bảng công việc. Các cột là trạng thái: Cần làm, Đang làm, Chờ duyệt, Hoàn thành.
> Có thể kéo thả để chuyển trạng thái, mở một thẻ để xem mô tả, hạn, tiến độ các bước con.
> Nhưng trong phần demo này, em sẽ **không nhập tay một dòng nào**.
> Em chỉ… gửi email. Và thầy/cô sẽ thấy nó tự chảy vào đây."

### Cảnh 4 — Email #1 → tự thành CÔNG VIỆC (1:55 – 2:50)

**Thao tác:**
1. Cắt sang Gmail, mở nháp #1, bấm **Gửi**.
2. Cắt về app, để nguyên màn hình. Sau vài giây, **chuông thông báo nảy số đỏ**.
3. Mở chuông → thấy thông báo mang đúng tiêu đề công việc → bấm vào thông báo → app nhảy thẳng
   vào **modal chi tiết công việc**.
4. Trỏ lần lượt: tiêu đề, mô tả, mức ưu tiên, hạn.
5. Đóng modal, chỉ vào card mới nằm đúng cột *Cần làm*.

**Lời thoại (khi đang gửi):**
> "Em vừa gửi một email nhờ nộp báo cáo quý 3 trước thứ Sáu. Một email rất đời thường."

**Lời thoại (khi chờ — chèn câu thoại chờ ở mục 4 nếu lâu):**
> "Trong lúc chờ, thầy/cô để ý: em không mở app, không bấm nút nào cả. Hệ thống tự chạy."

**Lời thoại (khi thông báo nảy):**
> "Và… trong vài giây, một thông báo bật lên. Bấm vào là nhảy thẳng tới công việc vừa được tạo.
> Tiêu đề đã được viết gọn. Mô tả đã được agent tóm tắt lại thành nội dung rõ ràng.
> Mức ưu tiên thì suy ra từ câu 'sếp cần gấp'. Còn hạn — lấy từ chính chữ 'trước thứ Sáu'.
> Em chưa hề gõ một chữ nào. Đây là toàn bộ giá trị: email biến thành việc, kèm ngữ cảnh,
> thay vì nằm chết trong hộp thư."

### Cảnh 5 — Email #2 → tự thành LỊCH (2:50 – 3:30)

**Thao tác:** Gửi nháp #2 → chờ chuông → bấm thông báo (app nhảy vào trang **Lịch**) → chỉ vào
khung giờ 9h sáng mai, tiêu đề và địa điểm "phòng họp tầng 3". Bấm vào sự kiện để mở form chi tiết.

**Lời thoại:**
> "Nhưng đây mới là phần khó. Không phải email nào cũng là 'việc cần làm'.
> Cùng là một email, nhưng ý định bên trong khác nhau.
> Email này là lời hẹn họp — và nó được đưa sang **Lịch**, không phải vào bảng công việc.
> Đúng ngày, đúng giờ, đúng địa điểm: chín giờ sáng mai, phòng họp tầng ba.
> Nếu làm thủ công, người dùng sẽ phải đọc, mở lịch, chọn ngày, chọn giờ, gõ địa điểm.
> Còn ở đây, agent phân biệt được đâu là việc, đâu là lịch hẹn.
> Đây là phần em tâm đắc nhất về mặt xử lý."

### Cảnh 6 — Email #3 → tự thành SỰ KIỆN, truy vết được nguồn (3:30 – 4:05)

**Thao tác:** Gửi nháp #3 → bấm thông báo → trang **Sự kiện** → mở chi tiết sự kiện → trỏ vào
dòng **Nguồn email**.

**Lời thoại:**
> "Còn thư mời hội thảo, tiệc, sự kiện — loại không phải việc, cũng chẳng phải cuộc hẹn cá nhân —
> thì vào mục **Sự kiện**.
> Mở chi tiết ra, ta thấy cả **'Nguồn email'**: tức là truy vết được sự kiện này sinh ra từ
> đúng email nào. Không có chuyện 'tự nhiên thấy trên lịch mà không biết từ đâu ra'.
> Với em, khả năng truy vết quan trọng ngang với khả năng tự động — vì người dùng cần tin được
> cái máy đang làm việc thay mình."

### Cảnh 7 — Email #4 → vào BACKLOG, không email nào bị mất (4:05 – 4:40)

**Thao tác:** Gửi nháp #4 → bấm thông báo → trang **Backlog** → trỏ vào các cột **Người gửi**,
**Phân loại**, **Lý do** → mở chi tiết, trỏ vào **Bản lưu object storage**. Đổi trạng thái mục này
sang *Đã xử lý* để cho thấy đây là hàng đợi có thể dọn.

**Lời thoại:**
> "Và đây là phần em tâm đắc nhất về mặt thiết kế.
> Email không phải việc, không phải lịch — newsletter, quảng cáo — hệ thống **không ném đi**.
> Nó đưa vào **Backlog**, kèm người gửi, nhãn phân loại, lý do vì sao nó ở đây,
> và cả **bản lưu email gốc** trên object storage.
> Người dùng có thể đánh dấu đã xử lý, hoặc lưu trữ.
> Nghĩa là: phân loại đúng thì được việc. Phân loại chưa chắc thì vẫn còn nguyên email,
> chỉ là nó đang chờ người dùng quyết. **Không có email nào biến mất.**"

### Cảnh 8 — Kết: từ hộp thư hỗn loạn thành hệ thống (4:40 – 5:05)

**Thao tác:** Mở **Báo cáo**, lia qua thẻ tổng quan task + biểu đồ xu hướng + thẻ **Backlog theo nhóm**.
Rồi quay lại board đã có card mới nằm đúng cột. Kết ở logo.

**Lời thoại:**
> "Và vì mọi thứ đều được gắn nhãn, người dùng có báo cáo: bao nhiêu việc, bao nhiêu lịch, bao nhiêu
> sự kiện, bao nhiêu email phải xử lý tay.
> Bốn email lúc nãy đã thành bốn mục nằm đúng chỗ, và em không phân loại cái nào.
> Điểm khác biệt không nằm ở chỗ có thêm một hộp thư nữa.
> Nó nằm ở chỗ: **người dùng không còn phải làm người phân loại — có một agent làm việc đó thay họ.**
> Phần trình bày của em đến đây là hết. Em xin cảm ơn thầy/cô đã theo dõi ạ."

**Text overlay cuối:** `Task Manager — hộp thư tự biết việc gì ra việc nấy.`

---

## 3. B-roll & text overlay gợi ý

| Thời điểm | B-roll | Text overlay |
|---|---|---|
| 0:00 | Cuộn Gmail unread | `47 email mới · 0 việc được ghi lại` |
| 0:20 | Lia sidebar 4 mục | `Công việc · Lịch · Sự kiện · Backlog` |
| 0:55 | Ô tích "Cho phép đọc Gmail" | `chỉ đọc — không gửi, không xoá` |
| 1:25 | Kéo thả card Kanban | `board sống` |
| 1:55 | Sơ đồ luồng: `Gmail → Agent → Công việc / Lịch / Sự kiện / Backlog` | `1 email vào, 4 ngả ra` |
| 2:30 | Đồng hồ đếm vài giây | `~5 giây` |
| 3:30 | Trường "Nguồn email" | `truy vết được` |
| 4:05 | Backlog + bản lưu | `không email nào bị mất` |
| 5:00 | Logo | Tagline |

**Sơ đồ luồng (dựng 1 slide, chiếu nhanh ở Cảnh 5 hoặc 8):**
```
Email mới trong Gmail
        │
        ▼
   Agent phân tích  ──►  Công việc   (việc cần làm, có hạn)
        │          ──►  Lịch        (có thời gian)
        │          ──►  Sự kiện     (thư mời, hội thảo, tiệc)
        └──────────►  Backlog     (không chắc / không phải việc)  →  vẫn lưu email gốc
```

---

## 4. Câu thoại chờ (lấp khoảng trễ)

Đọc khi đang chờ email về — chọn 1, đừng đọc hết:

1. **Giải thích phía sau (bản thân thiện):**
   > "Trong lúc chờ, em xin nói rõ chuyện gì đang xảy ra. Gmail báo có thư mới, hệ thống đọc email gốc,
   > đưa cho agent phân tích, rồi mới gọi sang đúng service để tạo. Ngay cả bản email gốc cũng được
   > lưu lại — nên nếu cần, người dùng luôn tra được email nào đã sinh ra mục này."

2. **Bản kỹ thuật (nên dùng khi demo trước giáo viên):**
   > "Luồng thật là: Gmail đẩy thông báo qua Pub/Sub, hệ thống đọc theo history id đã lưu —
   > nên thông báo trùng hay đến lệch thứ tự cũng không tạo trùng. Sau đó agent phân loại thành
   > đúng một nhãn: task, schedule, event, hoặc other. Nếu mô hình lỗi, có rule từ khoá dự phòng."

3. **Bản nêu bật độ tin cậy:**
   > "Và nếu agent không đủ dữ kiện để chắc chắn — nó **không đoán bừa**. Nó đẩy vào Backlog để
   > người dùng quyết. Thà phải xem lại một email, còn hơn mất một việc quan trọng."

4. **Bản nói về quyền riêng tư:**
   > "Một điều nữa: hệ thống chỉ có quyền đọc. Nó không gửi, không xoá, không sửa thư.
   > Và người dùng có thể ngắt kết nối Gmail bất cứ lúc nào."

> **Hậu kỳ:** nếu khoảng chờ dài, cắt và chèn **tua nhanh 4x** kèm text `đang xử lý…`, đừng để
> khán giả nhìn màn hình đứng im quá 5 giây.

---

## 5. Bản ngắn 60–90 giây

Dùng cho social/giới thiệu nhanh. Bỏ Cảnh 1 và 3, gộp lại.

| Thời gian | Hình | Lời thoại |
|---|---|---|
| 0:00–0:08 | Cuộn Gmail unread | "Hoá đơn, deadline, thư mời họp, newsletter — tất cả lẫn trong một hộp thư. Người dùng không phân loại nổi." |
| 0:08–0:18 | Tích "Cho phép đọc Gmail" → board | "Task Manager đọc Gmail. Chỉ đọc, không gửi, không xoá. Và từ đây, người dùng không nhập tay gì nữa." |
| 0:18–0:38 | Gửi email #1 → chuông → task | "Email nhờ nộp báo cáo → vài giây sau thành công việc, đúng mức ưu tiên, đúng hạn." |
| 0:38–0:52 | Gửi email #2, #3 → Lịch, Sự kiện | "Hẹn họp → thành lịch. Thư mời hội thảo → thành sự kiện, có cả nguồn email." |
| 0:52–1:10 | Gửi email #4 → Backlog | "Còn newsletter thì vào Backlog — kèm lý do và bản lưu email gốc. Không email nào bị mất." |
| 1:10–1:20 | Board + logo | "Một agent đọc, phân loại và sắp việc thay người dùng. Task Manager." |

---

## 6. Phân đoạn kỹ thuật (tuỳ chọn)

Khi demo trước giáo viên, **nên thay Cảnh 1 bằng phân đoạn ~40 giây này** để thể hiện chiều sâu
kỹ thuật (vẫn giữ tổng thời lượng ~5 phút):

**Hình:** Slide kiến trúc đơn giản, rồi cắt về app.

**Lời thoại:**
> "Về mặt kỹ thuật, có mấy điểm em xin nói rõ.
> **Một:** khi người dùng tích 'cho phép đọc Gmail', hệ thống xin scope chỉ-đọc và một refresh token,
> nên nó chạy nền kể cả khi người dùng không mở app.
> **Hai:** Gmail đẩy thông báo qua Pub/Sub. Hệ thống không tin history id trong thông báo —
> nó đọc checkpoint đã lưu rồi lấy đủ các thư mới. Nhờ vậy thông báo trùng hoặc đến lệch thứ tự
> cũng không tạo ra task trùng.
> **Ba:** agent trả về đúng một nhãn trong bốn: task, schedule, event, other. Nếu gọi mô hình lỗi,
> có rule từ khoá dự phòng, nên luồng không chết.
> **Bốn:** email gốc và tệp đính kèm được lưu lên object storage trước khi phân loại — nên dù
> phân loại thế nào, bản gốc vẫn còn.
> Và cuối cùng, mỗi kết quả tạo ra đều bắn một thông báo realtime về cho người dùng."

**Text overlay:** `OAuth chỉ-đọc · checkpoint chống trùng · fallback rule · lưu bản gốc trước khi phân loại`

---

## 7. Những gì KHÔNG nên nói (tránh nói quá)

- ❌ "AI hiểu 100% email" → ✅ "Khi không chắc, nó đưa vào Backlog để người dùng xử lý."
- ❌ "Tự động trả lời email" → ✅ "Chỉ đọc để tạo việc. Không gửi, không xoá."
- ❌ "Thay thế Gmail" → ✅ "Gmail vẫn là hộp thư. Task Manager là lớp phân loại ngồi trên nó."
- ❌ "Tức thì" → ✅ "Trong vài giây." (đúng với độ trễ Pub/Sub thực tế)

---

## 8. Phương án B khi demo

- **Email không về trong lúc demo:** quay nhiều take, để camera chạy, gửi email, nghỉ 1 phút rồi
  quay lại cảnh chuông nảy. Hoặc gửi trước 4 email, quay phần "hậu quả" trước, quay phần "gửi"
  sau rồi dựng ngược thứ tự.
- **Chuông không nảy:** kiểm tra service `notification` + soketi, rồi tải lại trang; thông báo
  vẫn nằm trong danh sách chuông dù không nảy realtime.
- **Agent phân loại sai một email:** **giữ lại và nói thật** — "đây là lúc nó không chắc, và nó
  đưa vào Backlog" — trung thực sẽ thuyết phục hơn một màn demo hoàn hảo.
- **Giáo viên hỏi thêm:** chuẩn bị sẵn 2 câu trả lời — (1) chi phí/độ trễ gọi mô hình và cách
  fallback; (2) quyền riêng tư: chỉ-đọc, có thể ngắt kết nối, và bản gốc vẫn nằm trong storage
  của hệ thống.

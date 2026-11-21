<!-- Sinh từ _tools/evidence/ket-qua-test.json, nfr-ket-qua.json và kết quả go test / vitest / make quality-all ngày 18/11/2026. Chạy lại api_tests.py rồi sinh lại để cập nhật. -->
# CHƯƠNG 5: TRIỂN KHAI VÀ KIỂM THỬ HỆ THỐNG

## 5.1. Triển khai hệ thống

### 5.1.1. Môi trường phát triển và triển khai

Hệ thống được phát triển và chạy thử nghiệm trên môi trường được mô tả trong Bảng 5.1.

[BANG: Bảng 5.1: Môi trường phát triển và triển khai]
| STT | Thành phần | Phiên bản | Ghi chú |
|---|---|---|---|
| 1 | Hệ điều hành | Ubuntu 24.04.4 LTS trên WSL2 (Linux 6.18) | Máy phát triển, chạy toàn bộ hệ thống |
| 2 | Go | 1.26.4 (module khai báo go 1.25) | Biên dịch 13 dịch vụ |
| 3 | Node.js | 22.23.2 | Biên dịch giao diện, chạy Vitest |
| 4 | Docker / Docker Compose | 29.7.2 / 5.5.0 | Đóng gói và khởi chạy hệ thống |
| 5 | PostgreSQL | 17 (image postgres:17-alpine) | Cơ sở dữ liệu |
| 6 | Apache Kafka | 3.9.1 | Hàng đợi thông điệp |
| 7 | Soketi | 1.6 | Máy chủ WebSocket |
| 8 | RustFS | bản mới nhất | Kho đối tượng S3 |
| 9 | buf | 1.73.0 | Sinh mã từ tệp proto |

### 5.1.2. Tổ chức mã nguồn

Mã nguồn được tổ chức dạng *monorepo*:

- `common/`: tệp proto, mã sinh tự động và bộ mã lỗi chuẩn dùng chung.
- `service/`: 13 dịch vụ, mỗi dịch vụ một Go module riêng với cấu trúc thư mục thống nhất.
- `frontend/`: ứng dụng React, Dockerfile và cấu hình nginx.
- `deployments/`: tệp Docker Compose và 18 migration SQL.
- `tools/quality`: công cụ kiểm tra quy tắc kiến trúc; `scripts/`: sinh mã, cài công cụ, kiểm thử nhanh.
- `Makefile`: các lệnh chuẩn như `make gen`, `make up`, `make test`, `make quality-all`.

Quy trình phát triển: mỗi tính năng được làm trên một nhánh riêng tách từ nhánh chính, mở pull request để rà soát rồi mới hợp nhất.

### 5.1.3. Triển khai bằng Docker Compose

Toàn bộ hệ thống được khởi động bằng một lệnh `make up`. Docker Compose lần lượt khởi động PostgreSQL, chạy container `migrate` để cập nhật lược đồ, khởi động RustFS, Soketi, Kafka (container `kafka-init` tạo sẵn các topic), sau đó là 12 dịch vụ (mỗi dịch vụ 2 container), worker và giao diện. Mỗi container có kiểm tra sức khoẻ (health check) qua đường dẫn `/healthz`; dịch vụ phụ thuộc chỉ khởi động khi dịch vụ nó cần đã sẵn sàng. Khi triển khai thử nghiệm, hệ thống gồm 31 container đang chạy, trong đó dịch vụ worker chạy 2 bản.

Số bản worker thay đổi được bằng tham số `--scale worker=N`; số worker xử lý song song tối đa bằng số partition của topic (mặc định 6).

### 5.1.4. Đưa hệ thống ra Internet

Chỉ container giao diện được mở cổng, và chỉ trên địa chỉ nội bộ `127.0.0.1:3000`. Để truy cập từ Internet và để Google Pub/Sub gửi thông báo đẩy về webhook, hệ thống bật thêm container `cloudflared` (lệnh `make up-tunnel`) tạo đường hầm từ máy chủ tới Cloudflare với tên miền HTTPS. Trường hợp không có địa chỉ công khai, dịch vụ xử lý thư có thể chuyển sang chế độ *pull* chủ động kéo thông báo từ Pub/Sub.

### 5.1.5. Kiểm soát chất lượng mã nguồn

Lệnh `make quality-all` chạy định dạng mã (`gofmt`), phân tích tĩnh (`go vet`), kiểm tra quy tắc kiến trúc khai báo trong `quality.json` và kiểm tra bộ mã lỗi của giao diện khớp với bản gốc. Kết quả chạy ngày 18/11/2026 được trình bày ở mục 5.3.

## 5.2. Kiểm thử chức năng

### 5.2.1. Phương pháp kiểm thử

Kiểm thử chức năng được thực hiện **trên hệ thống chạy thật** (Docker Compose, cổng `localhost:3000`), gọi API qua nginx đúng như giao diện. Tài khoản chính là `menc1lhp@gmail.com`; tài khoản `22010328@st.phenikaa-uni.edu.vn` (tài khoản thứ hai của cùng sinh viên) chỉ dùng cho các trường hợp cần hai người (kết bạn, nhắn tin, giao task qua chat). Các trường hợp kiểm thử được viết thành kịch bản tự động (`docs/bao-cao/_tools/evidence/api_tests.py`): mỗi trường hợp gửi dữ liệu đầu vào, so sánh phản hồi thực tế (mã HTTP, mã lỗi chuẩn, dữ liệu trả về) với kết quả mong đợi và ghi trạng thái. Kết quả: **49/49 trường hợp đạt**.

Riêng chức năng phân loại email được đánh giá trên **email thật** gửi tới hộp thư Gmail của tài khoản chính, vì luồng này phụ thuộc Gmail và Pub/Sub.

### 5.2.2. Kiểm thử chức năng xác thực

Bảng 5.2 trình bày 5 trường hợp kiểm thử nhóm xác thực; 5/5 trường hợp đạt.

[BANG: Bảng 5.2: Kết quả kiểm thử chức năng xác thực]
| STT | Chức năng | Dữ liệu đầu vào | Kết quả mong đợi | Kết quả thực tế | Trạng thái |
|---|---|---|---|---|---|
| 1 | Lấy hồ sơ bằng session token hợp lệ | Token của menc1lhp@gmail.com | Trả hồ sơ đúng email | HTTP 200, email menc1lhp@gmail.com | Pass |
| 2 | Gọi API khi chưa đăng nhập | Không gửi token | Từ chối, mã lỗi IDENTITY_AUTH_TOKEN_MISSING | HTTP 401, mã lỗi IDENTITY_AUTH_TOKEN_MISSING | Pass |
| 3 | Token bị sửa chữ ký | Token sai chữ ký | Từ chối, mã lỗi IDENTITY_AUTH_TOKEN_INVALID | HTTP 401, mã lỗi IDENTITY_AUTH_TOKEN_INVALID | Pass |
| 4 | Xin quyền nghe kênh thông báo của mình | private-noti-internal-<id của mình> | Trả chữ ký auth | HTTP 200, có auth: True | Pass |
| 5 | Xin quyền nghe kênh của người khác | private-noti-internal-<id người khác> | Từ chối, mã lỗi IDENTITY_AUTH_SOKETI_CHANNEL_FORBIDDEN | HTTP 403, mã lỗi IDENTITY_AUTH_SOKETI_CHANNEL_FORBIDDEN | Pass |

### 5.2.3. Kiểm thử chức năng dự án

Bảng 5.3 trình bày 4 trường hợp kiểm thử nhóm dự án; 4/4 trường hợp đạt.

[BANG: Bảng 5.3: Kết quả kiểm thử chức năng dự án]
| STT | Chức năng | Dữ liệu đầu vào | Kết quả mong đợi | Kết quả thực tế | Trạng thái |
|---|---|---|---|---|---|
| 1 | Tạo dự án mới | name = "Đồ án tốt nghiệp" | Tạo thành công | HTTP 200, id facb0b56… | Pass |
| 2 | Tạo dự án trùng tên (khác hoa thường) | name = "ĐỒ ÁN TỐT NGHIỆP" | Từ chối, mã lỗi PROJECT_NAME_DUPLICATE | HTTP 409, mã lỗi PROJECT_NAME_DUPLICATE | Pass |
| 3 | Lưu trữ dự án mặc định | Dự án "Chung" | Từ chối, mã lỗi PROJECT_DEFAULT_NOT_ARCHIVABLE | HTTP 400, mã lỗi PROJECT_DEFAULT_NOT_ARCHIVABLE | Pass |
| 4 | Tạo task trong dự án đã lưu trữ | project_id = dự án đã archive | Từ chối, mã lỗi PROJECT_ARCHIVED | HTTP 400, mã lỗi PROJECT_ARCHIVED | Pass |

### 5.2.4. Kiểm thử chức năng công việc

Bảng 5.4 trình bày 10 trường hợp kiểm thử nhóm công việc; 10/10 trường hợp đạt.

[BANG: Bảng 5.4: Kết quả kiểm thử chức năng công việc]
| STT | Chức năng | Dữ liệu đầu vào | Kết quả mong đợi | Kết quả thực tế | Trạng thái |
|---|---|---|---|---|---|
| 1 | Tạo task thiếu tiêu đề | title rỗng | Từ chối, mã lỗi TASK_TITLE_REQUIRED | HTTP 400, mã lỗi TASK_TITLE_REQUIRED | Pass |
| 2 | Tạo task hợp lệ | Tiêu đề, mô tả, ưu tiên cao, hạn +3 ngày | Tạo thành công, trạng thái mặc định To Do | HTTP 200, trạng thái To Do | Pass |
| 3 | Tạo task con | 3 task con của task "Viết Chương 4" | Tạo thành công, cùng dự án với cha | 3/3 task con, cùng dự án: True | Pass |
| 4 | Kéo task To Do sang Completed (không có luật) | To Do -> Completed | Từ chối, mã lỗi TASK_TRANSITION_NOT_ALLOWED | HTTP 400, mã lỗi TASK_TRANSITION_NOT_ALLOWED | Pass |
| 5 | Kéo task To Do sang On Progress (có luật) | To Do -> On Progress | Đổi trạng thái thành công | HTTP 200 | Pass |
| 6 | Hoàn thành task | On Progress -> In Review -> Completed | Ghi thời điểm hoàn thành (completedAt) | HTTP 200, completedAt có | Pass |
| 7 | Xoá task còn task con | Xoá task "Viết Chương 4" | Từ chối, mã lỗi TASK_HAS_SUBTASKS | HTTP 400, mã lỗi TASK_HAS_SUBTASKS | Pass |
| 8 | Hoàn thành task lặp lại hằng ngày | recurrence_rule = FREQ=DAILY, chuyển tới Completed | Tự sinh lần kế tiếp, hạn +1 ngày | HTTP 200, lần kế tiếp: có, hạn 2026-11-19 | Pass |
| 9 | Tìm kiếm task theo từ khoá | query = "kafka" | Trả các task có chữ Kafka | HTTP 200, 2 kết quả | Pass |
| 10 | Nhắc việc đúng giờ | remind_at = hiện tại + 5 giây | Nhận thông báo TASK_REMINDER | Thông báo: Nhắc việc: Nộp bản nháp báo cáo cho giảng viên | Pass |

### 5.2.5. Kiểm thử chức năng lịch và sự kiện

Bảng 5.5 trình bày 4 trường hợp kiểm thử nhóm lịch và sự kiện; 4/4 trường hợp đạt.

[BANG: Bảng 5.5: Kết quả kiểm thử chức năng lịch và sự kiện]
| STT | Chức năng | Dữ liệu đầu vào | Kết quả mong đợi | Kết quả thực tế | Trạng thái |
|---|---|---|---|---|---|
| 1 | Tạo lịch có giờ kết thúc trước giờ bắt đầu | end_at < start_at | Từ chối, mã lỗi CALENDAR_TIME_RANGE_INVALID | HTTP 400, mã lỗi CALENDAR_TIME_RANGE_INVALID | Pass |
| 2 | Tạo lịch hợp lệ và xem theo khoảng thời gian | Họp nhóm 9:00–10:30, lọc ±1 ngày | Lịch xuất hiện trong khoảng lọc | HTTP 200, có trong danh sách: True | Pass |
| 3 | Tạo sự kiện và xác nhận | Hội thảo, PLANNED -> CONFIRMED | Sự kiện chuyển sang đã xác nhận | HTTP 200/200, trạng thái EVENT_STATUS_CONFIRMED | Pass |
| 4 | Tạo sự kiện có khoảng thời gian sai | end_at < start_at | Từ chối, mã lỗi EVENT_TIME_RANGE_INVALID | HTTP 400, mã lỗi EVENT_TIME_RANGE_INVALID | Pass |

### 5.2.6. Kiểm thử chức năng email và backlog

Bảng 5.6 trình bày 5 trường hợp kiểm thử nhóm email và backlog; 5/5 trường hợp đạt.

[BANG: Bảng 5.6: Kết quả kiểm thử chức năng email và backlog]
| STT | Chức năng | Dữ liệu đầu vào | Kết quả mong đợi | Kết quả thực tế | Trạng thái |
|---|---|---|---|---|---|
| 1 | Email Gmail thật được phân loại tự động | Thư gửi tới menc1lhp@gmail.com (newsletter, yêu cầu công việc, lịch báo cáo) | Email được lưu thành bản ghi kèm người gửi, lý do | 4 email trong backlog, ví dụ: "TED newsletter: How to calm your anxiety…" (Newsletter quảng cáo nội dung, không cần) | Pass |
| 2 | Newsletter được phân loại là other | Email TED, Lumosity | category = other, vào backlog | 2 newsletter, category other | Pass |
| 3 | Email công việc không khớp dự án nào | Email yêu cầu xem tài liệu / lịch báo cáo, không nhắc tên dự án | Vào backlog với lý do no_project_matched | 2 email, category schedule, task | Pass |
| 4 | Đánh dấu backlog đã xử lý | status = TRIAGED | Cập nhật thành công | HTTP 200, BACKLOG_STATUS_TRIAGED | Pass |
| 5 | Thông báo khi email được xử lý | Các email ở trên | Có thông báo *_CREATED | 6 thông báo | Pass |

### 5.2.7. Kiểm thử chức năng tệp đính kèm

Bảng 5.7 trình bày 4 trường hợp kiểm thử nhóm tệp đính kèm; 4/4 trường hợp đạt.

[BANG: Bảng 5.7: Kết quả kiểm thử chức năng tệp đính kèm]
| STT | Chức năng | Dữ liệu đầu vào | Kết quả mong đợi | Kết quả thực tế | Trạng thái |
|---|---|---|---|---|---|
| 1 | Tải tệp lên task | de-cuong-do-an.txt, 2400 byte | Tải lên thành công | HTTP 200, kích thước 2400 byte | Pass |
| 2 | Tải tệp xuống | Tệp vừa tải lên | Nội dung giống hệt bản gốc | HTTP 200, giống bản gốc: True | Pass |
| 3 | Tải tệp vượt giới hạn | Tệp 11 MiB (giới hạn 10 MiB) | Từ chối, mã lỗi ATTACHMENT_FILE_TOO_LARGE | HTTP 429, mã lỗi ATTACHMENT_FILE_TOO_LARGE | Pass |
| 4 | Loại đối tượng sở hữu không hợp lệ | owner_type = abc | Từ chối, mã lỗi ATTACHMENT_OWNER_TYPE_INVALID | HTTP 400, mã lỗi ATTACHMENT_OWNER_TYPE_INVALID | Pass |

### 5.2.8. Kiểm thử chức năng báo cáo

Bảng 5.8 trình bày 4 trường hợp kiểm thử nhóm báo cáo; 4/4 trường hợp đạt.

[BANG: Bảng 5.8: Kết quả kiểm thử chức năng báo cáo]
| STT | Chức năng | Dữ liệu đầu vào | Kết quả mong đợi | Kết quả thực tế | Trạng thái |
|---|---|---|---|---|---|
| 1 | Báo cáo tổng quan | 30 ngày gần nhất | Trả tổng số task, tỉ lệ hoàn thành, quá hạn | HTTP 200, tổng 13, hoàn thành 15.38% | Pass |
| 2 | Tỉ lệ task theo trạng thái | Không lọc | Tổng tỉ lệ các trạng thái = 100% | HTTP 200, 4 trạng thái, tổng 100.0% | Pass |
| 3 | Biểu đồ tạo mới / hoàn thành theo tuần | interval = WEEK | Đủ các mốc tuần, có số tạo mới và hoàn thành | HTTP 200, 5 mốc | Pass |
| 4 | Thống kê lịch, sự kiện, backlog | 30 ngày tới | Trả số liệu lịch, sự kiện, backlog | HTTP 200, khoá: profileId, from, to | Pass |

### 5.2.9. Kiểm thử chức năng bạn bè và nhắn tin

Bảng 5.9 trình bày 10 trường hợp kiểm thử nhóm bạn bè và nhắn tin; 10/10 trường hợp đạt.

[BANG: Bảng 5.9: Kết quả kiểm thử chức năng bạn bè và nhắn tin]
| STT | Chức năng | Dữ liệu đầu vào | Kết quả mong đợi | Kết quả thực tế | Trạng thái |
|---|---|---|---|---|---|
| 1 | Tìm người dùng theo email | 22010328@st.phenikaa-uni.edu.vn | Tìm thấy hồ sơ | HTTP 200, tìm thấy: True | Pass |
| 2 | Tự kết bạn với chính mình | profile_id = chính mình | Từ chối, mã lỗi SOCIAL_CANNOT_FRIEND_SELF | HTTP 400, mã lỗi SOCIAL_CANNOT_FRIEND_SELF | Pass |
| 3 | Huỷ kết bạn | Huỷ kết bạn với 22010328@st.phenikaa-uni.edu.vn | Hai người không còn là bạn bè | HTTP 200, còn là bạn: False | Pass |
| 4 | Gửi lời mời kết bạn | Tới 22010328@st.phenikaa-uni.edu.vn | Tạo lời mời đang chờ | HTTP 200 | Pass |
| 5 | Chấp nhận lời mời kết bạn | Tài khoản 22010328 chấp nhận | Hai người trở thành bạn bè | HTTP 200, đã là bạn: True | Pass |
| 6 | Gửi tin nhắn rỗng | body rỗng | Từ chối, mã lỗi MESSAGE_BODY_REQUIRED | HTTP 400, mã lỗi MESSAGE_BODY_REQUIRED | Pass |
| 7 | Nhắn tin cho người chưa kết bạn | Người nhận không phải bạn bè | Từ chối, mã lỗi MESSAGE_NOT_FRIENDS | HTTP 403, mã lỗi MESSAGE_NOT_FRIENDS | Pass |
| 8 | Gửi tin nhắn cho bạn bè và lưu lịch sử | Tin nhắn văn bản tới 22010328 | Người nhận xem được tin trong lịch sử (qua Kafka → worker) | HTTP 200, có trong lịch sử người nhận: True | Pass |
| 9 | Giao task qua chat bằng lệnh /task | /task Chuẩn bị tài liệu họp nhóm, ưu tiên cao, hạn +1 ngày | Task được tạo trong danh sách của người nhận, tin nhắn có thẻ task | HTTP 200, thẻ task: True, task thuộc người nhận: True | Pass |
| 10 | Người giao được báo khi task đổi trạng thái | Người nhận chuyển task sang On Progress | Người giao nhận thông báo TASK_STATUS_CHANGED | Thông báo: Task “Chuẩn bị tài liệu họp nhóm” chuyển sang On Progress | Pass |

### 5.2.10. Kiểm thử chức năng thông báo

Bảng 5.10 trình bày 3 trường hợp kiểm thử nhóm thông báo; 3/3 trường hợp đạt.

[BANG: Bảng 5.10: Kết quả kiểm thử chức năng thông báo]
| STT | Chức năng | Dữ liệu đầu vào | Kết quả mong đợi | Kết quả thực tế | Trạng thái |
|---|---|---|---|---|---|
| 1 | Đếm thông báo chưa đọc | Tài khoản chính | Trả số lượng chưa đọc | HTTP 200, 7 chưa đọc | Pass |
| 2 | Chỉ xem được thông báo của mình | Token tài khoản 22010328 | Không thấy thông báo của tài khoản khác | HTTP 200, chỉ của mình: True | Pass |
| 3 | Xem thông báo khi chưa đăng nhập | Không gửi token | Bị từ chối (401) | HTTP 401 | Pass |

## 5.3. Kiểm thử tự động và kiểm thử phi chức năng

### 5.3.1. Kiểm thử đơn vị và kiểm tra mã nguồn

[BANG: Bảng 5.11: Kết quả kiểm thử đơn vị và kiểm tra mã nguồn]
| STT | Hạng mục | Công cụ | Quy mô | Kết quả |
|---|---|---|---|---|
| 1 | Kiểm thử đơn vị backend | go test | 409 hàm kiểm thử, 74 tệp, 38 package của 13 dịch vụ | Tất cả đạt |
| 2 | Kiểm thử đơn vị giao diện | Vitest | 174 kiểm thử, 13 tệp | 174/174 đạt |
| 3 | Định dạng mã | gofmt | Toàn bộ module | Đạt |
| 4 | Phân tích tĩnh | go vet | Toàn bộ module | Đạt |
| 5 | Quy tắc kiến trúc: hướng phụ thuộc, tầng phẳng, tệp bắt buộc, cmd dùng Fx | tools/quality | 13 dịch vụ | Đạt |
| 6 | Quy tắc kiến trúc: không comment trong mã Go | tools/quality | 13 dịch vụ | Chưa đạt: 20 vi phạm ở 6 dịch vụ (task, project, mail-provider, calendar, event, report) |

Các vi phạm ở hạng mục 6 là chú thích còn sót trong mã nguồn (phần lớn ở tệp kiểm thử và tệp mới bổ sung), không ảnh hưởng chức năng; cần dọn trước khi hoàn tất.

### 5.3.2. Kiểm thử phi chức năng

Các chỉ số dưới đây được đo trên máy phát triển (một máy, toàn bộ 31 container), mỗi phép đo lặp nhiều lần qua cổng nginx; vì vậy chỉ phản ánh mức tải nhẹ của một người dùng, chưa phải kiểm thử tải.

[BANG: Bảng 5.12: Kết quả kiểm thử phi chức năng]
| STT | Chỉ số | Cách đo | Kết quả |
|---|---|---|---|
| 1 | Độ trễ: Danh sách task (list) | 30 lần gọi | trung vị 2.7 ms; p95 4.3 ms |
| 2 | Độ trễ: Tìm kiếm task | 30 lần gọi | trung vị 2.9 ms; p95 3.3 ms |
| 3 | Độ trễ: Báo cáo tổng quan | 30 lần gọi | trung vị 3.6 ms; p95 4.4 ms |
| 4 | Độ trễ: Đếm thông báo chưa đọc | 30 lần gọi | trung vị 2.3 ms; p95 2.6 ms |
| 5 | Độ trễ: Lịch sử tin nhắn | 30 lần gọi | trung vị 2.3 ms; p95 2.7 ms |
| 6 | Độ trễ: Gửi tin nhắn (API trả về) | 10 lần gọi | trung vị 4.7 ms; lớn nhất 89.1 ms |
| 7 | Độ trễ: Gửi tin -> đã lưu vào CSDL (qua Kafka, worker) | 10 lần gọi | trung vị 61.4 ms; lớn nhất 146.4 ms |
| 8 | Mở rộng worker | 2 bản worker cùng consumer group message-worker | Kafka chia 6 partition thành 3 + 3, độ trễ tiêu thụ (lag) bằng 0 |
| 9 | Thời gian thực: nhắc việc | Đặt remind_at sau 5 giây | Nhận thông báo TASK_REMINDER trong một chu kỳ nhắc (30 giây) |
| 10 | Bảo mật: cô lập dữ liệu | Dùng token tài khoản này đọc kênh, thông báo của tài khoản khác | Bị từ chối (403) / chỉ thấy dữ liệu của mình |
| 11 | Độ tin cậy: không mất email | Email newsletter và email thiếu dữ kiện | Đều nằm trong backlog kèm lý do |

## 5.4. Đánh giá kết quả

- **Chức năng**: 49/49 trường hợp kiểm thử chức năng trên hệ thống thật đạt, bao phủ các yêu cầu chức năng ở Bảng 3.1 có API phía máy chủ, gồm cả các trường hợp lỗi (dữ liệu sai, vi phạm luật chuyển trạng thái, truy cập dữ liệu người khác). Các yêu cầu thuần giao diện (FR-07 thêm nhanh, FR-10 các chế độ xem, FR-13 việc hôm nay) được minh chứng bằng ảnh chụp ở mục 4.6.
- **Tự động hoá email**: hệ thống đã phân loại đúng email thật: newsletter vào nhóm *other*, email yêu cầu công việc và lịch hẹn được nhận diện đúng nhóm nhưng vào backlog do chưa khớp dự án nào (lý do `no_project_matched`), đúng thiết kế.
- **Thời gian thực và bất đồng bộ**: thông báo, tin nhắn hiển thị tức thời; tin nhắn được lưu vào cơ sở dữ liệu sau khoảng 61 ms qua Kafka; thẻ task giao qua chat tự cập nhật trạng thái.
- **Chất lượng mã**: kiểm thử đơn vị backend và giao diện đều đạt; còn 20 vi phạm quy tắc không comment cần dọn.
- **Điểm cần cải thiện phát hiện khi kiểm thử**: lỗi tệp vượt giới hạn trả mã HTTP 429 (do ánh xạ từ mã gRPC ResourceExhausted), nên đổi sang 413 cho đúng ngữ nghĩa; một số gateway chưa kiểm tra token (đã nêu ở mục 4.3.4).

<!-- Nguồn: docs/system/02-architecture.md §3-4, 03-business.md, 04-user-stories.md, 05-service-calls-and-data.md §5; service/message/README.md, service/worker/README.md (chat-task) -->
# CHƯƠNG 3: PHÂN TÍCH HỆ THỐNG

## 3.1. Yêu cầu hệ thống

### 3.1.1. Mô tả hệ thống

Task Manager là ứng dụng web giúp một người dùng quản lý tập trung công việc, lịch, sự kiện và các email cần xử lý. Người dùng đăng nhập bằng tài khoản Google, không cần tạo mật khẩu riêng. Khi đăng nhập lần đầu, hệ thống tự tạo sẵn bộ trạng thái công việc, luật chuyển trạng thái và một dự án mặc định để người dùng có thể làm việc ngay.

Công việc được hiển thị trên **bảng Kanban**: mỗi cột là một trạng thái do người dùng định nghĩa, mỗi thẻ là một công việc. Người dùng kéo thả thẻ để đổi trạng thái; hệ thống chỉ cho phép những bước chuyển đã được khai báo trong luật chuyển trạng thái. Công việc có thể chia thành công việc con, gắn vào dự án, đặt hạn, đặt nhắc việc, lặp lại theo chu kỳ và đính kèm tệp. Bên cạnh đó, người dùng quản lý **lịch** (khối thời gian trên lịch) và **sự kiện** (có trạng thái dự kiến, đã xác nhận, đã huỷ).

Nếu người dùng cho phép đọc Gmail, hệ thống **tự động đọc mỗi email mới**, dùng trí tuệ nhân tạo phân loại thành công việc, lịch hoặc sự kiện và tạo bản ghi tương ứng; email không phân loại được hoặc thiếu thông tin được đưa vào **backlog** để người dùng tự xử lý. Mọi đối tượng được tạo tự động đều kèm **thông báo thời gian thực**. Người dùng có thể xem **báo cáo** tiến độ, **kết bạn**, **nhắn tin** và **giao công việc cho bạn bè ngay trong cuộc trò chuyện** bằng lệnh `/task`.

Về tổ chức, hệ thống gồm ba lớp:

- **Lớp giao diện**: ứng dụng web React chạy trên trình duyệt, nhận thông báo và tin nhắn qua WebSocket.
- **Lớp xử lý nghiệp vụ**: 13 dịch vụ Go, mỗi dịch vụ phụ trách một miền nghiệp vụ, giao tiếp qua gRPC và Kafka; nginx là cổng vào duy nhất.
- **Lớp dữ liệu**: PostgreSQL (mỗi dịch vụ một schema), kho đối tượng RustFS cho tệp và email gốc.

### 3.1.2. Quy trình làm việc của người dùng

Hình 3.1 mô tả hành trình của người dùng từ khi mở ứng dụng tới khi đăng xuất.

[HINH: h3-01-quy-trinh-nguoi-dung.png | Hình 3.1: Sơ đồ quy trình làm việc của người dùng]

Các bước chính:

- **Đăng nhập**: người dùng bấm đăng nhập Google, có thể tích "Cho phép đọc Gmail". Với người dùng mới, hệ thống tự tạo dữ liệu mặc định.
- **Làm việc với công việc**: tạo công việc, kéo thả trên bảng Kanban để đổi trạng thái, mở chi tiết để xem công việc con và tệp đính kèm.
- **Lập kế hoạch**: thêm, sửa lịch và sự kiện; tổ chức công việc theo dự án.
- **Xử lý backlog**: xem các email chưa phân loại, chuyển trạng thái đã xử lý hoặc lưu trữ.
- **Theo dõi**: xem báo cáo, nhận thông báo thời gian thực và bấm vào thông báo để mở đúng đối tượng.
- **Giao tiếp**: kết bạn, nhắn tin, giao công việc cho bạn bè.

### 3.1.3. Quy trình tự động xử lý email

Quy trình xử lý email (Hình 3.2) chạy hoàn toàn tự động, không cần người dùng thao tác.

[HINH: h3-02-quy-trinh-email.png | Hình 3.2: Sơ đồ quy trình tự động xử lý email]

Khi hộp thư có thay đổi, Google Pub/Sub báo cho dịch vụ xử lý thư. Dịch vụ không dùng mốc lịch sử trong thông báo mà đọc **mốc đã xử lý lưu trong cơ sở dữ liệu**, lấy toàn bộ thư mới kể từ mốc đó, lưu bản gốc lên kho S3, rồi phân loại bằng DeepSeek (hoặc luật từ khoá khi DeepSeek không khả dụng). Email thuộc nhóm công việc, lịch, sự kiện, đủ thông tin và **khớp được một dự án** của người dùng được tạo thành bản ghi tương ứng trong dự án đó; trường hợp còn lại vào backlog kèm lý do (ví dụ `no_project_matched` khi không khớp dự án nào). Sau khi xử lý xong, hệ thống tạo thông báo và mới tiến mốc lịch sử, nhờ đó thông báo trùng lặp hoặc đến sai thứ tự không làm mất email.

### 3.1.4. Yêu cầu chức năng

Bảng 3.1 liệt kê các yêu cầu chức năng của hệ thống.

[BANG: Bảng 3.1: Danh sách yêu cầu chức năng]
| Mã | Phân hệ | Yêu cầu chức năng |
|---|---|---|
| FR-01 | Tài khoản | Đăng nhập bằng tài khoản Google; phiên đăng nhập có hiệu lực 30 ngày |
| FR-02 | Tài khoản | Cho phép hệ thống đọc Gmail (tuỳ chọn khi đăng nhập) |
| FR-03 | Tài khoản | Tự tạo bộ trạng thái, luật chuyển và dự án mặc định cho người dùng mới |
| FR-04 | Dự án | Tạo, đổi tên dự án; tên không trùng trong cùng người dùng |
| FR-05 | Dự án | Lưu trữ, khôi phục dự án; không lưu trữ được dự án mặc định |
| FR-06 | Công việc | Tạo, sửa, xoá công việc (tiêu đề, mô tả, ưu tiên, ngày bắt đầu, hạn, dự án) |
| FR-07 | Công việc | Thêm nhanh công việc bằng một dòng văn bản (nhận diện ngày, giờ, ưu tiên, dự án) |
| FR-08 | Công việc | Chia công việc thành công việc con; hiển thị tiến độ theo công việc con |
| FR-09 | Công việc | Kéo thả trên bảng Kanban để đổi trạng thái, chỉ theo luật chuyển đã khai báo |
| FR-10 | Công việc | Xem dạng Kanban, bảng, danh sách, dòng thời gian; tìm kiếm và lọc |
| FR-11 | Công việc | Đặt thời điểm nhắc việc và nhận thông báo đúng giờ |
| FR-12 | Công việc | Đặt công việc lặp lại; hoàn thành một lần thì tự sinh lần kế tiếp |
| FR-13 | Công việc | Xem việc quá hạn, hôm nay, sắp tới trên trang Việc hôm nay |
| FR-14 | Công việc | Quản lý trạng thái và luật chuyển trạng thái |
| FR-15 | Kế hoạch | Xem lịch theo ngày, tuần, tháng; tạo, sửa, xoá lịch |
| FR-16 | Kế hoạch | Tạo, sửa, xoá sự kiện; xác nhận hoặc huỷ sự kiện |
| FR-17 | Hộp thư | Tự động phân loại email mới thành công việc, lịch, sự kiện hoặc backlog |
| FR-18 | Hộp thư | Xem, lọc, sửa, đổi trạng thái xử lý các mục backlog |
| FR-19 | Tệp đính kèm | Tải lên, tải xuống, xoá tệp của công việc, lịch, sự kiện, backlog |
| FR-20 | Báo cáo | Xem tổng quan, tỉ lệ theo trạng thái, biểu đồ theo thời gian, hoạt động |
| FR-21 | Thông báo | Nhận thông báo thời gian thực; đếm chưa đọc; đánh dấu đã đọc |
| FR-22 | Bạn bè | Tìm theo email, gửi, chấp nhận, từ chối, huỷ lời mời; huỷ kết bạn |
| FR-23 | Nhắn tin | Nhắn tin 1-1 thời gian thực với bạn bè; xem lịch sử trò chuyện |
| FR-24 | Nhắn tin | Giao công việc cho bạn bè bằng lệnh `/task`; người giao được báo khi trạng thái đổi |

### 3.1.5. Yêu cầu phi chức năng

[BANG: Bảng 3.2: Danh sách yêu cầu phi chức năng]
| Mã | Nhóm | Yêu cầu | Cách đáp ứng |
|---|---|---|---|
| NFR-01 | Bảo mật | Không lưu mật khẩu; người dùng chỉ truy cập dữ liệu của mình | Đăng nhập Google OAuth 2.0 + PKCE; session JWT; kênh realtime riêng tư được ký uỷ quyền |
| NFR-02 | Thời gian thực | Thông báo, tin nhắn hiện ngay không cần tải lại trang | WebSocket qua Soketi |
| NFR-03 | Độ tin cậy | Không mất email, không mất tin nhắn | Backlog cho mọi email không phân loại được; Kafka đảm bảo ít nhất một lần, ghi luỹ đẳng |
| NFR-04 | Khả dụng | Lỗi của chức năng phụ không làm hỏng thao tác chính | Gửi thông báo, khởi tạo dữ liệu mặc định là nỗ lực tối đa (best-effort) |
| NFR-05 | Khả năng mở rộng | Tăng năng lực xử lý khi tải tăng | Microservices; tăng số bản chạy worker theo số partition Kafka |
| NFR-06 | Khả năng bảo trì | Mã nguồn rõ ràng, nhất quán | Kiến trúc phân tầng, kiểm tra quy tắc kiến trúc tự động, kiểm thử đơn vị |
| NFR-07 | Khả dụng ngôn ngữ | Giao diện tiếng Việt và tiếng Anh | i18next; bộ mã lỗi chuẩn có thông điệp hai ngôn ngữ |
| NFR-08 | Triển khai | Cài đặt và khởi chạy đơn giản | Docker Compose: một lệnh khởi động toàn bộ hệ thống |

## 3.2. Các tác nhân tác động đến hệ thống

### 3.2.1. Khách

Khách là người chưa đăng nhập. Khách chỉ truy cập được trang đăng nhập và thực hiện đăng nhập bằng Google; sau khi đăng nhập thành công, khách trở thành người dùng.

### 3.2.2. Người dùng

Người dùng là cá nhân đã đăng nhập, là tác nhân chính của hệ thống. Người dùng thực hiện toàn bộ các chức năng quản lý dự án, công việc, lịch, sự kiện, backlog, tệp đính kèm, xem báo cáo, thông báo, kết bạn và nhắn tin. Mọi thao tác chỉ tác động lên dữ liệu của chính người dùng đó; riêng chức năng giao công việc qua chat tạo công việc trong danh sách của người nhận.

### 3.2.3. Tác nhân tự động

Tác nhân tự động là các tiến trình chạy nền trong hệ thống, hoạt động không cần người dùng thao tác:

- **Bộ xử lý email**: nhận thông báo từ Gmail, phân loại và tạo bản ghi.
- **Bộ gia hạn theo dõi Gmail**: định kỳ gia hạn đăng ký theo dõi hộp thư trước khi hết hạn 7 ngày.
- **Bộ nhắc việc**: định kỳ tìm công việc đến giờ nhắc và gửi thông báo.
- **Worker Kafka**: lưu tin nhắn vào cơ sở dữ liệu và cập nhật thẻ công việc được giao qua chat.

### 3.2.4. Hệ thống bên ngoài

- **Google OAuth / OpenID Connect**: xác thực người dùng, cấp token truy cập Gmail.
- **Gmail API và Google Cloud Pub/Sub**: cung cấp nội dung email và thông báo khi hộp thư thay đổi.
- **DeepSeek API**: phân loại email bằng mô hình ngôn ngữ lớn.

## 3.3. Sơ đồ phân rã chức năng

### 3.3.1. Phân rã chức năng tổng thể

Hệ thống được phân rã thành năm nhóm chức năng chính (Hình 3.3).

[HINH: h3-03-phan-ra-tong-the.png | Hình 3.3: Sơ đồ phân rã chức năng tổng thể]

### 3.3.2. Nhóm quản lý công việc và kế hoạch

Nhóm quản lý công việc là trọng tâm của hệ thống, gồm quản lý dự án, công việc, trạng thái và trang việc hôm nay. Nhóm kế hoạch gồm lịch cá nhân và sự kiện (Hình 3.4).

[HINH: h3-04-phan-ra-cong-viec-ke-hoach.png | Hình 3.4: Sơ đồ phân rã chức năng quản lý công việc và kế hoạch]

### 3.3.3. Nhóm hộp thư thông minh, tổng quan và giao tiếp

Nhóm hộp thư thông minh xử lý email tự động, backlog và tệp đính kèm. Nhóm tổng quan và giao tiếp gồm báo cáo, thông báo, bạn bè và nhắn tin (Hình 3.5).

[HINH: h3-05-phan-ra-hop-thu-giao-tiep.png | Hình 3.5: Sơ đồ phân rã chức năng hộp thư thông minh, tổng quan và giao tiếp]

## 3.4. Sơ đồ Use Case

### 3.4.1. Sơ đồ Use Case tổng quát

Hình 3.6 thể hiện các use case của hệ thống và tác nhân tương ứng. Người dùng tham gia các use case quản lý và giao tiếp; tác nhân tự động thực hiện phân loại email và nhắc việc, có sự tham gia của các hệ thống bên ngoài.

[HINH: h3-06-use-case.png | Hình 3.6: Sơ đồ Use Case tổng quát]

### 3.4.2. Đặc tả các Use Case chính

[BANG: Bảng 3.3: Đặc tả Use Case Đăng nhập bằng Google]
| Mục | Nội dung |
|---|---|
| Tác nhân | Khách; Google OAuth |
| Mô tả | Khách đăng nhập bằng tài khoản Google, tuỳ chọn cho phép hệ thống đọc Gmail |
| Tiền điều kiện | Khách có tài khoản Google; chưa đăng nhập |
| Luồng chính | 1. Khách bấm "Đăng nhập Google", có thể tích "Cho phép đọc Gmail". 2. Hệ thống chuyển sang Google. 3. Khách đăng nhập, đồng ý cấp quyền. 4. Hệ thống xác minh và tìm hoặc tạo hồ sơ. 5. Hệ thống cấp phiên đăng nhập và mở trang chính |
| Luồng thay thế | 4a. Người dùng mới: tạo sẵn 4 trạng thái, 5 luật chuyển, dự án "Chung". 4b. Có tích đọc Gmail: đăng ký theo dõi hộp thư |
| Ngoại lệ | Khách từ chối cấp quyền hoặc đổi mã thất bại: hiển thị thông báo lỗi, quay lại trang đăng nhập |
| Hậu điều kiện | Người dùng đăng nhập thành công, phiên có hiệu lực 30 ngày |

[BANG: Bảng 3.4: Đặc tả Use Case Đổi trạng thái công việc]
| Mục | Nội dung |
|---|---|
| Tác nhân | Người dùng |
| Mô tả | Người dùng kéo thả thẻ công việc sang cột trạng thái khác trên bảng Kanban |
| Tiền điều kiện | Đã đăng nhập; công việc thuộc người dùng |
| Luồng chính | 1. Người dùng kéo thẻ sang cột mới. 2. Hệ thống kiểm tra tồn tại luật chuyển từ trạng thái hiện tại sang trạng thái đích. 3. Hệ thống cập nhật trạng thái, ghi lịch sử. 4. Thẻ hiển thị ở cột mới |
| Luồng thay thế | 3a. Trạng thái đích thuộc nhóm hoàn thành: ghi thời điểm hoàn thành; nếu là công việc lặp lại thì sinh lần kế tiếp. 3b. Công việc được giao qua chat: báo cho người giao |
| Ngoại lệ | Không có luật chuyển: hiển thị thông báo lỗi, thẻ trở về cột cũ |
| Hậu điều kiện | Trạng thái công việc được cập nhật và ghi lịch sử |

[BANG: Bảng 3.5: Đặc tả Use Case Tự động phân loại email]
| Mục | Nội dung |
|---|---|
| Tác nhân | Tác nhân tự động; Gmail, Pub/Sub; DeepSeek |
| Mô tả | Mỗi email mới được phân loại và tạo thành bản ghi tương ứng |
| Tiền điều kiện | Người dùng đã cho phép đọc Gmail; đăng ký theo dõi còn hiệu lực |
| Luồng chính | 1. Pub/Sub báo hộp thư thay đổi. 2. Hệ thống lấy thư mới từ mốc đã lưu. 3. Lưu email gốc lên S3. 4. Phân loại bằng DeepSeek. 5. Tạo công việc, lịch hoặc sự kiện. 6. Gửi thông báo. 7. Tiến mốc lịch sử |
| Luồng thay thế | 4a. DeepSeek lỗi hoặc không cấu hình: phân loại bằng luật từ khoá. 5a. Nhóm khác hoặc thiếu thông tin: tạo mục backlog kèm lý do. 5b. Không khớp dự án nào (độ tin cậy dưới ngưỡng): tạo mục backlog với lý do no_project_matched |
| Ngoại lệ | Mốc lịch sử quá cũ: đặt lại mốc theo hồ sơ Gmail hiện tại và ghi cảnh báo |
| Hậu điều kiện | Email được lưu thành một bản ghi, người dùng nhận thông báo |

[BANG: Bảng 3.6: Đặc tả Use Case Nhắn tin]
| Mục | Nội dung |
|---|---|
| Tác nhân | Người dùng (người gửi, người nhận) |
| Mô tả | Hai người dùng là bạn bè nhắn tin trực tiếp theo thời gian thực |
| Tiền điều kiện | Đã đăng nhập; hai người đã là bạn bè |
| Luồng chính | 1. Người gửi nhập tin (1 đến 2000 ký tự) và gửi. 2. Hệ thống kiểm tra quan hệ bạn bè. 3. Hệ thống đẩy tin tới người nhận và các thiết bị của người gửi. 4. Hệ thống lưu tin bất đồng bộ |
| Ngoại lệ | Không phải bạn bè: từ chối gửi. Nội dung rỗng hoặc quá dài: báo lỗi |
| Hậu điều kiện | Tin nhắn hiển thị ở hai phía và được lưu vào lịch sử |

[BANG: Bảng 3.7: Đặc tả Use Case Giao công việc qua chat]
| Mục | Nội dung |
|---|---|
| Tác nhân | Người dùng (người giao, người nhận) |
| Mô tả | Người giao dùng lệnh `/task` trong cuộc trò chuyện để tạo công việc cho bạn bè |
| Tiền điều kiện | Đã đăng nhập; hai người là bạn bè |
| Luồng chính | 1. Người giao nhập `/task` kèm tiêu đề, ưu tiên, hạn. 2. Hệ thống tạo công việc thuộc người nhận, trong dự án mặc định, ghi nhận người giao. 3. Tin nhắn kèm thẻ công việc hiện ở hai phía. 4. Khi người nhận đổi trạng thái, thẻ trong tin nhắn được cập nhật và người giao nhận thông báo |
| Ngoại lệ | Không phải bạn bè hoặc thiếu tiêu đề: từ chối tạo |
| Hậu điều kiện | Người nhận có công việc mới; người giao theo dõi được trạng thái |

## 3.5. Sơ đồ tuần tự cho các chức năng chính

### 3.5.1. Chức năng đăng nhập bằng Google

Hình 3.7 mô tả luồng đăng nhập. Dịch vụ định danh điều phối toàn bộ luồng OAuth với Google; với người dùng mới, dịch vụ gọi dịch vụ công việc và dịch vụ dự án để tạo dữ liệu mặc định; nếu người dùng cho phép đọc Gmail, dịch vụ gọi dịch vụ xử lý thư để đăng ký theo dõi hộp thư. Cuối cùng, người dùng được chuyển về giao diện kèm session token.

[HINH: h3-07-tt-dang-nhap.png | Hình 3.7: Sơ đồ tuần tự chức năng đăng nhập bằng Google]

### 3.5.2. Chức năng tạo công việc

Khi tạo công việc (Hình 3.8), dịch vụ công việc kiểm tra dữ liệu và gọi dịch vụ dự án để xác minh dự án tồn tại, thuộc người dùng và chưa bị lưu trữ, sau đó gán trạng thái mặc định và lưu công việc cùng bản ghi lịch sử trạng thái.

[HINH: h3-08-tt-tao-task.png | Hình 3.8: Sơ đồ tuần tự chức năng tạo công việc]

### 3.5.3. Chức năng đổi trạng thái công việc

Đổi trạng thái (Hình 3.9) là chức năng thể hiện rõ nghiệp vụ vòng đời công việc: hệ thống chỉ chấp nhận bước chuyển khi tồn tại luật chuyển tương ứng. Khi công việc hoàn thành, hệ thống ghi thời điểm hoàn thành, sinh lần lặp kế tiếp đối với công việc lặp lại, và phát sự kiện lên Kafka đối với công việc được giao qua chat.

[HINH: h3-09-tt-doi-trang-thai.png | Hình 3.9: Sơ đồ tuần tự chức năng đổi trạng thái công việc]

Hình 3.10 là vòng đời mặc định được tạo cho mỗi người dùng mới, gồm 4 trạng thái và 5 luật chuyển. Người dùng có thể thêm trạng thái và luật chuyển để phù hợp cách làm việc riêng.

[HINH: h3-16-trang-thai-task.png | Hình 3.10: Sơ đồ trạng thái vòng đời công việc mặc định]

### 3.5.4. Chức năng nhắc việc

Bộ nhắc việc chạy nền trong dịch vụ công việc (Hình 3.11). Mỗi chu kỳ, bộ nhắc *nhận* các công việc đến giờ nhắc (đánh dấu để không gửi trùng), gửi thông báo sang dịch vụ thông báo; nếu gửi lỗi, công việc được trả lại để nhắc ở chu kỳ sau.

[HINH: h3-10-tt-nhac-viec.png | Hình 3.11: Sơ đồ tuần tự chức năng nhắc việc]

### 3.5.5. Chức năng tự động phân loại email

Hình 3.12 mô tả chi tiết các thành phần tham gia xử lý một email: Gmail và Pub/Sub, dịch vụ xử lý thư, kho S3, DeepSeek, các dịch vụ đích và dịch vụ thông báo. Dịch vụ xử lý thư xác nhận thông báo Pub/Sub ngay và xử lý trong hàng đợi nội bộ để không làm Pub/Sub phải chờ.

[HINH: h3-11-tt-phan-loai-email.png | Hình 3.12: Sơ đồ tuần tự chức năng tự động phân loại email]

### 3.5.6. Chức năng đính kèm tệp

Tệp được tải lên dạng multipart (Hình 3.13). Dịch vụ tệp đính kèm kiểm tra kích thước (mặc định tối đa 10 MiB), lưu nội dung lên kho S3 và lưu thông tin tệp vào cơ sở dữ liệu. Khi tải xuống, dịch vụ đọc thông tin tệp rồi lấy nội dung từ kho S3.

[HINH: h3-12-tt-dinh-kem.png | Hình 3.13: Sơ đồ tuần tự chức năng đính kèm tệp]

### 3.5.7. Chức năng xem báo cáo

Dịch vụ báo cáo không có cơ sở dữ liệu riêng (Hình 3.14): nó gọi song song các dịch vụ công việc, sự kiện, lịch và backlog, sau đó tổng hợp, tính tỉ lệ và chia dữ liệu theo mốc thời gian (ngày, tuần, tháng).

[HINH: h3-13-tt-bao-cao.png | Hình 3.14: Sơ đồ tuần tự chức năng xem báo cáo]

### 3.5.8. Chức năng nhắn tin

Nhắn tin áp dụng nguyên tắc *hiển thị trước, lưu sau* (Hình 3.15): sau khi xác minh hai người là bạn bè, dịch vụ nhắn tin đẩy tin ngay tới người nhận qua Soketi, đồng thời ghi tin vào Kafka; worker tiêu thụ và lưu tin vào cơ sở dữ liệu, bỏ qua bản trùng.

[HINH: h3-14-tt-nhan-tin.png | Hình 3.15: Sơ đồ tuần tự chức năng nhắn tin]

### 3.5.9. Chức năng giao công việc qua chat

Chức năng giao công việc qua chat (Hình 3.16) kết nối phân hệ nhắn tin với phân hệ công việc. Dịch vụ nhắn tin gọi dịch vụ công việc tạo công việc cho người nhận và gửi tin nhắn kèm thẻ công việc. Khi người nhận đổi trạng thái, dịch vụ công việc phát sự kiện lên topic `task.events`; worker cập nhật thẻ trong tin nhắn, đẩy cập nhật tới hai phía và tạo thông báo cho người giao.

[HINH: h3-15-tt-giao-task-chat.png | Hình 3.16: Sơ đồ tuần tự chức năng giao công việc qua chat]

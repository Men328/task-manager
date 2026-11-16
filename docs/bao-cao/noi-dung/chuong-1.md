<!-- Nguồn: docs/system/03-business.md §1, §3; docs/system/04-user-stories.md; docs/thuyet-trinh/cong-nghe-do-an.xlsx; README.md -->
# CHƯƠNG 1: TỔNG QUAN ĐỒ ÁN

## 1.1. Giới thiệu đồ án

Trong bối cảnh chuyển đổi số diễn ra mạnh mẽ, khối lượng công việc mà mỗi cá nhân phải theo dõi hằng ngày ngày càng lớn và đến từ nhiều nguồn khác nhau: yêu cầu từ giảng viên hoặc cấp trên, lịch họp, sự kiện cần tham dự, hạn nộp báo cáo, hoá đơn cần thanh toán. Phần lớn các đầu việc này được gửi tới qua **email**, trong khi người dùng lại ghi chép công việc ở một ứng dụng việc cần làm (to-do), quản lý lịch ở một ứng dụng lịch khác và trao đổi với bạn bè qua một ứng dụng nhắn tin riêng.

Sự phân tán này gây ra nhiều bất tiện. Người dùng phải tự đọc từng email rồi chép tay sang công cụ quản lý công việc, dễ bỏ sót các yêu cầu quan trọng. Trạng thái công việc không được theo dõi nhất quán, không có cái nhìn tổng quan về tiến độ, và việc nhắc hạn phụ thuộc vào trí nhớ. Các công cụ phổ biến như Trello, Todoist hay Google Calendar giải quyết tốt từng mảng riêng lẻ nhưng chưa tự động biến email thành công việc có cấu trúc.

Xuất phát từ thực tế đó, đồ án xây dựng **hệ thống quản lý công việc cá nhân Task Manager**: một ứng dụng web giúp một người dùng quản lý tập trung công việc, lịch và sự kiện tại một nơi. Điểm nổi bật của hệ thống là khả năng **tự động đọc hộp thư Gmail và phân loại từng email mới** bằng mô hình ngôn ngữ lớn (DeepSeek) thành công việc, lịch hẹn, sự kiện hoặc mục chờ xử lý (backlog), sau đó thông báo tức thời cho người dùng. Bên cạnh đó, hệ thống hỗ trợ bảng Kanban với vòng đời trạng thái do người dùng tự định nghĩa, nhắc việc, công việc lặp lại, tệp đính kèm, báo cáo thống kê, kết bạn và nhắn tin thời gian thực.

Về mặt kỹ thuật, hệ thống được xây dựng theo **kiến trúc vi dịch vụ (microservices)** gồm 13 dịch vụ viết bằng ngôn ngữ Go, giao tiếp nội bộ qua gRPC, kết hợp giao diện web React. Dữ liệu được lưu trữ trên PostgreSQL và kho đối tượng chuẩn S3; tin nhắn được xử lý bất đồng bộ qua Apache Kafka; thông báo và tin nhắn được đẩy tới trình duyệt theo thời gian thực qua máy chủ WebSocket Soketi. Toàn bộ hệ thống được đóng gói và triển khai bằng Docker Compose.

## 1.2. Mục tiêu đồ án

Đồ án hướng tới xây dựng một hệ thống quản lý công việc cá nhân hoàn chỉnh, có khả năng vận hành thực tế, với các mục tiêu cụ thể sau:

- **Quản lý công việc trực quan**: tạo, sửa, xoá công việc và công việc con; hiển thị trên bảng Kanban, cho phép kéo thả để đổi trạng thái; người dùng tự định nghĩa bộ trạng thái và luật chuyển trạng thái (vòng đời công việc).
- **Quản lý kế hoạch thời gian**: quản lý dự án (nhóm công việc), lịch cá nhân, sự kiện có trạng thái xác nhận; hỗ trợ nhắc việc và công việc lặp lại theo chu kỳ.
- **Tự động hoá hộp thư**: theo dõi Gmail của người dùng, tự động phân loại email mới bằng trí tuệ nhân tạo và tạo công việc, lịch, sự kiện tương ứng; email không phân loại được đưa vào backlog để không email nào bị bỏ sót.
- **Hỗ trợ làm việc**: đính kèm tệp cho công việc, lịch, sự kiện, backlog; báo cáo thống kê tiến độ bằng biểu đồ trực quan.
- **Kết nối và thông báo thời gian thực**: thông báo tức thời khi có công việc mới, nhắc hạn, lời mời kết bạn; kết bạn và nhắn tin trực tiếp giữa hai người dùng.
- **Kiến trúc hiện đại, dễ mở rộng**: áp dụng kiến trúc microservices, giao tiếp gRPC, xử lý bất đồng bộ qua hàng đợi thông điệp; đóng gói toàn bộ hệ thống bằng Docker để triển khai bằng một lệnh.

## 1.3. Phạm vi đồ án

### 1.3.1. Đối tượng người dùng

Hệ thống phục vụ **người dùng cá nhân**: mỗi người dùng sở hữu và chỉ thao tác trên dữ liệu của chính mình. Hệ thống không có khái niệm tổ chức, nhóm làm việc hay vai trò quản trị viên. Ngoài người dùng, hệ thống có các **tác nhân tự động** chạy nền: bộ xử lý email, bộ nhắc việc và bộ lưu tin nhắn.

### 1.3.2. Phạm vi chức năng

Đồ án bao gồm các nhóm chức năng sau:

- Tài khoản: đăng nhập bằng Google, cho phép đọc Gmail, tự khởi tạo dữ liệu mặc định cho người dùng mới.
- Dự án và công việc: dự án, công việc cha – con, trạng thái và luật chuyển trạng thái, bảng Kanban, tìm kiếm, lọc, nhắc việc, công việc lặp lại, trang việc hôm nay.
- Kế hoạch: lịch cá nhân và sự kiện.
- Hộp thư thông minh: tự động phân loại email, backlog, lưu trữ email gốc.
- Tệp đính kèm, báo cáo thống kê, thông báo thời gian thực, kết bạn và nhắn tin.

Các nội dung **chưa nằm trong phạm vi** đồ án: lặp lại và mời người tham gia cho lịch, sự kiện; đồng bộ hai chiều với Google Calendar; hỗ trợ hộp thư ngoài Gmail; ứng dụng di động; hệ thống giám sát và ghi log tập trung.

### 1.3.3. Phạm vi công nghệ

- **Phía máy chủ**: ngôn ngữ Go, gRPC và Protocol Buffers, grpc-gateway, thư viện Uber Fx.
- **Phía giao diện**: React, TypeScript, Vite, Mantine UI, Redux Toolkit, FullCalendar.
- **Lưu trữ và hạ tầng**: PostgreSQL, RustFS (kho đối tượng chuẩn S3), Apache Kafka, Soketi (WebSocket), nginx.
- **Dịch vụ bên ngoài**: Google OAuth 2.0, Gmail API, Google Cloud Pub/Sub, DeepSeek API.
- **Triển khai**: Docker, Docker Compose, Cloudflare Tunnel.

## 1.4. Phương pháp nghiên cứu

Đồ án được thực hiện theo các phương pháp sau:

- **Nghiên cứu tài liệu**: tìm hiểu tài liệu chính thức về kiến trúc microservices, gRPC, Apache Kafka, giao thức OAuth 2.0 và OpenID Connect, Gmail API, Google Cloud Pub/Sub và các mô hình ngôn ngữ lớn.
- **Khảo sát ứng dụng tương tự**: phân tích các ứng dụng Trello, Todoist, Google Calendar để xác định các tính năng cốt lõi của một hệ thống quản lý công việc cá nhân, từ đó đề xuất tính năng khác biệt là tự động hoá hộp thư.
- **Phân tích và thiết kế trước khi lập trình**: xác định yêu cầu, xây dựng sơ đồ Use Case, sơ đồ tuần tự, sơ đồ thực thể liên kết; định nghĩa giao diện giữa các dịch vụ bằng tệp Protocol Buffers trước khi cài đặt (interface-first).
- **Phát triển lặp**: phát triển theo từng dịch vụ và từng tính năng nhỏ; mỗi tính năng được phát triển trên một nhánh mã nguồn riêng, kiểm thử và rà soát trước khi hợp nhất vào nhánh chính.
- **Kiểm thử và đánh giá**: viết kiểm thử đơn vị cho tầng nghiệp vụ, kiểm thử chức năng trên hệ thống chạy thực tế và tự động kiểm tra quy tắc kiến trúc mã nguồn.

## 1.5. Bố cục báo cáo

Ngoài phần mở đầu và kết luận, báo cáo gồm năm chương:

- **Chương 1 – Tổng quan đồ án**: giới thiệu bối cảnh, mục tiêu, phạm vi và phương pháp thực hiện.
- **Chương 2 – Cơ sở lý thuyết và công nghệ sử dụng**: trình bày kiến trúc microservices, các cơ chế giao tiếp giữa dịch vụ, công nghệ phía máy chủ, phía giao diện, lưu trữ, xác thực, tích hợp Gmail và trí tuệ nhân tạo, cùng các nguyên tắc thiết kế.
- **Chương 3 – Phân tích hệ thống**: phân tích yêu cầu chức năng và phi chức năng, tác nhân, sơ đồ phân rã chức năng, sơ đồ Use Case và sơ đồ tuần tự của các chức năng chính.
- **Chương 4 – Thiết kế hệ thống**: thiết kế kiến trúc tổng thể, giao tiếp giữa các dịch vụ, giao diện lập trình (API), cơ sở dữ liệu, xử lý nghiệp vụ trọng tâm và giao diện người dùng.
- **Chương 5 – Triển khai và kiểm thử hệ thống**: môi trường và cách triển khai, kết quả kiểm thử chức năng và phi chức năng, đánh giá kết quả.

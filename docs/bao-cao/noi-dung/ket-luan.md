<!-- Kết luận: theo bố cục báo cáo mẫu (kết quả đạt được → hạn chế → hướng phát triển → lời cảm ơn). Tiêu đề "KẾT LUẬN" do build_khung.py thêm. -->
Sau quá trình nghiên cứu và xây dựng, đồ án "Hệ thống quản lí công việc, lịch cá nhân" đã hoàn thành với đầy đủ các chức năng đề ra. Hệ thống được xây dựng bằng ngôn ngữ Go theo kiến trúc vi dịch vụ, giao tiếp qua gRPC và Apache Kafka, giao diện React, được đóng gói và triển khai bằng Docker, và đã được kiểm thử trên môi trường chạy thật.

## 1. Kết quả đạt được

- Xây dựng thành công hệ thống quản lý công việc cá nhân gồm **13 dịch vụ** và giao diện web, triển khai bằng một lệnh Docker Compose.
- Hoàn thành chức năng **quản lý công việc** trên bảng Kanban với vòng đời trạng thái do người dùng tự định nghĩa bằng luật chuyển trạng thái, công việc con, dự án, nhắc việc, công việc lặp lại, tìm kiếm và nhiều chế độ xem.
- Hoàn thành chức năng **lịch, sự kiện, tệp đính kèm và báo cáo thống kê** trực quan theo khoảng thời gian.
- Tích hợp thành công **Gmail, Google Pub/Sub và mô hình ngôn ngữ lớn DeepSeek** để tự động phân loại email mới thành công việc, lịch, sự kiện hoặc backlog, có cơ chế dự phòng bằng luật từ khoá và bảo đảm không email nào bị mất; đã kiểm chứng trên email thật.
- Xây dựng chức năng **thời gian thực**: thông báo, kết bạn, nhắn tin và giao công việc ngay trong cuộc trò chuyện, sử dụng WebSocket (Soketi) và xử lý bất đồng bộ qua Kafka.
- Bảo đảm **bảo mật**: đăng nhập bằng Google OAuth 2.0 với PKCE, phiên đăng nhập JWT, kênh thời gian thực riêng tư được ký uỷ quyền.
- Kết quả kiểm thử: **49/49 trường hợp kiểm thử chức năng** trên hệ thống thật đạt; 409 hàm kiểm thử đơn vị backend và 174 kiểm thử giao diện đạt; độ trễ phản hồi API dưới 10 ms ở mức tải một người dùng.
- Về bản thân, em đã củng cố và vận dụng được kiến thức về kiến trúc microservices, gRPC và Protocol Buffers, xử lý bất đồng bộ với hàng đợi thông điệp, thiết kế cơ sở dữ liệu, xác thực OAuth 2.0, tích hợp trí tuệ nhân tạo và quy trình phát triển phần mềm theo nhánh, rà soát mã nguồn.

## 2. Hạn chế của đồ án

Do thời gian và kinh nghiệm còn hạn chế, đồ án vẫn còn một số điểm chưa hoàn thiện:

- Một số gateway (công việc, lịch, sự kiện, backlog, tệp đính kèm, báo cáo) chưa tự kiểm tra session token mà nhận mã người dùng từ yêu cầu; việc phân quyền đầy đủ mới áp dụng ở các dịch vụ định danh, thông báo, bạn bè và nhắn tin.
- Chức năng xử lý email chỉ hỗ trợ Gmail; chưa chống trùng theo mã thư của Gmail nên trong trường hợp hiếm (đặt lại mốc lịch sử, Pub/Sub gửi lại) có thể tạo bản ghi trùng.
- Lịch và sự kiện chưa hỗ trợ lặp lại, mời người tham gia và đồng bộ hai chiều với Google Calendar.
- Chưa có hệ thống giám sát, ghi log và truy vết tập trung cho nhiều dịch vụ; chưa thực hiện kiểm thử tải với nhiều người dùng đồng thời.
- Mã nguồn còn 20 vi phạm quy tắc kiến trúc về chú thích trong mã cần dọn; mã lỗi tệp vượt giới hạn đang trả HTTP 429 thay vì 413.
- Chưa có ứng dụng di động.

## 3. Hướng phát triển

Trong thời gian tới, hệ thống có thể được mở rộng theo các hướng sau:

- Chuyển việc xác thực về một **cổng API tập trung**, kiểm tra session token cho mọi dịch vụ; bổ sung chống trùng email theo mã thư Gmail.
- Bổ sung **giám sát và truy vết** (metrics, log tập trung, distributed tracing với OpenTelemetry) và kiểm thử tải để đánh giá khả năng mở rộng.
- Hỗ trợ thêm hộp thư Outlook, đồng bộ hai chiều với Google Calendar, lịch và sự kiện lặp lại.
- Mở rộng sang **làm việc nhóm**: dự án chung, giao việc cho nhiều người, phân quyền theo vai trò.
- Phát triển **ứng dụng di động** sử dụng lại các API hiện có, thông báo đẩy trên điện thoại.
- Triển khai lên hạ tầng đám mây với Kubernetes để tự động mở rộng các dịch vụ theo tải.

Em xin chân thành cảm ơn giảng viên hướng dẫn đã tận tình hỗ trợ và góp ý trong suốt quá trình thực hiện đồ án. Em rất mong nhận được thêm ý kiến đóng góp của giảng viên và hội đồng để hoàn thiện và nâng cao chất lượng của hệ thống trong tương lai.

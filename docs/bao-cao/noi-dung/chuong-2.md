<!-- Nguồn: docs/system/01..05, README.md, service/*/README.md, quality.json, frontend/package.json, go.mod. Số [n] khớp danh mục tài liệu tham khảo trong build_khung.py -->
# CHƯƠNG 2: CƠ SỞ LÝ THUYẾT VÀ CÔNG NGHỆ SỬ DỤNG

## 2.1. Kiến trúc Microservices

### 2.1.1. Tổng quan

Kiến trúc vi dịch vụ (microservices) là phong cách xây dựng một ứng dụng dưới dạng tập hợp các dịch vụ nhỏ, mỗi dịch vụ chạy trong một tiến trình riêng, đảm nhận một năng lực nghiệp vụ cụ thể và giao tiếp với nhau qua các cơ chế nhẹ như HTTP hoặc RPC [2]. Thuật ngữ này được sử dụng rộng rãi từ khoảng năm 2012 và được James Lewis cùng Martin Fowler hệ thống hoá trong bài viết năm 2014 [2]; Sam Newman tiếp tục phát triển thành các nguyên tắc thiết kế và vận hành trong cuốn *Building Microservices* [1].

Đối lập với microservices là **kiến trúc nguyên khối (monolith)**, trong đó toàn bộ chức năng của hệ thống được đóng gói trong một ứng dụng duy nhất, dùng chung một cơ sở dữ liệu và được triển khai như một khối. Kiến trúc nguyên khối đơn giản khi hệ thống còn nhỏ, nhưng khi số chức năng tăng lên, mọi thay đổi dù nhỏ đều phải biên dịch và triển khai lại toàn bộ ứng dụng; một lỗi ở một phân hệ có thể làm sập cả hệ thống; và không thể mở rộng riêng phần chịu tải lớn. Microservices giải quyết các vấn đề này bằng cách tách hệ thống theo ranh giới nghiệp vụ, như minh hoạ ở Hình 2.1.

[HINH: h2-1-monolith-microservices.png | Hình 2.1: So sánh kiến trúc Monolith và Microservices]

### 2.1.2. Đặc điểm và nguyên tắc

Một hệ thống microservices thường tuân theo các nguyên tắc sau [1], [2]:

- **Mỗi dịch vụ một năng lực nghiệp vụ**: dịch vụ được tổ chức quanh một miền nghiệp vụ (ví dụ: công việc, lịch, thông báo) thay vì theo tầng kỹ thuật.
- **Triển khai độc lập**: mỗi dịch vụ được xây dựng, kiểm thử và triển khai riêng mà không cần triển khai lại các dịch vụ khác.
- **Dữ liệu phi tập trung (database per service)**: mỗi dịch vụ sở hữu dữ liệu của mình; dịch vụ khác chỉ truy cập dữ liệu đó thông qua giao diện lập trình (API) của dịch vụ sở hữu.
- **Giao tiếp qua mạng**: đồng bộ (gọi và chờ kết quả, như REST hoặc gRPC) hoặc bất đồng bộ (gửi thông điệp vào hàng đợi, như Kafka).
- **Thiết kế cho lỗi**: vì lời gọi qua mạng có thể thất bại, mỗi dịch vụ phải xử lý được khi dịch vụ khác chậm hoặc không phản hồi.

Trong đồ án, hệ thống được chia thành 13 dịch vụ theo miền nghiệp vụ: định danh (identity), dự án (project), công việc (task), lịch (calendar), sự kiện (event), backlog, tệp đính kèm (attachment), báo cáo (report), thông báo (notification), xử lý thư (mail-provider), bạn bè (social), nhắn tin (message) và dịch vụ chạy nền (worker). Mỗi dịch vụ sở hữu một *schema* riêng trên PostgreSQL.

### 2.1.3. Ưu điểm

- **Phát triển và triển khai độc lập**: có thể sửa, kiểm thử và triển khai dịch vụ báo cáo mà không ảnh hưởng tới dịch vụ công việc.
- **Mở rộng theo nhu cầu**: chỉ cần tăng số bản chạy của thành phần chịu tải lớn. Trong đồ án, dịch vụ worker lưu tin nhắn được thiết kế để chạy nhiều bản song song.
- **Cô lập lỗi**: khi một dịch vụ gặp sự cố, các chức năng khác vẫn hoạt động; ví dụ dịch vụ báo cáo lỗi thì bảng Kanban và lịch vẫn dùng được.
- **Tự do công nghệ và mã nguồn dễ hiểu**: mỗi dịch vụ nhỏ, tập trung vào một nghiệp vụ nên dễ đọc, dễ bảo trì.

### 2.1.4. Nhược điểm và thách thức

- **Độ phức tạp vận hành**: nhiều tiến trình, nhiều cấu hình, cần công cụ đóng gói và điều phối (Docker, Docker Compose).
- **Nhất quán dữ liệu phân tán**: một thao tác có thể chạm dữ liệu của nhiều dịch vụ nên không thể dùng một giao dịch cơ sở dữ liệu duy nhất; hệ thống phải chấp nhận *nhất quán cuối cùng* (eventual consistency) [3].
- **Độ trễ và lỗi mạng**: lời gọi giữa các dịch vụ chậm hơn lời gọi hàm trong cùng tiến trình và có thể thất bại.
- **Khó gỡ lỗi**: một yêu cầu đi qua nhiều dịch vụ nên việc lần vết lỗi phức tạp hơn.

Đồ án xử lý các thách thức trên bằng: định nghĩa giao diện chặt chẽ bằng Protocol Buffers; coi các tác vụ phụ (gửi thông báo, khởi tạo dữ liệu mặc định) là *nỗ lực tối đa* (best-effort) để lỗi của chúng không làm hỏng thao tác chính; thiết kế các thao tác ghi *luỹ đẳng* (idempotent) để có thể xử lý lại an toàn.

## 2.2. Giao tiếp giữa các dịch vụ

### 2.2.1. gRPC và Protocol Buffers

**gRPC** là framework gọi thủ tục từ xa (Remote Procedure Call – RPC) mã nguồn mở do Google công bố năm 2015, hiện thuộc Cloud Native Computing Foundation [5]. Với gRPC, dịch vụ gọi sử dụng phương thức của dịch vụ ở xa giống như gọi một hàm cục bộ. gRPC truyền dữ liệu trên giao thức **HTTP/2**, hỗ trợ ghép kênh nhiều lời gọi trên một kết nối, nén tiêu đề và truyền luồng hai chiều.

**Protocol Buffers (Protobuf)** là ngôn ngữ định nghĩa giao diện và định dạng tuần tự hoá dữ liệu nhị phân do Google phát triển [6]. Lập trình viên mô tả các *service* (tập phương thức) và *message* (cấu trúc dữ liệu) trong tệp `.proto`; trình biên dịch sinh ra mã client và server cho nhiều ngôn ngữ. So với JSON, dữ liệu Protobuf nhỏ gọn hơn, tuần tự hoá nhanh hơn và có kiểu dữ liệu chặt chẽ. Tệp `.proto` đóng vai trò **hợp đồng** giữa các dịch vụ: thay đổi không tương thích sẽ bị phát hiện ngay khi biên dịch.

[HINH: h2-2-grpc.png | Hình 2.2: Mô hình giao tiếp gRPC client – server]

Trong đồ án, toàn bộ giao diện giữa các dịch vụ được định nghĩa tập trung trong thư mục `common/proto`, mã nguồn được sinh tự động bằng công cụ *buf*. Mọi lời gọi nội bộ giữa các dịch vụ (ví dụ dịch vụ xử lý thư gọi `CreateTask` của dịch vụ công việc) đều dùng gRPC.

### 2.2.2. RESTful API và grpc-gateway

**REST** (Representational State Transfer) là phong cách kiến trúc cho hệ thống phân tán do Roy Fielding đề xuất năm 2000 [8]. Một API theo phong cách REST coi dữ liệu là các *tài nguyên* được định danh bằng URL, thao tác bằng các phương thức HTTP chuẩn (GET để đọc, POST để tạo, PATCH để cập nhật, DELETE để xoá), không lưu trạng thái phiên ở máy chủ và thường trao đổi dữ liệu dạng JSON.

Trình duyệt không gọi trực tiếp gRPC một cách thuận tiện, vì vậy đồ án sử dụng **grpc-gateway** [7]: một trình sinh mã đọc chú thích `google.api.http` trong tệp `.proto` và sinh ra một máy chủ HTTP chuyển đổi yêu cầu REST/JSON thành lời gọi gRPC tương ứng. Nhờ đó, mỗi phương thức chỉ cần khai báo một lần trong proto nhưng được phục vụ đồng thời qua gRPC (nội bộ) và REST (cho giao diện web). Ví dụ, khai báo `post: "/v1/tasks/{id}/status"` cho phương thức `ChangeTaskStatus` tạo ra API REST `POST /v1/tasks/{id}/status`.

### 2.2.3. Hàng đợi thông điệp Apache Kafka

**Apache Kafka** là nền tảng truyền luồng sự kiện phân tán, được phát triển tại LinkedIn và trở thành dự án của Apache Software Foundation [9]. Kafka hoạt động như một nhật ký (log) chỉ ghi nối thêm, bền vững và có khả năng mở rộng ngang. Các khái niệm chính:

- **Topic**: kênh logic chứa các thông điệp cùng loại, ví dụ `chat.messages`.
- **Partition**: mỗi topic được chia thành nhiều phân vùng; thông điệp trong cùng một partition được giữ đúng thứ tự. Thông điệp có cùng *khoá* (key) luôn vào cùng một partition.
- **Producer**: thành phần ghi thông điệp vào topic.
- **Consumer group**: nhóm các consumer cùng xử lý một topic; mỗi partition chỉ được giao cho một consumer trong nhóm, nhờ đó có thể tăng số consumer để xử lý song song (tối đa bằng số partition).
- **Offset**: vị trí của thông điệp trong partition; consumer *xác nhận* (commit) offset sau khi xử lý xong.

Nếu consumer chỉ commit offset sau khi xử lý thành công, hệ thống đạt đảm bảo **ít nhất một lần** (at-least-once): thông điệp không bị mất nhưng có thể được xử lý lại, do đó thao tác xử lý cần luỹ đẳng [3].

[HINH: h2-3-kafka.png | Hình 2.3: Mô hình producer – topic/partition – consumer group của Kafka]

Trong đồ án, dịch vụ nhắn tin là producer, ghi tin nhắn vào topic `chat.messages` với khoá là mã cuộc trò chuyện, để tin nhắn của cùng một cuộc trò chuyện luôn được xử lý đúng thứ tự. Dịch vụ worker là consumer group, lưu tin nhắn vào cơ sở dữ liệu và chỉ commit sau khi ghi thành công.

### 2.2.4. WebSocket và Soketi

Giao thức HTTP truyền thống hoạt động theo mô hình hỏi – đáp: máy chủ chỉ trả dữ liệu khi trình duyệt gửi yêu cầu. Để máy chủ chủ động đẩy dữ liệu (thông báo, tin nhắn mới), cần một kết nối hai chiều luôn mở. **WebSocket** (RFC 6455) [10] là giao thức cung cấp kênh truyền song công trên một kết nối TCP, được khởi tạo bằng một yêu cầu HTTP nâng cấp (upgrade).

**Soketi** là máy chủ WebSocket mã nguồn mở, tương thích giao thức của dịch vụ Pusher Channels [11]. Mô hình hoạt động gồm: trình duyệt đăng ký (*subscribe*) vào các *kênh*; máy chủ ứng dụng gửi *sự kiện* vào kênh qua REST API của Soketi; Soketi đẩy sự kiện tới mọi trình duyệt đang nghe kênh đó. Với **kênh riêng tư** (private channel, tên bắt đầu bằng `private-`), trình duyệt phải xin chữ ký uỷ quyền từ máy chủ ứng dụng: chữ ký là mã HMAC-SHA256 của chuỗi `socket_id:tên_kênh` với khoá bí mật của ứng dụng, đảm bảo chỉ người được phép mới nghe được kênh.

Đồ án dùng hai loại kênh riêng tư cho mỗi người dùng: kênh thông báo `private-noti-internal-<mã người dùng>` và kênh tin nhắn `private-chat-<mã người dùng>`; dịch vụ định danh chỉ ký uỷ quyền cho đúng chủ kênh.

## 2.3. Ngôn ngữ Go và kiến trúc mã nguồn backend

### 2.3.1. Tổng quan ngôn ngữ Go

**Go** (Golang) là ngôn ngữ lập trình mã nguồn mở do Robert Griesemer, Rob Pike và Ken Thompson thiết kế tại Google, công bố năm 2009 [4]. Go là ngôn ngữ biên dịch, kiểu tĩnh, có bộ thu gom rác, cú pháp ngắn gọn và đặc biệt mạnh về xử lý đồng thời nhờ *goroutine* (luồng nhẹ) và *channel* (kênh trao đổi giữa các goroutine). Chương trình Go được biên dịch thành một tệp thực thi độc lập, khởi động nhanh và tốn ít bộ nhớ, rất phù hợp để đóng gói thành container cho kiến trúc microservices.

Đồ án sử dụng Go 1.25; mỗi dịch vụ là một *Go module* riêng, được gom lại bằng tệp `go.work` khi phát triển. Mỗi dịch vụ có hai chương trình: máy chủ gRPC (chứa nghiệp vụ) và máy chủ HTTP (grpc-gateway).

### 2.3.2. Dependency Injection với Uber Fx

**Tiêm phụ thuộc** (Dependency Injection – DI) là kỹ thuật trong đó một đối tượng nhận các phụ thuộc của nó từ bên ngoài thay vì tự khởi tạo, giúp các thành phần lỏng lẻo và dễ thay thế khi kiểm thử. **Uber Fx** là framework DI cho Go [12]: lập trình viên khai báo các hàm khởi tạo (*provider*), Fx tự xây dựng đồ thị phụ thuộc, khởi tạo theo đúng thứ tự và quản lý vòng đời ứng dụng thông qua các móc `OnStart`/`OnStop` (mở và đóng kết nối cơ sở dữ liệu, khởi động và dừng máy chủ một cách an toàn).

Trong đồ án, tầng `cmd` của mọi dịch vụ bắt buộc dùng Fx; đây là nơi duy nhất ghép các cài đặt cụ thể (PostgreSQL, gRPC client, S3) vào các interface mà tầng nghiệp vụ cần.

### 2.3.3. Kiến trúc phân tầng và nguyên lý đảo ngược phụ thuộc

Mã nguồn bên trong mỗi dịch vụ được tổ chức theo **kiến trúc lục giác** (Hexagonal Architecture, còn gọi là Ports and Adapters) do Alistair Cockburn đề xuất [13], cùng tinh thần với *Clean Architecture* của Robert C. Martin [14]: phần lõi nghiệp vụ không phụ thuộc vào chi tiết kỹ thuật bên ngoài (cơ sở dữ liệu, mạng); thay vào đó, lõi khai báo các *cổng* (port – interface) và các *bộ chuyển đổi* (adapter) bên ngoài cài đặt các cổng đó.

[HINH: h2-4-layered.png | Hình 2.4: Kiến trúc phân tầng bên trong một dịch vụ]

Mỗi dịch vụ trong đồ án gồm các tầng:

- **model**: đối tượng miền và lỗi miền; chỉ dùng thư viện chuẩn, không phụ thuộc tầng nào.
- **service**: nghiệp vụ; khai báo các interface cần dùng trong tệp `interfaces.go` và chỉ phụ thuộc vào `model`.
- **repository**: các adapter cài đặt interface của tầng service (truy vấn PostgreSQL, gọi dịch vụ khác qua gRPC, lưu tệp lên S3, gọi API bên ngoài).
- **dependency**: kiểm tra dữ liệu đầu vào, chuyển đổi giữa model và message Protobuf, ánh xạ lỗi sang mã lỗi chuẩn.
- **handler**: tầng giao tiếp gRPC mỏng, nhận yêu cầu, gọi service và trả kết quả.

Nhờ ngôn ngữ Go cài đặt interface một cách ngầm định (structural typing), tầng repository không cần tham chiếu tới tầng service; hướng phụ thuộc luôn đi vào lõi. Các quy tắc này được kiểm tra tự động bằng công cụ trong `tools/quality` dựa trên tệp cấu hình `quality.json`.

## 2.4. Công nghệ phía giao diện (Frontend)

Giao diện người dùng là ứng dụng một trang (Single Page Application – SPA) được xây dựng bằng các công nghệ sau:

- **React 19** [15]: thư viện JavaScript xây dựng giao diện theo mô hình thành phần (component); giao diện được mô tả dưới dạng hàm của trạng thái và React tự cập nhật phần thay đổi.
- **TypeScript**: bổ sung kiểu tĩnh cho JavaScript, phát hiện lỗi khi biên dịch.
- **Vite**: công cụ phát triển và đóng gói mã nguồn, khởi động nhanh và cập nhật nóng khi lập trình.
- **Mantine UI** [16]: thư viện thành phần giao diện (nút, bảng, hộp thoại, thông báo) cho React.
- **Redux Toolkit**: quản lý trạng thái tập trung của ứng dụng (bảng công việc, lịch, thông báo, tin nhắn).
- **React Router**: điều hướng giữa các trang mà không tải lại.
- **FullCalendar**: hiển thị lịch theo ngày, tuần, tháng và hỗ trợ kéo thả.
- **i18next**: đa ngôn ngữ (tiếng Việt, tiếng Anh).
- **pusher-js**: thư viện kết nối WebSocket tới Soketi để nhận thông báo và tin nhắn thời gian thực.
- **Vitest**: kiểm thử đơn vị cho các hàm xử lý phía giao diện.

## 2.5. Lưu trữ dữ liệu

### 2.5.1. Hệ quản trị cơ sở dữ liệu PostgreSQL

**PostgreSQL** là hệ quản trị cơ sở dữ liệu quan hệ – đối tượng mã nguồn mở, có nguồn gốc từ dự án POSTGRES tại Đại học California, Berkeley [17]. PostgreSQL tuân thủ chặt chẽ chuẩn SQL, hỗ trợ đầy đủ giao dịch ACID, khoá ngoại, ràng buộc kiểm tra (CHECK), chỉ mục đa dạng và nhiều phần mở rộng.

Đồ án sử dụng PostgreSQL 17 với các đặc điểm:

- **Mỗi dịch vụ một schema** (`identity`, `task`, `calendar`, `event`, …) trong cùng một cơ sở dữ liệu: ranh giới dữ liệu rõ ràng nhưng vận hành đơn giản.
- **Ràng buộc toàn vẹn** bằng khoá ngoại tới hồ sơ người dùng (xoá người dùng thì xoá dữ liệu liên quan) và ràng buộc CHECK (ví dụ thời gian kết thúc không sớm hơn thời gian bắt đầu).
- **Tìm kiếm công việc** bằng phần mở rộng `pg_trgm` (chỉ mục trigram) để tìm theo từ khoá nhanh.
- **Quản lý phiên bản lược đồ** bằng các tệp migration SQL đánh số tuần tự, chạy tự động khi khởi động hệ thống.
- Kết nối từ Go qua thư viện **pgx** với nhóm kết nối (connection pool).

### 2.5.2. Lưu trữ đối tượng chuẩn S3 (RustFS)

Tệp có kích thước lớn (tệp đính kèm, email gốc) không nên lưu trực tiếp trong cơ sở dữ liệu quan hệ vì làm cơ sở dữ liệu phình to, sao lưu chậm và tốn tài nguyên. Giải pháp phổ biến là **lưu trữ đối tượng** (object storage): mỗi tệp là một *đối tượng* nằm trong một *bucket* và được định danh bằng một *khoá* (object key); cơ sở dữ liệu chỉ lưu siêu dữ liệu và khoá của tệp. API của dịch vụ Amazon S3 [18] đã trở thành chuẩn giao tiếp chung cho loại lưu trữ này.

Đồ án sử dụng **RustFS**, máy chủ lưu trữ đối tượng mã nguồn mở viết bằng Rust và tương thích API S3, chạy ngay trong hệ thống Docker. Do tương thích S3, mã nguồn dùng thư viện client `minio-go` và có thể chuyển sang Amazon S3 hoặc dịch vụ tương đương chỉ bằng thay đổi cấu hình. Tệp đính kèm được lưu theo khoá `attachments/{người dùng}/{loại đối tượng}/{mã đối tượng}/…`, email gốc theo khoá `mail/{người dùng}/{ngày}/{mã thư}.json`.

## 2.6. Xác thực và phân quyền

### 2.6.1. OAuth 2.0 và OpenID Connect

**OAuth 2.0** (RFC 6749) [19] là khung uỷ quyền cho phép một ứng dụng truy cập tài nguyên của người dùng tại một dịch vụ khác (ví dụ hộp thư Gmail) mà không cần biết mật khẩu của người dùng. Người dùng đăng nhập và đồng ý cấp quyền trực tiếp tại máy chủ cấp quyền (Google); ứng dụng nhận về **access token** có thời hạn ngắn để gọi API, và **refresh token** để lấy access token mới khi hết hạn. Phạm vi quyền được xác định bằng **scope**, ví dụ `gmail.readonly` chỉ cho phép đọc thư.

**OpenID Connect** [21] là lớp định danh xây dựng trên OAuth 2.0, cho phép ứng dụng biết *người dùng là ai* (email, tên, ảnh đại diện) thông qua `id_token` và endpoint UserInfo.

Luồng được sử dụng là **Authorization Code** kết hợp **PKCE** (Proof Key for Code Exchange, RFC 7636) [20]: ứng dụng sinh một chuỗi ngẫu nhiên `code_verifier`, gửi kèm giá trị băm `code_challenge` khi chuyển hướng người dùng sang Google, và chỉ đổi được mã uỷ quyền (`code`) lấy token khi xuất trình đúng `code_verifier`. Tham số `state` được dùng để chống tấn công giả mạo yêu cầu (CSRF). Nhờ vậy, kể cả khi mã uỷ quyền bị lộ, kẻ tấn công cũng không đổi được token.

[HINH: h2-5-oauth-pkce.png | Hình 2.5: Luồng OAuth 2.0 Authorization Code với PKCE]

### 2.6.2. JSON Web Token (JWT)

**JSON Web Token** (RFC 7519) [22] là chuẩn biểu diễn các *khẳng định* (claim) giữa hai bên dưới dạng chuỗi gọn nhẹ. Một JWT gồm ba phần mã hoá Base64URL, ngăn cách bởi dấu chấm: *header* (thuật toán ký), *payload* (các claim như `sub` – định danh người dùng, `exp` – thời điểm hết hạn) và *signature* (chữ ký). Máy chủ chỉ cần kiểm tra chữ ký để tin tưởng nội dung token mà không phải tra cứu cơ sở dữ liệu, phù hợp với kiến trúc nhiều dịch vụ.

Trong đồ án, sau khi đăng nhập Google thành công, dịch vụ định danh phát hành **session token** dạng JWT ký bằng thuật toán HMAC-SHA256 (HS256) với khoá bí mật dùng chung. Các dịch vụ thông báo, bạn bè và nhắn tin tự kiểm tra chữ ký token và lấy mã người dùng từ claim `sub`, không nhận mã người dùng do trình duyệt gửi lên, nên người dùng chỉ thao tác được trên dữ liệu của chính mình.

## 2.7. Tích hợp Gmail và trí tuệ nhân tạo

### 2.7.1. Gmail API và Google Cloud Pub/Sub

**Gmail API** [23] cho phép ứng dụng đọc và quản lý hộp thư của người dùng sau khi được cấp quyền OAuth. Thay vì liên tục hỏi máy chủ xem có thư mới không (polling), Gmail hỗ trợ cơ chế **thông báo đẩy** qua **Google Cloud Pub/Sub** [24], dịch vụ truyền thông điệp theo mô hình *phát hành – đăng ký* (publish – subscribe):

- Ứng dụng gọi `users.watch` để đăng ký theo dõi hộp thư; Gmail sẽ phát một thông điệp vào *topic* Pub/Sub mỗi khi hộp thư thay đổi. Đăng ký có hạn tối đa 7 ngày và cần được gia hạn định kỳ.
- Thông điệp chỉ chứa địa chỉ email và một mốc lịch sử `historyId`; ứng dụng gọi `users.history.list` từ mốc đã lưu để lấy danh sách thư mới, sau đó gọi `users.messages.get` để lấy nội dung.
- Ứng dụng nhận thông điệp Pub/Sub theo hai cách: **push** (Pub/Sub gọi vào một địa chỉ HTTPS công khai của ứng dụng, kèm token OIDC để xác thực) hoặc **pull** (ứng dụng chủ động kéo thông điệp về).

Pub/Sub đảm bảo giao thông điệp *ít nhất một lần*, nên ứng dụng phải chịu được thông điệp trùng lặp hoặc đến sai thứ tự. Đồ án giải quyết bằng cách không tin `historyId` trong thông điệp mà luôn đọc từ *mốc đã xử lý* lưu trong cơ sở dữ liệu và chỉ tiến mốc sau khi xử lý xong.

### 2.7.2. Mô hình ngôn ngữ lớn (LLM) và DeepSeek

**Mô hình ngôn ngữ lớn** (Large Language Model – LLM) là mạng nơ-ron học sâu, phần lớn dựa trên kiến trúc **Transformer** [25], được huấn luyện trên lượng văn bản rất lớn để dự đoán từ tiếp theo. Nhờ đó, LLM có khả năng hiểu và sinh ngôn ngữ tự nhiên, có thể thực hiện nhiều nhiệm vụ như tóm tắt, trích xuất thông tin, phân loại văn bản chỉ thông qua *lời nhắc* (prompt) mà không cần huấn luyện lại.

**DeepSeek** là công ty trí tuệ nhân tạo cung cấp các mô hình ngôn ngữ lớn qua API có định dạng tương thích với API của OpenAI [26]. Đồ án gửi tiêu đề, đoạn trích và nội dung email tới mô hình `deepseek-chat` cùng một lời nhắc hệ thống mô tả rõ bốn nhóm phân loại (`task` – công việc, `schedule` – lịch hẹn, `event` – sự kiện, `other` – khác) và yêu cầu mô hình trả về **duy nhất một đối tượng JSON** gồm nhóm phân loại, tiêu đề, mô tả, độ ưu tiên, các mốc thời gian, địa điểm và lý do. Việc ép định dạng JSON giúp kết quả của LLM được xử lý tự động một cách tin cậy.

LLM có thể trả lời sai, chậm hoặc không khả dụng. Vì vậy đồ án thiết kế **cơ chế dự phòng**: khi không cấu hình khoá API, khi gọi lỗi hoặc khi kết quả không hợp lệ, hệ thống chuyển sang **bộ luật từ khoá** (chấm điểm theo các từ khoá tiếng Việt và tiếng Anh như "hội thảo", "cuộc họp", "deadline", "hoá đơn"). Email không đủ thông tin để tạo bản ghi được đưa vào backlog kèm lý do, đảm bảo không email nào bị mất.

## 2.8. Công nghệ triển khai

- **Docker** [27]: nền tảng đóng gói ứng dụng cùng toàn bộ thư viện và cấu hình cần thiết vào *container*, giúp ứng dụng chạy giống nhau trên mọi máy. Mỗi dịch vụ Go được đóng gói thành một *image* chứa hai chương trình gRPC và HTTP.
- **Docker Compose**: công cụ khai báo và khởi chạy nhiều container cùng lúc bằng một tệp YAML. Toàn bộ hệ thống (cơ sở dữ liệu, Kafka, Soketi, RustFS, 13 dịch vụ và giao diện) được khởi động bằng một lệnh; số bản chạy của dịch vụ worker có thể thay đổi bằng tham số `--scale`.
- **nginx** [28]: máy chủ web và *reverse proxy*; trong đồ án, nginx vừa phục vụ giao diện web, vừa là **cổng vào duy nhất** chuyển tiếp các yêu cầu `/api/<dịch vụ>/…` tới đúng dịch vụ bên trong, nên chỉ cần công khai một cổng mạng.
- **Cloudflare Tunnel** [29]: tạo đường hầm mã hoá từ máy chạy hệ thống tới mạng Cloudflare, cho phép truy cập hệ thống qua tên miền HTTPS mà không cần mở cổng mạng. Đây là điều kiện để Google Pub/Sub gửi thông báo đẩy về hệ thống.

## 2.9. Tổng hợp công nghệ sử dụng

Bảng 2.1 tổng hợp các công nghệ chính được sử dụng trong đồ án cùng phiên bản và vai trò.

[BANG: Bảng 2.1: Các công nghệ sử dụng trong đồ án]
| STT | Thành phần | Công nghệ / phiên bản | Vai trò |
|---|---|---|---|
| 1 | Ngôn ngữ backend | Go 1.25 | Viết toàn bộ 13 dịch vụ |
| 2 | Giao tiếp nội bộ | gRPC 1.83, Protocol Buffers, buf | Gọi thủ tục từ xa giữa các dịch vụ |
| 3 | API cho giao diện | grpc-gateway v2.30 | Chuyển REST/JSON thành gRPC |
| 4 | Dependency Injection | Uber Fx 1.24 | Ghép phụ thuộc, quản lý vòng đời dịch vụ |
| 5 | Cơ sở dữ liệu | PostgreSQL 17, pgx v5 | Lưu dữ liệu nghiệp vụ, mỗi dịch vụ một schema |
| 6 | Lưu trữ đối tượng | RustFS, minio-go v7 | Lưu tệp đính kèm, email gốc (chuẩn S3) |
| 7 | Hàng đợi thông điệp | Apache Kafka 3.9, kafka-go | Xử lý bất đồng bộ tin nhắn |
| 8 | Thời gian thực | Soketi 1.6, pusher-js 8 | Đẩy thông báo, tin nhắn tới trình duyệt |
| 9 | Xác thực | Google OAuth 2.0 / OIDC, JWT (golang-jwt v5) | Đăng nhập, phiên làm việc |
| 10 | Tích hợp email | Gmail API, Google Cloud Pub/Sub | Theo dõi và đọc thư mới |
| 11 | Trí tuệ nhân tạo | DeepSeek API (deepseek-chat) | Phân loại email |
| 12 | Công việc lặp lại | rrule-go | Tính lần lặp tiếp theo theo quy tắc RRULE |
| 13 | Giao diện | React 19, TypeScript, Vite, Mantine 9 | Ứng dụng web một trang |
| 14 | Trạng thái, điều hướng | Redux Toolkit 2, React Router 7 | Quản lý trạng thái, chuyển trang |
| 15 | Lịch, đa ngôn ngữ | FullCalendar 6, i18next | Hiển thị lịch, tiếng Việt/tiếng Anh |
| 16 | Cổng vào | nginx | Phục vụ giao diện, chuyển tiếp API |
| 17 | Triển khai | Docker, Docker Compose, Cloudflare Tunnel | Đóng gói, khởi chạy, công khai hệ thống |
| 18 | Kiểm thử, chất lượng | go test, Vitest, gofmt, go vet | Kiểm thử đơn vị, kiểm tra mã nguồn |

## 2.10. Nguyên tắc thiết kế hệ thống

### 2.10.1. Nguyên tắc SOLID

SOLID là tập năm nguyên tắc thiết kế hướng đối tượng do Robert C. Martin tổng hợp [14], được áp dụng trong đồ án như sau:

- **S – Đơn trách nhiệm** (Single Responsibility): mỗi dịch vụ đảm nhận một miền nghiệp vụ; bên trong, mỗi tầng một trách nhiệm (handler chỉ giao tiếp, service chỉ nghiệp vụ, repository chỉ truy cập dữ liệu).
- **O – Mở để mở rộng, đóng để sửa đổi** (Open/Closed): thêm nguồn lưu trữ mới chỉ cần viết adapter mới cài đặt interface sẵn có, không sửa tầng nghiệp vụ; ví dụ cùng một interface lưu trữ có cả bản PostgreSQL và bản trong bộ nhớ.
- **L – Thay thế Liskov** (Liskov Substitution): mọi cài đặt của một interface (PostgreSQL, bộ nhớ, client giả khi kiểm thử) đều thay thế được cho nhau mà không làm sai nghiệp vụ.
- **I – Phân tách interface** (Interface Segregation): tầng nghiệp vụ khai báo các interface nhỏ đúng nhu cầu, ví dụ `ProjectChecker` chỉ có một phương thức kiểm tra dự án.
- **D – Đảo ngược phụ thuộc** (Dependency Inversion): tầng nghiệp vụ phụ thuộc vào interface do chính nó khai báo; cài đặt cụ thể được tiêm vào ở tầng `cmd` bằng Uber Fx.

### 2.10.2. Thiết kế RESTful API

API của hệ thống tuân theo các quy ước: đường dẫn theo tài nguyên và có phiên bản (`/v1/tasks`, `/v1/schedules/{id}`); dùng đúng phương thức HTTP cho từng thao tác; dữ liệu JSON với tên trường dạng *lowerCamelCase*; lỗi trả về theo **bộ mã lỗi chuẩn** gồm mã lỗi nghiệp vụ (ví dụ `TASK_TRANSITION_NOT_ALLOWED`) để giao diện hiển thị thông báo thân thiện theo ngôn ngữ của người dùng thay vì thông điệp kỹ thuật.

### 2.10.3. Chuẩn hoá cơ sở dữ liệu

Chuẩn hoá cơ sở dữ liệu, bắt nguồn từ mô hình quan hệ của E. F. Codd [30], là quá trình tổ chức bảng để giảm dư thừa và tránh bất thường khi cập nhật. Các dạng chuẩn thường dùng gồm: dạng chuẩn 1 (giá trị nguyên tố), dạng chuẩn 2 (thuộc tính phụ thuộc đầy đủ vào khoá) và dạng chuẩn 3 (không phụ thuộc bắc cầu). Đồ án áp dụng bằng cách tách trạng thái công việc, luật chuyển trạng thái và lịch sử chuyển trạng thái thành các bảng riêng thay vì lưu lặp trong bảng công việc; tách hồ sơ người dùng và liên kết tài khoản Google; dùng bảng tệp đính kèm chung với liên kết đa hình (`owner_type`, `owner_id`) cho nhiều loại đối tượng.

### 2.10.4. Tính luỹ đẳng và nhất quán cuối cùng

Một thao tác là **luỹ đẳng** (idempotent) nếu thực hiện nhiều lần cho cùng kết quả như thực hiện một lần [3]. Trong hệ thống phân tán, thông điệp có thể bị gửi lại (Kafka, Pub/Sub đều đảm bảo *ít nhất một lần*), nên các thao tác xử lý thông điệp phải luỹ đẳng. Đồ án áp dụng nguyên tắc này ở nhiều nơi: worker ghi tin nhắn với `ON CONFLICT DO NOTHING` theo mã tin nhắn nên xử lý lại không tạo bản trùng; thao tác tạo bộ trạng thái mặc định chỉ bổ sung phần còn thiếu; bộ xử lý email chỉ tiến mốc lịch sử sau khi xử lý xong.

**Nhất quán cuối cùng** (eventual consistency) chấp nhận dữ liệu giữa các thành phần có thể lệch nhau trong thời gian ngắn nhưng sẽ hội tụ về trạng thái đúng. Ví dụ, tin nhắn được hiển thị ngay cho người nhận qua WebSocket rồi mới được lưu vào cơ sở dữ liệu sau vài mili giây thông qua Kafka; đổi lại, người dùng nhận tin nhắn tức thì và hệ thống chịu tải tốt hơn.

/* Kịch bản mô phỏng các luồng (trừ luồng mail, nằm ở mail.js).
   Mỗi bước: S(từ, tới, nhãn dễ hiểu, loại, giải thích dễ hiểu, chi tiết kỹ thuật, tuỳ chọn).
   Chi tiết kỹ thuật (API, bảng dữ liệu, mã lỗi) hiển thị trong mục thu gọn. */
(function () {
  'use strict';

  const S = (f, t, label, kind, detail, tech, extra) => Object.assign({ f, t, label, kind, detail, tech }, extra || {});

  const STATUSES = [
    { slug: 'todo', name: 'To Do', vi: 'Cần làm', color: '#9aa0ae', category: 'TODO' },
    { slug: 'on-progress', name: 'On Progress', vi: 'Đang làm', color: '#f59f00', category: 'IN_PROGRESS' },
    { slug: 'in-review', name: 'In Review', vi: 'Chờ duyệt', color: '#37b24d', category: 'IN_PROGRESS' },
    { slug: 'completed', name: 'Completed', vi: 'Hoàn thành', color: '#f06595', category: 'DONE' },
  ];
  const TRANSITIONS = [
    ['todo', 'on-progress'], ['on-progress', 'in-review'], ['in-review', 'completed'],
    ['in-review', 'on-progress'], ['completed', 'on-progress'],
  ];
  const statusName = (slug) => STATUSES.find((s) => s.slug === slug).name;

  const SCENARIOS = [];

  // ------------------------------------------------------------------ Đăng nhập
  SCENARIOS.push({
    id: 'login', group: 'Tài khoản', icon: '🔐', title: 'Đăng nhập bằng Google',
    desc: 'Bạn bấm "Đăng nhập bằng Google". Google xác nhận bạn là ai, hệ thống tạo tài khoản (nếu là lần đầu) và chuẩn bị sẵn dữ liệu ban đầu. Nếu bạn cho phép, hệ thống bắt đầu theo dõi Gmail để tự tạo việc từ email.',
    lanes: ['browser', 'nginx', 'identity', 'google', 'pg', 'task', 'project', 'mail', 'gmail', 'soketi'],
    options: [
      { id: 'isNew', type: 'checkbox', label: 'Đây là lần đầu bạn đăng nhập', value: true },
      { id: 'gmail', type: 'checkbox', label: 'Bạn tích "Cho phép đọc Gmail"', value: true },
    ],
    build(o) {
      const st = [
        S('browser', 'nginx', 'Bấm "Đăng nhập bằng Google"', 'http', 'Trình duyệt gửi yêu cầu đăng nhập vào hệ thống.', `GET /api/identity/v1/auth/google/login${o.gmail ? '?gmail=1' : ''}`),
        S('nginx', 'identity', 'Chuyển yêu cầu tới bộ phận Tài khoản', 'http', 'Cổng vào thấy đây là việc đăng nhập nên chuyển cho bộ phận Tài khoản.', 'nginx proxy /api/identity/ → identity:8081', { ret: 'lệnh chuyển sang trang Google' }),
        S('identity', 'browser', 'Chuyển bạn sang trang đăng nhập của Google', 'http', 'Hệ thống không giữ mật khẩu Google của bạn — việc đăng nhập do chính Google làm.' + (o.gmail ? ' Kèm theo xin thêm quyền đọc Gmail.' : ''), ['302 → accounts.google.com', 'state ký HMAC + PKCE', o.gmail ? 'scope gmail.readonly, access_type=offline' : 'scope openid email profile'], { reply: true }),
        S('browser', 'google', 'Bạn chọn tài khoản và bấm "Đồng ý"', 'ext', 'Bạn đăng nhập ngay trên trang của Google.'),
        S('google', 'browser', 'Google xác nhận xong, đưa bạn quay lại', 'ext', 'Google trả bạn về hệ thống kèm một "mã xác nhận" dùng một lần.', '302 /v1/auth/google/callback?code&state', { reply: true }),
        S('browser', 'identity', 'Mang mã xác nhận về cho hệ thống', 'http', 'Trình duyệt đưa mã xác nhận cho bộ phận Tài khoản.', 'GET /v1/auth/google/callback'),
        S('identity', 'google', 'Đổi mã xác nhận lấy quyền truy cập', 'ext', 'Bộ phận Tài khoản hỏi lại Google để chắc chắn mã là thật, và nhận về quyền truy cập' + (o.gmail ? ' (gồm cả quyền đọc Gmail lâu dài)' : '') + '.', 'POST oauth2.googleapis.com/token (code + PKCE verifier)', { ret: 'quyền truy cập' }),
        S('identity', 'google', 'Hỏi thông tin cá nhân của bạn', 'ext', 'Lấy email, tên, ảnh đại diện từ Google.', 'GET openidconnect.googleapis.com/v1/userinfo', { ret: 'email, tên, ảnh' }),
        S('identity', 'identity', 'Tìm hoặc tạo tài khoản của bạn', 'self', 'Kiểm tra email này đã có tài khoản chưa.', 'LoginWithProvider (gRPC identity-grpc :9081)'),
      ];
      if (o.isNew) {
        st.push(
          S('identity', 'pg', 'Lưu tài khoản mới', 'db', 'Email chưa có trong hệ thống nên tạo hồ sơ mới cho bạn.', ['C identity.profiles', 'C identity.auth_providers']),
          S('identity', 'task', 'Nhờ tạo sẵn các cột trạng thái', 'grpc', 'Người mới cần sẵn các cột Cần làm / Đang làm / Chờ duyệt / Hoàn thành để dùng ngay.', 'SeedDefaultStatuses(profile_id)', { ret: 'xong' }),
          S('task', 'pg', 'Lưu 4 trạng thái và luật chuyển trạng thái', 'db', 'Ví dụ luật: chỉ được chuyển "Chờ duyệt" sang "Hoàn thành", không nhảy thẳng từ "Cần làm".', ['C task.task_statuses', 'C task.status_transitions']),
          S('identity', 'project', 'Nhờ tạo sẵn dự án "Chung"', 'grpc', 'Mọi người dùng có một dự án mặc định tên "Chung".', 'SeedDefaultProject(profile_id)', { ret: 'xong' }),
          S('project', 'pg', 'Lưu dự án "Chung"', 'db', 'Dự án này không thể bị lưu trữ (archive).', 'C project.projects (is_default)'),
        );
      } else {
        st.push(S('identity', 'pg', 'Tìm thấy tài khoản cũ, ghi lại lần đăng nhập', 'db', 'Bạn đã có tài khoản nên chỉ cập nhật thời điểm đăng nhập.', ['R identity.auth_providers', 'U identity.profiles.last_login_at']));
      }
      if (o.gmail) {
        st.push(
          S('identity', 'mail', 'Giao quyền đọc Gmail cho Bộ xử lý email', 'grpc', 'Vì bạn đã cho phép, bộ phận Tài khoản chuyển quyền đọc Gmail sang Bộ xử lý email giữ.', 'Subscribe(profile_id, email, access_token, refresh_token)', { ret: 'xong' }),
          S('mail', 'gmail', 'Đăng ký: "có thư mới thì báo cho tôi"', 'ext', 'Từ giờ mỗi khi hộp thư đến có thay đổi, Google sẽ báo cho hệ thống. Đăng ký này có hạn 7 ngày và được tự gia hạn.', 'users.watch(topic MAIL_PUBSUB_TOPIC)', { ret: 'mốc hộp thư hiện tại' }),
          S('mail', 'pg', 'Ghi nhớ quyền truy cập và mốc hộp thư', 'db', 'Mốc này giúp lần sau biết thư nào là thư mới.', ['UPSERT mail_provider.sessions', 'UPSERT mail_provider.noti_indexes (historyId)']),
        );
      }
      st.push(
        S('identity', 'browser', 'Đăng nhập thành công — cấp "vé vào cửa"', 'http', 'Trình duyệt nhận một vé (phiên đăng nhập). Các lần thao tác sau chỉ cần đưa vé này ra, không phải đăng nhập lại.', ['302 /auth/callback#token=<JWT>', 'JWT HS256, ký bằng SESSION_SECRET, sub = profile_id'], { reply: true }),
        S('browser', 'identity', 'Lấy hồ sơ của bạn để hiển thị', 'http', 'Hiện tên và ảnh đại diện ở góc màn hình.', 'GET /v1/auth/me (Bearer token)', { ret: 'hồ sơ' }),
        S('browser', 'identity', 'Xin phép nghe thông báo riêng của bạn', 'http', 'Mỗi người có một kênh thông báo riêng; chỉ chủ kênh mới được nghe.', ['POST /v1/auth/soketi', 'auth = HMAC_SHA256(app_secret, socket_id:channel)'], { ret: 'giấy phép' }),
        S('browser', 'soketi', 'Mở kênh thông báo tức thì', 'ws', 'Từ đây chuông thông báo sẽ tự sáng lên khi có tin, không cần tải lại trang.', 'subscribe private-noti-internal-<profile_id>'),
      );
      return st;
    },
  });

  // ------------------------------------------------------------------ Tạo task
  SCENARIOS.push({
    id: 'create-task', group: 'Công việc', icon: '📝', title: 'Tạo công việc mới',
    desc: 'Bạn bấm "Thêm công việc" trên bảng. Hệ thống kiểm tra dự án còn dùng được không rồi mới lưu. Bạn có thể đính kèm thêm file.',
    lanes: ['browser', 'nginx', 'task', 'project', 'attachment', 'pg', 's3'],
    options: [
      { id: 'archived', type: 'checkbox', label: 'Dự án đã bị lưu trữ (archive)', value: false },
      { id: 'attach', type: 'checkbox', label: 'Đính kèm 1 file', value: true },
    ],
    build(o) {
      const st = [
        S('browser', 'nginx', 'Bấm "Lưu" công việc mới', 'http', 'Gửi tiêu đề, dự án, độ ưu tiên, hạn chót… lên hệ thống.', 'POST /api/task/v1/tasks {profile_id, project_id, title, priority, due_at}'),
        S('nginx', 'task', 'Chuyển tới bộ phận Công việc', 'grpc', 'Cổng vào chuyển yêu cầu cho bộ phận Công việc.', 'task:8082 → gRPC CreateTask (:9082)'),
        S('task', 'project', 'Hỏi: dự án này còn dùng được không?', 'grpc', 'Bộ phận Công việc phải đợi câu trả lời: nếu dự án của người khác hoặc đã bị lưu trữ thì không được thêm việc vào.', 'ProjectService.GetProject', { ret: 'dự án hợp lệ' }),
        S('project', 'pg', 'Tra thông tin dự án', 'db', null, 'R project.projects'),
      ];
      if (o.archived) {
        st.push(
          S('project', 'task', 'Dự án đã bị lưu trữ', 'error', 'Dự án báo lại: không được thêm việc mới.', 'is_archived = true', { reply: true }),
          S('task', 'browser', 'Báo lỗi: dự án đã lưu trữ', 'error', 'Màn hình hiện thông báo đỏ. Không có gì được lưu.', 'PROJECT_ARCHIVED (common/errorcode)', { reply: true }),
        );
        return st;
      }
      st.push(
        S('task', 'pg', 'Lấy trạng thái mặc định ("To Do")', 'db', 'Công việc mới luôn bắt đầu ở cột "To Do".', 'R task.task_statuses (is_default)'),
        S('task', 'pg', 'Lưu công việc và ghi lịch sử', 'db', 'Lưu công việc, đồng thời ghi lại "đã tạo lúc nào" để tra cứu sau.', ['C task.tasks', 'C task.task_status_logs']),
        S('task', 'browser', 'Lưu xong — gửi công việc mới về', 'http', 'Thẻ công việc mới hiện ra ở cột "To Do".', '200 {task}', { reply: true }),
      );
      if (o.attach) {
        st.push(
          S('browser', 'nginx', 'Tải file đính kèm lên', 'http', 'Gửi file và cho biết file này thuộc công việc vừa tạo.', 'POST /api/attachment/v1/attachments (multipart, owner_type=task)'),
          S('nginx', 'attachment', 'Chuyển tới bộ phận Tệp đính kèm', 'grpc', null, 'attachment:8089 → gRPC'),
          S('attachment', 's3', 'Cất file vào kho', 'db', 'Nội dung file được cất trong kho lưu tệp.', 'S3 PUT attachments/{profile}/task/{id}/{uuid}.ext'),
          S('attachment', 'pg', 'Ghi lại tên file, dung lượng, thuộc công việc nào', 'db', null, 'C attachment.attachments'),
          S('attachment', 'browser', 'Tải lên xong', 'http', 'File hiện trong chi tiết công việc.', '200 {attachment}', { reply: true }),
        );
      }
      return st;
    },
  });

  // ------------------------------------------------------------------ Đổi trạng thái
  SCENARIOS.push({
    id: 'status', group: 'Công việc', icon: '🔀', title: 'Kéo thẻ sang cột khác',
    desc: 'Bạn kéo một thẻ công việc sang cột khác trên bảng. Hệ thống kiểm tra việc chuyển đó có được phép không (ví dụ không được nhảy thẳng từ "To Do" sang "Completed"). Nếu là việc lặp lại và vừa hoàn thành, hệ thống tự tạo lần tiếp theo.',
    lanes: ['browser', 'nginx', 'task', 'pg'],
    options: [
      { id: 'from', type: 'select', label: 'Thẻ đang ở cột', value: 'in-review', options: STATUSES.map((s) => [s.slug, s.name]) },
      { id: 'to', type: 'select', label: 'Kéo sang cột', value: 'completed', options: STATUSES.map((s) => [s.slug, s.name]) },
      { id: 'recurring', type: 'checkbox', label: 'Đây là việc lặp lại (VD: mỗi tuần)', value: true },
    ],
    extraTitle: 'Bảng công việc',
    extra(o) {
      const allowed = TRANSITIONS.some(([a, b]) => a === o.from && b === o.to);
      const cols = STATUSES.map((s) => `<div class="col ${s.slug === o.from ? 'from' : ''} ${s.slug === o.to ? 'to' : ''}" style="--c:${s.color}"><b>${s.name}</b>${s.vi}${s.slug === o.from ? '<div>⬅ thẻ đang ở đây</div>' : ''}${s.slug === o.to ? '<div>➡ thả vào đây</div>' : ''}</div>`).join('');
      const rules = TRANSITIONS.map(([a, b]) => `<span class="chip ${a === o.from && b === o.to ? 'hit' : ''}">${statusName(a)} → ${statusName(b)}</span>`).join('');
      return `<div class="mini-kanban">${cols}</div>
        <p style="margin:10px 0 4px"><b>Những lần chuyển được phép:</b></p>${rules}
        <p style="margin:8px 0 0">Kết quả: ${o.from === o.to ? '<span class="chip">thả lại chỗ cũ — không có gì xảy ra</span>' : allowed ? '<span class="chip ok">✓ được phép</span>' : '<span class="chip bad">✗ không được phép — thẻ quay về cột cũ</span>'}</p>`;
    },
    build(o) {
      const allowed = TRANSITIONS.some(([a, b]) => a === o.from && b === o.to);
      const toDone = STATUSES.find((s) => s.slug === o.to).category === 'DONE';
      if (o.from === o.to) {
        return [S('browser', 'browser', 'Thả thẻ lại đúng cột cũ — không cần làm gì', 'self', 'Không có gì thay đổi nên trình duyệt không gửi yêu cầu nào.')];
      }
      const st = [
        S('browser', 'nginx', `Kéo thẻ từ "${statusName(o.from)}" sang "${statusName(o.to)}"`, 'http', 'Trình duyệt báo cho hệ thống biết bạn muốn đổi trạng thái công việc.', `POST /api/task/v1/tasks/{id}/status {status_id: "${o.to}"}`),
        S('nginx', 'task', 'Chuyển tới bộ phận Công việc', 'grpc', null, 'gRPC ChangeTaskStatus'),
        S('task', 'pg', 'Lấy thông tin công việc và cột đích', 'db', null, ['R task.tasks', 'R task.task_statuses']),
        S('task', 'pg', `Tra luật: có được chuyển "${statusName(o.from)}" → "${statusName(o.to)}" không?`, 'db', 'Mỗi người có một bộ luật chuyển trạng thái riêng.', 'R task.status_transitions (from → to, is_active)'),
      ];
      if (!allowed) {
        st.push(
          S('task', 'task', 'Không có luật cho phép → từ chối', 'decision', 'Không được nhảy cóc trạng thái.'),
          S('task', 'browser', 'Báo lỗi: không được chuyển như vậy', 'error', 'Màn hình hiện thông báo đỏ và thẻ tự quay về cột cũ.', 'TASK_TRANSITION_NOT_ALLOWED', { reply: true }),
        );
        return st;
      }
      st.push(
        S('task', 'task', 'Được phép chuyển', 'decision'),
        S('task', 'pg', 'Cập nhật trạng thái' + (toDone ? ' + ghi giờ hoàn thành' : ''), 'db', toDone ? 'Vì đây là cột "Hoàn thành" nên ghi lại thời điểm xong việc.' : null, 'U task.tasks (status_id' + (toDone ? ', completed_at' : '') + ')'),
        S('task', 'pg', 'Ghi lịch sử thay đổi', 'db', 'Để sau này xem được ai đổi gì, lúc nào.', 'C task.task_status_logs'),
      );
      if (toDone && o.recurring) {
        st.push(S('task', 'pg', 'Tự tạo lần lặp tiếp theo', 'db', 'Việc lặp lại vừa xong ⇒ hệ thống tính ngày của lần kế tiếp và tạo sẵn một thẻ mới ở cột "To Do".', ['SpawnNextOccurrence (RRULE)', 'C task.tasks', 'U recurrence_next_id']));
      }
      st.push(S('task', 'browser', 'Đổi xong — cập nhật bảng', 'http', toDone && o.recurring ? 'Thẻ nằm ở cột mới, và thẻ của lần tiếp theo xuất hiện ở "To Do".' : 'Thẻ nằm yên ở cột mới.', toDone && o.recurring ? '200 {task, next_task}' : '200 {task}', { reply: true }));
      return st;
    },
  });

  // ------------------------------------------------------------------ Nhắc việc
  SCENARIOS.push({
    id: 'reminder', group: 'Công việc', icon: '⏰', title: 'Nhắc việc đúng giờ',
    desc: 'Bạn đặt giờ nhắc cho một công việc. Một "đồng hồ" chạy ngầm trong hệ thống cứ vài giây lại xem có việc nào đến giờ nhắc chưa, rồi gửi thông báo cho bạn. Nếu gửi lỗi thì lần sau thử lại, không bỏ sót.',
    lanes: ['task', 'pg', 'notification', 'soketi', 'browser'],
    options: [{ id: 'fail', type: 'checkbox', label: 'Lần gửi thông báo đầu tiên bị lỗi', value: false }],
    build(o) {
      const st = [
        S('task', 'task', 'Đồng hồ ngầm điểm giờ', 'self', 'Không ai bấm gì cả — bộ phận Công việc tự kiểm tra định kỳ.', 'ReminderWorker (ticker trong task-grpc)'),
        S('task', 'pg', 'Tìm và "giữ chỗ" các việc đã đến giờ nhắc', 'db', 'Đánh dấu luôn là "đang nhắc" để không nhắc trùng một việc hai lần.', 'ClaimDueReminders: remind_at <= now AND reminded_at IS NULL → set reminded_at'),
        S('task', 'notification', 'Nhờ gửi thông báo "Đến giờ làm việc này"', 'grpc', 'Phải đợi kết quả: gửi được thì xong, lỗi thì bỏ đánh dấu để lần sau gửi lại.', 'CreateNotice(TASK_REMINDER, target TASK)', { ret: 'đã tạo thông báo' }),
      ];
      if (o.fail) {
        st.push(
          S('notification', 'task', 'Gửi thất bại', 'error', 'Bộ phận Thông báo đang gặp sự cố.', 'gRPC Unavailable', { reply: true }),
          S('task', 'pg', 'Bỏ đánh dấu để lần sau thử lại', 'db', 'Nhờ vậy lời nhắc không bị mất.', 'ReleaseReminder: reminded_at = NULL'),
          S('task', 'task', 'Lần điểm giờ tiếp theo', 'self'),
          S('task', 'pg', 'Tìm lại — thấy việc vẫn chưa được nhắc', 'db', null, 'ClaimDueReminders'),
          S('task', 'notification', 'Nhờ gửi thông báo lần nữa', 'grpc', null, 'CreateNotice(TASK_REMINDER)', { ret: 'đã tạo thông báo' }),
        );
      }
      st.push(
        S('notification', 'pg', 'Lưu thông báo', 'db', 'Để bạn xem lại trong danh sách thông báo.', 'C notification.notices'),
        S('notification', 'soketi', 'Phát thông báo lên kênh riêng của bạn', 'http', null, 'POST /apps/{id}/events notice.created → private-noti-internal-<profile_id>', { ret: 'đã phát' }),
        S('soketi', 'browser', 'Chuông sáng lên trên màn hình', 'ws', 'Số thông báo chưa đọc tăng thêm 1.', 'FE gọi lại GET /v1/notices/unread-count'),
      );
      return st;
    },
  });

  // ------------------------------------------------------------------ Kết bạn
  SCENARIOS.push({
    id: 'friend', group: 'Liên lạc', icon: '🤝', title: 'Kết bạn',
    desc: 'Bạn (A) nhập email của B để gửi lời mời kết bạn. B nhận thông báo ngay lập tức và bấm Chấp nhận; bạn lại nhận được thông báo. Từ đó hai người nhắn tin được cho nhau.',
    lanes: ['browser', 'social', 'identity', 'pg', 'notification', 'soketi', 'peer'],
    options: [
      { id: 'found', type: 'checkbox', label: 'Email của B có tài khoản trong hệ thống', value: true },
      { id: 'accept', type: 'checkbox', label: 'B bấm "Chấp nhận"', value: true },
    ],
    build(o) {
      const st = [
        S('browser', 'social', 'Gửi lời mời kết bạn tới email của B', 'http', 'Đi qua Cổng vào tới bộ phận Bạn bè.', 'POST /api/social/v1/friend-requests {email} (JWT, requester = sub)', { ret: 'đã gửi lời mời' }),
        S('social', 'identity', 'Hỏi: email này là của ai?', 'grpc', 'Bộ phận Bạn bè không giữ danh sách người dùng nên phải hỏi bộ phận Tài khoản và đợi trả lời.', 'FindProfileByEmail', { ret: 'thông tin của B' }),
      ];
      if (!o.found) {
        st.push(
          S('identity', 'social', 'Không tìm thấy ai có email này', 'error', null, null, { reply: true }),
          S('social', 'browser', 'Báo: không có người dùng với email này', 'error', 'Không có lời mời nào được tạo.', null, { reply: true }),
        );
        return st;
      }
      st.push(
        S('social', 'pg', 'Lưu lời mời (đang chờ B trả lời)', 'db', 'Không thể tự kết bạn với chính mình, và mỗi cặp chỉ có một lời mời.', 'C social.friendships (status = pending)'),
        S('social', 'notification', 'Nhờ báo cho B: "A muốn kết bạn"', 'grpc', 'Nếu gửi thông báo lỗi thì lời mời vẫn còn, chỉ là B không thấy chuông.', 'CreateNotice(FRIEND_REQUEST → B)', { ret: 'đã tạo thông báo' }),
        S('notification', 'pg', 'Lưu thông báo cho B', 'db', null, 'C notification.notices'),
        S('notification', 'soketi', 'Phát lên kênh riêng của B', 'http', null, 'publish notice.created', { ret: 'đã phát' }),
        S('soketi', 'peer', 'Chuông của B sáng lên', 'ws', 'B thấy lời mời ngay lập tức.'),
      );
      if (o.accept) {
        st.push(
          S('peer', 'social', 'B bấm "Chấp nhận"', 'http', 'Đây là một yêu cầu mới, riêng của B.', 'POST /v1/friend-requests/{id}/accept', { ret: 'đã là bạn bè' }),
          S('social', 'pg', 'Ghi nhận: A và B đã là bạn', 'db', null, 'U social.friendships (status = accepted)'),
          S('social', 'notification', 'Nhờ báo cho A: "B đã đồng ý"', 'grpc', null, 'CreateNotice(FRIEND_ACCEPTED → A)', { ret: 'đã tạo thông báo' }),
          S('notification', 'soketi', 'Phát lên kênh riêng của A', 'http', null, 'publish notice.created', { ret: 'đã phát' }),
          S('soketi', 'browser', 'Chuông của A sáng lên', 'ws', 'Từ giờ A và B nhắn tin được cho nhau.'),
        );
      }
      return st;
    },
  });

  // ------------------------------------------------------------------ Nhắn tin
  SCENARIOS.push({
    id: 'chat', group: 'Liên lạc', icon: '💬', title: 'Nhắn tin',
    desc: 'Bạn gửi tin cho một người bạn. Tin hiện ngay trên màn hình người nhận, còn việc cất vào cơ sở dữ liệu được làm sau bởi một bộ phận chạy ngầm — nhờ vậy nhắn tin rất nhanh mà không bao giờ mất hay trùng tin.',
    lanes: ['browser', 'message', 'social', 'soketi', 'peer', 'kafka', 'worker', 'pg'],
    options: [
      { id: 'friends', type: 'checkbox', label: 'Hai người đã là bạn bè', value: true },
      { id: 'crash', type: 'checkbox', label: 'Bộ lưu tin nhắn bị sập giữa chừng', value: false },
    ],
    build(o) {
      const st = [
        S('browser', 'message', 'Bấm "Gửi" tin nhắn', 'http', 'Đi qua Cổng vào tới bộ phận Tin nhắn.', 'POST /api/message/v1/conversations/{peer}/messages {body, client_id} (JWT)'),
        S('message', 'message', 'Nhận tin cần gửi', 'self', null, 'gRPC SendMessage (sender = sub)'),
        S('message', 'social', 'Hỏi: hai người này có phải bạn bè không?', 'grpc', 'Chỉ bạn bè mới được nhắn cho nhau, nên phải đợi câu trả lời trước khi gửi tin đi.', 'AreFriends(sender, peer) → R social.friendships', { ret: o.friends ? 'có, là bạn bè' : 'không phải bạn bè' }),
      ];
      if (!o.friends) {
        st.push(
          S('social', 'message', 'Không phải bạn bè', 'error', null, null, { reply: true }),
          S('message', 'browser', 'Báo lỗi: chỉ nhắn được cho bạn bè', 'error', 'Tin không được gửi đi.', 'MESSAGE_NOT_FRIENDS', { reply: true }),
        );
        return st;
      }
      st.push(
        S('message', 'soketi', 'Phát tin nhắn tới người nhận (và các tab khác của bạn)', 'http', null, 'publish message.created → private-chat-<peer>, private-chat-<sender>', { ret: 'đã phát' }),
        S('soketi', 'peer', 'Tin nhắn hiện ngay trên màn hình người nhận', 'ws'),
        S('message', 'kafka', 'Bỏ tin vào hàng đợi để lưu sau', 'async', 'Không đợi lưu xong — nhờ vậy bạn không phải chờ. Tin của cùng một cuộc trò chuyện luôn giữ đúng thứ tự.', 'produce chat.messages, key = conversation_id, acks=all, retry 10'),
        S('message', 'browser', 'Gửi thành công', 'http', 'Tin của bạn chuyển từ "đang gửi" sang "đã gửi".', '200 {message} (khớp client_id)', { reply: true }),
        S('kafka', 'worker', 'Bộ lưu tin nhắn lấy tin từ hàng đợi', 'async', 'Việc này chạy ngầm, độc lập — không ai phải đợi.', 'consume group message-worker'),
        S('worker', 'pg', 'Cất tin nhắn vào cơ sở dữ liệu', 'db', 'Mỗi tin có một mã riêng, nên lỡ cất lại lần nữa cũng không bị trùng.', ['INSERT message.messages ON CONFLICT DO NOTHING', 'UPSERT message.conversations (last_message)']),
      );
      if (o.crash) {
        st.push(
          S('worker', 'worker', '💥 Bộ lưu tin nhắn bị sập trước khi kịp báo "xong"', 'error', 'Hàng đợi chưa nhận được chữ "xong" nên coi như tin chưa được lưu.'),
          S('kafka', 'worker', 'Hàng đợi giao lại tin cho một bộ lưu khác', 'async', 'Có nhiều bộ lưu chạy song song để thay thế nhau.', 'rebalance → worker #2'),
          S('worker', 'pg', 'Cất lại — phát hiện đã có, bỏ qua', 'db', 'Không sinh ra tin trùng.', 'ON CONFLICT DO NOTHING → 0 dòng mới'),
        );
      }
      st.push(S('worker', 'kafka', 'Báo hàng đợi: "đã lưu xong"', 'async', 'Chỉ báo sau khi lưu thành công — sập trước bước này thì tin sẽ được giao lại.', 'commit offset'));
      return st;
    },
  });

  // ------------------------------------------------------------------ Báo cáo
  SCENARIOS.push({
    id: 'report', group: 'Khác', icon: '📊', title: 'Xem báo cáo',
    desc: 'Bạn mở trang Báo cáo. Bộ phận Báo cáo không tự lưu dữ liệu, mà đi hỏi từng bộ phận khác rồi cộng dồn lại thành biểu đồ.',
    lanes: ['browser', 'nginx', 'report', 'task', 'event', 'calendar', 'backlog', 'pg'],
    options: [{ id: 'kind', type: 'select', label: 'Loại báo cáo', value: 'activity', options: [['activity', 'Hoạt động (sự kiện, lịch, email chờ)'], ['overview', 'Tổng quan công việc']] }],
    build(o) {
      const st = [
        S('browser', 'nginx', 'Mở trang Báo cáo', 'http', 'Trình duyệt xin số liệu trong khoảng thời gian bạn chọn.', `GET /api/report/v1/reports/${o.kind}?profile_id&from&to`),
        S('nginx', 'report', 'Chuyển tới bộ phận Báo cáo', 'grpc', null, `gRPC Get${o.kind === 'activity' ? 'Activity' : 'Overview'}Report`),
      ];
      if (o.kind === 'activity') {
        st.push(
          S('report', 'event', 'Hỏi danh sách sự kiện', 'grpc', 'Đợi bộ phận Sự kiện gửi số liệu.', 'ListEvents', { ret: 'các sự kiện' }),
          S('event', 'pg', 'Đọc sự kiện', 'db', null, 'R event.events'),
          S('report', 'calendar', 'Hỏi danh sách lịch', 'grpc', null, 'ListSchedules', { ret: 'các lịch' }),
          S('calendar', 'pg', 'Đọc lịch', 'db', null, 'R calendar.schedules'),
          S('report', 'backlog', 'Hỏi danh sách email đang chờ xử lý', 'grpc', null, 'ListBacklogs', { ret: 'các email chờ' }),
          S('backlog', 'pg', 'Đọc hộp chờ', 'db', null, 'R backlog.backlogs'),
        );
      } else {
        st.push(
          S('report', 'task', 'Hỏi danh sách công việc và trạng thái', 'grpc', null, 'ListTasks / ListStatuses', { ret: 'công việc + trạng thái' }),
          S('task', 'pg', 'Đọc công việc', 'db', null, ['R task.tasks', 'R task.task_statuses']),
        );
      }
      st.push(
        S('report', 'report', 'Cộng dồn, nhóm theo ngày / trạng thái', 'self', 'Tính toán ngay trong bộ nhớ, không lưu lại.'),
        S('report', 'browser', 'Gửi số liệu để vẽ biểu đồ', 'http', 'Trang hiện các thẻ số liệu, biểu đồ tròn, biểu đồ đường.', '200 {cards, series, ...}', { reply: true }),
      );
      return st;
    },
  });

  // ------------------------------------------------------------------ Gia hạn watch
  SCENARIOS.push({
    id: 'watch-renew', group: 'Email', icon: '♻️', title: 'Tự gia hạn theo dõi Gmail',
    desc: 'Đăng ký "có thư mới thì báo" với Gmail chỉ có hạn 7 ngày. Hệ thống tự kiểm tra mỗi 12 giờ và gia hạn trước khi hết hạn, để bạn không phải đăng nhập lại.',
    lanes: ['mail', 'pg', 'google', 'gmail'],
    options: [{ id: 'expired', type: 'checkbox', label: 'Quyền truy cập Gmail đã hết hạn', value: true }],
    build(o) {
      const st = [
        S('mail', 'mail', 'Đồng hồ ngầm điểm giờ (12 giờ / lần)', 'self', null, 'MAIL_WATCH_RENEW_INTERVAL=12h'),
        S('mail', 'pg', 'Tìm các đăng ký sắp hết hạn (còn dưới 1 ngày)', 'db', null, 'R mail_provider.noti_indexes.watch_expires_at < now + 24h'),
        S('mail', 'pg', 'Lấy quyền truy cập Gmail đã lưu', 'db', null, 'R mail_provider.sessions'),
      ];
      if (o.expired) {
        st.push(
          S('mail', 'google', 'Xin cấp lại quyền truy cập mới', 'ext', 'Quyền cũ hết hạn sau khoảng 1 giờ; dùng "quyền dài hạn" để xin quyền mới mà không cần bạn đăng nhập lại.', 'POST oauth2 token (refresh_token)', { ret: 'quyền mới' }),
          S('mail', 'pg', 'Lưu quyền mới', 'db', null, 'U mail_provider.sessions'),
        );
      }
      st.push(
        S('mail', 'gmail', 'Đăng ký lại "có thư mới thì báo"', 'ext', 'Gia hạn thêm 7 ngày.', 'users.watch(topic)', { ret: 'hạn mới' }),
        S('mail', 'pg', 'Ghi lại hạn mới', 'db', null, 'U mail_provider.noti_indexes.watch_expires_at'),
      );
      return st;
    },
  });

  window.Scenarios = { list: SCENARIOS, S };
})();

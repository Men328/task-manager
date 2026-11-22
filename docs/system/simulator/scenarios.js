/* Kịch bản mô phỏng các luồng (trừ luồng mail, nằm ở mail.js). */
(function () {
  'use strict';

  const S = (f, t, label, kind, detail, store, extra) => Object.assign({ f, t, label, kind, detail, store }, extra || {});

  const STATUSES = [
    { slug: 'todo', name: 'To Do', color: '#9aa0ae', category: 'TODO' },
    { slug: 'on-progress', name: 'On Progress', color: '#f59f00', category: 'IN_PROGRESS' },
    { slug: 'in-review', name: 'In Review', color: '#37b24d', category: 'IN_PROGRESS' },
    { slug: 'completed', name: 'Completed', color: '#f06595', category: 'DONE' },
  ];
  const TRANSITIONS = [
    ['todo', 'on-progress'], ['on-progress', 'in-review'], ['in-review', 'completed'],
    ['in-review', 'on-progress'], ['completed', 'on-progress'],
  ];
  const statusName = (slug) => STATUSES.find((s) => s.slug === slug).name;

  const SCENARIOS = [];

  // ------------------------------------------------------------------ Đăng nhập
  SCENARIOS.push({
    id: 'login', group: 'Tài khoản', icon: '🔐', title: 'Đăng nhập Google',
    desc: 'OAuth Google (PKCE) → identity tạo/đọc profile → seed dữ liệu mặc định cho người mới → (tuỳ chọn) đăng ký watch Gmail → trả session JWT và mở kênh realtime.',
    lanes: ['browser', 'nginx', 'identity', 'google', 'pg', 'task', 'project', 'mail', 'gmail', 'soketi'],
    options: [
      { id: 'isNew', type: 'checkbox', label: 'Người dùng lần đầu đăng nhập', value: true },
      { id: 'gmail', type: 'checkbox', label: 'Tích "Cho phép đọc Gmail"', value: true },
    ],
    build(o) {
      const st = [
        S('browser', 'nginx', `GET /api/identity/v1/auth/google/login${o.gmail ? '?gmail=1' : ''}`, 'http', 'Người dùng bấm "Đăng nhập bằng Google".'),
        S('nginx', 'identity', 'Forward tới identity:8081', 'http', 'nginx proxy theo tiền tố <code>/api/identity/</code>.'),
        S('identity', 'browser', '302 → accounts.google.com', 'http', 'Sinh <code>state</code> ký HMAC + PKCE code_verifier. Scope: <code>openid email profile</code>' + (o.gmail ? ' + <code>gmail.readonly</code>, <code>access_type=offline</code>' : '') + '.', null, { reply: true }),
        S('browser', 'google', 'Đăng nhập + đồng ý cấp quyền', 'ext', 'Người dùng chọn tài khoản Google và đồng ý.'),
        S('google', 'browser', '302 /v1/auth/google/callback?code&state', 'ext', null, null, { reply: true }),
        S('browser', 'identity', 'GET /v1/auth/google/callback', 'http', 'identity kiểm tra chữ ký <code>state</code>.'),
        S('identity', 'google', 'POST oauth2.googleapis.com/token', 'ext', 'Đổi <code>code</code> + code_verifier lấy access_token' + (o.gmail ? ' và refresh_token' : '') + '.'),
        S('identity', 'google', 'GET openidconnect.../userinfo', 'ext', 'Lấy subject, email, tên, avatar.'),
        S('identity', 'identity', 'LoginWithProvider (gRPC identity-grpc)', 'grpc', 'Gateway HTTP gọi sang identity-grpc :9081.'),
      ];
      if (o.isNew) {
        st.push(
          S('identity', 'pg', 'Tạo profile mới', 'db', 'Email chưa tồn tại ⇒ tạo profile + liên kết provider google.', ['C identity.profiles', 'C identity.auth_providers']),
          S('identity', 'task', 'SeedDefaultStatuses(profile_id)', 'grpc', 'Tạo bộ status mặc định cho người dùng mới.'),
          S('task', 'pg', 'Ghi 4 status + 5 rule lifecycle', 'db', 'To Do → On Progress → In Review → Completed (và các đường quay lại).', ['C task.task_statuses', 'C task.status_transitions']),
          S('identity', 'project', 'SeedDefaultProject(profile_id)', 'grpc', 'Tạo project mặc định.'),
          S('project', 'pg', 'Tạo project "Chung" (is_default)', 'db', 'Project mặc định không archive được.', ['C project.projects']),
        );
      } else {
        st.push(S('identity', 'pg', 'Đọc profile có sẵn, cập nhật last_login_at', 'db', null, ['R identity.auth_providers', 'U identity.profiles']));
      }
      if (o.gmail) {
        st.push(
          S('identity', 'mail', 'Subscribe(profile_id, email, tokens)', 'grpc', 'Chỉ khi người dùng tích "Cho phép đọc Gmail".'),
          S('mail', 'gmail', 'users.watch(topic MAIL_PUBSUB_TOPIC)', 'ext', 'Gmail sẽ báo về Pub/Sub mỗi khi INBOX thay đổi. Hạn watch tối đa 7 ngày (có worker gia hạn).'),
          S('mail', 'pg', 'Lưu token + checkpoint historyId', 'db', null, ['UPSERT mail_provider.sessions', 'UPSERT mail_provider.noti_indexes']),
        );
      }
      st.push(
        S('identity', 'browser', '302 /auth/callback#token=<JWT>', 'http', 'Session JWT HS256 ký bằng <code>SESSION_SECRET</code>, claim <code>sub</code> = profile_id.', null, { reply: true }),
        S('browser', 'identity', 'GET /v1/auth/me (Bearer)', 'http', 'SPA lưu token, tải hồ sơ.'),
        S('browser', 'identity', 'POST /v1/auth/soketi', 'http', 'Xin chữ ký subscribe kênh private: <code>HMAC_SHA256(app_secret, socket_id:channel)</code>, chỉ cho kênh của chính mình.'),
        S('browser', 'soketi', 'Subscribe private-noti-internal-<id>', 'ws', 'Từ đây chuông thông báo nhận realtime.'),
      );
      return st;
    },
  });

  // ------------------------------------------------------------------ Tạo task
  SCENARIOS.push({
    id: 'create-task', group: 'Công việc', icon: '📝', title: 'Tạo task từ Board',
    desc: 'Browser gọi task service; task kiểm tra project qua gRPC rồi mới ghi DB. Có thể kèm upload tệp đính kèm lên RustFS.',
    lanes: ['browser', 'nginx', 'task', 'project', 'attachment', 'pg', 's3'],
    options: [
      { id: 'archived', type: 'checkbox', label: 'Project đã bị archive', value: false },
      { id: 'attach', type: 'checkbox', label: 'Đính kèm 1 tệp', value: true },
    ],
    build(o) {
      const st = [
        S('browser', 'nginx', 'POST /api/task/v1/tasks', 'http', 'Body: <code>{profile_id, project_id, title, priority, due_at, ...}</code>'),
        S('nginx', 'task', 'task:8082 → gRPC CreateTask (:9082)', 'grpc', 'grpc-gateway chuyển REST JSON sang gRPC.'),
        S('task', 'project', 'ProjectService.GetProject', 'grpc', 'Chặn project của người khác hoặc đã archive.'),
        S('project', 'pg', 'Đọc project', 'db', null, ['R project.projects']),
      ];
      if (o.archived) {
        st.push(
          S('project', 'task', 'is_archived = true', 'error', null, null, { reply: true }),
          S('task', 'browser', 'Lỗi PROJECT_ARCHIVED', 'error', 'FE dịch mã lỗi chuẩn (<code>common/errorcode</code>) sang thông báo vi/en. Không có gì được ghi.', null, { reply: true }),
        );
        return st;
      }
      st.push(
        S('task', 'pg', 'Lấy status mặc định (To Do)', 'db', null, ['R task.task_statuses']),
        S('task', 'pg', 'Ghi task + audit log', 'db', 'Trong cùng transaction.', ['C task.tasks', 'C task.task_status_logs']),
        S('task', 'browser', '200 {task}', 'http', 'Board thêm thẻ mới vào cột To Do.', null, { reply: true }),
      );
      if (o.attach) {
        st.push(
          S('browser', 'nginx', 'POST /api/attachment/v1/attachments (multipart)', 'http', '<code>owner_type=task</code>, <code>owner_id=&lt;task_id&gt;</code>. Route upload viết tay ở gateway.'),
          S('nginx', 'attachment', 'attachment:8089 → gRPC', 'grpc'),
          S('attachment', 's3', 'PUT attachments/{profile}/task/{id}/{uuid}.ext', 'db', 'Nội dung tệp vào RustFS.', ['S3 PUT']),
          S('attachment', 'pg', 'Lưu metadata tệp', 'db', 'Liên kết đa hình, không có FK tới task.', ['C attachment.attachments']),
          S('attachment', 'browser', '200 {attachment}', 'http', null, null, { reply: true }),
        );
      }
      return st;
    },
  });

  // ------------------------------------------------------------------ Đổi trạng thái
  SCENARIOS.push({
    id: 'status', group: 'Công việc', icon: '🔀', title: 'Kéo thả đổi trạng thái',
    desc: 'Mỗi lần kéo thẻ sang cột khác, task service kiểm tra rule lifecycle (allowlist). Nếu chuyển sang Completed mà task lặp lại thì sinh lần kế tiếp.',
    lanes: ['browser', 'nginx', 'task', 'pg'],
    options: [
      { id: 'from', type: 'select', label: 'Từ', value: 'in-review', options: STATUSES.map((s) => [s.slug, s.name]) },
      { id: 'to', type: 'select', label: 'Sang', value: 'completed', options: STATUSES.map((s) => [s.slug, s.name]) },
      { id: 'recurring', type: 'checkbox', label: 'Task lặp lại (RRULE)', value: true },
    ],
    extra(o) {
      const allowed = TRANSITIONS.some(([a, b]) => a === o.from && b === o.to);
      const cols = STATUSES.map((s) => `<div class="col ${s.slug === o.from ? 'from' : ''} ${s.slug === o.to ? 'to' : ''}" style="--c:${s.color}"><b>${s.name}</b>${s.category}${s.slug === o.from ? '<div>⬅ thẻ đang ở đây</div>' : ''}${s.slug === o.to ? '<div>➡ thả vào đây</div>' : ''}</div>`).join('');
      const rules = TRANSITIONS.map(([a, b]) => `<span class="chip ${a === o.from && b === o.to ? 'hit' : ''}">${statusName(a)} → ${statusName(b)}</span>`).join('');
      return `<div class="mini-kanban">${cols}</div>
        <p style="margin:10px 0 4px"><b>Rule lifecycle mặc định</b> (<code>task.status_transitions</code>):</p>${rules}
        <p style="margin:8px 0 0">Kết quả: ${allowed ? '<span class="chip ok">✓ hợp lệ</span>' : '<span class="chip bad">✗ TASK_TRANSITION_NOT_ALLOWED</span>'}</p>`;
    },
    build(o) {
      const allowed = TRANSITIONS.some(([a, b]) => a === o.from && b === o.to);
      const toDone = STATUSES.find((s) => s.slug === o.to).category === 'DONE';
      const st = [
        S('browser', 'nginx', 'POST /api/task/v1/tasks/{id}/status', 'http', `Body <code>{status_id: "${o.to}", note}</code> — kéo thẻ từ <b>${statusName(o.from)}</b> sang <b>${statusName(o.to)}</b>.`),
        S('nginx', 'task', 'gRPC ChangeTaskStatus', 'grpc'),
        S('task', 'pg', 'Đọc task + status đích', 'db', null, ['R task.tasks', 'R task.task_statuses']),
        S('task', 'pg', `IsAllowed? ${statusName(o.from)} → ${statusName(o.to)}`, 'db', 'Tìm rule đang active trong allowlist.', ['R task.status_transitions']),
      ];
      if (o.from === o.to) {
        st.splice(2);
        st.push(S('browser', 'browser', 'Thả vào cùng cột → không gọi API', 'self'));
        return st;
      }
      if (!allowed) {
        st.push(
          S('task', 'task', 'Không có rule → từ chối', 'decision'),
          S('task', 'browser', 'Lỗi TASK_TRANSITION_NOT_ALLOWED', 'error', 'UI hiện thông báo đỏ và trả thẻ về cột cũ.', null, { reply: true }),
        );
        return st;
      }
      st.push(
        S('task', 'task', 'Rule hợp lệ', 'decision'),
        S('task', 'pg', 'Cập nhật status' + (toDone ? ' + completed_at' : ''), 'db', toDone ? 'Status đích thuộc category DONE ⇒ ghi completed_at.' : null, ['U task.tasks']),
        S('task', 'pg', 'Ghi audit log', 'db', null, ['C task.task_status_logs']),
      );
      if (toDone && o.recurring) {
        st.push(S('task', 'pg', 'SpawnNextOccurrence', 'db', 'Task lặp lại vừa DONE ⇒ tính lần kế tiếp theo RRULE và tạo task mới ở To Do.', ['C task.tasks (lần kế tiếp)', 'U recurrence_next_id']));
      }
      st.push(S('task', 'browser', toDone && o.recurring ? '200 {task, next_task}' : '200 {task}', 'http', 'Board cập nhật thẻ.', null, { reply: true }));
      return st;
    },
  });

  // ------------------------------------------------------------------ Nhắc việc
  SCENARIOS.push({
    id: 'reminder', group: 'Công việc', icon: '⏰', title: 'Nhắc việc (reminder)',
    desc: 'ReminderWorker chạy theo chu kỳ trong task-grpc, "claim" các task đến giờ nhắc rồi gọi notification. Gửi lỗi thì trả lại để lần sau thử tiếp.',
    lanes: ['task', 'pg', 'notification', 'soketi', 'browser'],
    options: [{ id: 'fail', type: 'checkbox', label: 'notification lỗi lần đầu', value: false }],
    build(o) {
      const st = [
        S('task', 'task', 'Ticker tới chu kỳ', 'self', 'Worker nền trong task-grpc, không có API.'),
        S('task', 'pg', 'ClaimDueReminders', 'db', '<code>remind_at &lt;= now AND reminded_at IS NULL</code> → đặt reminded_at (claim nằm ở DB, không ở RAM).', ['U task.tasks.reminded_at']),
        S('task', 'notification', 'CreateNotice(TASK_REMINDER, target TASK)', 'grpc'),
      ];
      if (o.fail) {
        st.push(
          S('notification', 'task', 'Lỗi (unavailable)', 'error', null, null, { reply: true }),
          S('task', 'pg', 'ReleaseReminder', 'db', 'Bỏ claim để chu kỳ sau thử lại.', ['U task.tasks.reminded_at = NULL']),
          S('task', 'task', 'Chu kỳ kế tiếp', 'self'),
          S('task', 'pg', 'ClaimDueReminders (lần 2)', 'db', null, ['U task.tasks.reminded_at']),
          S('task', 'notification', 'CreateNotice(TASK_REMINDER)', 'grpc'),
        );
      }
      st.push(
        S('notification', 'pg', 'Lưu notice', 'db', null, ['C notification.notices']),
        S('notification', 'soketi', 'POST /apps/{id}/events notice.created', 'http', 'Kênh <code>private-noti-internal-&lt;profile_id&gt;</code>.'),
        S('soketi', 'browser', 'WebSocket notice.created', 'ws', 'FE gọi lại <code>GET /v1/notices/unread-count</code> để cập nhật badge chuông.'),
      );
      return st;
    },
  });

  // ------------------------------------------------------------------ Kết bạn
  SCENARIOS.push({
    id: 'friend', group: 'Giao tiếp', icon: '🤝', title: 'Kết bạn',
    desc: 'A tìm B theo email và gửi lời mời; B nhận thông báo realtime và chấp nhận; A nhận thông báo lại.',
    lanes: ['browser', 'social', 'identity', 'pg', 'notification', 'soketi', 'peer'],
    options: [
      { id: 'found', type: 'checkbox', label: 'Email của B tồn tại', value: true },
      { id: 'accept', type: 'checkbox', label: 'B chấp nhận lời mời', value: true },
    ],
    build(o) {
      const st = [
        S('browser', 'social', 'POST /api/social/v1/friend-requests {email}', 'http', 'Qua nginx. Gateway social tự verify JWT, lấy requester từ <code>sub</code>.'),
        S('social', 'identity', 'FindProfileByEmail', 'grpc'),
      ];
      if (!o.found) {
        st.push(S('identity', 'social', 'Không tìm thấy', 'error', null, null, { reply: true }), S('social', 'browser', 'Lỗi: không có người dùng với email này', 'error', null, null, { reply: true }));
        return st;
      }
      st.push(
        S('social', 'pg', 'Tạo lời mời pending', 'db', 'Ràng buộc: không tự kết bạn, cặp không thứ tự là duy nhất.', ['C social.friendships (pending)']),
        S('social', 'notification', 'CreateNotice(FRIEND_REQUEST → B)', 'grpc'),
        S('notification', 'pg', 'Lưu notice', 'db', null, ['C notification.notices']),
        S('notification', 'soketi', 'publish notice.created', 'http'),
        S('soketi', 'peer', 'WebSocket → chuông của B', 'ws'),
      );
      if (o.accept) {
        st.push(
          S('peer', 'social', 'POST /v1/friend-requests/{id}/accept', 'http'),
          S('social', 'pg', 'status = accepted', 'db', null, ['U social.friendships']),
          S('social', 'notification', 'CreateNotice(FRIEND_ACCEPTED → A)', 'grpc'),
          S('notification', 'soketi', 'publish notice.created', 'http'),
          S('soketi', 'browser', 'WebSocket → chuông của A', 'ws', 'Từ giờ A và B nhắn tin được cho nhau.'),
        );
      }
      return st;
    },
  });

  // ------------------------------------------------------------------ Nhắn tin
  SCENARIOS.push({
    id: 'chat', group: 'Giao tiếp', icon: '💬', title: 'Nhắn tin realtime',
    desc: '"Realtime trước, lưu sau": message-grpc broadcast qua Soketi ngay, đồng thời đẩy Kafka; worker ghi DB idempotent (at-least-once).',
    lanes: ['browser', 'message', 'social', 'soketi', 'peer', 'kafka', 'worker', 'pg'],
    options: [
      { id: 'friends', type: 'checkbox', label: 'Hai người là bạn bè', value: true },
      { id: 'crash', type: 'checkbox', label: 'Worker chết trước khi commit offset', value: false },
    ],
    build(o) {
      const st = [
        S('browser', 'message', 'POST /api/message/v1/conversations/{peer}/messages', 'http', 'Body <code>{body, client_id}</code>, Bearer JWT. sender = <code>sub</code>.'),
        S('message', 'message', 'gRPC SendMessage', 'grpc'),
        S('message', 'social', 'AreFriends(sender, peer)', 'grpc', null, ['R social.friendships']),
      ];
      if (!o.friends) {
        st.push(S('social', 'message', 'false', 'error', null, null, { reply: true }), S('message', 'browser', 'Lỗi MESSAGE_NOT_FRIENDS', 'error', null, null, { reply: true }));
        return st;
      }
      st.push(
        S('message', 'soketi', 'publish message.created ×2 kênh', 'http', '<code>private-chat-&lt;peer&gt;</code> và <code>private-chat-&lt;sender&gt;</code> (đồng bộ nhiều tab).'),
        S('soketi', 'peer', 'WebSocket: tin nhắn hiện ngay', 'ws'),
        S('message', 'kafka', 'produce chat.messages (async)', 'async', 'key = conversation_id ⇒ cùng partition ⇒ giữ thứ tự. acks=all, retry 10 lần.'),
        S('message', 'browser', '200 {message}', 'http', 'Khớp <code>client_id</code> để thay tin "đang gửi".', null, { reply: true }),
        S('kafka', 'worker', 'consume (group message-worker)', 'async'),
        S('worker', 'pg', 'INSERT ... ON CONFLICT DO NOTHING', 'db', 'id = UUIDv7 ⇒ ghi lại cùng tin không bị trùng.', ['C message.messages', 'UPSERT message.conversations']),
      );
      if (o.crash) {
        st.push(
          S('worker', 'worker', '💥 Worker #1 chết trước khi commit', 'error', 'Offset chưa commit ⇒ Kafka sẽ giao lại.'),
          S('kafka', 'worker', 'Rebalance → giao lại cho worker #2', 'async'),
          S('worker', 'pg', 'INSERT lại → ON CONFLICT DO NOTHING', 'db', 'Không sinh bản ghi trùng (idempotent).', ['message.messages: 0 dòng mới']),
        );
      }
      st.push(S('worker', 'kafka', 'commit offset', 'async', 'Chỉ commit SAU khi ghi DB thành công.'));
      return st;
    },
  });

  // ------------------------------------------------------------------ Báo cáo
  SCENARIOS.push({
    id: 'report', group: 'Khác', icon: '📊', title: 'Báo cáo hoạt động',
    desc: 'report không có DB: gọi gRPC sang các service khác và tổng hợp trong RAM.',
    lanes: ['browser', 'nginx', 'report', 'task', 'event', 'calendar', 'backlog', 'pg'],
    options: [{ id: 'kind', type: 'select', label: 'Loại báo cáo', value: 'activity', options: [['activity', 'Activity'], ['overview', 'Tổng quan (task)']] }],
    build(o) {
      const st = [
        S('browser', 'nginx', `GET /api/report/v1/reports/${o.kind}?profile_id&from&to`, 'http'),
        S('nginx', 'report', `gRPC Get${o.kind === 'activity' ? 'Activity' : 'Overview'}Report`, 'grpc'),
      ];
      if (o.kind === 'activity') {
        st.push(
          S('report', 'event', 'ListEvents', 'grpc'), S('event', 'pg', 'Đọc sự kiện', 'db', null, ['R event.events']),
          S('report', 'calendar', 'ListSchedules', 'grpc'), S('calendar', 'pg', 'Đọc lịch', 'db', null, ['R calendar.schedules']),
          S('report', 'backlog', 'ListBacklogs', 'grpc'), S('backlog', 'pg', 'Đọc backlog', 'db', null, ['R backlog.backlogs']),
        );
      } else {
        st.push(S('report', 'task', 'ListTasks / statuses', 'grpc'), S('task', 'pg', 'Đọc task', 'db', null, ['R task.tasks', 'R task.task_statuses']));
      }
      st.push(
        S('report', 'report', 'Tổng hợp trong RAM', 'self', 'Đếm, nhóm theo ngày / status.'),
        S('report', 'browser', '200 {cards, series, ...}', 'http', 'Trang /reports vẽ thẻ, donut, line chart.', null, { reply: true }),
      );
      return st;
    },
  });

  // ------------------------------------------------------------------ Gia hạn watch
  SCENARIOS.push({
    id: 'watch-renew', group: 'Email', icon: '♻️', title: 'Gia hạn watch Gmail',
    desc: 'users.watch chỉ sống tối đa 7 ngày. Worker nền trong mail-provider-grpc quét định kỳ và gia hạn trước khi hết hạn.',
    lanes: ['mail', 'pg', 'google', 'gmail'],
    options: [{ id: 'expired', type: 'checkbox', label: 'Access token đã hết hạn', value: true }],
    build(o) {
      const st = [
        S('mail', 'mail', 'Ticker MAIL_WATCH_RENEW_INTERVAL (12h)', 'self'),
        S('mail', 'pg', 'Tìm watch sắp hết hạn (< 24h)', 'db', null, ['R mail_provider.noti_indexes.watch_expires_at']),
        S('mail', 'pg', 'Đọc token', 'db', null, ['R mail_provider.sessions']),
      ];
      if (o.expired) {
        st.push(S('mail', 'google', 'POST token (refresh_token)', 'ext'), S('mail', 'pg', 'Lưu access_token mới', 'db', null, ['U mail_provider.sessions']));
      }
      st.push(
        S('mail', 'gmail', 'users.watch(topic)', 'ext'),
        S('mail', 'pg', 'Cập nhật hạn watch', 'db', null, ['U mail_provider.noti_indexes']),
      );
      return st;
    },
  });

  window.Scenarios = { list: SCENARIOS, S };
})();

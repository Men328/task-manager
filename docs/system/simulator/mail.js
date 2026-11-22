/* Luồng mail: bản port JS của bộ phân loại từ khoá + khớp project trong
   service/mail-provider/internal/service/{classifier.go, project_match.go, worker.go}. */
(function () {
  'use strict';
  const { S } = window.Scenarios;
  const esc = window.Sim.esc;

  // ---------------------------------------------------------------- port của classifier.go
  const KEYWORDS = {
    event: ['sự kiện', 'hội thảo', 'hội nghị', 'conference', 'webinar', 'workshop', 'seminar', 'sinh nhật', 'birthday', 'đám cưới', 'wedding', 'tiệc', 'party', 'khai trương', 'ra mắt', 'lễ', 'ceremony', 'thiệp mời', 'invitation', 'mời bạn', 'mời tham dự', 'mời tham gia', 'event'],
    schedule: ['lịch', 'lịch hẹn', 'appointment', 'schedule', 'calendar', 'cuộc họp', 'meeting', 'họp', 'đặt phòng', 'booking', 'reminder', 'nhắc lịch', 'ca làm', 'shift', 'call', 'zoom', 'google meet'],
    task: ['cần', 'hoàn thành', 'nộp', 'thanh toán', 'hóa đơn', 'hoá đơn', 'invoice', 'deadline', 'hạn chót', 'yêu cầu', 'action required', 'todo', 'to-do', 'task', 'công việc', 'xử lý', 'kiểm tra', 'duyệt', 'review', 'báo cáo', 'xác nhận'],
  };
  const ISO_TIME = /(\d{4})-(\d{1,2})-(\d{1,2})(?:[ t](\d{1,2}):(\d{2}))?/;
  const DAY_FIRST = /\b(\d{1,2})[/](\d{1,2})(?:[/](\d{2,4}))?(?:\s+(\d{1,2}):(\d{2}))?/;

  function keywordScore(text, subject, list) {
    let score = 0;
    const hits = [];
    for (const kw of list) {
      if (!text.includes(kw)) continue;
      const inSubject = subject.includes(kw);
      score += inSubject ? 2 : 1;
      hits.push({ kw, inSubject });
    }
    return { score, hits };
  }

  function parseFirstTime(text) {
    const valid = (y, mo, d, h, mi) => h >= 0 && h <= 23 && mi >= 0 && mi <= 59 && y >= 1970 && y <= 2200 && mo >= 1 && mo <= 12 && d >= 1 && d <= 31;
    const n = (v) => (v ? parseInt(v, 10) : 0);
    let m = ISO_TIME.exec(text);
    if (m && valid(n(m[1]), n(m[2]), n(m[3]), n(m[4]), n(m[5]))) return { y: n(m[1]), mo: n(m[2]), d: n(m[3]), h: n(m[4]), mi: n(m[5]), raw: m[0] };
    m = DAY_FIRST.exec(text);
    if (m) {
      let y = m[3] ? n(m[3]) : new Date().getUTCFullYear();
      if (m[3] && y < 100) y += 2000;
      if (valid(y, n(m[2]), n(m[1]), n(m[4]), n(m[5]))) return { y, mo: n(m[2]), d: n(m[1]), h: n(m[4]), mi: n(m[5]), raw: m[0] };
    }
    return null;
  }
  const fmtTime = (t) => t && `${t.y}-${String(t.mo).padStart(2, '0')}-${String(t.d).padStart(2, '0')}T${String(t.h).padStart(2, '0')}:${String(t.mi).padStart(2, '0')}:00Z`;

  // ---------------------------------------------------------------- port của project_match.go
  const STOP = new Set(('và của cho các những là có được hàng khách công việc bạn anh chị em quý xin chào cảm ơn thông báo ngày giờ tại với này đã sẽ không một người từ về theo như khi để trong trên mới the and for with to of a an in on').split(' '));
  function tokens(s) {
    const out = new Set();
    for (const t of s.toLowerCase().split(/[^\p{L}\p{N}]+/u)) {
      if ([...t].length < 2 || STOP.has(t)) continue;
      out.add(t);
    }
    return out;
  }
  function matchProjects(text, projects) {
    const emailTokens = tokens(text);
    let bestId = '';
    let best = 0;
    const rows = projects.map((p) => {
      const pt = tokens(p.name + ' ' + p.desc);
      const overlap = [...pt].filter((t) => emailTokens.has(t));
      let score = 0;
      if (pt.size && overlap.length >= 2) {
        const denom = Math.max(3, Math.min(5, pt.size));
        score = Math.min(1, overlap.length / denom);
        if (score > best) { best = score; bestId = p.id; }
      }
      return { p, tokens: [...pt], overlap, score };
    });
    return { bestId, best, rows };
  }

  // ---------------------------------------------------------------- phân tích + dispatch (worker.go)
  function analyze(st) {
    const subject = st.subject.trim().toLowerCase();
    const text = [st.subject, st.body].join('\n').toLowerCase();
    const projects = parseProjects(st.projects);
    const scores = {
      event: keywordScore(text, subject, KEYWORDS.event),
      schedule: keywordScore(text, subject, KEYWORDS.schedule),
      task: keywordScore(text, subject, KEYWORDS.task),
    };
    // thứ tự so sánh strict ">" ⇒ hoà điểm: event > schedule > task
    let ruleCat = 'other';
    let best = 0;
    for (const c of ['event', 'schedule', 'task']) if (scores[c].score > best) { ruleCat = c; best = scores[c].score; }
    const time = parseFirstTime(text);
    const pm = matchProjects(st.subject + ' ' + st.body, projects);

    const decision = [];
    let source; // 'ai' | 'rule' | 'off'
    let category;
    let reason = '';
    let actionable;
    if (st.classifier === 'ai-ok') {
      source = 'ai';
      category = st.aiCategory === 'auto' ? ruleCat : st.aiCategory;
      actionable = category !== 'other';
      reason = category === 'other' ? 'newsletter / không đủ dữ kiện (AI)' : '';
      decision.push(['✓', `DeepSeek trả về JSON hợp lệ: <b>${category}</b>`]);
    } else if (st.classifier === 'off') {
      source = 'off';
      category = 'other';
      reason = 'classifier_disabled';
      decision.push(['✗', 'Không có DEEPSEEK_API_KEY và MAIL_RULE_FALLBACK_ENABLED=false']);
    } else {
      source = 'rule';
      category = ruleCat;
      actionable = category !== 'other';
      if (category === 'other') reason = 'no_rule_matched';
      decision.push([st.classifier === 'ai-fail' ? '⚠' : 'ℹ', st.classifier === 'ai-fail' ? 'DeepSeek lỗi → chuyển sang rule từ khoá dự phòng' : 'Thiếu DEEPSEEK_API_KEY → dùng rule từ khoá']);
      decision.push(['✓', `Điểm cao nhất: <b>${category}</b>${best ? ` (${best} điểm)` : ' (không khớp từ khoá nào)'}`]);
    }

    const title = st.subject.trim() || st.body.trim().split('\n')[0] || 'Email không có tiêu đề';
    const start = (category === 'schedule' || category === 'event') ? time : null;
    const conf = pm.best;
    const project = projects.find((p) => p.id === pm.bestId);
    let dest = 'backlog';
    if (category === 'task') {
      if (!actionable || !title) { reason = 'not_actionable'; decision.push(['✗', 'Task không actionable / thiếu title']); }
      else if (!project || conf < st.threshold) { reason = 'no_project_matched'; decision.push(['✗', `Không project nào đạt ngưỡng ${st.threshold} (cao nhất ${conf.toFixed(2)})`]); }
      else { dest = 'task'; decision.push(['✓', `Khớp project <b>${esc(project.name)}</b> (${conf.toFixed(2)} ≥ ${st.threshold})`]); }
    } else if (category === 'schedule' || category === 'event') {
      if (!start) { reason = category === 'schedule' ? 'missing_schedule_time' : 'missing_event_time'; decision.push(['✗', 'Không bóc được thời gian bắt đầu (cần dạng 2026-11-25 09:30 hoặc 25/11/2026 09:30)']); }
      else {
        decision.push(['✓', `Thời gian bắt đầu: <code>${fmtTime(start)}</code>`]);
        if (!project || conf < st.threshold) { reason = 'no_project_matched'; decision.push(['✗', `Không project nào đạt ngưỡng ${st.threshold} (cao nhất ${conf.toFixed(2)})`]); }
        else { dest = category === 'schedule' ? 'calendar' : 'event'; decision.push(['✓', `Khớp project <b>${esc(project.name)}</b> (${conf.toFixed(2)} ≥ ${st.threshold})`]); }
      }
    } else {
      decision.push(['→', 'category = other ⇒ luôn vào backlog']);
    }
    if (dest !== 'backlog' && st.createFail) {
      decision.push(['✗', `Tạo bản ghi ở ${dest} thất bại ⇒ giữ email trong backlog`]);
      reason = 'create_failed';
      dest = 'backlog';
    }
    if (dest === 'backlog' && !reason) reason = 'unclassified';

    const draft = {
      category, is_actionable: !!actionable, title, description: st.body.trim().slice(0, 120) + (st.body.trim().length > 120 ? '…' : ''),
      priority: 'medium', start_at: fmtTime(start) || '', project_id: pm.bestId, project_confidence: +conf.toFixed(2),
      reason: category === 'other' ? reason : '',
    };
    return { scores, ruleCat, time, start, pm, projects, project, source, category, reason, dest, decision, draft, title };
  }

  function parseProjects(text) {
    return text.split('\n').map((l) => l.trim()).filter(Boolean).map((l, i) => {
      const [name, ...rest] = l.split('|');
      return { id: `p${i + 1}`, name: name.trim(), desc: rest.join('|').trim() };
    });
  }

  // ---------------------------------------------------------------- dữ liệu mẫu
  const PRESETS = [
    { name: '📝 Yêu cầu nộp báo cáo', subject: '[Đồ án tốt nghiệp] Yêu cầu nộp báo cáo tiến độ', body: 'Chào em,\nEm cần hoàn thành và nộp báo cáo tiến độ đồ án tốt nghiệp trước deadline 30/11/2026. Thầy sẽ duyệt trong tuần sau.\nPhenikaa University' },
    { name: '📅 Cuộc họp có giờ', subject: 'Lịch họp sprint dự án Website bán hàng', body: 'Mời cả team tham gia cuộc họp qua Google Meet lúc 2026-11-25 09:30 để chốt phần phát triển trang thanh toán cho website bán hàng online.' },
    { name: '🎤 Hội thảo có giờ', subject: 'Mời tham dự hội thảo AI tại Phenikaa', body: 'Hội thảo diễn ra lúc 05/12/2026 08:00 tại hội trường A, chủ đề ứng dụng AI trong đồ án tốt nghiệp.' },
    { name: '💌 Thiệp mời không giờ', subject: 'Thiệp mời đám cưới', body: 'Trân trọng mời bạn tới chung vui tiệc cưới của chúng mình vào cuối tuần này nhé!' },
    { name: '🧾 Hoá đơn (không khớp dự án)', subject: 'Thanh toán hoá đơn điện tháng 11', body: 'Quý khách cần thanh toán hoá đơn tiền điện trước 2026-11-30 để tránh gián đoạn dịch vụ.' },
    { name: '📰 Bản tin', subject: 'Bản tin công nghệ tuần 47', body: 'Tổng hợp tin tức công nghệ nổi bật trong tuần. Hủy đăng ký nhận thư tại đây.' },
  ];

  const state = {
    from: 'thay.huong@phenikaa-uni.edu.vn',
    subject: PRESETS[0].subject,
    body: PRESETS[0].body,
    projects: 'Đồ án tốt nghiệp | Báo cáo, tiến độ đồ án tốt nghiệp tại Phenikaa\nWebsite bán hàng | Phát triển website thương mại điện tử bán hàng online\nMarketing Q4 | Chiến dịch quảng cáo mạng xã hội quý 4',
    classifier: 'rule', aiCategory: 'auto', threshold: 0.6,
    delivery: 'push', attachments: true, tokenExpired: false, queueFull: false, duplicate: false, createFail: false,
  };

  // ---------------------------------------------------------------- giao diện nhập liệu
  function mountPanel(host, onChange) {
    host.innerHTML = `
      <div class="grid grid-2">
        <div class="card">
          <h3>✉️ Email đến <span class="sub">sửa tuỳ ý hoặc chọn mẫu</span></h3>
          <div class="preset-row">${PRESETS.map((p, i) => `<button class="btn" data-preset="${i}">${p.name}</button>`).join('')}</div>
          <div class="field"><label>Người gửi</label><input data-k="from"></div>
          <div class="field"><label>Tiêu đề (từ khoá ở tiêu đề được nhân đôi điểm)</label><input data-k="subject"></div>
          <div class="field"><label>Nội dung</label><textarea data-k="body" rows="4"></textarea></div>
        </div>
        <div class="card">
          <h3>⚙️ Cấu hình & tình huống</h3>
          <div class="field"><label>Project của người dùng (mỗi dòng: Tên | Mô tả) — project mặc định "Chung" luôn bị loại</label><textarea data-k="projects" rows="3"></textarea></div>
          <div class="field-row">
            <div class="field"><label>Bộ phân loại</label>
              <select data-k="classifier">
                <option value="rule">Thiếu DEEPSEEK_API_KEY → rule từ khoá</option>
                <option value="ai-fail">DeepSeek lỗi/timeout → rule dự phòng</option>
                <option value="ai-ok">DeepSeek trả lời (giả lập)</option>
                <option value="off">Tắt cả AI và rule</option>
              </select></div>
            <div class="field" data-show="ai-ok"><label>AI trả về category</label>
              <select data-k="aiCategory"><option value="auto">Giống kết quả rule</option><option>task</option><option>schedule</option><option>event</option><option>other</option></select></div>
            <div class="field"><label>PROJECT_MATCH_THRESHOLD</label><input data-k="threshold" type="number" step="0.05" min="0" max="1"></div>
            <div class="field"><label>Pub/Sub</label>
              <select data-k="delivery"><option value="push">Push (webhook qua tunnel)</option><option value="pull">Pull (subscriptions.pull)</option></select></div>
          </div>
          <div class="options">
            <label class="opt"><input type="checkbox" data-k="attachments"> Có file đính kèm</label>
            <label class="opt"><input type="checkbox" data-k="tokenExpired"> Access token hết hạn</label>
            <label class="opt"><input type="checkbox" data-k="queueFull"> Queue RAM đầy</label>
            <label class="opt"><input type="checkbox" data-k="duplicate"> Pub/Sub gửi trùng notice</label>
            <label class="opt"><input type="checkbox" data-k="createFail"> Service đích lỗi khi tạo</label>
          </div>
        </div>
      </div>`;
    const inputs = host.querySelectorAll('[data-k]');
    const sync = () => {
      inputs.forEach((i) => {
        if (i.type === 'checkbox') i.checked = !!state[i.dataset.k]; else i.value = state[i.dataset.k];
      });
      host.querySelector('[data-show="ai-ok"]').classList.toggle('hidden', state.classifier !== 'ai-ok');
    };
    sync();
    let t;
    inputs.forEach((i) => i.addEventListener(i.tagName === 'SELECT' || i.type === 'checkbox' ? 'change' : 'input', () => {
      state[i.dataset.k] = i.type === 'checkbox' ? i.checked : i.type === 'number' ? parseFloat(i.value) || 0 : i.value;
      sync();
      clearTimeout(t);
      t = setTimeout(onChange, 250);
    }));
    host.querySelectorAll('[data-preset]').forEach((b) => b.addEventListener('click', () => {
      const p = PRESETS[+b.dataset.preset];
      state.subject = p.subject; state.body = p.body;
      sync(); onChange();
    }));
  }

  const DEST = {
    task: { icon: '📝', name: 'Task mới trên Board', svc: 'task', notice: 'TASK_CREATED', rpc: 'CreateTask', table: 'task.tasks' },
    calendar: { icon: '📅', name: 'Lịch mới trên Calendar', svc: 'calendar', notice: 'SCHEDULE_CREATED', rpc: 'CreateSchedule', table: 'calendar.schedules' },
    event: { icon: '🎉', name: 'Sự kiện mới', svc: 'event', notice: 'EVENT_CREATED', rpc: 'CreateEvent', table: 'event.events' },
    backlog: { icon: '📥', name: 'Backlog (chờ phân loại tay)', svc: 'backlog', notice: 'BACKLOG_CREATED', rpc: 'CreateBacklog', table: 'backlog.backlogs' },
  };

  function extra() {
    const a = analyze(state);
    const d = DEST[a.dest];
    const scoreRows = ['event', 'schedule', 'task'].map((c) => {
      const s = a.scores[c];
      const max = Math.max(1, ...Object.values(a.scores).map((x) => x.score));
      return `<tr class="${a.ruleCat === c ? 'win' : ''}"><td><b>${c}</b></td><td>${s.score}<div class="bar"><div style="width:${(s.score / max) * 100}%"></div></div></td>
        <td>${s.hits.length ? s.hits.map((h) => `<span class="chip hit">${esc(h.kw)}${h.inSubject ? ' ×2' : ''}</span>`).join('') : '<span class="sub">—</span>'}</td></tr>`;
    }).join('');
    const projRows = a.pm.rows.map((r) => `<tr class="${r.p.id === a.pm.bestId && r.score >= state.threshold ? 'win' : ''}"><td><b>${esc(r.p.name)}</b></td>
        <td>${r.tokens.map((t) => `<span class="chip ${r.overlap.includes(t) ? 'hit' : ''}">${esc(t)}</span>`).join('')}</td>
        <td>${r.score.toFixed(2)}<div class="bar"><div style="width:${r.score * 100}%;background:${r.score >= state.threshold ? 'var(--k-grpc)' : 'var(--k-error)'}"></div></div></td></tr>`).join('');
    const reasonTxt = a.dest === 'backlog' ? ` · reason = <code>${a.reason}</code>` : '';
    return `
      <div class="grid grid-2">
        <div>
          <h3 style="margin:0 0 6px;font-size:13px">1. Chấm điểm từ khoá <span class="sub">(classifier.go — luôn chạy để so sánh)</span></h3>
          <table class="score-table"><tr><th>Loại</th><th>Điểm</th><th>Từ khoá khớp</th></tr>${scoreRows}</table>
          <p class="sub" style="margin:6px 0 0;font-size:12px;color:var(--muted)">Hoà điểm ưu tiên event &gt; schedule &gt; task. Thời gian bóc được: ${a.time ? `<code>${esc(a.time.raw)}</code> → <code>${fmtTime(a.time)}</code>` : '<i>không có</i>'}</p>
          <h3 style="margin:14px 0 6px;font-size:13px">2. Khớp project <span class="sub">(điểm = token trùng / max(min(số token,5),3), cần ≥ 2 token trùng)</span></h3>
          <table class="score-table"><tr><th>Project</th><th>Token (tô đậm = có trong email)</th><th>Điểm</th></tr>${projRows || '<tr><td colspan="3">Không có project ứng viên</td></tr>'}</table>
        </div>
        <div>
          <h3 style="margin:0 0 6px;font-size:13px">3. Bản nháp (MailDraft) ${a.source === 'ai' ? '<span class="sub">— giả lập JSON DeepSeek</span>' : ''}</h3>
          <pre class="json mono">${esc(JSON.stringify(a.draft, null, 2))}</pre>
          <h3 style="margin:14px 0 6px;font-size:13px">4. Dispatch <span class="sub">(worker.go)</span></h3>
          <ul class="decision">${a.decision.map(([m, t]) => `<li><span class="mk">${m}</span><span>${t}</span></li>`).join('')}</ul>
          <div class="result-card"><div class="big">${d.icon}</div><div><div class="t">${d.name}</div>
            <div class="s">${d.rpc} → <code>${d.table}</code>${reasonTxt} · thông báo <code>${d.notice}</code></div></div></div>
          ${a.source === 'ai' ? '<p class="note" style="margin-top:10px">Trang tĩnh không gọi được DeepSeek, nên câu trả lời AI được giả lập (lấy theo rule hoặc theo category bạn chọn). Thực tế AI hiểu ngữ nghĩa và tự chọn project_id / project_confidence.</p>' : ''}
        </div>
      </div>`;
  }

  // ---------------------------------------------------------------- các bước
  function build() {
    const a = analyze(state);
    const d = DEST[a.dest];
    const st = [S('gmail', 'pubsub', 'Hộp thư INBOX có thư mới', 'ext', `Từ <code>${esc(state.from)}</code>: “${esc(state.subject)}”. Gmail báo về topic nhờ <code>users.watch</code> đã đăng ký lúc đăng nhập.`)];
    const deliver = (label) => {
      if (state.delivery === 'push') {
        st.push(
          S('pubsub', 'nginx', label || 'POST /api/mail/v1/notifications', 'ext', 'Push qua Cloudflare tunnel, kèm OIDC JWT. Payload chỉ có <code>emailAddress</code> + <code>historyId</code> — không có nội dung thư.'),
          S('nginx', 'mail', 'mail-provider:8084 → HandleNotification', 'grpc', 'Gateway xác thực OIDC (<code>MAIL_PUBSUB_AUDIENCE</code>), giải mã envelope, gọi gRPC :9084.'),
        );
      } else {
        st.push(S('mail', 'pubsub', label || 'subscriptions.pull (long-poll)', 'ext', 'Pull worker chạy khi đặt <code>MAIL_PULL_SUBSCRIPTION</code>; không cần endpoint công khai.'));
      }
    };
    deliver();
    if (state.queueFull) {
      st.push(
        S('mail', 'mail', 'Queue RAM đầy (MAIL_QUEUE_SIZE=256)', 'error', 'Lỗi tạm thời ⇒ KHÔNG ack để Pub/Sub gửi lại sau. Email không mất.'),
        S('mail', 'pubsub', 'Không ack / trả lỗi', 'error', null, null, { reply: true }),
      );
      deliver(state.delivery === 'push' ? 'Gửi lại notice (retry)' : 'Pull lại notice');
    }
    st.push(S('mail', 'mail', 'Enqueue → ack / 200 ngay', 'async', 'Xử lý nặng chạy ở 2 worker nền (<code>MAIL_WORKER_COUNT</code>).'));
    st.push(S('mail', 'pg', 'Đọc token + checkpoint historyId', 'db', 'Không tin historyId trong notice — dùng checkpoint đã lưu.', ['R mail_provider.sessions', 'R mail_provider.noti_indexes']));
    if (state.tokenExpired) {
      st.push(
        S('mail', 'google', 'POST oauth2 token (refresh_token)', 'ext', 'Access token hết hạn ⇒ refresh.'),
        S('mail', 'pg', 'Lưu access_token mới', 'db', null, ['U mail_provider.sessions']),
      );
    }
    st.push(
      S('mail', 'gmail', 'users.history.list(startHistoryId)', 'ext', 'Lấy mọi message mới kể từ checkpoint.'),
      S('mail', 'gmail', 'users.messages.get' + (state.attachments ? ' + attachments' : ''), 'ext', 'Lấy tiêu đề, nội dung (cắt ở 32 KB)' + (state.attachments ? ', tối đa 5 tệp ≤ 10 MiB' : '') + '.'),
      S('mail', 's3', 'PUT email gốc' + (state.attachments ? ' + tệp đính kèm' : ''), 'db', 'Lưu trước khi phân loại để luôn truy vết được.', ['mail/{profile}/{yyyymmdd}/{msg}.json'].concat(state.attachments ? ['mail/.../{msg}/attachments/{file}'] : [])),
      S('mail', 'project', 'ListProjects (ứng viên)', 'grpc', `Bỏ project mặc định và đã archive ⇒ ${a.projects.length} ứng viên.`, ['R project.projects']),
    );
    if (state.classifier === 'ai-ok') {
      st.push(
        S('mail', 'deepseek', 'POST /chat/completions', 'ext', 'Prompt gồm email + danh sách PROJECTS, yêu cầu trả đúng 1 JSON object.'),
        S('deepseek', 'mail', `JSON {category: "${a.category}"}`, 'ext', null, null, { reply: true }),
      );
    } else if (state.classifier === 'ai-fail') {
      st.push(
        S('mail', 'deepseek', 'POST /chat/completions', 'ext'),
        S('deepseek', 'mail', 'Lỗi / timeout / category lạ', 'error', null, null, { reply: true }),
        S('mail', 'mail', `Rule từ khoá dự phòng → ${a.category}`, 'self', 'classifyByRules: chấm điểm subject + body.'),
      );
    } else if (state.classifier === 'rule') {
      st.push(S('mail', 'mail', `Rule từ khoá → ${a.category}`, 'self', 'Không có DEEPSEEK_API_KEY ⇒ classifyByRules.'));
    } else {
      st.push(S('mail', 'mail', 'Không có bộ phân loại → other', 'self', 'reason = classifier_disabled'));
    }
    st.push(S('mail', 'mail', `Dispatch → ${d.svc}${a.dest === 'backlog' ? ` (${a.reason})` : ''}`, 'decision', a.decision.map(([m, t]) => `${m} ${t}`).join('<br>')));

    if (state.createFail && a.reason === 'create_failed') {
      const tried = a.category === 'schedule' ? 'calendar' : a.category;
      st.push(
        S('mail', tried, `${DEST[tried === 'calendar' ? 'calendar' : tried].rpc}`, 'grpc'),
        S(tried, 'mail', 'Lỗi khi tạo', 'error', null, null, { reply: true }),
      );
    }
    st.push(S('mail', d.svc, `${d.rpc}${a.dest === 'backlog' ? ` reason=${a.reason}` : ''}`, 'grpc', a.dest === 'backlog' ? 'Backlog lưu kèm sender, source (message id) và object_key email gốc — không email nào bị mất.' : `Gắn vào project <b>${esc(a.project ? a.project.name : '')}</b>.`));
    if (a.dest !== 'backlog') st.push(S(d.svc, 'project', 'GetProject (kiểm tra project)', 'grpc', null, ['R project.projects']));
    st.push(
      S(d.svc, 'pg', 'Ghi bản ghi mới', 'db', null, ['C ' + d.table]),
      S('mail', 'notification', `CreateNotice(${d.notice})`, 'grpc', 'Best-effort: lỗi chỉ log, không xử lý lại email.'),
      S('notification', 'pg', 'Lưu notice', 'db', null, ['C notification.notices']),
      S('notification', 'soketi', 'publish notice.created', 'http', 'Kênh <code>private-noti-internal-&lt;profile_id&gt;</code>.'),
      S('soketi', 'browser', 'WebSocket → chuông +1', 'ws', `Người dùng thấy: ${d.icon} ${d.name}.`),
      S('mail', 'pg', 'Tiến checkpoint historyId', 'db', 'Chỉ tiến SAU khi xử lý xong ⇒ notice trùng / lệch thứ tự không làm mất thư.', ['U mail_provider.noti_indexes']),
    );
    if (state.duplicate) {
      deliver(state.delivery === 'push' ? 'Notice trùng (cùng historyId)' : 'Pull nhận bản sao notice');
      st.push(
        S('mail', 'pg', 'Đọc checkpoint (đã tiến)', 'db', null, ['R mail_provider.noti_indexes']),
        S('mail', 'gmail', 'users.history.list → rỗng', 'ext', 'Không có message mới sau checkpoint.'),
        S('mail', 'mail', 'Không tạo bản ghi trùng', 'decision', 'Idempotent nhờ checkpoint historyId.'),
      );
    }
    return st;
  }

  window.Scenarios.list.unshift({
    id: 'mail', group: 'Email', icon: '📧', title: 'Email → Task / Lịch / Sự kiện',
    desc: 'Nhập một email bất kỳ và xem nó đi qua toàn hệ thống: Gmail → Pub/Sub → mail-provider → lưu S3 → phân loại → khớp project → tạo task / lịch / sự kiện hoặc vào backlog → thông báo realtime. Phần phân loại từ khoá và khớp project chạy đúng thuật toán trong code Go.',
    lanes: ['gmail', 'pubsub', 'nginx', 'mail', 'google', 'pg', 's3', 'project', 'deepseek', 'task', 'calendar', 'event', 'backlog', 'notification', 'soketi', 'browser'],
    options: [],
    mountPanel, extra, build,
    extraTitle: 'Kết quả phân loại',
  });
})();

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
      reason = category === 'other' ? 'ai_other' : '';
      decision.push(['✓', `AI đọc email và kết luận: đây là <b>${CAT[category]}</b>`]);
    } else if (st.classifier === 'off') {
      source = 'off';
      category = 'other';
      reason = 'classifier_disabled';
      decision.push(['✗', 'Tính năng tự phân loại đang tắt nên không đoán được']);
    } else {
      source = 'rule';
      category = ruleCat;
      actionable = category !== 'other';
      if (category === 'other') reason = 'no_rule_matched';
      decision.push([st.classifier === 'ai-fail' ? '⚠' : 'ℹ', st.classifier === 'ai-fail' ? 'AI bị lỗi nên hệ thống chuyển sang cách dự phòng: đếm từ khoá' : 'Chưa bật AI nên hệ thống đoán bằng cách đếm từ khoá']);
      decision.push(['✓', best ? `Nhiều từ khoá của loại <b>${CAT[category]}</b> nhất (${best} điểm)` : 'Không thấy từ khoá quen thuộc nào → coi là <b>thư thường</b>']);
    }

    const title = st.subject.trim() || st.body.trim().split('\n')[0] || 'Email không có tiêu đề';
    const start = (category === 'schedule' || category === 'event') ? time : null;
    const conf = pm.best;
    const project = projects.find((p) => p.id === pm.bestId);
    const noProject = () => decision.push(['✗', `Không dự án nào đủ giống (giống nhất mới ${pct(conf)}, cần từ ${pct(st.threshold)})`]);
    const okProject = () => decision.push(['✓', `Thuộc dự án <b>${esc(project.name)}</b> (giống ${pct(conf)})`]);
    let dest = 'backlog';
    if (category === 'task') {
      if (!actionable || !title) { reason = 'not_actionable'; decision.push(['✗', 'Không thấy việc gì cụ thể cần làm']); }
      else if (!project || conf < st.threshold) { reason = 'no_project_matched'; noProject(); }
      else { dest = 'task'; okProject(); }
    } else if (category === 'schedule' || category === 'event') {
      if (!start) { reason = category === 'schedule' ? 'missing_schedule_time' : 'missing_event_time'; decision.push(['✗', 'Không tìm thấy ngày giờ cụ thể trong email (VD cần dạng 25/11/2026 09:30)']); }
      else {
        decision.push(['✓', `Tìm thấy thời gian: <b>${fmtNice(start)}</b>`]);
        if (!project || conf < st.threshold) { reason = 'no_project_matched'; noProject(); }
        else { dest = category === 'schedule' ? 'calendar' : 'event'; okProject(); }
      }
    } else {
      decision.push(['→', 'Thư thường thì luôn để vào Hộp chờ xử lý cho bạn tự xem']);
    }
    if (dest !== 'backlog' && st.createFail) {
      decision.push(['✗', 'Lưu bị lỗi ⇒ giữ email trong Hộp chờ xử lý để không mất']);
      reason = 'create_failed';
      dest = 'backlog';
    }
    if (dest === 'backlog' && !reason) reason = 'unclassified';

    const draft = {
      category, is_actionable: !!actionable, title, description: st.body.trim().slice(0, 120) + (st.body.trim().length > 120 ? '…' : ''),
      priority: 'medium', start_at: fmtTime(start) || '', project_id: pm.bestId, project_confidence: +conf.toFixed(2),
      reason: category === 'other' ? reason : '',
    };
    return { scores, ruleCat, time, start, pm, projects, project, source, category, reason, dest, decision, draft, title, conf };
  }

  const CAT = { task: 'công việc', schedule: 'lịch hẹn / cuộc họp', event: 'sự kiện', other: 'thư thường' };
  const REASON = {
    no_project_matched: 'không biết email thuộc dự án nào',
    missing_schedule_time: 'là lịch hẹn nhưng không ghi ngày giờ',
    missing_event_time: 'là sự kiện nhưng không ghi ngày giờ',
    not_actionable: 'không có việc cụ thể cần làm',
    no_rule_matched: 'là thư thường (tin tức, quảng cáo…)',
    ai_other: 'AI cho rằng đây là thư thường',
    classifier_disabled: 'tính năng tự phân loại đang tắt',
    create_failed: 'lưu vào nơi đích bị lỗi',
    unclassified: 'không xếp được',
  };
  const pct = (x) => `${Math.round(x * 100)}%`;
  const fmtNice = (t) => `${String(t.d).padStart(2, '0')}/${String(t.mo).padStart(2, '0')}/${t.y} ${String(t.h).padStart(2, '0')}:${String(t.mi).padStart(2, '0')}`;

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
    { name: '🧾 Hoá đơn (không thuộc dự án)', subject: 'Thanh toán hoá đơn điện tháng 11', body: 'Quý khách cần thanh toán hoá đơn tiền điện trước 2026-11-30 để tránh gián đoạn dịch vụ.' },
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
          <h3>✉️ Thử gửi một email <span class="sub">chọn mẫu hoặc tự gõ</span></h3>
          <div class="preset-row">${PRESETS.map((p, i) => `<button class="btn" data-preset="${i}">${p.name}</button>`).join('')}</div>
          <div class="field"><label>Người gửi</label><input data-k="from"></div>
          <div class="field"><label>Tiêu đề</label><input data-k="subject"></div>
          <div class="field"><label>Nội dung</label><textarea data-k="body" rows="4"></textarea></div>
        </div>
        <div class="card">
          <h3>⚙️ Thiết lập & tình huống</h3>
          <div class="field"><label>Các dự án của bạn (mỗi dòng: Tên | Mô tả ngắn)</label><textarea data-k="projects" rows="3"></textarea></div>
          <div class="field-row">
            <div class="field"><label>Cách hệ thống đọc hiểu email</label>
              <select data-k="classifier">
                <option value="rule">Đếm từ khoá (chưa bật AI)</option>
                <option value="ai-fail">AI bị lỗi → tự chuyển sang đếm từ khoá</option>
                <option value="ai-ok">Dùng AI (mô phỏng)</option>
                <option value="off">Tắt tự phân loại</option>
              </select></div>
            <div class="field" data-show="ai-ok"><label>Giả sử AI kết luận là</label>
              <select data-k="aiCategory"><option value="auto">Giống kết quả đếm từ khoá</option><option value="task">Công việc</option><option value="schedule">Lịch hẹn / cuộc họp</option><option value="event">Sự kiện</option><option value="other">Thư thường</option></select></div>
            <div class="field"><label>Email phải giống dự án tối thiểu (0–1)</label><input data-k="threshold" type="number" step="0.05" min="0" max="1"></div>
            <div class="field"><label>Google báo thư mới bằng cách</label>
              <select data-k="delivery"><option value="push">Google chủ động gọi vào hệ thống</option><option value="pull">Hệ thống tự hỏi Google liên tục</option></select></div>
          </div>
          <div class="options">
            <label class="opt"><input type="checkbox" data-k="attachments"> Email có file đính kèm</label>
            <label class="opt"><input type="checkbox" data-k="tokenExpired"> Quyền đọc Gmail đã hết hạn</label>
            <label class="opt"><input type="checkbox" data-k="queueFull"> Hệ thống đang quá tải</label>
            <label class="opt"><input type="checkbox" data-k="duplicate"> Google báo trùng 2 lần</label>
            <label class="opt"><input type="checkbox" data-k="createFail"> Lưu vào nơi đích bị lỗi</label>
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
    task: { icon: '📝', name: 'Thành công việc mới trên bảng', svc: 'task', short: 'công việc', notice: 'TASK_CREATED', rpc: 'CreateTask', table: 'task.tasks' },
    calendar: { icon: '📅', name: 'Thành lịch hẹn mới trong Lịch', svc: 'calendar', short: 'lịch hẹn', notice: 'SCHEDULE_CREATED', rpc: 'CreateSchedule', table: 'calendar.schedules' },
    event: { icon: '🎉', name: 'Thành sự kiện mới', svc: 'event', short: 'sự kiện', notice: 'EVENT_CREATED', rpc: 'CreateEvent', table: 'event.events' },
    backlog: { icon: '📥', name: 'Vào Hộp chờ xử lý (bạn tự xem)', svc: 'backlog', short: 'mục trong Hộp chờ', notice: 'BACKLOG_CREATED', rpc: 'CreateBacklog', table: 'backlog.backlogs' },
  };

  function extra() {
    const a = analyze(state);
    const d = DEST[a.dest];
    const max = Math.max(1, ...Object.values(a.scores).map((x) => x.score));
    const scoreRows = ['event', 'schedule', 'task'].map((c) => {
      const s = a.scores[c];
      return `<tr class="${a.ruleCat === c ? 'win' : ''}"><td><b>${CAT[c][0].toUpperCase() + CAT[c].slice(1)}</b></td><td>${s.score}<div class="bar"><div style="width:${(s.score / max) * 100}%"></div></div></td>
        <td>${s.hits.length ? s.hits.map((h) => `<span class="chip hit">${esc(h.kw)}${h.inSubject ? ' ×2' : ''}</span>`).join('') : '<span class="sub">—</span>'}</td></tr>`;
    }).join('');
    const projRows = a.pm.rows.map((r) => `<tr class="${r.p.id === a.pm.bestId && r.score >= state.threshold ? 'win' : ''}"><td><b>${esc(r.p.name)}</b></td>
        <td>${r.tokens.map((t) => `<span class="chip ${r.overlap.includes(t) ? 'hit' : ''}">${esc(t)}</span>`).join('')}</td>
        <td>${pct(r.score)}<div class="bar"><div style="width:${r.score * 100}%;background:${r.score >= state.threshold ? 'var(--k-grpc)' : 'var(--k-error)'}"></div></div></td></tr>`).join('');
    const understood = [
      ['Loại email', CAT[a.category]],
      ['Tiêu đề', esc(a.title)],
      ['Thời gian', a.start ? fmtNice(a.start) : '—'],
      ['Dự án', a.project && a.conf >= state.threshold ? esc(a.project.name) : '— (không rõ)'],
    ];
    return `
      <div class="grid grid-2">
        <div>
          <h3 style="margin:0 0 6px;font-size:13px">1. Email này giống loại nào? <span class="sub">đếm các từ khoá quen thuộc</span></h3>
          <table class="score-table"><tr><th>Loại</th><th>Điểm</th><th>Từ khoá tìm thấy</th></tr>${scoreRows}</table>
          <p style="margin:6px 0 0;font-size:12px;color:var(--muted)">Từ khoá nằm ở tiêu đề được tính gấp đôi (×2). Bằng điểm thì ưu tiên: sự kiện → lịch hẹn → công việc.
            Ngày giờ tìm thấy: ${a.time ? `<b>${fmtNice(a.time)}</b> (từ chữ “${esc(a.time.raw)}”)` : '<i>không có</i>'}</p>
          <h3 style="margin:14px 0 6px;font-size:13px">2. Email thuộc dự án nào? <span class="sub">so các từ trong tên + mô tả dự án với email</span></h3>
          <table class="score-table"><tr><th>Dự án</th><th>Các từ (tô màu = có trong email)</th><th>Giống</th></tr>${projRows || '<tr><td colspan="3">Bạn chưa có dự án nào</td></tr>'}</table>
          <p style="margin:6px 0 0;font-size:12px;color:var(--muted)">Cần trùng ít nhất 2 từ, và độ giống từ ${pct(state.threshold)} trở lên. Dự án mặc định "Chung" không được tính.</p>
        </div>
        <div>
          <h3 style="margin:0 0 6px;font-size:13px">3. Hệ thống hiểu email là ${a.source === 'ai' ? '<span class="sub">(AI — mô phỏng)</span>' : ''}</h3>
          <table class="score-table">${understood.map(([k, v]) => `<tr><td style="width:90px;color:var(--muted)">${k}</td><td><b>${v}</b></td></tr>`).join('')}</table>
          <h3 style="margin:14px 0 6px;font-size:13px">4. Quyết định</h3>
          <ul class="decision">${a.decision.map(([m, t]) => `<li><span class="mk">${m}</span><span>${t}</span></li>`).join('')}</ul>
          <div class="result-card"><div class="big">${d.icon}</div><div><div class="t">${d.name}</div>
            <div class="s">${a.dest === 'backlog' ? `Lý do: ${REASON[a.reason] || a.reason}. ` : ''}Bạn nhận một thông báo ở chuông.</div></div></div>
          <details class="tech" style="margin-top:10px"><summary>Chi tiết kỹ thuật</summary>
            <p class="note" style="margin:6px 0">${d.rpc} → <code>${d.table}</code>${a.dest === 'backlog' ? ` · reason = <code>${a.reason}</code>` : ''} · notice <code>${d.notice}</code></p>
            <pre class="json mono">${esc(JSON.stringify(a.draft, null, 2))}</pre></details>
          ${a.source === 'ai' ? '<p class="note" style="margin-top:10px">Trang này không gọi AI thật, nên kết quả AI chỉ là giả định (theo lựa chọn ở trên). AI thật hiểu ý nghĩa câu chữ tốt hơn đếm từ khoá.</p>' : ''}
        </div>
      </div>`;
  }

  // ---------------------------------------------------------------- các bước
  function build() {
    const a = analyze(state);
    const d = DEST[a.dest];
    const st = [S('gmail', 'pubsub', 'Hộp thư của bạn có thư mới', 'ext', `Thư từ ${esc(state.from)}: “${esc(state.subject)}”. Gmail chỉ báo "có thay đổi" cho dịch vụ báo tin của Google, chưa gửi nội dung thư.`, 'Gmail users.watch → Pub/Sub topic', { call: 'async' })];
    const deliver = (label) => {
      if (state.delivery === 'push') {
        st.push(
          S('pubsub', 'nginx', label || 'Google gọi vào hệ thống: "có thư mới"', 'ext', 'Google đợi hệ thống trả lời "đã nhận"; nếu không nhận được câu trả lời, Google sẽ báo lại sau.', ['POST /api/mail/v1/notifications', 'qua Cloudflare tunnel, OIDC JWT', 'payload: emailAddress + historyId'], { ret: 'đã nhận' }),
          S('nginx', 'mail', 'Chuyển tới Bộ xử lý email', 'grpc', 'Cổng vào kiểm tra đúng là Google gửi rồi chuyển tiếp.', ['mail-provider:8084 → HandleNotification (gRPC :9084)', 'verify OIDC MAIL_PUBSUB_AUDIENCE']),
        );
      } else {
        st.push(S('mail', 'pubsub', label || 'Hỏi Google: "có tin gì mới không?"', 'ext', 'Hệ thống liên tục hỏi Google; có tin thì Google trả về ngay.', 'subscriptions.pull (long-poll), MAIL_PULL_SUBSCRIPTION', { ret: 'tin báo có thư mới' }));
      }
    };
    const acceptNotice = () => {
      st.push(S('mail', 'mail', 'Ghi tin báo vào danh sách việc cần làm', 'self', 'Chưa xử lý ngay — chỉ xếp hàng để trả lời Google thật nhanh.', 'queue trong RAM (MAIL_QUEUE_SIZE=256)'));
      if (state.delivery === 'push') {
        st.push(S('mail', 'pubsub', 'Trả lời Google: "đã nhận"', 'http', 'Google yên tâm, không báo lại nữa. Việc nặng (đọc thư, phân loại) sẽ làm ngầm ngay sau đây.', '200 OK', { reply: true }));
      } else {
        st.push(S('mail', 'pubsub', 'Báo Google: "đã nhận"', 'ext', 'Để Google không gửi lại tin này.', 'acknowledge(ackIds)', { ret: 'OK' }));
      }
    };
    deliver();
    if (state.queueFull) {
      st.push(S('mail', 'mail', 'Danh sách việc đã đầy — chưa nhận được', 'error', 'Hệ thống đang quá tải nên cố ý chưa nhận. Thư không mất vì Google sẽ báo lại.', 'queue đầy, không ack'));
      if (state.delivery === 'push') st.push(S('mail', 'pubsub', 'Trả lời Google: "chưa nhận được, báo lại sau"', 'error', null, '503', { reply: true }));
      else st.push(S('mail', 'mail', 'Không báo "đã nhận"', 'error', 'Hết thời gian chờ, Google sẽ tự gửi lại tin này.', 'không acknowledge'));
      deliver(state.delivery === 'push' ? 'Google báo lại lần nữa' : 'Hỏi lại Google');
    }
    acceptNotice();
    st.push(S('mail', 'mail', 'Bắt đầu xử lý ngầm', 'self', 'Từ đây không ai bên ngoài phải đợi nữa.', 'worker nền, MAIL_WORKER_COUNT=2'));
    st.push(S('mail', 'pg', 'Lấy quyền đọc Gmail và "mốc" lần đọc trước', 'db', 'Mốc này cho biết thư nào đã đọc rồi, để chỉ lấy thư mới.', ['R mail_provider.sessions', 'R mail_provider.noti_indexes (historyId)']));
    if (state.tokenExpired) {
      st.push(
        S('mail', 'google', 'Xin cấp lại quyền đọc Gmail', 'ext', 'Quyền cũ đã hết hạn; xin quyền mới mà không cần bạn đăng nhập lại.', 'POST oauth2 token (refresh_token)', { ret: 'quyền mới' }),
        S('mail', 'pg', 'Lưu quyền mới', 'db', null, 'U mail_provider.sessions'),
      );
    }
    st.push(
      S('mail', 'gmail', 'Hỏi Gmail: "từ mốc trước tới giờ có thư nào mới?"', 'ext', null, 'users.history.list(startHistoryId)', { ret: 'danh sách thư mới' }),
      S('mail', 'gmail', 'Lấy nội dung thư' + (state.attachments ? ' và file đính kèm' : ''), 'ext', 'Lấy tiêu đề, người gửi, nội dung' + (state.attachments ? ' và tối đa 5 file (mỗi file ≤ 10 MB)' : '') + '.', 'users.messages.get' + (state.attachments ? ' + attachments.get' : ''), { ret: 'nội dung thư' + (state.attachments ? ' + file' : '') }),
      S('mail', 's3', 'Cất bản gốc của thư' + (state.attachments ? ' và file' : '') + ' vào kho', 'db', 'Để sau này luôn xem lại được thư gốc, dù hệ thống xếp nó vào đâu.', ['mail/{profile}/{yyyymmdd}/{msg}.json'].concat(state.attachments ? ['mail/{profile}/{yyyymmdd}/{msg}/attachments/{file}'] : [])),
      S('mail', 'project', 'Lấy danh sách dự án của bạn', 'grpc', `Để đoán xem thư thuộc dự án nào. Có ${a.projects.length} dự án để so (không tính "Chung" và dự án đã lưu trữ).`, 'ListProjects → R project.projects', { ret: `${a.projects.length} dự án` }),
    );
    if (state.classifier === 'ai-ok') {
      st.push(
        S('mail', 'deepseek', 'Nhờ AI đọc thư', 'ext', 'Gửi nội dung thư và danh sách dự án cho AI, nhờ AI cho biết: loại thư, tiêu đề, thời gian, thuộc dự án nào.', 'POST /chat/completions (DeepSeek), yêu cầu trả JSON'),
        S('deepseek', 'mail', `AI trả lời: đây là ${CAT[a.category]}`, 'ext', null, `{"category": "${a.category}", ...}`, { reply: true }),
      );
    } else if (state.classifier === 'ai-fail') {
      st.push(
        S('mail', 'deepseek', 'Nhờ AI đọc thư', 'ext', null, 'POST /chat/completions'),
        S('deepseek', 'mail', 'AI bị lỗi / không trả lời', 'error', null, 'timeout / 5xx / category không hợp lệ', { reply: true }),
        S('mail', 'mail', `Dự phòng: đếm từ khoá → ${CAT[a.category]}`, 'self', 'Không để email bị kẹt chỉ vì AI lỗi.', 'classifyByRules'),
      );
    } else if (state.classifier === 'rule') {
      st.push(S('mail', 'mail', `Đếm từ khoá → ${CAT[a.category]}`, 'self', 'Chưa bật AI nên hệ thống tự đoán bằng từ khoá.', 'DEEPSEEK_API_KEY rỗng → classifyByRules'));
    } else {
      st.push(S('mail', 'mail', 'Không phân loại → coi là thư thường', 'self', null, 'reason = classifier_disabled'));
    }
    st.push(S('mail', 'mail', `Quyết định: ${d.name.toLowerCase()}`, 'decision', a.decision.map(([m, t]) => `${m} ${t}`).join('<br>')));

    if (state.createFail && a.reason === 'create_failed') {
      const tried = a.category === 'schedule' ? 'calendar' : a.category;
      st.push(
        S('mail', tried, `Nhờ tạo ${DEST[tried].short}`, 'grpc', null, DEST[tried].rpc),
        S(tried, 'mail', 'Lưu bị lỗi', 'error', 'Vì vậy thư sẽ được giữ trong Hộp chờ xử lý.', null, { reply: true }),
      );
    }
    st.push(S('mail', d.svc, `Nhờ tạo ${d.short}`, 'grpc',
      (a.dest === 'backlog' ? `Thư được giữ lại kèm lý do “${REASON[a.reason] || a.reason}” để bạn tự xem — không email nào bị mất.` : `Gắn vào dự án “${esc(a.project ? a.project.name : '')}”.`)
        + ` Bộ xử lý email <b>đợi</b> kết quả, vì cần mã của ${d.short} vừa tạo để gắn vào thông báo.`,
      `${d.rpc}${a.dest === 'backlog' ? ` (reason=${a.reason})` : ''}`, { ret: `mã ${d.short} mới` }));
    if (a.dest !== 'backlog') st.push(S(d.svc, 'project', 'Kiểm tra lại dự án còn dùng được', 'grpc', 'Chắc chắn dự án chưa bị lưu trữ trước khi thêm vào.', 'GetProject → R project.projects', { ret: 'dự án hợp lệ' }));
    st.push(
      S(d.svc, 'pg', `Lưu ${d.short}`, 'db', `Lưu xong mới gửi kết quả lại cho Bộ xử lý email ở bước kế tiếp.`, 'C ' + d.table),
      S('mail', 'notification', 'Nhờ báo cho bạn biết', 'grpc', 'Nếu bước này lỗi thì chỉ mất thông báo, còn ' + d.short + ' vẫn được giữ.', `CreateNotice(${d.notice})`, { ret: 'đã tạo thông báo' }),
      S('notification', 'pg', 'Lưu thông báo', 'db', null, 'C notification.notices'),
      S('notification', 'soketi', 'Phát thông báo lên kênh riêng của bạn', 'http', null, 'publish notice.created → private-noti-internal-<profile_id>', { ret: 'đã phát' }),
      S('soketi', 'browser', 'Chuông sáng lên trên màn hình', 'ws', `Bạn thấy: ${d.icon} ${d.name}.`),
      S('mail', 'pg', 'Dời "mốc" tới thư vừa xử lý', 'db', 'Chỉ dời mốc khi đã xử lý xong, nên dù Google báo trùng hay báo lộn thứ tự cũng không mất thư.', 'U mail_provider.noti_indexes (historyId)'),
    );
    if (state.duplicate) {
      deliver(state.delivery === 'push' ? 'Google báo trùng lần nữa' : 'Lại nhận được tin báo cũ');
      acceptNotice();
      st.push(
        S('mail', 'pg', 'Lấy "mốc" (đã được dời ở trên)', 'db', null, 'R mail_provider.noti_indexes'),
        S('mail', 'gmail', 'Hỏi Gmail: "từ mốc mới có thư nào không?"', 'ext', null, 'users.history.list(startHistoryId)', { ret: 'không có thư mới' }),
        S('mail', 'mail', 'Không làm gì — không tạo trùng', 'decision', 'Nhờ có "mốc", một thư không bao giờ bị xử lý hai lần.'),
      );
    }
    return st;
  }

  window.Scenarios.list.unshift({
    id: 'mail', group: 'Email', icon: '📧', title: 'Email tự thành công việc / lịch',
    desc: 'Gõ thử một email và xem hệ thống tự làm gì: đọc thư mới từ Gmail, đoán xem đó là công việc, lịch hẹn, sự kiện hay thư thường, đoán thuộc dự án nào, rồi tạo giúp bạn và báo lên chuông. Thư không xếp được sẽ vào "Hộp chờ xử lý" để bạn tự xem — không thư nào bị mất.',
    lanes: ['gmail', 'pubsub', 'nginx', 'mail', 'google', 'pg', 's3', 'project', 'deepseek', 'task', 'calendar', 'event', 'backlog', 'notification', 'soketi', 'browser'],
    options: [],
    mountPanel, extra, build,
    extraTitle: 'Hệ thống xử lý email này thế nào?',
  });
})();

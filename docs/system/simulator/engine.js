/* Engine: danh mục thành phần, vẽ sơ đồ kiến trúc, sơ đồ tuần tự và trình phát từng bước. */
(function () {
  'use strict';

  const SVG_NS = 'http://www.w3.org/2000/svg';
  const NODE_W = 140;
  const NODE_H = 44;

  // Mỗi thành phần: vị trí trên sơ đồ kiến trúc + thông tin hiển thị khi bấm vào.
  // label = tên dễ hiểu cho người dùng; sub = tên kỹ thuật trong code.
  const NODES = {
    browser: { label: 'Trình duyệt', sub: 'ứng dụng web của bạn', x: 85, y: 250, cls: 'client',
      info: { 'Là gì': 'Trang web Task Manager bạn đang mở trên máy.', 'Làm gì': 'Gửi yêu cầu khi bạn bấm nút, và nhận thông báo tức thì (chuông, tin nhắn).', 'Kỹ thuật': 'React SPA; gọi /api/* qua nginx, WebSocket tới Soketi' } },
    peer: { label: 'Trình duyệt B', sub: 'người thứ hai', x: 85, y: 400, cls: 'client',
      info: { 'Là gì': 'Máy của một người dùng khác — người được kết bạn hoặc nhận tin nhắn.' } },
    nginx: { label: 'Cổng vào', sub: 'nginx', x: 250, y: 325, cls: 'edge',
      info: { 'Là gì': 'Cửa duy nhất để đi vào hệ thống từ bên ngoài.', 'Làm gì': 'Nhận mọi yêu cầu rồi chuyển đúng tới bộ phận phụ trách, sau đó chuyển kết quả ngược lại.', 'Kỹ thuật': 'nginx reverse proxy /api/<service>/; host 127.0.0.1:3000' } },

    identity: { label: 'Tài khoản', sub: 'identity', x: 440, y: 50, cls: 'svc',
      info: { 'Là gì': 'Bộ phận quản lý tài khoản người dùng.', 'Làm gì': 'Đăng nhập bằng Google, cấp "vé vào cửa" (phiên đăng nhập), tạo sẵn dữ liệu ban đầu cho người mới.', 'Lưu ở': 'Hồ sơ người dùng', 'Kỹ thuật': 'identity :8081/:9081 — JWT HS256, identity.profiles, identity.auth_providers' } },
    project: { label: 'Dự án', sub: 'project', x: 440, y: 115, cls: 'svc',
      info: { 'Là gì': 'Bộ phận quản lý dự án (nhóm công việc).', 'Làm gì': 'Tạo, đổi tên, lưu trữ dự án. Mỗi người có sẵn dự án "Chung".', 'Lưu ở': 'Danh sách dự án', 'Kỹ thuật': 'project :8090/:9090 — project.projects' } },
    task: { label: 'Công việc', sub: 'task', x: 440, y: 180, cls: 'svc',
      info: { 'Là gì': 'Bộ phận quản lý công việc (task).', 'Làm gì': 'Tạo/sửa công việc, đổi trạng thái (To Do → Completed), nhắc việc đúng giờ, công việc lặp lại.', 'Lưu ở': 'Công việc, trạng thái, lịch sử thay đổi', 'Kỹ thuật': 'task :8082/:9082 — task.tasks, task_statuses, status_transitions, task_status_logs' } },
    calendar: { label: 'Lịch', sub: 'calendar', x: 440, y: 245, cls: 'svc',
      info: { 'Là gì': 'Bộ phận quản lý lịch cá nhân.', 'Làm gì': 'Lưu các cuộc hẹn, cuộc họp có giờ bắt đầu/kết thúc.', 'Kỹ thuật': 'calendar :8083/:9083 — calendar.schedules' } },
    event: { label: 'Sự kiện', sub: 'event', x: 440, y: 310, cls: 'svc',
      info: { 'Là gì': 'Bộ phận quản lý sự kiện.', 'Làm gì': 'Lưu hội thảo, tiệc, lễ… và trạng thái dự kiến / xác nhận / huỷ.', 'Kỹ thuật': 'event :8085/:9085 — event.events' } },
    backlog: { label: 'Hộp chờ xử lý', sub: 'backlog', x: 440, y: 375, cls: 'svc',
      info: { 'Là gì': 'Nơi giữ các email hệ thống chưa tự xếp được.', 'Làm gì': 'Giữ email kèm lý do để bạn tự xem và phân loại — không email nào bị mất.', 'Kỹ thuật': 'backlog :8086/:9086 — backlog.backlogs' } },
    attachment: { label: 'Tệp đính kèm', sub: 'attachment', x: 440, y: 440, cls: 'svc',
      info: { 'Là gì': 'Bộ phận quản lý file đính kèm.', 'Làm gì': 'Nhận file bạn tải lên, cất vào kho tệp, cho tải về.', 'Kỹ thuật': 'attachment :8089/:9089 — attachment.attachments + RustFS' } },
    report: { label: 'Báo cáo', sub: 'report', x: 440, y: 505, cls: 'svc',
      info: { 'Là gì': 'Bộ phận làm báo cáo.', 'Làm gì': 'Hỏi số liệu từ các bộ phận khác rồi tổng hợp thành biểu đồ. Bản thân không lưu dữ liệu.', 'Kỹ thuật': 'report :8087/:9087 — không có DB, gọi gRPC' } },

    notification: { label: 'Thông báo', sub: 'notification', x: 640, y: 70, cls: 'svc',
      info: { 'Là gì': 'Bộ phận thông báo (cái chuông).', 'Làm gì': 'Lưu thông báo, đếm số chưa đọc, và đẩy ngay lên màn hình.', 'Kỹ thuật': 'notification :8088/:9088 — notification.notices; publish Soketi' } },
    social: { label: 'Bạn bè', sub: 'social', x: 640, y: 150, cls: 'svc',
      info: { 'Là gì': 'Bộ phận kết bạn.', 'Làm gì': 'Tìm người theo email, gửi / chấp nhận / huỷ lời mời kết bạn.', 'Kỹ thuật': 'social :8091/:9091 — social.friendships' } },
    message: { label: 'Tin nhắn', sub: 'message', x: 640, y: 230, cls: 'svc',
      info: { 'Là gì': 'Bộ phận nhắn tin giữa hai người bạn.', 'Làm gì': 'Chuyển tin nhắn tới người nhận ngay lập tức, rồi gửi sang hàng đợi để lưu sau.', 'Kỹ thuật': 'message :8092/:9092 — Soketi + Kafka, đọc message.*' } },
    worker: { label: 'Bộ lưu tin nhắn', sub: 'worker', x: 640, y: 310, cls: 'svc',
      info: { 'Là gì': 'Bộ phận chạy ngầm chuyên lưu tin nhắn.', 'Làm gì': 'Lấy tin nhắn từ hàng đợi và cất vào cơ sở dữ liệu, không bao giờ lưu trùng.', 'Kỹ thuật': 'worker ×N — consume Kafka chat.messages, ghi message.messages' } },
    mail: { label: 'Bộ xử lý email', sub: 'mail-provider', x: 640, y: 460, cls: 'svc',
      info: { 'Là gì': 'Bộ phận đọc Gmail của bạn.', 'Làm gì': 'Khi có thư mới: đọc thư, đoán xem đó là công việc / lịch / sự kiện, rồi tạo giúp bạn.', 'Kỹ thuật': 'mail-provider :8084/:9084 — Gmail API, Pub/Sub, DeepSeek, RustFS' } },

    pg: { label: 'Cơ sở dữ liệu', sub: 'PostgreSQL', x: 840, y: 90, cls: 'infra',
      info: { 'Là gì': 'Nơi lưu trữ lâu dài mọi dữ liệu (như một cuốn sổ cái).', 'Làm gì': 'Mỗi bộ phận có một ngăn riêng của mình.', 'Kỹ thuật': 'postgres:5432, DB task_manager, schema riêng mỗi service' } },
    s3: { label: 'Kho lưu tệp', sub: 'RustFS (S3)', x: 840, y: 210, cls: 'infra',
      info: { 'Là gì': 'Kho chứa file.', 'Làm gì': 'Cất bản gốc email và các file đính kèm.', 'Kỹ thuật': 'rustfs:9000, bucket task-manager' } },
    kafka: { label: 'Hàng đợi', sub: 'Kafka', x: 840, y: 330, cls: 'infra',
      info: { 'Là gì': 'Hàng đợi tin nhắn — như một băng chuyền.', 'Làm gì': 'Giữ tin nhắn theo đúng thứ tự cho tới khi được lưu xong.', 'Kỹ thuật': 'kafka:9092, topic chat.messages ×6 partition' } },
    soketi: { label: 'Kênh tức thì', sub: 'Soketi', x: 840, y: 450, cls: 'infra',
      info: { 'Là gì': 'Đường dây luôn mở giữa hệ thống và trình duyệt.', 'Làm gì': 'Đẩy thông báo, tin nhắn lên màn hình ngay lập tức mà không cần tải lại trang.', 'Kỹ thuật': 'soketi:6001, WebSocket giao thức Pusher' } },

    google: { label: 'Đăng nhập Google', sub: 'Google OAuth', x: 1050, y: 90, cls: 'ext',
      info: { 'Là gì': 'Dịch vụ đăng nhập của Google.', 'Làm gì': 'Xác nhận bạn là ai và cấp quyền đọc Gmail nếu bạn đồng ý.' } },
    gmail: { label: 'Gmail', sub: 'Gmail API', x: 1050, y: 210, cls: 'ext',
      info: { 'Là gì': 'Hộp thư Gmail của bạn.', 'Làm gì': 'Cho hệ thống đọc thư mới (chỉ đọc, không gửi).' } },
    pubsub: { label: 'Google báo tin', sub: 'Pub/Sub', x: 1050, y: 330, cls: 'ext',
      info: { 'Là gì': 'Dịch vụ của Google chuyên báo tin.', 'Làm gì': 'Báo cho hệ thống biết "hộp thư vừa có thay đổi".' } },
    deepseek: { label: 'AI đọc email', sub: 'DeepSeek', x: 1050, y: 450, cls: 'ext',
      info: { 'Là gì': 'Trí tuệ nhân tạo bên ngoài.', 'Làm gì': 'Đọc nội dung email và đoán xem đó là công việc, lịch, sự kiện hay thư thường.' } },
  };

  const GROUPS = [
    { label: 'Người dùng', x: 10, y: 205, w: 150, h: 250 },
    { label: 'Chức năng chính', x: 362, y: 12, w: 156, h: 525 },
    { label: 'Liên lạc & email', x: 562, y: 32, w: 156, h: 460 },
    { label: 'Nơi lưu trữ', x: 762, y: 52, w: 156, h: 440 },
    { label: 'Dịch vụ bên ngoài', x: 972, y: 52, w: 156, h: 440 },
  ];

  const KIND_LABEL = {
    http: 'Gửi yêu cầu qua web', grpc: 'Bộ phận nhờ bộ phận', db: 'Lưu / lấy dữ liệu', async: 'Gửi đi, không đợi',
    ext: 'Nhờ dịch vụ bên ngoài', ws: 'Đẩy lên màn hình ngay', self: 'Tự xử lý', decision: 'Tự quyết định', error: 'Lỗi / bị từ chối',
    reply: 'Gửi kết quả về',
  };

  // ---------------------------------------------------------------- Mô hình gọi đồng bộ / bất đồng bộ
  // Mỗi bước được gán "mode":
  //   sync   — bên gọi đứng chờ bên được gọi trả lời (gRPC / HTTP / API ngoài)
  //   db     — đọc/ghi Postgres hoặc S3, chờ kết quả ngay trong cùng bước
  //   async  — gửi đi rồi làm tiếp, không chờ (Kafka, WebSocket, sự kiện Gmail)
  //   return — bên được gọi trả kết quả về cho bên gọi
  //   local  — tự xử lý bên trong, không gọi ai
  // expandCalls() duyệt kịch bản như một call stack và tự chèn bước "trả về" ngay khi
  // luồng điều khiển quay lại bên gọi, để thấy rõ ai chờ ai.
  const STORES = new Set(['pg', 's3']);
  function modeOf(s) {
    if (s.f === s.t) return 'local';
    if (s.reply) return 'return';
    if (s.call) return s.call;
    if (s.kind === 'async' || s.kind === 'ws') return 'async';
    if (s.kind === 'db' || STORES.has(s.t)) return 'db';
    return 'sync';
  }

  function expandCalls(raw) {
    const out = [];
    const stack = [];
    const snap = () => stack.map((fr) => ({ caller: fr.caller, callee: fr.callee, idx: fr.idx }));
    const emitReturn = (src) => {
      const fr = stack.pop();
      const call = out[fr.idx];
      const r = Object.assign({}, src || {}, {
        f: fr.callee, t: fr.caller, mode: 'return', retOf: fr.idx, reply: true,
        kind: src && src.kind === 'error' ? 'error' : 'reply',
        label: src ? src.label : `Trả về: ${call.ret || 'kết quả'}`,
        auto: !src,
      });
      call.retAt = out.length;
      r.stack = snap();
      out.push(r);
      return fr;
    };
    for (const s0 of raw) {
      const s = Object.assign({}, s0);
      s.mode = modeOf(s);
      if (s.mode === 'return') {
        const i = stack.map((fr) => fr.callee).lastIndexOf(s.f);
        if (i < 0) { s.stack = snap(); out.push(s); continue; }
        while (stack.length - 1 > i) emitReturn();
        const target = s.t;
        let fr = emitReturn(s);
        // phản hồi đi ngược qua các tầng trung gian (vd task → nginx → browser)
        while (fr.caller !== target && stack.length && stack[stack.length - 1].callee === fr.caller) {
          const via = fr.caller;
          fr = emitReturn({ label: `Chuyển tiếp: ${s.label}`, kind: s.kind, detail: `${NODES[via].label} không tự xử lý gì, chỉ chuyển kết quả về cho bên đã gửi yêu cầu.` });
        }
        continue;
      }
      // điều khiển quay về s.f ⇒ mọi lời gọi s.f đang chờ phía trên đã xong, trả kết quả trước
      while (stack.length && stack[stack.length - 1].callee !== s.f) emitReturn();
      if (s.mode === 'sync') stack.push({ caller: s.f, callee: s.t, idx: out.length });
      s.stack = snap();
      out.push(s);
    }
    while (stack.length) emitReturn();
    return out;
  }

  // Câu giải thích dễ hiểu cho một bước.
  function describe(steps, i) {
    const s = steps[i];
    if (!s) return '';
    const A = `<b>${esc(NODES[s.f].label)}</b>`;
    const B = `<b>${esc(NODES[s.t].label)}</b>`;
    switch (s.mode) {
      case 'sync':
        return `${A} nhờ ${B} làm việc này và <b>đợi</b> ${B} làm xong mới đi tiếp. `
          + (s.retAt != null ? `${B} sẽ gửi kết quả lại ở <a href="#" data-goto="${s.retAt}">bước ${s.retAt + 1}</a>.` : '');
      case 'db':
        return s.t === 's3'
          ? `${A} cất / lấy file trong ${B}. Việc này xong ngay trong bước này.`
          : `${A} ghi / đọc dữ liệu trong ${B}. Việc này xong ngay trong bước này.`;
      case 'async':
        return s.kind === 'ws'
          ? `${A} đẩy thông tin lên màn hình ${B} <b>ngay lập tức</b>, không cần tải lại trang và không cần ai trả lời.`
          : `${A} gửi sang ${B} rồi <b>làm việc khác luôn</b>, không đợi ${B} trả lời.`;
      case 'return':
        return s.retOf != null
          ? `${A} đã làm xong và <b>gửi kết quả</b> lại cho ${B} (trả lời cho <a href="#" data-goto="${s.retOf}">bước ${s.retOf + 1}</a>). ${B} hết phải đợi và làm tiếp.`
          : `${A} gửi kết quả lại cho ${B}.`;
      default:
        return s.kind === 'decision' ? `${A} tự xem xét thông tin đã có để quyết định làm gì tiếp theo.` : `${A} tự xử lý, không cần nhờ ai.`;
    }
  }

  const MODE_BADGE = {
    sync: (s) => `⏳ Nhờ và đợi kết quả${s.retAt != null ? ` (có ở bước ${s.retAt + 1})` : ''}`,
    db: () => '🗄️ Lưu / lấy dữ liệu, xong ngay',
    async: () => '🚀 Gửi đi, không đợi trả lời',
    return: (s) => `↩️ Gửi kết quả${s.retOf != null ? ` cho bước ${s.retOf + 1}` : ''}`,
    local: () => '⚙️ Tự xử lý',
  };

  function el(tag, attrs, parent) {
    const node = document.createElementNS(SVG_NS, tag);
    for (const key in attrs || {}) node.setAttribute(key, attrs[key]);
    if (parent) parent.appendChild(node);
    return node;
  }

  function kindColor(kind) {
    return `var(--k-${kind === 'decision' ? 'self' : kind})`;
  }

  // ---------------------------------------------------------------- Map
  class ArchMap {
    constructor(container, { onNodeClick } = {}) {
      this.container = container;
      this.onNodeClick = onNodeClick;
      this.svg = el('svg', { class: 'map', viewBox: '0 0 1140 560', role: 'img', 'aria-label': 'Sơ đồ kiến trúc' });
      container.innerHTML = '';
      container.appendChild(this.svg);
      this.edgeLayer = el('g', {}, this.svg);
      this.nodeLayer = el('g', {}, this.svg);
      this.packetLayer = el('g', {}, this.svg);
      this.nodeEls = {};
      this.edgeEls = {};
      this.drawGroups();
      this.drawNodes();
    }

    drawGroups() {
      for (const g of GROUPS) {
        el('rect', { class: 'group-box', x: g.x, y: g.y, width: g.w, height: g.h, rx: 12 }, this.edgeLayer);
        const t = el('text', { class: 'group-label', x: g.x + 10, y: g.y - 3 }, this.edgeLayer);
        t.textContent = g.label;
      }
    }

    drawNodes() {
      for (const id in NODES) {
        const n = NODES[id];
        const g = el('g', { class: 'node', 'data-id': id, transform: `translate(${n.x - NODE_W / 2},${n.y - NODE_H / 2})` }, this.nodeLayer);
        el('rect', { width: NODE_W, height: NODE_H, rx: 9, fill: `var(--n-${n.cls})`, stroke: 'var(--border)' }, g);
        const t1 = el('text', { x: 10, y: 19 }, g);
        t1.textContent = n.label;
        const t2 = el('text', { x: 10, y: 34, class: 'small' }, g);
        t2.textContent = n.sub;
        g.addEventListener('click', () => this.onNodeClick && this.onNodeClick(id));
        this.nodeEls[id] = g;
      }
    }

    edgePath(a, b) {
      const A = NODES[a];
      const B = NODES[b];
      const dx = B.x - A.x;
      if (Math.abs(dx) < 40) {
        // cùng cột: vòng ra bên phải
        const x1 = A.x + NODE_W / 2;
        const x2 = B.x + NODE_W / 2;
        const bulge = 30 + Math.min(60, Math.abs(B.y - A.y) / 6);
        return `M${x1},${A.y} C${x1 + bulge},${A.y} ${x2 + bulge},${B.y} ${x2},${B.y}`;
      }
      const dir = Math.sign(dx);
      const x1 = A.x + dir * NODE_W / 2;
      const x2 = B.x - dir * NODE_W / 2;
      const c = Math.max(40, Math.abs(x2 - x1) / 2);
      return `M${x1},${A.y} C${x1 + dir * c},${A.y} ${x2 - dir * c},${B.y} ${x2},${B.y}`;
    }

    key(a, b) { return a < b ? `${a}|${b}` : `${b}|${a}`; }

    // Vẽ các cạnh có trong kịch bản và làm mờ các node không liên quan.
    setSteps(steps) {
      this.edgeLayer.querySelectorAll('.edge').forEach((e) => e.remove());
      this.packetLayer.innerHTML = '';
      this.edgeEls = {};
      const used = new Set();
      for (const s of steps) {
        used.add(s.f); used.add(s.t);
        if (s.f === s.t) continue;
        const k = this.key(s.f, s.t);
        if (this.edgeEls[k]) continue;
        const [a, b] = k.split('|');
        const path = el('path', { class: 'edge', d: this.edgePath(a, b), stroke: kindColor(s.kind) }, this.edgeLayer);
        this.edgeEls[k] = { path, a, b };
      }
      for (const id in this.nodeEls) {
        this.nodeEls[id].classList.toggle('dim', steps.length > 0 && !used.has(id));
        this.nodeEls[id].classList.remove('active');
      }
      this.used = used;
      this.fit();
    }

    showAllEdges(pairs) {
      this.setSteps(pairs.map(([f, t, kind]) => ({ f, t, kind })));
      for (const k in this.edgeEls) this.edgeEls[k].path.classList.add('done');
    }

    highlight(steps, idx) {
      for (const k in this.edgeEls) this.edgeEls[k].path.classList.remove('active', 'done', 'waiting');
      for (const id in this.nodeEls) this.nodeEls[id].classList.remove('active');
      for (let i = 0; i <= idx && i < steps.length; i++) {
        const s = steps[i];
        if (s.f === s.t) continue;
        const e = this.edgeEls[this.key(s.f, s.t)];
        if (e) e.path.classList.add('done');
      }
      const cur = steps[idx];
      if (!cur) return;
      // các lời gọi đồng bộ còn đang chờ trả lời: nét đứt chạy
      for (const fr of cur.stack || []) {
        const w = this.edgeEls[this.key(fr.caller, fr.callee)];
        if (w) w.path.classList.add('waiting');
      }
      this.nodeEls[cur.f] && this.nodeEls[cur.f].classList.add('active');
      this.nodeEls[cur.t] && this.nodeEls[cur.t].classList.add('active');
      const e = this.edgeEls[this.key(cur.f, cur.t)];
      if (e) {
        e.path.classList.add('active');
        e.path.setAttribute('stroke', kindColor(cur.kind));
      }
    }

    animatePacket(step, duration) {
      return new Promise((resolve) => {
        if (step.f === step.t) {
          const g = this.nodeEls[step.f];
          const n = NODES[step.f];
          const ring = el('circle', { cx: n.x, cy: n.y, r: 20, fill: 'none', stroke: kindColor(step.kind), 'stroke-width': 3 }, this.packetLayer);
          const start = performance.now();
          const tick = (now) => {
            const p = Math.min(1, (now - start) / duration);
            ring.setAttribute('r', 20 + p * 60);
            ring.setAttribute('opacity', 1 - p);
            if (p < 1) requestAnimationFrame(tick); else { ring.remove(); resolve(); }
          };
          if (g) requestAnimationFrame(tick); else resolve();
          return;
        }
        const e = this.edgeEls[this.key(step.f, step.t)];
        if (!e) return resolve();
        const reverse = e.a !== step.f;
        const len = e.path.getTotalLength();
        const dot = el('circle', { r: 7, class: 'packet', fill: kindColor(step.kind), style: `color:${kindColor(step.kind)}` }, this.packetLayer);
        const start = performance.now();
        const tick = (now) => {
          const p = Math.min(1, (now - start) / duration);
          const eased = p < 0.5 ? 2 * p * p : 1 - Math.pow(-2 * p + 2, 2) / 2;
          const pt = e.path.getPointAtLength(len * (reverse ? 1 - eased : eased));
          dot.setAttribute('cx', pt.x);
          dot.setAttribute('cy', pt.y);
          if (p < 1) requestAnimationFrame(tick); else { dot.remove(); resolve(); }
        };
        requestAnimationFrame(tick);
      });
    }

    select(id) {
      for (const k in this.nodeEls) this.nodeEls[k].classList.toggle('selected', k === id);
    }

    // Co giãn viewBox theo nội dung thật (node + mọi cạnh cong) để không phần nào bị cắt.
    fit() {
      let box;
      try { box = this.svg.getBBox(); } catch (e) { return; }
      if (!box || !box.width) return; // đang bị ẩn — sẽ gọi lại khi hiện
      // Trong một tình huống: phóng to vào các bộ phận có tham gia (và các đường nối giữa chúng)
      // để chữ trên sơ đồ to, dễ đọc hơn.
      if (this.used && this.used.size && this.used.size < Object.keys(NODES).length) {
        let x1 = Infinity; let y1 = Infinity; let x2 = -Infinity; let y2 = -Infinity;
        for (const id of this.used) {
          const n = NODES[id];
          x1 = Math.min(x1, n.x - NODE_W / 2); x2 = Math.max(x2, n.x + NODE_W / 2);
          y1 = Math.min(y1, n.y - NODE_H / 2); y2 = Math.max(y2, n.y + NODE_H / 2);
        }
        for (const k in this.edgeEls) {
          const b = this.edgeEls[k].path.getBBox();
          x1 = Math.min(x1, b.x); x2 = Math.max(x2, b.x + b.width);
          y1 = Math.min(y1, b.y); y2 = Math.max(y2, b.y + b.height);
        }
        const P = 22;
        this.svg.setAttribute('viewBox', `${x1 - P} ${y1 - P - 12} ${x2 - x1 + 2 * P} ${y2 - y1 + 2 * P + 12}`);
        return;
      }
      const PAD = 14; // chừa chỗ cho viền active/selected và chấm sáng
      const x = Math.min(0, box.x - PAD);
      const y = Math.min(0, box.y - PAD);
      const w = Math.max(1140, box.x + box.width + PAD) - x;
      const h = Math.max(560, box.y + box.height + PAD) - y;
      this.svg.setAttribute('viewBox', `${x} ${y} ${w} ${h}`);
    }
  }

  // ---------------------------------------------------------------- Sequence diagram
  class SeqDiagram {
    constructor(container, { onStepClick } = {}) {
      this.container = container;
      this.onStepClick = onStepClick;
      this.msgEls = [];
    }

    render(steps, laneOrder) {
      const lanes = [];
      const add = (id) => { if (!lanes.includes(id)) lanes.push(id); };
      (laneOrder || []).forEach((id) => steps.some((s) => s.f === id || s.t === id) && add(id));
      steps.forEach((s) => { add(s.f); add(s.t); });

      const GAP = 140;
      const LEFT = 70;
      const TOP = 54;
      const ROW = 36;
      const width = LEFT + (lanes.length - 1) * GAP + 230;
      const height = TOP + steps.length * ROW + 24;
      const svg = el('svg', { class: 'seq', width, height, viewBox: `0 0 ${width} ${height}` });
      const defs = el('defs', {}, svg);
      for (const kind of Object.keys(KIND_LABEL)) {
        const m = el('marker', { id: `arr-${kind}`, viewBox: '0 0 10 10', refX: 9, refY: 5, markerWidth: 7, markerHeight: 7, orient: 'auto-start-reverse' }, defs);
        el('path', { d: 'M0,0 L10,5 L0,10 z', fill: kindColor(kind) }, m);
      }
      const xOf = (id) => LEFT + lanes.indexOf(id) * GAP;

      lanes.forEach((id) => {
        const x = xOf(id);
        el('line', { class: 'lifeline', x1: x, y1: 40, x2: x, y2: height - 6 }, svg);
        const g = el('g', { class: 'lane-head' }, svg);
        el('rect', { x: x - 64, y: 6, width: 128, height: 30, rx: 7, fill: `var(--n-${NODES[id].cls})`, stroke: 'var(--border)' }, g);
        const t = el('text', { x, y: 25, 'text-anchor': 'middle' }, g);
        t.textContent = NODES[id].label;
      });

      // thanh kích hoạt: bên được gọi đang bận xử lý, bên gọi đang chờ
      const actLayer = el('g', { class: 'acts' }, svg);
      steps.forEach((s, i) => {
        if (s.mode !== 'sync' || s.retAt == null) return;
        const y1 = TOP + i * ROW + 18;
        const y2 = TOP + s.retAt * ROW + 18;
        el('rect', { class: 'act', x: xOf(s.t) - 5, y: y1, width: 10, height: Math.max(6, y2 - y1), rx: 2, 'data-i': i }, actLayer);
      });

      this.msgEls = steps.map((s, i) => {
        const y = TOP + i * ROW + 18;
        const g = el('g', { class: `msg k-${s.kind}`, 'data-i': i }, svg);
        el('rect', { class: 'row-hl', x: 0, y: y - 22, width, height: ROW - 2, rx: 4 }, g);
        const color = kindColor(s.kind);
        const x1 = xOf(s.f);
        const x2 = xOf(s.t);
        const label = s.label.length > 46 ? s.label.slice(0, 44) + '…' : s.label;
        if (s.f === s.t) {
          el('path', { d: `M${x1},${y - 8} h26 v12 h-24`, fill: 'none', stroke: color, 'stroke-width': 1.6, 'marker-end': `url(#arr-${s.kind})` }, g);
          const t = el('text', { x: x1 + 32, y: y + 2 }, g);
          t.textContent = (s.kind === 'decision' ? '◆ ' : '') + label;
        } else {
          const dashed = s.mode === 'async' || s.mode === 'return';
          el('line', { x1, y1: y, x2: x2 + (x2 > x1 ? -2 : 2), y2: y, stroke: color, 'stroke-width': 1.6, 'stroke-dasharray': dashed ? '5 4' : '', 'marker-end': `url(#arr-${s.kind})` }, g);
          if (s.mode === 'db') {
            // đi và về trong cùng một bước
            el('line', { x1: x2 + (x2 > x1 ? -4 : 4), y1: y + 6, x2: x1 + (x2 > x1 ? 2 : -2), y2: y + 6, stroke: color, 'stroke-width': 1.2, 'stroke-dasharray': '3 3', opacity: 0.8, 'marker-end': `url(#arr-${s.kind})` }, g);
          }
          const mid = (x1 + x2) / 2;
          const span = Math.abs(x2 - x1);
          const t = el('text', { x: span > 200 ? mid : Math.min(x1, x2) + 6, y: y - 6, 'text-anchor': span > 200 ? 'middle' : 'start' }, g);
          t.textContent = (s.mode === 'return' ? '↩ ' : s.mode === 'async' ? '🚀 ' : '') + label;
        }
        // số thứ tự luôn nằm bên trái mũi tên để không đè lên nhãn
        const nx = Math.min(x1, x2) - 14;
        el('circle', { cx: nx, cy: y, r: 9, fill: color }, g);
        const n = el('text', { class: 'num', x: nx, y: y + 3.5, 'text-anchor': 'middle' }, g);
        n.textContent = i + 1;
        const title = el('title', {}, g);
        title.textContent = `${s.label}${s.detail ? '\n' + s.detail.replace(/<[^>]+>/g, '') : ''}`;
        g.addEventListener('click', () => this.onStepClick && this.onStepClick(i));
        return g;
      });

      this.container.innerHTML = '';
      this.container.appendChild(svg);
      this.svg = svg;
      this.fit();
    }

    // Mở rộng khung theo nhãn dài nhất để chữ ở lane cuối không bị cắt.
    fit() {
      const svg = this.svg;
      if (!svg) return;
      let box;
      try { box = svg.getBBox(); } catch (e) { return; }
      if (!box || !box.width) return;
      const PAD = 12;
      const x = Math.min(0, box.x - PAD);
      const y = Math.min(0, box.y - PAD);
      const w = box.x + box.width + PAD - x;
      const h = box.y + box.height + PAD - y;
      svg.setAttribute('viewBox', `${x} ${y} ${w} ${h}`);
      svg.setAttribute('width', w);
      svg.setAttribute('height', h);
      svg.querySelectorAll('.row-hl').forEach((r) => { r.setAttribute('x', x); r.setAttribute('width', w); });
    }

    highlight(idx) {
      this.msgEls.forEach((g, i) => {
        g.classList.toggle('future', i > idx);
        g.classList.toggle('active', i === idx);
      });
      const cur = this.msgEls[idx];
      if (cur) scrollWithin(this.container, cur.getBBox().y - 60);
    }
  }

  function scrollWithin(container, top) {
    const max = container.scrollHeight - container.clientHeight;
    container.scrollTo({ top: Math.max(0, Math.min(max, top)), behavior: 'smooth' });
  }

  // ---------------------------------------------------------------- Log
  class StepLog {
    constructor(container, { onStepClick } = {}) {
      this.container = container;
      this.onStepClick = onStepClick;
      this.items = [];
    }

    render(steps) {
      this.container.innerHTML = '';
      this.items = steps.map((s, i) => {
        const div = document.createElement('div');
        div.className = `log-item k-${s.kind}`;
        const from = NODES[s.f].label;
        const to = NODES[s.t].label;
        const tech = [].concat(s.tech || []);
        div.innerHTML = `
          <div class="top"><span class="n">${i + 1}</span>
            <div><div class="route">${esc(from)}${s.f === s.t ? '' : ' → ' + esc(to)}</div>
            <div class="label">${esc(s.label)}</div></div></div>
          <div class="mode mode-${s.mode}">${MODE_BADGE[s.mode](s)}</div>
          ${s.detail ? `<div class="detail">${s.detail}</div>` : ''}
          ${tech.length ? `<details class="tech"><summary>Chi tiết kỹ thuật</summary><div class="store">${tech.map((x) => `<code>${esc(x)}</code>`).join(' ')}</div></details>` : ''}`;
        div.querySelector('details') && div.querySelector('details').addEventListener('click', (e) => e.stopPropagation());
        div.addEventListener('click', () => this.onStepClick && this.onStepClick(i));
        this.container.appendChild(div);
        return div;
      });
    }

    highlight(idx) {
      this.items.forEach((d, i) => {
        d.classList.toggle('future', i > idx);
        d.classList.toggle('active', i === idx);
      });
      const cur = this.items[idx];
      if (cur) scrollWithin(this.container, cur.offsetTop - this.container.offsetTop - 80);
    }
  }

  // ---------------------------------------------------------------- Player
  class Player {
    constructor({ map, seq, log, onChange }) {
      this.map = map; this.seq = seq; this.log = log; this.onChange = onChange;
      this.steps = []; this.idx = -1; this.playing = false; this.speed = 1; this.token = 0;
    }

    load(rawSteps, laneOrder) {
      this.stop();
      const steps = expandCalls(rawSteps);
      this.steps = steps;
      this.idx = -1;
      this.map.setSteps(steps);
      this.seq.render(steps, laneOrder);
      this.log.render(steps);
      this.apply();
    }

    apply() {
      this.map.highlight(this.steps, this.idx);
      this.seq.highlight(this.idx);
      this.log.highlight(this.idx);
      this.onChange && this.onChange(this);
    }

    async goTo(i, animate = true) {
      if (i < -1 || i >= this.steps.length) return;
      this.idx = i;
      this.apply();
      if (animate && i >= 0) await this.map.animatePacket(this.steps[i], 700 / this.speed);
    }

    async next() { if (this.idx < this.steps.length - 1) await this.goTo(this.idx + 1); }
    prev() { if (this.idx >= 0) this.goTo(this.idx - 1, false); }
    reset() { this.stop(); this.goTo(-1, false); }

    async play() {
      if (this.playing) return;
      if (this.idx >= this.steps.length - 1) this.idx = -1;
      this.playing = true;
      const token = ++this.token;
      this.onChange && this.onChange(this);
      while (this.playing && token === this.token && this.idx < this.steps.length - 1) {
        await this.next();
        await wait(450 / this.speed);
      }
      if (token === this.token) this.playing = false;
      this.onChange && this.onChange(this);
    }

    stop() { this.playing = false; this.token++; this.onChange && this.onChange(this); }
  }

  function wait(ms) { return new Promise((r) => setTimeout(r, ms)); }
  function esc(s) {
    return String(s).replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));
  }

  window.Sim = { NODES, KIND_LABEL, ArchMap, SeqDiagram, StepLog, Player, esc, expandCalls, describe };
})();

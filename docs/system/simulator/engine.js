/* Engine: danh mục thành phần, vẽ sơ đồ kiến trúc, sơ đồ tuần tự và trình phát từng bước. */
(function () {
  'use strict';

  const SVG_NS = 'http://www.w3.org/2000/svg';
  const NODE_W = 140;
  const NODE_H = 44;

  // Mỗi thành phần: vị trí trên sơ đồ kiến trúc + thông tin hiển thị khi bấm vào.
  const NODES = {
    browser: { label: 'Browser (SPA)', sub: 'React + Mantine', x: 85, y: 250, cls: 'client',
      info: { 'Vai trò': 'Người dùng thao tác trên giao diện React SPA', 'Giao tiếp': 'HTTP /api/* qua nginx; WebSocket tới Soketi' } },
    peer: { label: 'Browser B', sub: 'người nhận', x: 85, y: 400, cls: 'client',
      info: { 'Vai trò': 'Người dùng thứ hai (bạn bè / người nhận tin nhắn)' } },
    nginx: { label: 'nginx', sub: 'frontend :3000', x: 250, y: 325, cls: 'edge',
      info: { 'Vai trò': 'Phục vụ SPA tĩnh + reverse proxy /api/<service>/ tới gateway HTTP', 'Cổng': 'host 127.0.0.1:3000 (duy nhất publish ra ngoài)' } },

    identity: { label: 'identity', sub: ':8081 / :9081', x: 440, y: 50, cls: 'svc',
      info: { 'Chức năng': 'Hồ sơ, OAuth Google, phát session JWT HS256, ký kênh Soketi, seed dữ liệu cho profile mới', 'Lưu trữ': 'identity.profiles, identity.auth_providers', 'Gọi tới': 'task (seed status), project (seed project), mail-provider (Subscribe), Google OAuth' } },
    project: { label: 'project', sub: ':8090 / :9090', x: 440, y: 115, cls: 'svc',
      info: { 'Chức năng': 'Project theo profile; project mặc định "Chung" không archive được', 'Lưu trữ': 'project.projects', 'Gọi tới': '— (service lá)' } },
    task: { label: 'task', sub: ':8082 / :9082', x: 440, y: 180, cls: 'svc',
      info: { 'Chức năng': 'Task cha/con, status & lifecycle, audit log, tìm kiếm, nhắc việc (ReminderWorker), task lặp lại (RRULE)', 'Lưu trữ': 'task.tasks, task_statuses, status_transitions, task_status_logs', 'Gọi tới': 'project (GetProject), notification (CreateNotice nhắc việc)' } },
    calendar: { label: 'calendar', sub: ':8083 / :9083', x: 440, y: 245, cls: 'svc',
      info: { 'Chức năng': 'Lịch cá nhân (khối thời gian, all-day, màu)', 'Lưu trữ': 'calendar.schedules', 'Gọi tới': 'project (GetProject)' } },
    event: { label: 'event', sub: ':8085 / :9085', x: 440, y: 310, cls: 'svc',
      info: { 'Chức năng': 'Sự kiện PLANNED → CONFIRMED → CANCELLED, truy vết email nguồn', 'Lưu trữ': 'event.events', 'Gọi tới': 'project (GetProject)' } },
    backlog: { label: 'backlog', sub: ':8086 / :9086', x: 440, y: 375, cls: 'svc',
      info: { 'Chức năng': 'Email không phân loại được (NEW / TRIAGED / ARCHIVED) kèm reason', 'Lưu trữ': 'backlog.backlogs', 'Gọi tới': '— (service lá)' } },
    attachment: { label: 'attachment', sub: ':8089 / :9089', x: 440, y: 440, cls: 'svc',
      info: { 'Chức năng': 'Tệp đính kèm đa hình (owner_type, owner_id), upload / download', 'Lưu trữ': 'attachment.attachments + RustFS', 'Gọi tới': 'RustFS (S3)' } },
    report: { label: 'report', sub: ':8087 / :9087', x: 440, y: 505, cls: 'svc',
      info: { 'Chức năng': 'Báo cáo tổng quan, status breakdown, time series, activity', 'Lưu trữ': 'Không có DB — tổng hợp qua gRPC', 'Gọi tới': 'task, calendar, event, backlog' } },

    notification: { label: 'notification', sub: ':8088 / :9088', x: 640, y: 70, cls: 'svc',
      info: { 'Chức năng': 'Lưu notice, đếm chưa đọc, đánh dấu đã đọc, push realtime', 'Lưu trữ': 'notification.notices', 'Gọi tới': 'Soketi (notice.created)' } },
    social: { label: 'social', sub: ':8091 / :9091', x: 640, y: 150, cls: 'svc',
      info: { 'Chức năng': 'Kết bạn: tìm theo email, gửi / chấp nhận / từ chối / huỷ lời mời', 'Lưu trữ': 'social.friendships', 'Gọi tới': 'identity, notification' } },
    message: { label: 'message', sub: ':8092 / :9092', x: 640, y: 230, cls: 'svc',
      info: { 'Chức năng': 'Chat 1-1: broadcast realtime + đẩy Kafka; đọc lịch sử', 'Lưu trữ': 'message.* (chỉ đọc)', 'Gọi tới': 'social (AreFriends), Soketi, Kafka' } },
    worker: { label: 'worker ×N', sub: 'health :8094', x: 640, y: 310, cls: 'svc',
      info: { 'Chức năng': 'Consume chat.messages, ghi tin nhắn idempotent', 'Lưu trữ': 'message.messages, message.conversations', 'Gọi tới': 'Kafka, Postgres' } },
    mail: { label: 'mail-provider', sub: ':8084 / :9084', x: 640, y: 460, cls: 'svc',
      info: { 'Chức năng': 'Watch Gmail, nhận Pub/Sub, lưu email gốc lên S3, phân loại (DeepSeek / từ khoá), dispatch', 'Lưu trữ': 'mail_provider.sessions, mail_provider.noti_indexes + RustFS', 'Gọi tới': 'project, task, calendar, event, backlog, notification, Gmail, DeepSeek, Pub/Sub' } },

    pg: { label: 'PostgreSQL', sub: 'schema / service', x: 840, y: 90, cls: 'infra',
      info: { 'Vai trò': 'DB task_manager, mỗi service sở hữu 1 schema riêng', 'Cổng': 'postgres:5432' } },
    s3: { label: 'RustFS (S3)', sub: 'bucket task-manager', x: 840, y: 210, cls: 'infra',
      info: { 'Vai trò': 'Object storage: email gốc + tệp đính kèm', 'Cổng': 'rustfs:9000' } },
    kafka: { label: 'Kafka', sub: 'chat.messages ×6', x: 840, y: 330, cls: 'infra',
      info: { 'Vai trò': 'Hàng đợi tin nhắn chat, key = conversation_id', 'Cổng': 'kafka:9092' } },
    soketi: { label: 'Soketi', sub: 'WebSocket :6001', x: 840, y: 450, cls: 'infra',
      info: { 'Vai trò': 'Realtime theo giao thức Pusher; kênh private ký bởi identity', 'Cổng': 'soketi:6001' } },

    google: { label: 'Google OAuth', sub: 'OIDC / token', x: 1050, y: 90, cls: 'ext',
      info: { 'Vai trò': 'Đăng nhập Google, đổi code lấy token, refresh token' } },
    gmail: { label: 'Gmail API', sub: 'watch / history', x: 1050, y: 210, cls: 'ext',
      info: { 'Vai trò': 'users.watch, users.history.list, users.messages.get' } },
    pubsub: { label: 'Google Pub/Sub', sub: 'push / pull', x: 1050, y: 330, cls: 'ext',
      info: { 'Vai trò': 'Báo cho mail-provider khi hộp thư thay đổi' } },
    deepseek: { label: 'DeepSeek LLM', sub: '/chat/completions', x: 1050, y: 450, cls: 'ext',
      info: { 'Vai trò': 'Phân loại email thành task / schedule / event / other' } },
  };

  const GROUPS = [
    { label: 'Client', x: 10, y: 205, w: 150, h: 250 },
    { label: 'Nghiệp vụ lõi', x: 362, y: 12, w: 156, h: 525 },
    { label: 'Giao tiếp & mail', x: 562, y: 32, w: 156, h: 460 },
    { label: 'Hạ tầng', x: 762, y: 52, w: 156, h: 440 },
    { label: 'Bên ngoài', x: 972, y: 52, w: 156, h: 440 },
  ];

  const KIND_LABEL = {
    http: 'HTTP / REST', grpc: 'gRPC nội bộ', db: 'Đọc / ghi dữ liệu', async: 'Bất đồng bộ (Kafka / queue)',
    ext: 'Gọi dịch vụ ngoài', ws: 'Realtime WebSocket', self: 'Xử lý nội bộ', decision: 'Rẽ nhánh / quyết định', error: 'Lỗi / từ chối',
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
      this.fit();
    }

    showAllEdges(pairs) {
      this.setSteps(pairs.map(([f, t, kind]) => ({ f, t, kind })));
      for (const k in this.edgeEls) this.edgeEls[k].path.classList.add('done');
    }

    highlight(steps, idx) {
      for (const k in this.edgeEls) this.edgeEls[k].path.classList.remove('active', 'done');
      for (const id in this.nodeEls) this.nodeEls[id].classList.remove('active');
      for (let i = 0; i <= idx && i < steps.length; i++) {
        const s = steps[i];
        if (s.f === s.t) continue;
        const e = this.edgeEls[this.key(s.f, s.t)];
        if (e) e.path.classList.add('done');
      }
      const cur = steps[idx];
      if (!cur) return;
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

      const GAP = 128;
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
        el('rect', { x: x - 58, y: 6, width: 116, height: 30, rx: 7, fill: `var(--n-${NODES[id].cls})`, stroke: 'var(--border)' }, g);
        const t = el('text', { x, y: 25, 'text-anchor': 'middle' }, g);
        t.textContent = NODES[id].label;
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
          const dashed = ['async', 'ws'].includes(s.kind) || s.reply;
          el('line', { x1, y1: y, x2: x2 + (x2 > x1 ? -2 : 2), y2: y, stroke: color, 'stroke-width': 1.6, 'stroke-dasharray': dashed ? '5 4' : '', 'marker-end': `url(#arr-${s.kind})` }, g);
          const mid = (x1 + x2) / 2;
          const span = Math.abs(x2 - x1);
          const t = el('text', { x: span > 200 ? mid : Math.min(x1, x2) + 6, y: y - 6, 'text-anchor': span > 200 ? 'middle' : 'start' }, g);
          t.textContent = label;
        }
        el('circle', { cx: x1 - 14 * (s.f === s.t || x2 > x1 ? 1 : -1), cy: y, r: 9, fill: color }, g);
        const n = el('text', { class: 'num', x: x1 - 14 * (s.f === s.t || x2 > x1 ? 1 : -1), y: y + 3.5, 'text-anchor': 'middle' }, g);
        n.textContent = i + 1;
        const title = el('title', {}, g);
        title.textContent = `${s.label}${s.detail ? '\n' + s.detail : ''}`;
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
        div.innerHTML = `
          <div class="top"><span class="n">${i + 1}</span>
            <div><div class="route">${esc(from)}${s.f === s.t ? '' : ' → ' + esc(to)} · ${KIND_LABEL[s.kind]}</div>
            <div class="label">${esc(s.label)}</div></div></div>
          ${s.detail ? `<div class="detail">${s.detail}</div>` : ''}
          ${s.store ? `<div class="store">${[].concat(s.store).map((x) => `<code>${esc(x)}</code>`).join(' ')}</div>` : ''}`;
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

    load(steps, laneOrder) {
      this.stop();
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

  window.Sim = { NODES, KIND_LABEL, ArchMap, SeqDiagram, StepLog, Player, esc };
})();

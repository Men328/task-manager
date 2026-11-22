/* Điều phối trang: điều hướng, form tuỳ chọn, gắn trình phát. */
(function () {
  'use strict';
  const { NODES, KIND_LABEL, ArchMap, SeqDiagram, StepLog, Player, esc } = window.Sim;
  const scenarios = window.Scenarios.list;
  const $ = (id) => document.getElementById(id);

  // Tất cả quan hệ giữa các thành phần (cho trang tổng quan).
  const ALL_EDGES = [
    ['browser', 'nginx', 'http'], ['peer', 'nginx', 'http'], ['browser', 'soketi', 'ws'], ['peer', 'soketi', 'ws'],
    ['nginx', 'identity', 'http'], ['nginx', 'task', 'http'], ['nginx', 'project', 'http'], ['nginx', 'calendar', 'http'],
    ['nginx', 'event', 'http'], ['nginx', 'backlog', 'http'], ['nginx', 'attachment', 'http'], ['nginx', 'report', 'http'],
    ['nginx', 'notification', 'http'], ['nginx', 'social', 'http'], ['nginx', 'message', 'http'], ['nginx', 'mail', 'http'],
    ['identity', 'task', 'grpc'], ['identity', 'project', 'grpc'], ['identity', 'mail', 'grpc'], ['identity', 'google', 'ext'],
    ['task', 'project', 'grpc'], ['calendar', 'project', 'grpc'], ['event', 'project', 'grpc'], ['task', 'notification', 'grpc'],
    ['report', 'task', 'grpc'], ['report', 'calendar', 'grpc'], ['report', 'event', 'grpc'], ['report', 'backlog', 'grpc'],
    ['mail', 'project', 'grpc'], ['mail', 'task', 'grpc'], ['mail', 'calendar', 'grpc'], ['mail', 'event', 'grpc'],
    ['mail', 'backlog', 'grpc'], ['mail', 'notification', 'grpc'], ['mail', 'gmail', 'ext'], ['mail', 'deepseek', 'ext'],
    ['mail', 's3', 'db'], ['mail', 'pg', 'db'], ['pubsub', 'mail', 'ext'], ['gmail', 'pubsub', 'ext'], ['mail', 'google', 'ext'],
    ['social', 'identity', 'grpc'], ['social', 'notification', 'grpc'], ['message', 'social', 'grpc'],
    ['message', 'kafka', 'async'], ['message', 'soketi', 'http'], ['kafka', 'worker', 'async'], ['notification', 'soketi', 'http'],
    ['attachment', 's3', 'db'], ['worker', 'pg', 'db'], ['identity', 'pg', 'db'], ['task', 'pg', 'db'], ['notification', 'pg', 'db'],
    ['social', 'pg', 'db'], ['message', 'pg', 'db'],
  ];

  const legendHtml = () => ['http', 'grpc', 'db', 'reply', 'async', 'ext', 'ws', 'error']
    .map((k) => `<span><i style="background:var(--k-${k})"></i>${KIND_LABEL[k]}</span>`).join('');

  // ---------------------------------------------------------------- điều hướng
  const nav = $('nav');
  const GROUP_ORDER = ['Bắt đầu', 'Email', 'Tài khoản', 'Công việc', 'Liên lạc', 'Khác'];
  scenarios.sort((a, b) => GROUP_ORDER.indexOf(a.group) - GROUP_ORDER.indexOf(b.group));
  const items = [{ id: 'overview', icon: '🗺️', title: 'Toàn cảnh hệ thống', group: 'Bắt đầu' }].concat(scenarios);
  let lastGroup = '';
  for (const it of items) {
    if (it.group !== lastGroup) {
      const g = document.createElement('div');
      g.className = 'nav-group';
      g.textContent = it.group;
      nav.appendChild(g);
      lastGroup = it.group;
    }
    const b = document.createElement('button');
    b.dataset.id = it.id;
    b.innerHTML = `<span class="ico">${it.icon}</span>${esc(it.title)}`;
    b.addEventListener('click', () => { location.hash = it.id; });
    nav.appendChild(b);
  }

  // ---------------------------------------------------------------- tổng quan
  const overviewMap = new ArchMap($('overview-map'), { onNodeClick: showNode });
  overviewMap.showAllEdges(ALL_EDGES);
  $('overview-legend').innerHTML = legendHtml();

  function showNode(id) {
    overviewMap.select(id);
    const n = NODES[id];
    const related = scenarios.filter((s) => s.build(defaults(s)).some((st) => st.f === id || st.t === id));
    $('node-info').innerHTML = `<h3>${esc(n.label)} <span class="sub">${esc(n.sub)}</span></h3>
      <dl>${Object.entries(n.info).map(([k, v]) => `<dt>${esc(k)}</dt><dd>${esc(v)}</dd>`).join('')}</dl>
      ${related.length ? `<p style="margin:10px 0 4px;color:var(--muted);font-size:12px">Có mặt trong các tình huống:</p>${related.map((s) => `<a href="#${s.id}" class="chip hit" style="text-decoration:none">${s.icon} ${esc(s.title)}</a>`).join('')}` : ''}`;
  }
  $('node-info').innerHTML = '<h3>ℹ️ Bộ phận này làm gì?</h3><p class="note">Bấm vào một ô trên sơ đồ ở trên để xem bộ phận đó là gì, làm việc gì, và có mặt trong những tình huống nào.</p>';
  $('scenario-cards').innerHTML = scenarios.map((s) => `<a href="#${s.id}" class="log-item" style="text-decoration:none;color:inherit;display:block">
      <div class="label">${s.icon} ${esc(s.title)}</div><div class="detail">${esc(s.desc)}</div></a>`).join('');

  // ---------------------------------------------------------------- mô phỏng
  let current = null;
  const opts = {};
  const player = new Player({
    map: new ArchMap($('map')),
    seq: new SeqDiagram($('seq'), { onStepClick: (i) => { player.stop(); player.goTo(i); } }),
    log: new StepLog($('log'), { onStepClick: (i) => { player.stop(); player.goTo(i); } }),
    onChange: updateToolbar,
  });
  $('legend').innerHTML = legendHtml();

  function defaults(s) {
    const o = {};
    for (const op of s.options || []) o[op.id] = op.value;
    return o;
  }

  function renderOptions(s) {
    const panel = $('panel');
    if (s.mountPanel) { s.mountPanel(panel, rebuild); return; }
    if (!s.options.length) { panel.innerHTML = ''; return; }
    panel.innerHTML = `<div class="card"><h3>⚙️ Thử các tình huống <span class="sub">đổi lựa chọn để xem các bước thay đổi theo</span></h3><div class="options">${
      s.options.map((op) => op.type === 'checkbox'
        ? `<label class="opt"><input type="checkbox" data-o="${op.id}" ${opts[s.id][op.id] ? 'checked' : ''}> ${esc(op.label)}</label>`
        : `<label class="opt">${esc(op.label)} <select data-o="${op.id}">${op.options.map(([v, l]) => `<option value="${v}" ${opts[s.id][op.id] === v ? 'selected' : ''}>${esc(l)}</option>`).join('')}</select></label>`).join('')
    }</div></div>`;
    panel.querySelectorAll('[data-o]').forEach((inp) => inp.addEventListener('change', () => {
      opts[s.id][inp.dataset.o] = inp.type === 'checkbox' ? inp.checked : inp.value;
      rebuild();
    }));
  }

  function rebuild() {
    const s = current;
    const o = opts[s.id];
    if (s.extra) {
      $('extra-card').classList.remove('hidden');
      $('extra-title').textContent = s.extraTitle || 'Minh hoạ';
      $('extra').innerHTML = s.extra(o);
    } else {
      $('extra-card').classList.add('hidden');
    }
    player.load(s.build(o), s.lanes);
  }

  function renderNow(p) {
    const box = $('now');
    const s = p.steps[p.idx];
    if (!s) {
      box.className = 'card now';
      box.innerHTML = `<div class="now-head"><span class="now-step">Sẵn sàng</span><span class="now-title">Bấm ▶ Chạy để xem tự động, hoặc “Bước ▶” để đi từng bước</span></div>
        <p class="now-detail">Mỗi bước sẽ được kể lại ở đây bằng lời: bộ phận nào đang làm gì, có phải đợi ai không. Khi một bộ phận nhờ bộ phận khác và phải đợi, sẽ luôn có một bước “↩ gửi kết quả” tương ứng.</p>`;
      return;
    }
    box.className = `card now k-${s.kind}`;
    const waits = (s.stack || []).map((fr) => `<span class="frame">${esc(NODES[fr.caller].label)} ⏳ đợi ${esc(NODES[fr.callee].label)}</span>`).join('');
    box.innerHTML = `<div class="now-head"><span class="now-step">Bước ${p.idx + 1}/${p.steps.length}</span>
        <span class="now-title">${esc(s.label)}</span></div>
      <p class="now-what">${window.Sim.describe(p.steps, p.idx)}</p>
      ${s.detail ? `<p class="now-detail">${s.detail}</p>` : ''}
      <div class="now-wait"><span class="lbl">Ai đang đợi ai:</span>${waits || '<span class="frame">không ai phải đợi</span>'}</div>`;
    box.querySelectorAll('[data-goto]').forEach((a) => a.addEventListener('click', (e) => {
      e.preventDefault(); player.stop(); player.goTo(+a.dataset.goto);
    }));
  }

  function updateToolbar(p) {
    renderNow(p);
    $('btn-play').textContent = p.playing ? '⏸ Dừng' : (p.idx >= p.steps.length - 1 && p.idx >= 0 ? '↺ Chạy lại' : '▶ Chạy');
    $('btn-prev').disabled = p.idx < 0;
    $('btn-next').disabled = p.idx >= p.steps.length - 1;
    $('progress').textContent = p.idx < 0 ? `${p.steps.length} bước` : `Bước ${p.idx + 1}/${p.steps.length}`;
  }

  $('btn-play').addEventListener('click', () => (player.playing ? player.stop() : player.play()));
  $('btn-next').addEventListener('click', () => { player.stop(); player.next(); });
  $('btn-prev').addEventListener('click', () => { player.stop(); player.prev(); });
  $('btn-reset').addEventListener('click', () => player.reset());
  $('speed').addEventListener('input', (e) => { player.speed = parseFloat(e.target.value); $('speed-v').textContent = `${player.speed}×`; });
  document.addEventListener('keydown', (e) => {
    if (!current || ['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement.tagName)) return;
    if (e.code === 'Space') { e.preventDefault(); player.playing ? player.stop() : player.play(); }
    else if (e.key === 'ArrowRight') { player.stop(); player.next(); }
    else if (e.key === 'ArrowLeft') { player.stop(); player.prev(); }
    else if (e.key === 'Home') player.reset();
  });

  // ---------------------------------------------------------------- router
  function route() {
    const id = location.hash.slice(1) || 'overview';
    nav.querySelectorAll('button').forEach((b) => b.classList.toggle('active', b.dataset.id === id));
    const s = scenarios.find((x) => x.id === id);
    player.stop();
    if (!s) {
      current = null;
      $('title').textContent = 'Toàn cảnh hệ thống';
      $('desc').textContent = 'Task Manager gồm nhiều bộ phận nhỏ, mỗi bộ phận lo một việc (tài khoản, công việc, lịch, email, tin nhắn…). Khi bạn bấm một nút trên trang web, yêu cầu đi qua Cổng vào tới bộ phận phụ trách; các bộ phận nhờ nhau khi cần, cất dữ liệu vào Cơ sở dữ liệu, và đẩy thông báo lên màn hình bạn ngay lập tức. Chọn một tình huống ở menu trái để xem từng bước diễn ra thế nào.';
      $('overview').classList.remove('hidden');
      $('sim').classList.add('hidden');
      overviewMap.fit();
      return;
    }
    current = s;
    if (!opts[s.id]) opts[s.id] = defaults(s);
    $('title').textContent = `${s.icon} ${s.title}`;
    $('desc').textContent = s.desc;
    $('overview').classList.add('hidden');
    $('sim').classList.remove('hidden');
    renderOptions(s);
    rebuild();
    window.scrollTo({ top: 0 });
  }
  window.addEventListener('hashchange', route);
  route();
})();

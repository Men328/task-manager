/**
 * Seed dữ liệu demo giống design/ui.png (4 cột + task mẫu + subtask để có progress)
 * kèm lịch, sự kiện và backlog để các trang mới cũng có dữ liệu.
 * Dùng để xem UI có dữ liệu thật ngay sau khi start service.
 *
 *   node scripts/seed-demo.mjs
 *
 * Chạy qua Docker (chỉ frontend publish ra host) thì trỏ URL vào nginx:
 *   IDENTITY_URL=http://127.0.0.1:3000/api/identity \
 *   TASK_URL=http://127.0.0.1:3000/api/task \
 *   CALENDAR_URL=http://127.0.0.1:3000/api/calendar \
 *   EVENT_URL=http://127.0.0.1:3000/api/event \
 *   BACKLOG_URL=http://127.0.0.1:3000/api/backlog \
 *   node scripts/seed-demo.mjs
 */
const ID = process.env.IDENTITY_URL ?? 'http://localhost:8081';
const TK = process.env.TASK_URL ?? 'http://localhost:8082';
const CAL = process.env.CALENDAR_URL ?? 'http://localhost:8083';
const EV = process.env.EVENT_URL ?? 'http://localhost:8085';
const BL = process.env.BACKLOG_URL ?? 'http://localhost:8086';

async function req(method, url, body) {
  const res = await fetch(url, {
    method,
    headers: { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await res.text();
  let data;
  try {
    data = text ? JSON.parse(text) : undefined;
  } catch {
    data = text;
  }
  if (!res.ok) {
    throw new Error(`${method} ${url} -> ${res.status} ${text.slice(0, 200)}`);
  }
  return data;
}

const DESC =
  'Lorem ipsum dolor sit amet consectetur. Nisi sed do eiusmod tempor incididunt ut labore et dolore magna.';

const profile = (
  await req('POST', `${ID}/v1/profiles`, {
    email: process.env.SEED_EMAIL ?? 'rico@layers.co',
    display_name: process.env.SEED_NAME ?? 'Rico Tandoor',
  })
).profile;
const pid = profile.id;

const defs = [
  ['To Do', 'todo', '#9aa0ae', 'TASK_STATUS_CATEGORY_TODO', true, false],
  ['On Progress', 'on-progress', '#f59f00', 'TASK_STATUS_CATEGORY_IN_PROGRESS', false, false],
  ['In Review', 'in-review', '#37b24d', 'TASK_STATUS_CATEGORY_IN_PROGRESS', false, false],
  ['Completed', 'completed', '#f06595', 'TASK_STATUS_CATEGORY_DONE', false, true],
];

const existingStatuses = (await req('GET', `${TK}/v1/statuses?profile_id=${pid}`)).statuses ?? [];
const statusBySlug = new Map(existingStatuses.map((status) => [status.slug, status]));

const S = {};
for (const [i, [name, slug, color, category, isDefault, isTerminal]] of defs.entries()) {
  const found = statusBySlug.get(slug);
  if (found) {
    S[slug] = found.id;
    continue;
  }
  const created = await req('POST', `${TK}/v1/statuses`, {
    profile_id: pid,
    name,
    slug,
    color,
    category,
    is_default: isDefault,
    is_terminal: isTerminal,
    position: i,
  });
  S[slug] = created.status.id;
}

const existingTransitions =
  (await req('GET', `${TK}/v1/transitions?profile_id=${pid}`)).transitions ?? [];
const transitionKeys = new Set(
  existingTransitions.map((rule) => `${rule.fromStatusId}->${rule.toStatusId}`),
);

for (const [from, to] of [
  ['todo', 'on-progress'],
  ['on-progress', 'in-review'],
  ['in-review', 'completed'],
  ['in-review', 'on-progress'],
  ['completed', 'on-progress'],
]) {
  if (transitionKeys.has(`${S[from]}->${S[to]}`)) {
    continue;
  }
  await req('POST', `${TK}/v1/transitions`, {
    profile_id: pid,
    from_status_id: S[from],
    to_status_id: S[to],
  });
}

const inDays = (n) => {
  const d = new Date();
  d.setDate(d.getDate() + n);
  return d.toISOString();
};

async function makeTask({ title, status, priority, dueAt, subtasks }) {
  const created = await req('POST', `${TK}/v1/tasks`, {
    profile_id: pid,
    title,
    status_id: S[status],
    priority,
    description: DESC,
    ...(dueAt ? { due_at: dueAt } : {}),
  });
  const taskId = created.task.id;

  if (subtasks) {
    const { total, done } = subtasks;
    for (let i = 0; i < total; i += 1) {
      await req('POST', `${TK}/v1/tasks`, {
        profile_id: pid,
        title: `${title} — bước ${i + 1}`,
        // tạo thẳng ở status đích để không phải đi qua transition
        status_id: i < done ? S.completed : S.todo,
        priority: 'TASK_PRIORITY_LOW',
        parent_task_id: taskId,
      });
    }
  }

  return taskId;
}

await makeTask({ title: 'CRM Structure Plan', status: 'todo', priority: 'TASK_PRIORITY_URGENT', dueAt: inDays(3) });
await makeTask({ title: 'CRM Layout Design', status: 'todo', priority: 'TASK_PRIORITY_URGENT', subtasks: { total: 10, done: 1 } });
await makeTask({ title: 'CRM Layout Draft', status: 'todo', priority: 'TASK_PRIORITY_URGENT', subtasks: { total: 4, done: 3 }, dueAt: inDays(-2) });

await makeTask({ title: 'CRM Layout Design', status: 'on-progress', priority: 'TASK_PRIORITY_URGENT', subtasks: { total: 4, done: 3 }, dueAt: inDays(5) });
await makeTask({ title: 'CRM Structure Plan', status: 'on-progress', priority: 'TASK_PRIORITY_LOW', subtasks: { total: 10, done: 1 } });

await makeTask({ title: 'CRM Structure Plan', status: 'in-review', priority: 'TASK_PRIORITY_LOW', subtasks: { total: 10, done: 1 } });
await makeTask({ title: 'CRM Layout Outline', status: 'in-review', priority: 'TASK_PRIORITY_MEDIUM', subtasks: { total: 4, done: 3 }, dueAt: inDays(9) });

await makeTask({ title: 'CRM Layout Design', status: 'completed', priority: 'TASK_PRIORITY_URGENT' });
await makeTask({ title: 'CRM Layout Draft', status: 'completed', priority: 'TASK_PRIORITY_URGENT' });

const at = (days, hour) => {
  const d = new Date();
  d.setDate(d.getDate() + days);
  d.setHours(hour, 0, 0, 0);
  return d.toISOString();
};

for (const [title, location, day, hour, allDay] of [
  ['Họp nhóm dự án', 'Zoom', 1, 9, false],
  ['Review thiết kế', 'Phòng họp A', 2, 14, false],
  ['Ngày nghỉ cá nhân', '', 5, 0, true],
]) {
  await req('POST', `${CAL}/v1/schedules`, {
    profile_id: pid,
    title,
    description: DESC,
    ...(location ? { location } : {}),
    start_at: at(day, hour),
    ...(allDay ? {} : { end_at: at(day, hour + 1) }),
    all_day: allDay,
  });
}

for (const [title, location, day, hour, status] of [
  ['Hội thảo AI 2026', 'Trung tâm hội nghị', 7, 8, 'EVENT_STATUS_CONFIRMED'],
  ['Tiệc ra mắt sản phẩm', 'Sky Lounge', 12, 18, 'EVENT_STATUS_PLANNED'],
  ['Buổi chia sẻ nội bộ', 'Online', 3, 15, 'EVENT_STATUS_CANCELLED'],
]) {
  await req('POST', `${EV}/v1/events`, {
    profile_id: pid,
    title,
    description: DESC,
    location,
    start_at: at(day, hour),
    end_at: at(day, hour + 2),
    status,
    source: `demo-${day}`,
  });
}

for (const [title, sender, category, reason, status] of [
  ['Newsletter tháng 10', 'promo@shop.vn', 'other', 'no_rule_matched', 'BACKLOG_STATUS_NEW'],
  ['Thông báo bảo trì hệ thống', 'noreply@saas.io', 'other', 'not_actionable', 'BACKLOG_STATUS_NEW'],
  ['Email mời họp thiếu thời gian', 'pm@company.com', 'schedule', 'missing_schedule_time', 'BACKLOG_STATUS_TRIAGED'],
]) {
  await req('POST', `${BL}/v1/backlogs`, {
    profile_id: pid,
    title,
    description: DESC,
    sender,
    category,
    reason,
    status,
    object_key: `mail/${pid}/demo/${encodeURIComponent(title)}.json`,
  });
}

console.log('SEED DONE. profileId =', pid);

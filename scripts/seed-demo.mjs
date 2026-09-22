/**
 * Seed dữ liệu demo giống design/ui.png (4 cột + task mẫu + subtask để có progress).
 * Dùng để xem UI có dữ liệu thật ngay sau khi start service.
 *
 *   node scripts/seed-demo.mjs
 */
const ID = process.env.IDENTITY_URL ?? 'http://localhost:8081';
const TK = process.env.TASK_URL ?? 'http://localhost:8082';
const WS = process.env.WORKSPACE_URL ?? 'http://localhost:8083';

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
    email: 'rico@layers.co',
    display_name: 'Rico Tandoor',
  })
).profile;
const pid = profile.id;

const workspace = (
  await req('POST', `${WS}/v1/workspaces`, {
    owner_profile_id: pid,
    name: 'Công việc của tôi',
    slug: 'cong-viec-cua-toi',
    is_default: true,
  })
).workspace;
const wid = workspace.id;

const defs = [
  ['To Do', 'todo', '#9aa0ae', 'TASK_STATUS_CATEGORY_TODO', true, false],
  ['On Progress', 'on-progress', '#f59f00', 'TASK_STATUS_CATEGORY_IN_PROGRESS', false, false],
  ['In Review', 'in-review', '#37b24d', 'TASK_STATUS_CATEGORY_IN_PROGRESS', false, false],
  ['Completed', 'completed', '#f06595', 'TASK_STATUS_CATEGORY_DONE', false, true],
];

const S = {};
for (const [i, [name, slug, color, category, isDefault, isTerminal]] of defs.entries()) {
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

for (const [from, to] of [
  ['todo', 'on-progress'],
  ['on-progress', 'in-review'],
  ['in-review', 'completed'],
  ['in-review', 'on-progress'],
  ['completed', 'on-progress'],
]) {
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
    workspace_id: wid,
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
        workspace_id: wid,
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

console.log('SEED DONE. profileId =', pid, 'workspaceId =', wid);

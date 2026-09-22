/**
 * Smoke test end-to-end cho base: gọi thẳng HTTP gateway của identity + task.
 * Chạy qua `make smoke` (script sẽ tự build & start service).
 *
 * Lưu ý mapping mã lỗi của grpc-gateway:
 *   InvalidArgument     -> 400
 *   FailedPrecondition  -> 400
 *   NotFound            -> 404
 *   AlreadyExists       -> 409
 */
const IDENTITY = process.env.IDENTITY_URL ?? 'http://localhost:8081';
const TASK = process.env.TASK_URL ?? 'http://localhost:8082';
const WORKSPACE = process.env.WORKSPACE_URL ?? 'http://localhost:8083';

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
  return { status: res.status, data };
}

let failures = 0;
function check(label, r, expectedStatus, assert) {
  const ok = r.status === expectedStatus && (assert ? assert(r) : true);
  if (!ok) failures += 1;
  const detail = ok ? '' : `  <- ${JSON.stringify(r.data).slice(0, 240)}`;
  console.log(`${ok ? 'PASS' : 'FAIL'}  ${label}  [http ${r.status}]${detail}`);
  return r.data;
}

function errorCodeOf(data) {
  const details = data?.details;
  if (Array.isArray(details)) {
    const info = details.find((item) => item && typeof item.reason === 'string');
    if (info) return info.reason;
  }
  return data?.error_code ?? data?.errorCode ?? null;
}

function checkCode(label, r, expectedStatus, expectedCode, extra) {
  return check(
    label,
    r,
    expectedStatus,
    (res) => errorCodeOf(res.data) === expectedCode && (extra ? extra(res) : true),
  );
}

// ---------------------------------------------------------------- health
check('GET /healthz (identity)', await req('GET', `${IDENTITY}/healthz`), 200);
check('GET /healthz (task)', await req('GET', `${TASK}/healthz`), 200);
check('GET /healthz (workspace)', await req('GET', `${WORKSPACE}/healthz`), 200);

// ---------------------------------------------------------------- identity
const profile = check(
  'POST /v1/profiles',
  await req('POST', `${IDENTITY}/v1/profiles`, { email: 'a@example.com', display_name: 'Nguyen Van A' }),
  200,
  (r) => r.data.profile?.displayName === 'Nguyen Van A' && Boolean(r.data.profile?.createdAt),
);
const profileId = profile.profile.id;

check('GET /v1/profiles/{id}', await req('GET', `${IDENTITY}/v1/profiles/${profileId}`), 200,
  (r) => r.data.profile?.email === 'a@example.com');
check('GET /v1/profiles', await req('GET', `${IDENTITY}/v1/profiles`), 200,
  (r) => Array.isArray(r.data.profiles) && r.data.profiles.length === 1);
checkCode('GET profile không tồn tại -> 404 + IDENTITY_PROFILE_NOT_FOUND',
  await req('GET', `${IDENTITY}/v1/profiles/khong-ton-tai`), 404, 'IDENTITY_PROFILE_NOT_FOUND');
checkCode('POST profile thiếu email -> 400 + IDENTITY_EMAIL_REQUIRED',
  await req('POST', `${IDENTITY}/v1/profiles`, { display_name: 'x' }), 400, 'IDENTITY_EMAIL_REQUIRED');

// ---------------------------------------------------------------- workspace
const workspace = check(
  'POST /v1/workspaces (default)',
  await req('POST', `${WORKSPACE}/v1/workspaces`, {
    owner_profile_id: profileId, name: 'Cá nhân', slug: 'personal', is_default: true,
  }),
  200,
  (r) => r.data.workspace?.slug === 'personal' && r.data.workspace?.isDefault === true,
);
const workspaceId = workspace.workspace.id;

check('GET /v1/workspaces?owner_profile_id=...',
  await req('GET', `${WORKSPACE}/v1/workspaces?owner_profile_id=${profileId}`), 200,
  (r) => r.data.workspaces?.length === 1 && r.data.workspaces[0].id === workspaceId);
checkCode('POST /v1/workspaces trùng slug -> 409 + WORKSPACE_SLUG_ALREADY_EXISTS',
  await req('POST', `${WORKSPACE}/v1/workspaces`, {
    owner_profile_id: profileId, name: 'Khác', slug: 'personal',
  }),
  409, 'WORKSPACE_SLUG_ALREADY_EXISTS');
check('PATCH /v1/workspaces/{id}',
  await req('PATCH', `${WORKSPACE}/v1/workspaces/${workspaceId}`, { name: 'Cá nhân (đổi tên)' }), 200,
  (r) => r.data.workspace?.name === 'Cá nhân (đổi tên)' && r.data.workspace?.slug === 'personal');

// ---------------------------------------------------------------- status
const todo = check(
  'POST /v1/statuses (Todo, default)',
  await req('POST', `${TASK}/v1/statuses`, {
    profile_id: profileId, name: 'Todo', slug: 'todo', is_default: true,
    category: 'TASK_STATUS_CATEGORY_TODO', color: '#868E96',
  }),
  200,
  (r) => r.data.status?.isDefault === true,
);
const doing = check(
  'POST /v1/statuses (Doing)',
  await req('POST', `${TASK}/v1/statuses`, {
    profile_id: profileId, name: 'Doing', slug: 'doing',
    category: 'TASK_STATUS_CATEGORY_IN_PROGRESS', color: '#228BE6',
  }),
  200,
);
const done = check(
  'POST /v1/statuses (Done, terminal)',
  await req('POST', `${TASK}/v1/statuses`, {
    profile_id: profileId, name: 'Done', slug: 'done', is_terminal: true,
    category: 'TASK_STATUS_CATEGORY_DONE', color: '#40C057',
  }),
  200,
  (r) => r.data.status?.isTerminal === true,
);
const todoId = todo.status.id;
const doingId = doing.status.id;
const doneId = done.status.id;

check('GET /v1/statuses?profile_id=...',
  await req('GET', `${TASK}/v1/statuses?profile_id=${profileId}`), 200,
  (r) => r.data.statuses?.length === 3);
checkCode('POST /v1/statuses trùng slug -> 409 + TASK_STATUS_SLUG_ALREADY_EXISTS',
  await req('POST', `${TASK}/v1/statuses`, {
    profile_id: profileId, name: 'Todo 2', slug: 'todo',
  }),
  409, 'TASK_STATUS_SLUG_ALREADY_EXISTS');

// ---------------------------------------------------- lifecycle (allowlist)
check('POST /v1/transitions Todo->Doing',
  await req('POST', `${TASK}/v1/transitions`, { profile_id: profileId, from_status_id: todoId, to_status_id: doingId }), 200);
check('POST /v1/transitions Doing->Done',
  await req('POST', `${TASK}/v1/transitions`, { profile_id: profileId, from_status_id: doingId, to_status_id: doneId }), 200);
checkCode('POST /v1/transitions trùng rule -> 409 + TASK_TRANSITION_ALREADY_EXISTS',
  await req('POST', `${TASK}/v1/transitions`, { profile_id: profileId, from_status_id: todoId, to_status_id: doingId }),
  409, 'TASK_TRANSITION_ALREADY_EXISTS');
checkCode('POST /v1/transitions from == to -> 400 + TASK_TRANSITION_SAME_STATUS',
  await req('POST', `${TASK}/v1/transitions`, { profile_id: profileId, from_status_id: todoId, to_status_id: todoId }),
  400, 'TASK_TRANSITION_SAME_STATUS');
check('POST /v1/transitions/validate Todo->Doing => allowed=true',
  await req('POST', `${TASK}/v1/transitions/validate`, { profile_id: profileId, from_status_id: todoId, to_status_id: doingId }), 200,
  (r) => r.data.allowed === true);
check('POST /v1/transitions/validate Todo->Done => allowed=false (không khai báo = cấm)',
  await req('POST', `${TASK}/v1/transitions/validate`, { profile_id: profileId, from_status_id: todoId, to_status_id: doneId }), 200,
  (r) => r.data.allowed === false);

// ---------------------------------------------------------------- task
checkCode('POST /v1/tasks thiếu workspace_id -> 400 + TASK_WORKSPACE_ID_REQUIRED',
  await req('POST', `${TASK}/v1/tasks`, { profile_id: profileId, title: 'Thiếu workspace' }),
  400, 'TASK_WORKSPACE_ID_REQUIRED');
checkCode('POST /v1/tasks workspace không tồn tại -> 404 + COMMON_NOT_FOUND',
  await req('POST', `${TASK}/v1/tasks`, {
    profile_id: profileId, workspace_id: '22222222-2222-2222-2222-222222222222', title: 'Workspace lạ',
  }),
  404, 'COMMON_NOT_FOUND');

const task = check(
  'POST /v1/tasks (không truyền status_id -> dùng status default)',
  await req('POST', `${TASK}/v1/tasks`, {
    profile_id: profileId, workspace_id: workspaceId, title: 'Viết báo cáo', priority: 'TASK_PRIORITY_HIGH',
  }),
  200,
  (r) => r.data.task?.statusId === todoId && r.data.task?.parentTaskId === '' &&
    r.data.task?.workspaceId === workspaceId,
);
const taskId = task.task.id;

const sub = check(
  'POST /v1/tasks (task con)',
  await req('POST', `${TASK}/v1/tasks`, {
    profile_id: profileId, workspace_id: workspaceId, title: 'Thu thập số liệu', parent_task_id: taskId,
  }),
  200,
  (r) => r.data.task?.parentTaskId === taskId,
);
const subId = sub.task.id;

checkCode('GET /v1/tasks thiếu workspace_id -> 400 + TASK_WORKSPACE_ID_REQUIRED',
  await req('GET', `${TASK}/v1/tasks?profile_id=${profileId}&root_only=true`),
  400, 'TASK_WORKSPACE_ID_REQUIRED');
check('GET /v1/tasks?workspace_id=...&root_only=true&include_subtasks=true',
  await req('GET', `${TASK}/v1/tasks?profile_id=${profileId}&workspace_id=${workspaceId}&root_only=true&include_subtasks=true`), 200,
  (r) => r.data.tasks?.length === 1 && r.data.tasks[0].subtasks?.length === 1);
check('GET /v1/tasks workspace khác -> rỗng',
  await req('GET', `${TASK}/v1/tasks?profile_id=${profileId}&workspace_id=22222222-2222-2222-2222-222222222222`), 200,
  (r) => (r.data.tasks?.length ?? 0) === 0);
check('GET /v1/tasks/{id}?include_subtasks=true',
  await req('GET', `${TASK}/v1/tasks/${taskId}?include_subtasks=true`), 200,
  (r) => r.data.task?.subtasks?.length === 1);

// --------------------------------------------- lifecycle enforcement
checkCode('POST /v1/tasks/{id}/status Todo->Done (không có rule) -> 400 + TASK_TRANSITION_NOT_ALLOWED',
  await req('POST', `${TASK}/v1/tasks/${taskId}/status`, { status_id: doneId }), 400, 'TASK_TRANSITION_NOT_ALLOWED',
  (r) => /status change isn't allowed/.test(r.data.message ?? ''));
check('POST /v1/tasks/{id}/status Todo->Doing (có rule)',
  await req('POST', `${TASK}/v1/tasks/${taskId}/status`, { status_id: doingId, note: 'bắt đầu làm' }), 200,
  (r) => r.data.task?.statusId === doingId && r.data.task?.completedAt == null);
check('POST /v1/tasks/{id}/status Doing->Done (tự set completed_at)',
  await req('POST', `${TASK}/v1/tasks/${taskId}/status`, { status_id: doneId }), 200,
  (r) => r.data.task?.statusId === doneId && Boolean(r.data.task?.completedAt));

// ---------------------------------------------------------------- ràng buộc
checkCode('DELETE /v1/tasks/{id} khi còn task con -> 400 + TASK_HAS_SUBTASKS',
  await req('DELETE', `${TASK}/v1/tasks/${taskId}`), 400, 'TASK_HAS_SUBTASKS');
checkCode('DELETE /v1/statuses/{id} khi đang được task dùng -> 400 + TASK_STATUS_IN_USE',
  await req('DELETE', `${TASK}/v1/statuses/${todoId}`), 400, 'TASK_STATUS_IN_USE');
check('PATCH /v1/tasks/{id}',
  await req('PATCH', `${TASK}/v1/tasks/${subId}`, { title: 'Thu thập số liệu (đã sửa)' }), 200,
  (r) => r.data.task?.title === 'Thu thập số liệu (đã sửa)');
checkCode('PATCH /v1/tasks/{id} tạo vòng lặp -> 400 + TASK_CYCLE_DETECTED',
  await req('PATCH', `${TASK}/v1/tasks/${taskId}`, { parent_task_id: subId }), 400, 'TASK_CYCLE_DETECTED');
check('DELETE /v1/tasks/{id} (con)', await req('DELETE', `${TASK}/v1/tasks/${subId}`), 200);
check('DELETE /v1/tasks/{id} (cha)', await req('DELETE', `${TASK}/v1/tasks/${taskId}`), 200);

console.log(failures === 0 ? '\nSMOKE: ALL PASS' : `\nSMOKE: ${failures} FAILED`);
process.exit(failures === 0 ? 0 : 1);

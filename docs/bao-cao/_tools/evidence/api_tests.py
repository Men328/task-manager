"""Kiểm thử chức năng qua API thật (qua nginx localhost:3000) + tạo dữ liệu mẫu cho ảnh giao diện.

python api_tests.py <token_men_file> <token_second_file> <out_json>

- Tài khoản chính: menc1lhp@gmail.com. Tài khoản phụ (chỉ cho case chat/giao task): 22010328@st.phenikaa-uni.edu.vn.
- Mỗi case ghi: nhóm, chức năng, đầu vào, mong đợi, thực tế (từ response thật), trạng thái Pass/Fail.
- Dữ liệu tạo ra được giữ lại làm dữ liệu mẫu cho ảnh chụp giao diện.
"""
import json
import os
import sys
import time
import uuid
from datetime import datetime, timedelta, timezone

import requests

BASE = "http://localhost:3000/api"
MEN = "fec32ca3-1feb-4f48-8ce9-ecda1dfa5286"
SECOND = "bbffa35b-d862-485b-a49f-6e828e10dc76"  # 22010328@st.phenikaa-uni.edu.vn
NON_FRIEND = "8064c833-c1f6-483c-8a0f-c6bc46b0af3c"

TOKEN_MEN = open(sys.argv[1]).read().strip()
TOKEN_SECOND = open(sys.argv[2]).read().strip()
OUT = sys.argv[3]
ONLY = [x.strip() for x in os.environ.get("ONLY", "").split(",") if x.strip()]


def want(n):
    return not ONLY or str(n) in ONLY

results = []
NOW = datetime.now(timezone.utc).replace(microsecond=0)


def iso(dt):
    return dt.strftime("%Y-%m-%dT%H:%M:%SZ")


def call(method, path, token=TOKEN_MEN, **kw):
    headers = kw.pop("headers", {})
    if token:
        headers["Authorization"] = f"Bearer {token}"
    r = requests.request(method, BASE + path, headers=headers, timeout=30, **kw)
    try:
        body = r.json()
    except ValueError:
        body = {"_raw": r.text[:200]}
    return r.status_code, body


def reason(body):
    for d in (body.get("details") or []) if isinstance(body, dict) else []:
        if d.get("reason"):
            return d["reason"]
    return ""


def record(group, func, inp, expect, actual, ok):
    results.append({"nhom": group, "chuc_nang": func, "dau_vao": inp, "mong_doi": expect,
                    "thuc_te": actual, "trang_thai": "Pass" if ok else "Fail"})
    print(("PASS " if ok else "FAIL ") + f"[{group}] {func}: {actual}")


def expect_error(group, func, inp, code, status, body):
    got = reason(body)
    record(group, func, inp, f"Từ chối, mã lỗi {code}", f"HTTP {status}, mã lỗi {got or '(không có)'}",
           status >= 400 and got == code)


def wait_for(fn, timeout=20, step=1.0):
    end = time.time() + timeout
    while time.time() < end:
        v = fn()
        if v:
            return v
        time.sleep(step)
    return None


# ------------------------------------------------------------------ dữ liệu chung
_, st = call("GET", f"/task/v1/statuses?profile_id={MEN}")
S = {s["name"]: s["id"] for s in st["taskStatuses"]} if "taskStatuses" in st else {s["name"]: s["id"] for s in next(v for v in st.values() if isinstance(v, list))}
_, pj = call("GET", f"/project/v1/projects?profile_id={MEN}")
PROJECTS = {p["name"]: p for p in pj["projects"]}
DEFAULT_PROJECT = next(p["id"] for p in pj["projects"] if p.get("isDefault"))

# ================================================================== 1. Xác thực
if want(1):
    G = "Xác thực"
    s, b = call("GET", "/identity/v1/auth/me")
    record(G, "Lấy hồ sơ bằng session token hợp lệ", "Token của menc1lhp@gmail.com", "Trả hồ sơ đúng email",
           f"HTTP {s}, email {b.get('profile', {}).get('email')}", s == 200 and b.get("profile", {}).get("email") == "menc1lhp@gmail.com")
    s, b = call("GET", "/identity/v1/auth/me", token=None)
    expect_error(G, "Gọi API khi chưa đăng nhập", "Không gửi token", "IDENTITY_AUTH_TOKEN_MISSING", s, b)
    s, b = call("GET", "/identity/v1/auth/me", token=TOKEN_MEN[:-4] + "abcd")
    expect_error(G, "Token bị sửa chữ ký", "Token sai chữ ký", "IDENTITY_AUTH_TOKEN_INVALID", s, b)
    s, b = call("POST", "/identity/v1/auth/soketi", data={"socket_id": "123.456", "channel_name": f"private-noti-internal-{MEN}"})
    record(G, "Xin quyền nghe kênh thông báo của mình", f"private-noti-internal-<id của mình>", "Trả chữ ký auth",
           f"HTTP {s}, có auth: {'auth' in b}", s == 200 and "auth" in b)
    s, b = call("POST", "/identity/v1/auth/soketi", data={"socket_id": "123.456", "channel_name": f"private-noti-internal-{SECOND}"})
    expect_error(G, "Xin quyền nghe kênh của người khác", "private-noti-internal-<id người khác>", "IDENTITY_AUTH_SOKETI_CHANNEL_FORBIDDEN", s, b)

# ================================================================== 2. Dự án
if want(2):
    G = "Dự án"
    NAME = "Đồ án tốt nghiệp"
    if NAME in PROJECTS:
        DOAN = PROJECTS[NAME]["id"]
        record(G, "Tạo dự án mới", f"name = \"{NAME}\"", "Tạo thành công", "Đã tồn tại từ lần chạy trước", True)
    else:
        s, b = call("POST", "/project/v1/projects", json={"profile_id": MEN, "name": NAME, "description": "Hệ thống quản lý công việc cá nhân, báo cáo đồ án"})
        DOAN = b.get("project", {}).get("id")
        record(G, "Tạo dự án mới", f"name = \"{NAME}\"", "Tạo thành công", f"HTTP {s}, id {str(DOAN)[:8]}…", s == 200 and bool(DOAN))
    s, b = call("POST", "/project/v1/projects", json={"profile_id": MEN, "name": NAME.upper()})
    expect_error(G, "Tạo dự án trùng tên (khác hoa thường)", f"name = \"{NAME.upper()}\"", "PROJECT_NAME_DUPLICATE", s, b)
    s, b = call("POST", f"/project/v1/projects/{DEFAULT_PROJECT}/archive", json={})
    expect_error(G, "Lưu trữ dự án mặc định", "Dự án \"Chung\"", "PROJECT_DEFAULT_NOT_ARCHIVABLE", s, b)
    s, b = call("POST", "/project/v1/projects", json={"profile_id": MEN, "name": f"Dự án tạm {uuid.uuid4().hex[:4]}"})
    TMP = b["project"]["id"]
    call("POST", f"/project/v1/projects/{TMP}/archive", json={})
    s, b = call("POST", "/task/v1/tasks", json={"profile_id": MEN, "project_id": TMP, "title": "Task vào dự án đã lưu trữ"})
    expect_error(G, "Tạo task trong dự án đã lưu trữ", "project_id = dự án đã archive", "PROJECT_ARCHIVED", s, b)

# ================================================================== 3. Công việc & vòng đời
if want(3):
    G = "Công việc"
    s, b = call("POST", "/task/v1/tasks", json={"profile_id": MEN, "project_id": DOAN, "title": ""})
    expect_error(G, "Tạo task thiếu tiêu đề", "title rỗng", "TASK_TITLE_REQUIRED", s, b)


    def new_task(title, **extra):
        body = {"profile_id": MEN, "project_id": DOAN, "title": title}
        body.update(extra)
        return call("POST", "/task/v1/tasks", json=body)


    s, b = new_task("Viết Chương 4 - Thiết kế hệ thống", description="Kiến trúc, API, CSDL, giao diện",
                    priority="TASK_PRIORITY_HIGH", due_at=iso(NOW + timedelta(days=3)))
    T1 = b.get("task", {})
    record(G, "Tạo task hợp lệ", "Tiêu đề, mô tả, ưu tiên cao, hạn +3 ngày", "Tạo thành công, trạng thái mặc định To Do",
           f"HTTP {s}, trạng thái {'To Do' if T1.get('statusId') == S['To Do'] else T1.get('statusId')}", s == 200 and T1.get("statusId") == S["To Do"])
    subs = []
    for t in ["Vẽ sơ đồ kiến trúc", "Mô tả cơ sở dữ liệu", "Chụp ảnh giao diện"]:
        s, b = new_task(t, parent_task_id=T1["id"])
        subs.append(b.get("task", {}))
    record(G, "Tạo task con", "3 task con của task \"Viết Chương 4\"", "Tạo thành công, cùng dự án với cha",
           f"{len([x for x in subs if x.get('id')])}/3 task con, cùng dự án: {all(x.get('projectId') == DOAN for x in subs)}",
           all(x.get("projectId") == DOAN for x in subs))

    s, b = call("POST", f"/task/v1/tasks/{T1['id']}/status", json={"status_id": S["Completed"]})
    expect_error(G, "Kéo task To Do sang Completed (không có luật)", "To Do -> Completed", "TASK_TRANSITION_NOT_ALLOWED", s, b)
    s, b = call("POST", f"/task/v1/tasks/{subs[0]['id']}/status", json={"status_id": S["On Progress"], "note": "bắt đầu"})
    record(G, "Kéo task To Do sang On Progress (có luật)", "To Do -> On Progress", "Đổi trạng thái thành công",
           f"HTTP {s}", s == 200 and b.get("task", {}).get("statusId") == S["On Progress"])
    for st_name in ["In Review", "Completed"]:
        s, b = call("POST", f"/task/v1/tasks/{subs[0]['id']}/status", json={"status_id": S[st_name]})
    done = b.get("task", {})
    record(G, "Hoàn thành task", "On Progress -> In Review -> Completed", "Ghi thời điểm hoàn thành (completedAt)",
           f"HTTP {s}, completedAt {'có' if done.get('completedAt') else 'không'}", s == 200 and bool(done.get("completedAt")))
    s, b = call("DELETE", f"/task/v1/tasks/{T1['id']}")
    expect_error(G, "Xoá task còn task con", "Xoá task \"Viết Chương 4\"", "TASK_HAS_SUBTASKS", s, b)

    s, b = new_task("Đọc tài liệu Kafka 30 phút", due_at=iso(NOW + timedelta(hours=2)), recurrence_rule="FREQ=DAILY",
                    recurrence_tz="Asia/Ho_Chi_Minh")
    R = b.get("task", {})
    for st_name in ["On Progress", "In Review", "Completed"]:
        s, b = call("POST", f"/task/v1/tasks/{R['id']}/status", json={"status_id": S[st_name]})
    nxt = b.get("nextTask") or {}
    record(G, "Hoàn thành task lặp lại hằng ngày", "recurrence_rule = FREQ=DAILY, chuyển tới Completed",
           "Tự sinh lần kế tiếp, hạn +1 ngày", f"HTTP {s}, lần kế tiếp: {'có, hạn ' + nxt.get('dueAt', '')[:10] if nxt else 'không'}",
           bool(nxt) and nxt.get("dueAt", "")[:10] == (datetime.fromisoformat(R["dueAt"].replace("Z", "+00:00")) + timedelta(days=1)).strftime("%Y-%m-%d"))

    s, b = call("GET", f"/task/v1/tasks?profile_id={MEN}&query=kafka")
    titles = [t["title"] for t in b.get("tasks", [])]
    record(G, "Tìm kiếm task theo từ khoá", "query = \"kafka\"", "Trả các task có chữ Kafka",
           f"HTTP {s}, {len(titles)} kết quả", s == 200 and len(titles) >= 1 and all("kafka" in t.lower() for t in titles))

    remind_at = datetime.now(timezone.utc) + timedelta(seconds=5)
    s, b = new_task("Nộp bản nháp báo cáo cho giảng viên", priority="TASK_PRIORITY_URGENT",
                    due_at=iso(NOW + timedelta(days=1)), remind_at=iso(remind_at))
    REM = b.get("task", {})


    def reminder_notice():
        _, nb = call("GET", "/notification/v1/notices?limit=30")
        return next((n for n in nb.get("notices", []) if n.get("targetId") == REM.get("id") and n.get("type") == "TASK_REMINDER"), None)


    got = wait_for(reminder_notice, timeout=90, step=5)
    record(G, "Nhắc việc đúng giờ", "remind_at = hiện tại + 5 giây", "Nhận thông báo TASK_REMINDER",
           f"Thông báo: {got.get('title') if got else 'không nhận được trong 90 giây'}", bool(got))

    # thêm vài task để bảng Kanban có đủ cột
    for title, st_name, prio, days, proj in [
        ("Chuẩn bị slide bảo vệ", "To Do", "TASK_PRIORITY_MEDIUM", 10, DOAN),
        ("Kiểm thử chức năng nhắn tin", "On Progress", "TASK_PRIORITY_HIGH", 2, DOAN),
        ("Rà soát Chương 3 với giảng viên", "In Review", "TASK_PRIORITY_MEDIUM", 1, DOAN),
        ("Gửi email xác nhận lịch họp nhóm", "To Do", "TASK_PRIORITY_LOW", 4, DEFAULT_PROJECT),
    ]:
        s, b = call("POST", "/task/v1/tasks", json={"profile_id": MEN, "project_id": proj, "title": title, "priority": prio,
                                                     "due_at": iso(NOW + timedelta(days=days))})
        tid = b.get("task", {}).get("id")
        path = {"On Progress": ["On Progress"], "In Review": ["On Progress", "In Review"]}.get(st_name, [])
        for x in path:
            call("POST", f"/task/v1/tasks/{tid}/status", json={"status_id": S[x]})

# ================================================================== 4. Lịch & sự kiện
if want(4):
    G = "Lịch và sự kiện"
    day = (NOW + timedelta(days=2)).replace(hour=2, minute=0, second=0)
    s, b = call("POST", "/calendar/v1/schedules", json={"profile_id": MEN, "project_id": DOAN, "title": "Họp nhóm đồ án",
                                                         "start_at": iso(day), "end_at": iso(day - timedelta(hours=1))})
    expect_error(G, "Tạo lịch có giờ kết thúc trước giờ bắt đầu", "end_at < start_at", "CALENDAR_TIME_RANGE_INVALID", s, b)
    s, b = call("POST", "/calendar/v1/schedules", json={"profile_id": MEN, "project_id": DOAN, "title": "Họp nhóm đồ án",
                                                         "location": "Phòng A9-301", "start_at": iso(day),
                                                         "end_at": iso(day + timedelta(hours=1, minutes=30)), "color": "#4C6EF5"})
    SCH = b.get("schedule", {})
    s2, b2 = call("GET", f"/calendar/v1/schedules?profile_id={MEN}&from={iso(day - timedelta(days=1))}&to={iso(day + timedelta(days=1))}")
    record(G, "Tạo lịch hợp lệ và xem theo khoảng thời gian", "Họp nhóm 9:00–10:30, lọc ±1 ngày", "Lịch xuất hiện trong khoảng lọc",
           f"HTTP {s}, có trong danh sách: {any(x['id'] == SCH.get('id') for x in b2.get('schedules', []))}",
           s == 200 and any(x["id"] == SCH.get("id") for x in b2.get("schedules", [])))
    for title, d, h, dur in [("Gặp giảng viên hướng dẫn", 4, 7, 1), ("Ôn tập bảo vệ", 6, 13, 2)]:
        st_ = (NOW + timedelta(days=d)).replace(hour=h, minute=0, second=0)
        call("POST", "/calendar/v1/schedules", json={"profile_id": MEN, "project_id": DOAN, "title": title,
                                                      "start_at": iso(st_), "end_at": iso(st_ + timedelta(hours=dur))})
    ev_day = (NOW + timedelta(days=5)).replace(hour=1, minute=30, second=0)
    s, b = call("POST", "/event/v1/events", json={"profile_id": MEN, "project_id": DOAN, "title": "Hội thảo Công nghệ phần mềm",
                                                   "location": "Hội trường A2", "start_at": iso(ev_day),
                                                   "end_at": iso(ev_day + timedelta(hours=3)), "status": "EVENT_STATUS_PLANNED"})
    EV = b.get("event", {})
    s2, b2 = call("PATCH", f"/event/v1/events/{EV.get('id')}", json={"status": "EVENT_STATUS_CONFIRMED"})
    record(G, "Tạo sự kiện và xác nhận", "Hội thảo, PLANNED -> CONFIRMED", "Sự kiện chuyển sang đã xác nhận",
           f"HTTP {s}/{s2}, trạng thái {b2.get('event', {}).get('status')}", s2 == 200 and b2.get("event", {}).get("status") == "EVENT_STATUS_CONFIRMED")
    call("POST", "/event/v1/events", json={"profile_id": MEN, "project_id": DOAN, "title": "Lễ bảo vệ đồ án tốt nghiệp",
                                           "start_at": iso(NOW + timedelta(days=20)), "status": "EVENT_STATUS_PLANNED"})
    s, b = call("POST", "/event/v1/events", json={"profile_id": MEN, "project_id": DOAN, "title": "Sự kiện lỗi",
                                                   "start_at": iso(ev_day), "end_at": iso(ev_day - timedelta(hours=1))})
    expect_error(G, "Tạo sự kiện có khoảng thời gian sai", "end_at < start_at", "EVENT_TIME_RANGE_INVALID", s, b)

# ================================================================== 5. Email & backlog
if want(5):
    G = "Email và backlog"
    s, b = call("GET", f"/backlog/v1/backlogs?profile_id={MEN}")
    bl = b.get("backlogs", [])
    from_mail = [x for x in bl if x.get("sender") and x.get("source")]
    record(G, "Email Gmail thật được phân loại tự động", "Thư gửi tới menc1lhp@gmail.com (newsletter, yêu cầu công việc, lịch báo cáo)",
           "Email được lưu thành bản ghi kèm người gửi, lý do",
           f"{len(from_mail)} email trong backlog, ví dụ: \"{from_mail[0]['title'][:40]}…\" ({from_mail[0]['reason'][:40]})" if from_mail else "Không có",
           len(from_mail) >= 1)
    news = [x for x in from_mail if x.get("category") == "other"]
    record(G, "Newsletter được phân loại là other", "Email TED, Lumosity", "category = other, vào backlog",
           f"{len(news)} newsletter, category other", len(news) >= 1)
    nm = [x for x in from_mail if x.get("reason") == "no_project_matched"]
    record(G, "Email công việc không khớp dự án nào", "Email yêu cầu xem tài liệu / lịch báo cáo, không nhắc tên dự án", "Vào backlog với lý do no_project_matched",
           f"{len(nm)} email, category {', '.join(sorted({x['category'] for x in nm}))}", len(nm) >= 1)
    if bl:
        s, b = call("PATCH", f"/backlog/v1/backlogs/{bl[0]['id']}", json={"status": "BACKLOG_STATUS_TRIAGED"})
        record(G, "Đánh dấu backlog đã xử lý", "status = TRIAGED", "Cập nhật thành công", f"HTTP {s}, {b.get('backlog', {}).get('status')}",
               s == 200 and b.get("backlog", {}).get("status") == "BACKLOG_STATUS_TRIAGED")
    _, nb = call("GET", "/notification/v1/notices?limit=50")
    mail_notices = [n for n in nb.get("notices", []) if n.get("type") in ("BACKLOG_CREATED", "TASK_CREATED", "SCHEDULE_CREATED", "EVENT_CREATED")]
    record(G, "Thông báo khi email được xử lý", "Các email ở trên", "Có thông báo *_CREATED", f"{len(mail_notices)} thông báo", len(mail_notices) >= 1)

# ================================================================== 6. Tệp đính kèm
if want(6):
    G = "Tệp đính kèm"
    content = ("Đề cương đồ án tốt nghiệp - Trần Thị Mến\n" * 40).encode()
    s, b = call("POST", "/attachment/v1/attachments/upload",
                data={"profile_id": MEN, "owner_type": "task", "owner_id": T1["id"]},
                files={"file": ("de-cuong-do-an.txt", content, "text/plain")})
    AT = b.get("attachment") or b  # API trả object tệp ở cấp ngoài cùng
    record(G, "Tải tệp lên task", "de-cuong-do-an.txt, " + str(len(content)) + " byte", "Tải lên thành công",
           f"HTTP {s}, kích thước {AT.get('size')}", s == 200 and int(AT.get("size", 0)) == len(content))
    r = requests.get(f"{BASE}/attachment/v1/attachments/{AT.get('id')}/download", headers={"Authorization": f"Bearer {TOKEN_MEN}"}, timeout=30)
    record(G, "Tải tệp xuống", "Tệp vừa tải lên", "Nội dung giống hệt bản gốc",
           f"HTTP {r.status_code}, giống bản gốc: {r.content == content}", r.status_code == 200 and r.content == content)
    s, b = call("POST", "/attachment/v1/attachments/upload",
                data={"profile_id": MEN, "owner_type": "task", "owner_id": T1["id"]},
                files={"file": ("qua-lon.bin", b"0" * (11 * 1024 * 1024), "application/octet-stream")})
    expect_error(G, "Tải tệp vượt giới hạn", "Tệp 11 MiB (giới hạn 10 MiB)", "ATTACHMENT_FILE_TOO_LARGE", s, b)
    s, b = call("POST", "/attachment/v1/attachments/upload",
                data={"profile_id": MEN, "owner_type": "abc", "owner_id": T1["id"]}, files={"file": ("a.txt", b"x", "text/plain")})
    expect_error(G, "Loại đối tượng sở hữu không hợp lệ", "owner_type = abc", "ATTACHMENT_OWNER_TYPE_INVALID", s, b)

# ================================================================== 7. Báo cáo
if want(7):
    G = "Báo cáo"
    _, tb = call("GET", f"/task/v1/tasks?profile_id={MEN}&include_subtasks=true")
    s, b = call("GET", f"/report/v1/reports/overview?profile_id={MEN}&from={iso(NOW - timedelta(days=30))}&to={iso(NOW + timedelta(days=1))}")
    ov = b.get("overview", b)
    record(G, "Báo cáo tổng quan", "30 ngày gần nhất", "Trả tổng số task, tỉ lệ hoàn thành, quá hạn",
           f"HTTP {s}, tổng {ov.get('totalTasks')}, hoàn thành {ov.get('completionRate')}%", s == 200 and int(ov.get("totalTasks", 0)) > 0)
    s, b = call("GET", f"/report/v1/reports/status-breakdown?profile_id={MEN}")
    items = b.get("items") or b.get("statuses") or next((v for v in b.values() if isinstance(v, list)), [])
    total_pct = round(sum(float(x.get("percent", x.get("percentage", 0))) for x in items), 1)
    record(G, "Tỉ lệ task theo trạng thái", "Không lọc", "Tổng tỉ lệ các trạng thái = 100%", f"HTTP {s}, {len(items)} trạng thái, tổng {total_pct}%",
           s == 200 and abs(total_pct - 100) < 0.6)
    s, b = call("GET", f"/report/v1/reports/timeseries?profile_id={MEN}&interval=REPORT_INTERVAL_WEEK")
    pts = b.get("points") or next((v for v in b.values() if isinstance(v, list)), [])
    record(G, "Biểu đồ tạo mới / hoàn thành theo tuần", "interval = WEEK", "Đủ các mốc tuần, có số tạo mới và hoàn thành",
           f"HTTP {s}, {len(pts)} mốc", s == 200 and len(pts) >= 4)
    s, b = call("GET", f"/report/v1/reports/activity?profile_id={MEN}&from={iso(NOW - timedelta(days=1))}&to={iso(NOW + timedelta(days=30))}")
    record(G, "Thống kê lịch, sự kiện, backlog", "30 ngày tới", "Trả số liệu lịch, sự kiện, backlog", f"HTTP {s}, khoá: {', '.join(list(b.keys())[:3])}",
           s == 200 and len(b) >= 2)

# ================================================================== 8. Bạn bè, nhắn tin, giao task qua chat
if want(8):
    G = "Bạn bè và nhắn tin"
    s, b = call("GET", "/social/v1/friends/search?email=22010328@st.phenikaa-uni.edu.vn")
    record(G, "Tìm người dùng theo email", "22010328@st.phenikaa-uni.edu.vn", "Tìm thấy hồ sơ",
           f"HTTP {s}, tìm thấy: {b.get('profile', {}).get('id') == SECOND}", s == 200 and b.get("profile", {}).get("id") == SECOND)
    s, b = call("POST", "/social/v1/friend-requests", json={"profile_id": MEN})
    expect_error(G, "Tự kết bạn với chính mình", "profile_id = chính mình", "SOCIAL_CANNOT_FRIEND_SELF", s, b)
    _, fr = call("GET", "/social/v1/friends")
    if any(f["profile"]["id"] == SECOND for f in fr.get("friends", [])):
        s, b = call("DELETE", f"/social/v1/friends/{SECOND}")
        _, fr = call("GET", "/social/v1/friends")
        record(G, "Huỷ kết bạn", "Huỷ kết bạn với 22010328@st.phenikaa-uni.edu.vn", "Hai người không còn là bạn bè",
               f"HTTP {s}, còn là bạn: {any(f['profile']['id'] == SECOND for f in fr.get('friends', []))}",
               s == 200 and not any(f["profile"]["id"] == SECOND for f in fr.get("friends", [])))
    s, b = call("POST", "/social/v1/friend-requests", json={"profile_id": SECOND})
    record(G, "Gửi lời mời kết bạn", "Tới 22010328@st.phenikaa-uni.edu.vn", "Tạo lời mời đang chờ", f"HTTP {s}", s == 200)
    _, rq = call("GET", "/social/v1/friend-requests", token=TOKEN_SECOND)
    incoming = rq.get("incoming", [])
    req = next((x for x in incoming if x.get("profile", {}).get("id") == MEN or x.get("requester", {}).get("id") == MEN), incoming[0] if incoming else {})
    s, b = call("POST", f"/social/v1/friend-requests/{req.get('id')}/accept", json={}, token=TOKEN_SECOND)
    _, fr = call("GET", "/social/v1/friends")
    record(G, "Chấp nhận lời mời kết bạn", "Tài khoản 22010328 chấp nhận", "Hai người trở thành bạn bè",
           f"HTTP {s}, đã là bạn: {any(f['profile']['id'] == SECOND for f in fr.get('friends', []))}",
           s == 200 and any(f["profile"]["id"] == SECOND for f in fr.get("friends", [])))

    s, b = call("POST", f"/message/v1/conversations/{SECOND}/messages", json={"body": "", "client_id": uuid.uuid4().hex})
    expect_error(G, "Gửi tin nhắn rỗng", "body rỗng", "MESSAGE_BODY_REQUIRED", s, b)
    s, b = call("POST", f"/message/v1/conversations/{NON_FRIEND}/messages", json={"body": "Xin chào", "client_id": uuid.uuid4().hex})
    expect_error(G, "Nhắn tin cho người chưa kết bạn", "Người nhận không phải bạn bè", "MESSAGE_NOT_FRIENDS", s, b)
    text = f"Chào bạn, mai mình họp nhóm lúc 9h nhé! ({datetime.now().strftime('%H:%M:%S')})"
    s, b = call("POST", f"/message/v1/conversations/{SECOND}/messages", json={"body": text, "client_id": uuid.uuid4().hex})
    call("POST", f"/message/v1/conversations/{MEN}/messages", json={"body": "Ok bạn, mình sẽ chuẩn bị tài liệu trước.", "client_id": uuid.uuid4().hex}, token=TOKEN_SECOND)


    def peer_sees():
        _, mb = call("GET", f"/message/v1/conversations/{MEN}/messages?limit=20", token=TOKEN_SECOND)
        return any(m.get("body") == text for m in mb.get("messages", []))


    saved = wait_for(peer_sees, timeout=20)
    record(G, "Gửi tin nhắn cho bạn bè và lưu lịch sử", "Tin nhắn văn bản tới 22010328", "Người nhận xem được tin trong lịch sử (qua Kafka → worker)",
           f"HTTP {s}, có trong lịch sử người nhận: {bool(saved)}", s == 200 and bool(saved))

    s, b = call("POST", f"/message/v1/conversations/{SECOND}/tasks",
                json={"title": "Chuẩn bị tài liệu họp nhóm", "description": "In 5 bản đề cương", "priority": "high",
                      "due_at": iso(NOW + timedelta(days=1)), "client_id": uuid.uuid4().hex})
    msg = b.get("message", {})
    task_ref = msg.get("task", {})
    _, bt = call("GET", f"/task/v1/tasks?profile_id={SECOND}&query=tài liệu họp nhóm", token=TOKEN_SECOND)
    _, gt = call("GET", f"/task/v1/tasks/{task_ref.get('taskId')}", token=TOKEN_SECOND)
    btask = gt.get("task", {}) if gt.get("task", {}).get("profileId") == SECOND else {}
    record(G, "Giao task qua chat bằng lệnh /task", "/task Chuẩn bị tài liệu họp nhóm, ưu tiên cao, hạn +1 ngày",
           "Task được tạo trong danh sách của người nhận, tin nhắn có thẻ task",
           f"HTTP {s}, thẻ task: {bool(task_ref)}, task thuộc người nhận: {bool(btask)}", s == 200 and bool(btask))
    _, bst = call("GET", f"/task/v1/statuses?profile_id={SECOND}", token=TOKEN_SECOND)
    BS = {x["name"]: x["id"] for x in next(v for v in bst.values() if isinstance(v, list))}
    call("POST", f"/task/v1/tasks/{btask.get('id')}/status", json={"status_id": BS.get("On Progress")}, token=TOKEN_SECOND)


    def men_notified():
        _, nb2 = call("GET", "/notification/v1/notices?limit=20")
        return next((n for n in nb2.get("notices", []) if n.get("type") == "TASK_STATUS_CHANGED"
                     and "Chuẩn bị tài liệu họp nhóm" in n.get("title", "")), None)


    got = wait_for(men_notified, timeout=30)
    record(G, "Người giao được báo khi task đổi trạng thái", "Người nhận chuyển task sang On Progress", "Người giao nhận thông báo TASK_STATUS_CHANGED",
           f"Thông báo: {got.get('title') if got else 'không nhận được trong 30 giây'}", bool(got))

# ================================================================== 9. Thông báo
if want(9):
    G = "Thông báo"
    s, b = call("GET", "/notification/v1/notices/unread-count")
    before = int(b.get("count", 0))
    record(G, "Đếm thông báo chưa đọc", "Tài khoản chính", "Trả số lượng chưa đọc", f"HTTP {s}, {before} chưa đọc", s == 200)
    s, b = call("GET", "/notification/v1/notices", token=TOKEN_SECOND)
    mine = all(n.get("profileId") in (None, SECOND) for n in b.get("notices", []))
    record(G, "Chỉ xem được thông báo của mình", "Token tài khoản 22010328", "Không thấy thông báo của tài khoản khác", f"HTTP {s}, chỉ của mình: {mine}", s == 200 and mine)
    s, b = call("GET", "/notification/v1/notices", token=None)
    record(G, "Xem thông báo khi chưa đăng nhập", "Không gửi token", "Bị từ chối (401)", f"HTTP {s}", s == 401)

if ONLY and os.path.exists(OUT):
    # gộp: thay kết quả của các nhóm vừa chạy, giữ nguyên nhóm khác và thứ tự
    old = json.load(open(OUT))
    groups_run = {r["nhom"] for r in results}
    merged, inserted = [], set()
    for r in old:
        if r["nhom"] in groups_run:
            if r["nhom"] not in inserted:
                merged += [x for x in results if x["nhom"] == r["nhom"]]
                inserted.add(r["nhom"])
        else:
            merged.append(r)
    merged += [x for x in results if x["nhom"] not in inserted]
    results = merged
json.dump(results, open(OUT, "w"), ensure_ascii=False, indent=1)
print(f"\n{sum(r['trang_thai'] == 'Pass' for r in results)}/{len(results)} pass -> {OUT}")

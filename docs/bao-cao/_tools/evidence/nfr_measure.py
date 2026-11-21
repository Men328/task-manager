"""Đo một số chỉ số phi chức năng trên hệ thống chạy thật.
python nfr_measure.py <token_men> <token_second> <out_json>"""
import json, statistics, sys, time, uuid
import requests
A = "http://localhost:3000/api"
MEN = "fec32ca3-1feb-4f48-8ce9-ecda1dfa5286"; SECOND = "bbffa35b-d862-485b-a49f-6e828e10dc76"  # 22010328@st.phenikaa-uni.edu.vn
TM, TB = (open(f).read().strip() for f in sys.argv[1:3])
H = lambda t: {"Authorization": f"Bearer {t}"}
out = {}
def lat(name, url, token=TM, n=30):
    xs = []
    for _ in range(n):
        t0 = time.perf_counter(); r = requests.get(A + url, headers=H(token), timeout=30); xs.append((time.perf_counter() - t0) * 1000)
        assert r.status_code == 200, (url, r.status_code)
    xs.sort(); out[name] = {"p50_ms": round(statistics.median(xs), 1), "p95_ms": round(xs[int(0.95 * n) - 1], 1), "n": n}
lat("Danh sách task (list)", f"/task/v1/tasks?profile_id={MEN}")
lat("Tìm kiếm task", f"/task/v1/tasks?profile_id={MEN}&query=chương")
lat("Báo cáo tổng quan", f"/report/v1/reports/overview?profile_id={MEN}")
lat("Đếm thông báo chưa đọc", "/notification/v1/notices/unread-count")
lat("Lịch sử tin nhắn", f"/message/v1/conversations/{SECOND}/messages?limit=50")
# thời gian từ khi gửi tin tới khi worker lưu xong (Kafka -> worker -> PostgreSQL)
ds = []
for i in range(10):
    body = f"Đo độ trễ lưu tin {i} {uuid.uuid4().hex[:6]}"
    t0 = time.perf_counter()
    requests.post(f"{A}/message/v1/conversations/{SECOND}/messages", headers=H(TM), json={"body": body, "client_id": uuid.uuid4().hex}, timeout=30)
    send_ms = (time.perf_counter() - t0) * 1000
    while True:
        ms = requests.get(f"{A}/message/v1/conversations/{MEN}/messages?limit=20", headers=H(TB), timeout=30).json().get("messages", [])
        if any(m.get("body") == body for m in ms): break
        if time.perf_counter() - t0 > 20: break
        time.sleep(0.05)
    ds.append(((time.perf_counter() - t0) * 1000, send_ms))
out["Gửi tin nhắn (API trả về)"] = {"p50_ms": round(statistics.median(d[1] for d in ds), 1), "max_ms": round(max(d[1] for d in ds), 1), "n": 10}
out["Gửi tin -> đã lưu vào CSDL (qua Kafka, worker)"] = {"p50_ms": round(statistics.median(d[0] for d in ds), 1), "max_ms": round(max(d[0] for d in ds), 1), "n": 10}
json.dump(out, open(sys.argv[3], "w"), ensure_ascii=False, indent=1)
print(json.dumps(out, ensure_ascii=False, indent=1))

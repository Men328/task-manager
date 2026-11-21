"""Cập nhật noi-dung/chuong-5.md từ kết quả kiểm thử thật.

python gen_chuong5.py

Sinh lại: mục 5.2 (phương pháp + bảng kết quả từng nhóm) từ evidence/ket-qua-test.json,
bảng kiểm thử phi chức năng từ evidence/nfr-ket-qua.json, và các con số X/Y ở mục 5.4.
Phần chữ còn lại của chương giữ nguyên.
"""
import json
import re
from pathlib import Path

BASE = Path(__file__).resolve().parent
MD = BASE.parent / "noi-dung" / "chuong-5.md"
R = json.loads((BASE / "evidence" / "ket-qua-test.json").read_text())
N = json.loads((BASE / "evidence" / "nfr-ket-qua.json").read_text())

MAIN_ACCOUNT = "menc1lhp@gmail.com"
SECOND_ACCOUNT = "22010328@st.phenikaa-uni.edu.vn"

groups = []
for r in R:
    if r["nhom"] not in groups:
        groups.append(r["nhom"])
total, passed = len(R), sum(r["trang_thai"] == "Pass" for r in R)


def cell(v):
    return str(v).replace("|", "/").replace("\n", " ")


def lower_first(s):
    return s[0].lower() + s[1:]


L = ["### 5.2.1. Phương pháp kiểm thử", ""]
L.append(
    "Kiểm thử chức năng được thực hiện **trên hệ thống chạy thật** (Docker Compose, cổng `localhost:3000`), "
    "gọi API qua nginx đúng như giao diện. Tài khoản chính là `" + MAIN_ACCOUNT + "`; tài khoản `" + SECOND_ACCOUNT
    + "` (tài khoản thứ hai của cùng sinh viên) chỉ dùng cho các trường hợp cần hai người (kết bạn, nhắn tin, "
    "giao task qua chat). Các trường hợp kiểm thử được viết thành kịch bản tự động "
    "(`docs/bao-cao/_tools/evidence/api_tests.py`): mỗi trường hợp gửi dữ liệu đầu vào, so sánh phản hồi thực tế "
    "(mã HTTP, mã lỗi chuẩn, dữ liệu trả về) với kết quả mong đợi và ghi trạng thái. "
    f"Kết quả: **{passed}/{total} trường hợp đạt**."
)
L += ["", "Riêng chức năng phân loại email được đánh giá trên **email thật** gửi tới hộp thư Gmail của tài khoản chính, "
      "vì luồng này phụ thuộc Gmail và Pub/Sub."]
tbl = 2
for i, g in enumerate(groups, 2):
    rows = [r for r in R if r["nhom"] == g]
    p = sum(r["trang_thai"] == "Pass" for r in rows)
    L += ["", f"### 5.2.{i}. Kiểm thử chức năng {lower_first(g)}", "",
          f"Bảng 5.{tbl} trình bày {len(rows)} trường hợp kiểm thử nhóm {g.lower()}; {p}/{len(rows)} trường hợp đạt.", "",
          f"[BANG: Bảng 5.{tbl}: Kết quả kiểm thử chức năng {lower_first(g)}]",
          "| STT | Chức năng | Dữ liệu đầu vào | Kết quả mong đợi | Kết quả thực tế | Trạng thái |",
          "|---|---|---|---|---|---|"]
    for k, r in enumerate(rows, 1):
        L.append(f"| {k} | {cell(r['chuc_nang'])} | {cell(r['dau_vao'])} | {cell(r['mong_doi'])} | "
                 f"{cell(r['thuc_te'])} | {r['trang_thai']} |")
    tbl += 1
sec52 = "\n".join(L) + "\n\n"

md = MD.read_text(encoding="utf-8")
a, b = md.index("### 5.2.1."), md.index("## 5.3.")
md = md[:a] + sec52 + md[b:]

# bảng phi chức năng: thay các dòng đo độ trễ, giữ các dòng định tính phía sau
m = re.search(r"(\[BANG: Bảng 5\.\d+: Kết quả kiểm thử phi chức năng\]\n\| STT .*\n\|---.*\n)((?:\|.*\n)+)", md)
head, body = m.group(1), m.group(2).splitlines()
qual = [r for r in body if "Độ trễ:" not in r]
new = []
for name, v in N.items():
    res = f"trung vị {v['p50_ms']} ms; " + (f"p95 {v['p95_ms']} ms" if "p95_ms" in v else f"lớn nhất {v['max_ms']} ms")
    new.append(f"| | Độ trễ: {name} | {v['n']} lần gọi | {res} |")
rows = [re.sub(r"^\| *\d* *\|", f"| {i} |", r) for i, r in enumerate(new + qual, 1)]
md = md[:m.start()] + head + "\n".join(rows) + "\n" + md[m.end():]

md = re.sub(r"\d+/\d+ trường hợp kiểm thử chức năng", f"{passed}/{total} trường hợp kiểm thử chức năng", md)
md = re.sub(r"sau khoảng \d+ ms qua Kafka",
            f"sau khoảng {round(N['Gửi tin -> đã lưu vào CSDL (qua Kafka, worker)']['p50_ms'])} ms qua Kafka", md)
MD.write_text(md, encoding="utf-8")
print(f"chuong-5.md: {passed}/{total} pass, {len(groups)} nhóm")

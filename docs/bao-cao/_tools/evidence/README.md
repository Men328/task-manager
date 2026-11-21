# Bằng chứng kiểm thử & ảnh giao diện (Chương 4, 5)

Chạy trên hệ thống đang chạy (`make up`, http://localhost:3000). Tài khoản chính `menc1lhp@gmail.com`,
tài khoản phụ `22010328@st.phenikaa-uni.edu.vn` (tài khoản thứ hai của Mến, chỉ cho case kết bạn / nhắn tin / giao task qua chat).

| Tệp | Việc |
|---|---|
| `mint_token.py` | Tạo session JWT local cho 1 profile (đọc `SESSION_SECRET` trong `deployments/docker/.env`, không in ra). Ghi token vào file **ngoài repo** (scratchpad). |
| `api_tests.py` | 48 test case chức năng qua API thật + tạo dữ liệu mẫu → JSON kết quả. |
| `nfr_measure.py` | Đo độ trễ API, thời gian lưu tin nhắn qua Kafka → `nfr-ket-qua.json`. |
| `shots.cjs` | Chụp 12 màn hình bằng Chromium headless (puppeteer-core) → `docs/bao-cao/hinh/h4-ui-*.png`. |
| `ket-qua-test.json`, `nfr-ket-qua.json` | Kết quả lần chạy 18/11/2026, nguồn sinh `noi-dung/chuong-5.md`. |

```bash
python mint_token.py fec32ca3-1feb-4f48-8ce9-ecda1dfa5286 menc1lhp@gmail.com /tmp/token-men.txt
python mint_token.py bbffa35b-d862-485b-a49f-6e828e10dc76 22010328@st.phenikaa-uni.edu.vn /tmp/token-second.txt
python api_tests.py /tmp/token-men.txt /tmp/token-second.txt ket-qua-test.json   # cần: pip install requests
ONLY=1,8,9 python api_tests.py ...   # chỉ chạy một số nhóm, gộp vào file kết quả cũ
python ../gen_chuong5.py              # cập nhật Chương 5 từ kết quả
```

Chạy lại `api_tests.py` sẽ **tạo thêm dữ liệu mẫu** (task, lịch, sự kiện) vào tài khoản chính.
Puppeteer gọi Chromium bản snap nên `node_modules` phải nằm trong `~/snap/chromium/common/`.

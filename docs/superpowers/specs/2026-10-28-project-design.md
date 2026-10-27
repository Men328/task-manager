# Project: nhóm task / schedule / event + tự detect project

Ngày: 2026-10-28 · Trạng thái: chờ duyệt

## 1. Ý định

Hiện mọi task, schedule, event chỉ thuộc `profile_id` — không có cách nhóm theo
công việc/dự án. Thêm khái niệm **Project**:

- Mỗi task / schedule / event **bắt buộc** thuộc đúng 1 project.
- Item sinh tự động từ email được **tự detect** vào project bằng cách so nội dung
  với **description** của các project; khớp cao nhất (và đủ ngưỡng) thì vào đó.
- Detect có thể sai → mỗi task / schedule / event **đổi được project** trên UI.

## 2. Mục tiêu / Không phải mục tiêu / Phản mục tiêu

**Mục tiêu**
- Bảng project + CRUD (không xoá, chỉ archive).
- Detect project trong luồng email của mail-provider.
- Đổi project trên từng task / schedule / event.
- Project switcher trên TopBar kiểu WorkOS; các trang làm việc trong đúng 1 project.

**Không phải mục tiêu**
- Detect project cho item người dùng tự tạo trên UI (họ tự chọn — bắt buộc).
- Chế độ "Tất cả project" (người dùng quyết định: luôn đứng trong 1 project).
- Status / lifecycle theo project (vẫn dùng chung theo profile).
- Chia sẻ project giữa nhiều người dùng.
- Xoá project.

**Phản mục tiêu** (chạy đúng kỹ thuật nhưng vẫn là thất bại)
- Email không liên quan bị ép vào project có điểm cao nhất dù khớp thấp → làm bẩn
  project thật, người dùng phải đổi tay liên tục. Không khớp đủ → **backlog**.
- Item rơi vào backlog vì không khớp project mà không có đường đưa ra → bị kẹt.

## 3. Ràng buộc từ hệ thống hiện tại

- 1 Postgres, mỗi service 1 schema; FK chéo schema đã có (`identity.profiles`).
- Service gọi nhau qua gRPC; proto ở `common/proto`, sinh code bằng `make gen`.
- Identity đã seed dữ liệu mặc định cho profile mới qua gRPC
  (`task_lifecycle_seeder.go` → `SeedDefaultStatuses`).
- Detect duy nhất nằm ở mail-provider: DeepSeek (`deepseek_client.go`) với luật
  keyword dự phòng (`classifier.go`), kết quả → `MailDraft` → worker gọi
  task/calendar/event/backlog client.
- Report lấy dữ liệu qua gRPC `ListTasks`… chứ không đọc DB trực tiếp.
- Migration mới tiếp theo: `000011`.

## 4. Approaches

| | Cách | Đánh đổi |
|---|---|---|
| **A (đã chọn)** | Service `project` riêng (schema `project`), 3 bảng item thêm `project_id NOT NULL` FK chéo schema. Detect = thêm danh sách project vào prompt DeepSeek hiện có (1 lời gọi). | Thêm 1 service để deploy; đổi lại ranh giới rõ, calendar/event không phụ thuộc task. |
| B | Đặt bảng project trong task service. | Ít hạ tầng hơn, nhưng calendar/event phải gọi task service để kiểm tra project; task service ôm khái niệm chung. |
| (loại) | So khớp bằng embedding. | DeepSeek không có embedding API → thêm nhà cung cấp + pgvector, trong khi LLM call hiện có đã làm được. |

Người dùng chọn **A**.

## 5. Thiết kế đã quyết

### 5.1 Dữ liệu

Migration `000011_init_project`:

```sql
CREATE SCHEMA IF NOT EXISTS project;

CREATE TABLE project.projects (
    id          uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id  uuid         NOT NULL REFERENCES identity.profiles (id) ON DELETE CASCADE,
    name        varchar(100) NOT NULL,
    description text         NOT NULL DEFAULT '',  -- dùng để detect
    color       varchar(9),
    is_default  boolean      NOT NULL DEFAULT false, -- project "Chung"
    is_archived boolean      NOT NULL DEFAULT false,
    created_at, updated_at ... (trigger updated_at như các schema khác),
    CHECK (btrim(name) <> ''), color format check như các bảng khác,
    UNIQUE (id, profile_id)
);
-- mỗi profile đúng 1 project mặc định
CREATE UNIQUE INDEX ... ON project.projects (profile_id) WHERE is_default;
-- tên không trùng trong profile (so không phân biệt hoa thường)
CREATE UNIQUE INDEX ... ON project.projects (profile_id, lower(name));
```

Cùng migration (theo thứ tự):
1. Tạo project "Chung" (`is_default = true`) cho **mọi profile hiện có**.
2. Thêm `project_id uuid` (nullable) vào `task.TASKS`, `calendar.schedules`, `event.events`.
3. Gán mọi row hiện có vào project "Chung" của profile đó.
4. `SET NOT NULL` + FK composite `(project_id, profile_id) → project.projects (id, profile_id)`
   (đảm bảo project cùng profile, theo pattern `fk_tasks_status`), `ON DELETE RESTRICT`.
5. Index `(profile_id, project_id)` trên cả 3 bảng.

Down migration đảo ngược (drop cột, drop schema).

Cập nhật `design/db_schema.dbml`.

### 5.2 Quy tắc nghiệp vụ

- **Không xoá project**, chỉ archive / unarchive. Không có `DeleteProject` RPC.
- Project "Chung": không archive được, không detect vào được; sửa được tên/màu.
- Project archived: không tham gia detect, không hiện trong ô chọn project khi
  tạo/đổi, item bên trong giữ nguyên và vẫn xem được (switcher có mục xem project
  đã archive trong trang quản lý).
- Không gán item vào project archived hoặc project của profile khác
  (service item kiểm tra qua project gRPC trước khi ghi; FK composite chặn khác profile).
- Profile mới: identity gọi `SeedDefaultProject` (giống `SeedDefaultStatuses`).

### 5.3 Service `project`

- Module `taskmanager/service/project`, layout giống các service khác
  (`cmd/grpc`, `cmd/http`, `internal/{config,dependency,handler,model,repository,service}`).
- Cổng **gRPC :9090 / HTTP :8090**; thêm vào docker-compose, Makefile (`run-project`),
  `go.work`, `scripts/run-service.sh`, smoke test, README.
- Proto `common/proto/project/v1/project.proto`, `ProjectService`:
  `CreateProject`, `GetProject`, `ListProjects` (lọc `include_archived`),
  `UpdateProject` (name, description, color), `ArchiveProject`, `UnarchiveProject`,
  `SeedDefaultProject`.
- Mã lỗi mới trong `common/errorcode/error_codes.json` (vd. project không tồn tại,
  project đã archive, không archive project mặc định, trùng tên) + sync sang frontend.

### 5.4 Thay đổi ở task / calendar / event

- Proto: thêm `project_id` vào message item, `Create*Request` (bắt buộc),
  `Update*Request` (optional — đây là cơ chế **đổi project**), `List*Request`
  (lọc theo project).
- Service validate project (tồn tại, cùng profile, chưa archive) qua project gRPC client.
- **Subtask** (`parent_task_id`): subtask luôn cùng project với task cha. Tạo subtask
  thì lấy project của cha; đổi project task cha thì chuyển cả cây con; không đổi
  project riêng một subtask.

### 5.5 Detect trong mail-provider

Luồng mới cho mỗi email:
1. Lấy danh sách project **không archive, không mặc định** của profile (project gRPC).
2. Gọi DeepSeek **1 lần** như hiện tại; prompt thêm danh sách project
   (`id`, `name`, `description`), JSON trả thêm `project_id` và `project_confidence` (0–1).
   `project_id` phải nằm trong danh sách đã gửi, không thì coi như không khớp.
3. Quyết định:
   - `category = other` → backlog (như cũ).
   - task/schedule/event và `project_confidence >= PROJECT_MATCH_THRESHOLD`
     → tạo item với `project_id` đó.
   - task/schedule/event nhưng không khớp đủ (hoặc user không có project nào) → **backlog**,
     `category` = loại đã phân loại, `reason = "no_project_matched"`.
4. **Luật dự phòng** (khi không có DeepSeek): điểm = số từ khoá (token đã chuẩn hoá,
   bỏ stopword) trùng giữa email và `name + description` của project, chuẩn hoá về 0–1;
   cùng ngưỡng.
- `PROJECT_MATCH_THRESHOLD` cấu hình qua env, mặc định 0.6.
- Notice tạo item kèm tên project trong body.

### 5.6 Backlog → item

Thêm thao tác trên trang Backlog: **"Chuyển thành task / lịch / sự kiện"** — form tạo
điền sẵn từ backlog (tiêu đề, mô tả, `category` gợi ý loại), bắt buộc chọn project;
tạo xong đánh dấu backlog `TRIAGED`. Thực hiện ở frontend bằng các API create sẵn có +
`UpdateBacklog`, không cần RPC mới.

### 5.7 Frontend

- **Project switcher** bên trái TopBar, trước ô search, theo flow WorkOS: chấm màu +
  tên project hiện tại + ▾. Dropdown: ô tìm, danh sách project đang hoạt động (✓ ở
  project hiện tại), "+ Tạo project", "Quản lý project".
- Project hiện tại lưu trong Redux slice mới `project` + `localStorage` (try/catch);
  nếu project lưu đã archive/không còn thì về project "Chung".
- **Trang Quản lý project** (route `/projects`): danh sách (kể cả archived, có lọc),
  tạo/sửa tên, màu, description (kèm gợi ý: mô tả rõ phạm vi, người liên quan, từ khoá —
  vì đây là thứ dùng để detect), archive/unarchive.
- Board, Calendar, Events, Report: chỉ dữ liệu của project hiện tại.
- Backlog: không lọc theo project.
- Statuses: không đổi.
- Form tạo/sửa task, schedule, event: ô **Project** bắt buộc, mặc định project hiện tại;
  đổi project trong form sửa. Đổi sang project khác thì item biến khỏi view hiện tại
  (kèm toast có nút "Đến project").
- Bấm notification về item ở project khác: fetch item → chuyển switcher sang project
  của item → mở item.
- i18n vi + en cho mọi chuỗi mới.

### 5.8 Report

Các RPC report nhận thêm `project_id` và truyền xuống `List*` của task/calendar/event.
Số liệu backlog trong report vẫn tính theo cả profile (backlog không có project).

## 6. Quyết định do builder/Claude tự đưa ra (cần người dùng kiểm tra)

- Project "Chung" bị loại khỏi detect; không archive được.
- Hệ quả: user chưa tạo project nào ngoài "Chung" thì **mọi email đều vào backlog**
  (khác hiện tại — đang tự tạo task/lịch/sự kiện).
- Ngưỡng mặc định 0.6, cấu hình qua env.
- Subtask luôn cùng project với cha; đổi cha kéo theo cả cây.
- Thao tác "Chuyển backlog thành item" nằm trong phạm vi.
- Bấm notification tự chuyển project.
- Cổng :9090 / :8090; fields project: name, description, color.
- Tên project không trùng trong 1 profile.

## 7. Để lại cho builder

- Câu chữ prompt DeepSeek và cách chuẩn hoá điểm luật dự phòng.
- Thiết kế chi tiết UI (dùng Mantine + tokens hiện có) và vị trí toast.
- Cách cache danh sách project trong mail-provider (nếu cần).
- Thứ tự triển khai và test, theo quy ước test hiện có của từng service.

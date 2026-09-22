# frontend

React + TypeScript (TSX) + Mantine + Vite. UI dựng theo `design/ui.png` (Kanban dashboard).

## Chạy

```bash
# cần backend đang chạy: make run-identity & make run-task
npm install
npm run dev          # http://localhost:5173
npm run build        # tsc -b && vite build
npm run typecheck    # chỉ type-check
```

Muốn có dữ liệu ngay để xem UI:

```bash
make seed-demo       # tạo 1 profile + 4 status + rule + task mẫu như trong design
```

## Cấu trúc

```
src/
├── main.tsx                    # Provider (redux) + MantineProvider + Notifications + Router
├── theme.ts                    # Mantine theme + color scheme manager + token trỏ CSS var
├── App.tsx                     # routes: / (board), /statuses + bootstrap fetchProfiles
├── types.ts                    # type khớp proto
├── styles/
│   ├── tokens.css              # token màu light/dark (--tm-*) + layout, đổi theo data-mantine-color-scheme
│   ├── global.css              # reset toàn cục (dùng var --tm-*)
│   └── scroll.module.css       # scrollbar mảnh dùng chung
├── i18n/                       # react-i18next: config JSON + resource JSON + hook
│   ├── config.json             # cấu hình i18n (ngôn ngữ, mặc định, detect, storage key)
│   ├── index.ts                # init i18next (nạp mọi locales/*.json + detect + fallback)
│   ├── types.ts                # type hoá config.json (không hardcode giá trị)
│   ├── useLanguage.ts          # hook đọc/đổi ngôn ngữ
│   └── locales/{vi,en}.json    # resource dịch
├── context/                    # Redux Toolkit: store + slice tách theo page
│   ├── store.ts                # configureStore (session, workspace, board, statusesPage)
│   ├── hooks.ts                # useAppDispatch / useAppSelector có type
│   ├── index.ts                # re-export store + hooks + use* của từng page
│   ├── seed.ts                 # seed bộ status mặc định (dùng chung 2 slice)
│   ├── session/                # slice dùng chung: profiles, profileId, query
│   ├── workspace/              # slice dùng chung: danh sách + workspace đang chọn
│   ├── board/                  # slice BoardPage: statuses/transitions/tasks, filter, CRUD
│   └── statuses/               # slice StatusesPage: statuses/transitions/tasks, seed
├── api/
│   ├── client.ts               # fetch wrapper, ApiError, getErrorCode/getErrorMessage, asList/unwrap envelope
│   ├── identity.ts             # /v1/profiles
│   ├── workspace.ts            # /v1/workspaces
│   └── task.ts                 # /v1/statuses, /v1/transitions, /v1/tasks
├── config/
│   └── error_codes.json        # MIRROR bộ mã lỗi chuẩn (sinh bằng `make error-codes`, không sửa tay)
├── lib/
│   ├── errorCatalog.ts         # đọc bộ mã lỗi + map mã -> message theo ngôn ngữ
│   ├── tokens.ts               # nhãn priority, màu status fallback, sort order
│   ├── session.ts              # token ở localStorage (key tm-session-token)
│   ├── workspace.ts            # workspace đang chọn ở localStorage (key tm-workspace-id) + slug
│   └── format.ts               # format ngày, initials, progress suy ra từ subtask
├── components/                 # mỗi component có <Name>.module.css đi kèm
│   ├── layout/                 # AppLayout (shell), TopBar, Sidebar, ThemeSwitcher, LanguageSwitcher
│   ├── workspace/              # WorkspaceSwitcher (select/create/edit) + WorkspaceFormModal
│   ├── board/                  # BoardHeader, KanbanBoard, KanbanColumn, TaskCard,
│   │                           # TaskTable, PriorityBadge, TaskProgress
│   ├── task/TaskFormModal.tsx  # modal tạo task
│   └── common/States.tsx       # ApiErrorAlert / CenteredPanel / LoadingBlock
└── pages/                      # mỗi page = 1 thư mục
    ├── BoardPage/              # index.tsx + BoardPage.module.css
    └── StatusesPage/           # index.tsx + StatusesPage.module.css
```

## State (Redux Toolkit)

- Store gom 4 slice: `session` (dùng chung), `workspace` (dùng chung), `board` (BoardPage),
  `statusesPage` (StatusesPage).
- Component dùng hook theo page: `useSession()`, `useWorkspace()`, `useBoard()`, `useStatuses()`,
  `useSidebarCounts()`. Muốn truy cập thô thì dùng `useAppSelector` / `useAppDispatch`.
- Async qua `createAsyncThunk` (fetch + mutate rồi refetch); derived data memo hoá bằng
  `createSelector` (cardsByStatus, visibleTaskCount, statusById, taskById, selectedWorkspace).

## Workspace switcher (sidebar)

Khối "Công việc của tôi" ở sidebar là workspace switcher (`components/workspace/`):

- **Chọn workspace** bằng `Select` — danh sách lấy từ `GET /v1/workspaces?owner_profile_id=...`.
  Khi chưa có workspace nào, thay cho `Select` là **nút tạo lớn** (kiểu dashed) để bấm vào mở form.
- **Tạo workspace qua modal**: nút `+` (hoặc nút lớn khi chưa có gì) mở `WorkspaceFormModal` để nhập
  tên/mô tả; submit gọi `POST /v1/workspaces` với slug tự sinh (`lib/workspace.ts`), rồi chọn luôn.
- **Sửa workspace**: icon bút chì mở cùng `WorkspaceFormModal` ở chế độ sửa → `PATCH /v1/workspaces/{id}`.
- **Nhớ workspace đang chọn**: id lưu ở localStorage key `tm-workspace-id`
  (`lib/workspace.ts`), khởi tạo lại khi reload; sau khi fetch, nếu id không còn tồn tại thì
  fallback về workspace `is_default` → đầu tiên → `null`.
- **Tạo task kèm workspace**: `TaskFormModal` đọc `readWorkspaceId()` và gửi `workspace_id` trong
  body `POST /v1/tasks` (bắt buộc). Board chỉ fetch khi đã chọn workspace.
- **List task bắt buộc `workspace_id`**: `GET /v1/tasks` trả 400 `TASK_WORKSPACE_ID_REQUIRED` nếu
  thiếu; board/trang Statuses đều truyền workspace đang chọn và tự refetch khi đổi workspace.
  Chưa chọn workspace thì board hiện panel "Chưa có không gian làm việc" thay vì gọi API.

## Styles

- Reset/global ở `src/styles/global.css`.
- Mỗi component/page có `<Name>.module.css`; **không** dùng `style={{}}` cho style tĩnh.
- Token màu/kích thước ở `src/styles/tokens.css` dưới dạng CSS variables `--tm-*` (2 block light/dark);
  `src/theme.ts` chỉ re-export tên token trỏ tới `var(--tm-*)` cho TSX dùng.
- CSS module tham chiếu trực tiếp `var(--tm-*)`. Prop Mantine (`fz`, `c`, `p`, `radius`, ...) vẫn giữ
  nguyên; với `c` truyền `tokens.*` (ví dụ `c={tokens.text}`) nên tự đổi theo theme. Icon Tabler dùng
  `style={{ color: tokens.* }}` (không dùng prop `color`) để `currentColor` ăn theo CSS var.
- Vài giá trị động (màu status từ API, % progress) set qua inline style tối thiểu.

## Đa ngôn ngữ (i18n)

- Dùng **`react-i18next`**. **Toàn bộ cấu hình + resource là JSON**, không hardcode trong TypeScript:
  - `src/i18n/config.json` — danh sách ngôn ngữ (`vi`, `en`), ngôn ngữ mặc định/fallback, thứ tự detect,
    cache và storage key.
  - `src/i18n/locales/<code>.json` — resource dịch.
- `src/i18n/types.ts` chỉ là lớp type hoá mỏng đọc từ `config.json` (không chứa giá trị).
- `src/i18n/index.ts` tự nạp **mọi** file `locales/*.json` bằng `import.meta.glob` và dựng
  `resources` theo tên file, nên không phải sửa code khi thêm ngôn ngữ.
- Khởi tạo một lần ở `src/i18n/index.ts` (được import trong `main.tsx`).
- Thứ tự detect: `?lng=` → `localStorage` (key `tm-language`) → `navigator`; lựa chọn được cache lại
  (tất cả đọc từ `config.json`).
- **Module chuyển ngôn ngữ**: `src/components/layout/LanguageSwitcher.tsx` (gắn ở TopBar) dùng hook
  `useLanguage()`; component dịch text qua `useTranslation()` + `t('key')`.
- **Toàn bộ text UI** đi qua `t()`: nav/sidebar/topbar, board header, kanban, table, form, states, cả 2
  page. Nhãn priority (`priority.*`) và status category (`category.*`) map qua `lib/tokens.ts`.
- `document.title`, meta description và `<html lang>` tự cập nhật theo ngôn ngữ; format ngày dùng
  locale hiện tại (`formatShortDate(value, i18n.language)`).
- Thêm ngôn ngữ: tạo `locales/<code>.json` (copy cấu trúc key từ `en.json`) và thêm một entry vào
  `languages` trong `config.json`. Không cần sửa `index.ts`/`types.ts`.
- Không dịch (cố ý): phím tắt `⌘K`, câu lệnh trong khối `Code`, và dữ liệu do người dùng/DB
  (tên status, email, display name...).
- Test nhanh: mở `/?lng=en` hoặc `/statuses?lng=vi`.

## Design tokens

Nguồn sự thật của màu/kích thước là **`src/styles/tokens.css`** — mỗi token có 2 giá trị (light/dark),
CSS module và TSX dùng chung biến `--tm-*`. Font: **Plus Jakarta Sans** (load từ Google Fonts trong
`index.html`).

| Token | Dùng cho |
|---|---|
| `appBg` / `surface` / `columnBg` | nền app / card / cột kanban |
| `border` / `borderStrong` | viền |
| `text` / `textMuted` / `textFaint` | 3 mức chữ |
| `accentTag` | nhãn category trên card |
| `brand` / `brandLight` / `navActiveBg` / `hoverBg` | màu nhấn, tab/nav active, hover |
| `inputBg` / `tableHeadBg` / `chipBg` / `surfaceAlt` | nền input, header bảng, chip đếm, panel phụ |
| `progressTrack` / `progressFill` / `danger` | thanh tiến độ, chữ/icon cảnh báo |
| `shadowCard` / `shadowCardHover` | đổ bóng card |
| `priority-*-bg` / `priority-*-fg` | màu badge priority (5 mức) |
| `radiusCard` / `radiusPanel` / `sidebarWidth` / `headerHeight` | kích thước shell |

Màu badge priority đọc token `--tm-priority-*` trong `PriorityBadge.module.css`; nhãn ở
`src/lib/tokens.ts` (`PRIORITY_LABEL_KEY`).

## Chế độ sáng / tối (light / dark mode)

- Dùng **color scheme của Mantine**. `MantineProvider` ở `src/main.tsx` nhận
  `defaultColorScheme="auto"` + `colorSchemeManager` (localStorage key `tm-color-scheme`, khai báo ở
  `src/theme.ts`).
- **Module chuyển chế độ**: `src/components/layout/ThemeSwitcher.tsx` (gắn ở TopBar) — 3 lựa chọn
  Sáng / Tối / Theo hệ thống, dịch qua `theme.*` trong `locales/{vi,en}.json`.
- `index.html` có inline script đọc cùng storage key và set `data-mantine-color-scheme` **trước khi
  paint** để tránh nháy màu (FOUC). Đổi `COLOR_SCHEME_STORAGE_KEY` thì phải đổi cả trong `index.html`.
- `tokens.css` khai báo `:root` (mặc định light) và `:root[data-mantine-color-scheme='dark']`; Mantine
  gắn attribute này lên `<html>`, nên mọi `var(--tm-*)` tự đổi mà không cần re-mount component.
- Palette Mantine `dark` trong `src/theme.ts` được chỉnh khớp token dark để input/menu/modal/button
  `variant="default"` đồng bộ với phần CSS module tự style.
- Thêm token mới: khai báo biến ở **cả 2 block** trong `tokens.css`, thêm tên vào `tokens` trong
  `src/theme.ts` nếu cần dùng trong TSX, rồi tham chiếu `var(--tm-*)`.

## Nối API

- Dev: Vite proxy `/api/identity/*` → `:8081`, `/api/task/*` → `:8082`, `/api/workspace/*` → `:8083`
  (bỏ prefix).
- Prod (Docker): nginx proxy y hệt, xem `nginx.conf`.
- Response của gateway là **camelCase** (protojson mặc định); **query param** phải snake_case
  (`?profile_id=...`); body nhận cả `snake_case` (đang dùng) và camelCase.

## Bộ mã lỗi & message hiển thị

- Backend gắn mã lỗi chuẩn vào mọi response lỗi (grpc-gateway trả ở `details[].reason`,
  `@type = google.rpc.ErrorInfo`). FE **không** show `message` thô / HTTP status / URL lên notification.
- `api/client.ts`:
  - `getErrorCode(error)` bóc mã từ `details[].reason` (hoặc field phẳng `error_code`).
  - `getErrorMessage(error)` map mã → message theo ngôn ngữ qua `lib/errorCatalog.ts`; mã lạ thì
    fallback theo HTTP status (`CLIENT_*`), lỗi mạng (`status 0`) → `CLIENT_NETWORK_ERROR`.
    Ở `import.meta.env.DEV`, lỗi thiếu mã được `console.warn` để debug (không lên UI).
- Bộ mã nằm ở `src/config/error_codes.json` — **mirror** của `common/errorcode/error_codes.json`
  (nguồn sự thật ở backend). Sinh lại bằng `make error-codes`; `make check-error-codes` phát hiện lệch.
- Thêm/sửa mã: sửa JSON canonical ở backend (kèm `message.vi` + `message.en`) → `make error-codes`.
  Không cần sửa component: mọi notification/alert đều đi qua `getErrorMessage`.

## Nghiệp vụ đã nối

| UI | API |
|---|---|
| Workspace switcher (chọn) | `GET /v1/workspaces?owner_profile_id=...` |
| Workspace switcher (tạo 1 thao tác) | `POST /v1/workspaces` |
| Workspace switcher (sửa) | `PATCH /v1/workspaces/{id}` |
| Cột kanban | `GET /v1/statuses` (sort theo `position`, màu theo `status.color`) |
| Card | `GET /v1/tasks?workspace_id=...&include_subtasks=true` (chỉ task gốc; workspace bắt buộc) |
| Progress trên card | suy ra từ task con: `done/total` theo `category = DONE` |
| Kéo thả card sang cột khác | `POST /v1/tasks/{id}/status` — **rule allowlist chặn ở backend**, UI hiện notification đỏ nếu bị cấm |
| Tạo task | `POST /v1/tasks` (kèm `workspace_id` từ localStorage) |
| Xoá task (tab Table) | `DELETE /v1/tasks/{id}` |
| Lifecycle | `GET /v1/transitions` |
| Nút "Tạo bộ status mặc định" | `POST /v1/statuses` + `POST /v1/transitions` |

## Chưa có trong base (đã có sẵn chỗ trên UI)

- Tab **List** / **Timeline** và chuông thông báo: chỉ hiển thị theo design, bấm vào có tooltip
  "Chưa có trong base".
- Chưa có field **category/tag**, comment, attachment → card dùng progress từ subtask và due date thay cho
  các con số ảo trong design.
- Chưa có form sửa status/rule (trang Statuses hiện read-only).

## Hướng personal hub (đang đơn giản hoá)

App target **người dùng cá nhân**, không có nhân sự / project / sprint. Menu chỉ giữ những gì phục vụ
việc gom task + quan sát/phân tích:

- **Công việc của tôi** (`/`) — kanban + table.
- **Trạng thái & vòng đời** (`/statuses`) — cấu hình cột cho board.

Đã bỏ: các navlink doanh nghiệp (Department/Employee/Payroll/Schedule/Design/Project Manager/HR/
Development), card **Upgrade PRO**, breadcrumb Dashboard/Project, nút **Invite**, cụm avatar nhóm.
Dự kiến bổ sung sau (dùng lại dữ liệu task hiện có, chưa cần backend mới): **Lịch** (theo `dueAt`/
`startAt`), **Hộp thư/Noti** (quá hạn, đến hạn, task chưa có trạng thái) và **Tổng quan/Insights**
(thống kê theo trạng thái & độ ưu tiên).

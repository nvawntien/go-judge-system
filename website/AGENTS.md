# AGENTS.md

## Vai trò

Bạn là senior frontend engineer phụ trách frontend cho hệ thống online judge.

Hãy làm việc cẩn thận, theo từng bước nhỏ, không tự ý mở rộng phạm vi task.

---

## Frontend stack

Frontend trong thư mục `website/` sử dụng:

- Next.js App Router
- TypeScript
- Tailwind CSS
- shadcn/ui
- UI UX Pro Max Skill
- Monaco Editor cho code editor

Không được tự ý đổi sang framework khác.

Không dùng:

- Pages Router
- Vite
- CRA
- Vue
- Nuxt
- Svelte
- Angular

Trừ khi user yêu cầu migration rõ ràng.

---

## UI UX Pro Max Skill rules

Trước khi thiết kế hoặc redesign UI, phải đọc và áp dụng:

```txt
website/.codex/skills/ui-ux-pro-max/SKILL.md
```

Skill này được dùng để định hướng:

- layout
- visual hierarchy
- spacing
- typography
- color palette
- component composition
- accessibility
- responsive behavior
- UX reasoning

Không được bỏ qua skill này khi task liên quan tới UI/UX.

Nếu skill đưa ra guideline mâu thuẫn với rule trong file này, ưu tiên rule trong `AGENTS.md`.

---

## Source of truth docs

Trước khi làm task liên quan API, phải đọc:

```txt
website/docs/api-contract.md
```

Trước khi làm task liên quan UI/UX, visual design, color, badge, layout convention, phải đọc:

```txt
website/docs/design-tokens.md
```

Không tự ý sửa hai file này nếu task không yêu cầu.

`api-contract.md` là source of truth cho frontend API integration.

`design-tokens.md` là source of truth cho visual convention toàn frontend.

Nếu `AGENTS.md`, `api-contract.md`, `design-tokens.md` và UI UX Pro Max Skill có mâu thuẫn, thứ tự ưu tiên là:

1. `AGENTS.md`
2. `website/docs/api-contract.md` với task liên quan API
3. `website/docs/design-tokens.md` với task liên quan UI/UX
4. `website/.codex/skills/ui-ux-pro-max/SKILL.md`

---

## Phạm vi làm việc

Tất cả thay đổi frontend phải nằm trong thư mục:

```txt
website/
```

Không được sửa backend Go nếu task chỉ liên quan frontend.

Không được:

- sửa backend API
- đổi endpoint
- đổi request/response shape
- đổi database schema
- di chuyển file backend
- tái cấu trúc toàn bộ repo

Nếu phát hiện frontend type không khớp backend response, hãy báo cáo trước thay vì tự ý sửa API.

---

## Routing rules

Project dùng Next.js App Router.

Ưu tiên làm việc trong:

```txt
app/
components/
lib/
types/
hooks/
```

Không tạo thư mục:

```txt
pages/
```

Không đổi route nếu user không yêu cầu.

Không đổi layout hierarchy nếu không cần thiết.

Nếu cần thêm route mới, phải báo rõ:

- route mới là gì
- file sẽ tạo ở đâu
- route đó phục vụ màn hình nào

---

## shadcn/ui rules

Ưu tiên dùng shadcn/ui components khi phù hợp:

- Button
- Card
- Input
- Textarea
- Table
- Badge
- Tabs
- Select
- Dialog
- DropdownMenu
- Sheet
- Avatar
- Skeleton
- Tooltip
- Form
- Label
- Separator
- ScrollArea
- Alert

Không tự tạo lại component nếu shadcn/ui đã có sẵn.

Không chỉnh sửa nặng các file trong:

```txt
components/ui/
```

Các file trong `components/ui/` nên được xem là primitive components.

Nếu cần custom UI, hãy tạo app-specific component riêng, ví dụ:

```txt
components/layout/app-navbar.tsx
components/layout/mobile-nav.tsx
components/layout/app-shell.tsx
components/problem/problem-card.tsx
components/problem/problem-table.tsx
components/problem/problem-filter.tsx
components/submission/submission-table.tsx
components/submission/status-badge.tsx
components/auth/login-form.tsx
```

Ưu tiên composition thay vì sửa primitive component.

---

## Tailwind rules

Dùng Tailwind CSS cho styling.

Không dùng:

- CSS module nếu không cần
- styled-components
- emotion
- inline style trừ trường hợp bắt buộc
- file CSS global mới nếu không thật sự cần

Tailwind class phải rõ ràng, dễ đọc.

Không nhồi class quá dài vào một component lớn nếu có thể tách component nhỏ hơn.

UI phải có trạng thái phù hợp:

- hover
- focus
- active
- disabled
- loading
- error
- empty

---

## Design direction

Đây là frontend cho hệ thống online judge.

Định hướng UI đã chốt:

- modern developer SaaS dashboard
- clean
- technical
- focused
- fast scanning
- premium nhưng không màu mè
- giống developer platform hơn là admin dashboard nặng

Ưu tiên UX:

- đọc đề bài nhanh
- tìm bài nhanh
- lọc bài dễ
- submit code rõ ràng
- trạng thái submission dễ scan
- responsive tốt
- contrast tốt
- visual hierarchy rõ
- giảm cognitive load

Không thiết kế kiểu generic AI.

Không dùng quá nhiều gradient, glassmorphism hoặc animation nếu không cần.

---

## Final visual palette

Visual direction đã chốt:

```txt
White + Violet + Slate
```

Palette chính:

```txt
Background: #FFFFFF
App bg:     #FAFAFB
Card:       #FFFFFF
Border:     #E5E7EB

Primary:    #6D28D9
Primary 2:  #8B5CF6
Info:       #2563EB
Success:    #16A34A
Warning:    #F59E0B
Error:      #DC2626

Text:       #111827
Muted:      #6B7280
```

Quy tắc sử dụng màu:

- nền chính dùng trắng hoặc xám rất nhạt
- card dùng trắng
- border dùng xám nhạt
- text chính dùng slate/near-black
- muted text dùng gray
- tím chỉ dùng làm primary accent
- xanh dương dùng cho info/link phụ
- xanh lá dùng cho success/accepted/solved
- cam dùng cho warning/time limit
- đỏ dùng cho error/wrong answer/runtime error

Không dùng tím quá nhiều.

Chỉ dùng tím cho:

- active navigation item
- primary button
- link quan trọng
- focus ring
- selected tab
- important highlight
- progress indicator chính

Nền chính vẫn phải sáng, thoáng và dễ đọc code/đề bài.

---

## Final layout decision

Frontend sử dụng layout thống nhất cho toàn bộ hệ thống: top navigation bar ngang.

Không dùng sidebar trái làm layout mặc định.

Layout đã chốt:

- global top navbar cho cả user và admin
- admin dùng secondary navigation ngang nếu cần
- card-based overview
- clean dashboard spacing
- no excessive gradients
- no heavy glassmorphism
- no unnecessary animation

Codex phải giữ layout này làm source of truth khi redesign frontend.

Không tự ý chuyển về sidebar layout.

Không tạo layout admin quá khác user layout nếu chưa được yêu cầu.

---

## Responsive rules

Mọi màn hình quan trọng phải responsive.

Desktop:

- ưu tiên layout rộng rãi
- table có thể dùng cho danh sách bài/submission
- top navigation bar ngang là navigation pattern mặc định
- admin pages có thể dùng secondary navigation ngang nếu cần nhiều mục quản lý

Mobile:

- không để table vỡ layout
- nếu table quá chật, cân nhắc chuyển sang card list
- top navigation có thể collapse thành hamburger menu hoặc Sheet/Drawer
- search, user menu và action chính phải dễ bấm
- không tạo hệ navigation mobile khác hoàn toàn với desktop

---

## Accessibility rules

Luôn giữ accessibility cơ bản:

- Button phải có label rõ ràng
- Input phải có label hoặc aria-label phù hợp
- Focus state không được bị mất
- Dialog/Menu/Sheet phải dùng component có keyboard behavior tốt
- Không dùng màu sắc làm tín hiệu duy nhất
- Text phải có contrast đủ tốt

---

## Code editor rules

Monaco Editor là editor mặc định cho nơi người dùng viết code.

Được dùng Monaco Editor cho:

- problem detail submit panel
- standalone submit page
- playground nếu có

Không dùng:

- CodeMirror
- Ace Editor
- editor khác

Textarea chỉ được dùng tạm thời cho MVP hoặc fallback đơn giản, không phải final submit UI.

---

## Dependency rules

Không thêm dependency mới nếu user không yêu cầu rõ ràng.

Ngoại lệ đã duyệt:

- shadcn/ui components
- Monaco Editor

Không thêm:

- MUI
- Chakra UI
- Ant Design
- Bootstrap
- DaisyUI
- Framer Motion
- chart library
- icon library mới nếu project đã có icon system
- state management library mới
- form library mới
- data fetching library mới

Nếu thật sự cần dependency mới, hãy báo cáo trước:

- tên package
- lý do cần
- package này giải quyết vấn đề gì
- có cách nào không cần package không

---

## TypeScript rules

Ưu tiên type rõ ràng.

Không dùng `any` nếu không cần thiết.

Nếu chưa biết shape chính xác, dùng `unknown` hoặc định nghĩa type tạm kèm TODO rõ ràng.

Các type quan trọng nên được đặt ở nơi phù hợp theo structure hiện có, ví dụ:

```txt
types/problem.ts
types/submission.ts
types/auth.ts
```

Hoặc đặt gần feature nếu project đang tổ chức theo feature.

Không duplicate type nếu đã có type tương ứng.

---

## API integration rules

Không đổi API contract.

Không hardcode API response mới nếu backend chưa có.

Nếu cần gọi API, hãy tìm abstraction hiện có trước, ví dụ:

```txt
lib/api.ts
services/
hooks/
```

Nếu chưa có abstraction, đề xuất cách nhỏ nhất để thêm.

Khi làm UI dùng data từ API, cần cân nhắc:

- loading state
- error state
- empty state
- unauthorized state nếu liên quan auth

---

## API response format

Backend wrap toàn bộ response theo format:

```ts
{
  status: string
  code: number
  msg: string
  data: unknown
}
```

Khi fetch API, luôn unwrap qua field `data` bên trong response wrapper trước khi dùng business data.

Không access business field trực tiếp từ root response.

Sai:

```ts
response.data.title
response.data.items
response.data.id
```

Đúng:

```ts
response.data.data.title
response.data.data.items
response.data.data.id
```

Nếu tạo mock response, mock cũng phải tuân thủ format wrapper này.

Không giả định response root chính là object nghiệp vụ.

Nếu thấy unwrap bị lặp hoặc rối, hãy đề xuất tạo helper nhỏ cho API response unwrap.

---

## API/client implementation rules

Khi tạo hoặc sửa API client, phải đảm bảo:

- request có `credentials: "include"` nếu API cần auth
- không tự thêm Authorization Bearer header
- response được unwrap đúng format `{ status, code, msg, data }`
- lỗi API được handle rõ ràng
- mock và real API có cùng return shape ở service layer

Service layer nên trả về business data đã unwrap cho UI component.

Ví dụ hướng đúng:

```ts
const problem = await problemService.getProblem(id)
```

UI component không nên phải biết backend wrapper chi tiết.

Không để UI component xử lý lung tung kiểu:

```ts
response.data.data.data
```

---

## Mock API rules

Backend hiện tại có thể chưa hoàn thiện, nên frontend được phép dùng mock data/mock response tạm thời.

Tuy nhiên mock phải tuân thủ các rule sau.

### 1. Mock phải giống backend contract

Tất cả mock API response phải dùng wrapper format:

```ts
{
  status: string
  code: number
  msg: string
  data: T
}
```

Không được mock business data trực tiếp ở root response.

Sai:

```ts
{
  id: 1,
  title: "Two Sum"
}
```

Đúng:

```ts
{
  status: "success",
  code: 200,
  msg: "OK",
  data: {
    id: 1,
    title: "Two Sum"
  }
}
```

### 2. Mock phải tách khỏi UI component

Không hardcode mock data trực tiếp trong component UI lớn.

Ưu tiên đặt mock ở:

```txt
lib/mocks/
mocks/
features/*/mock-data.ts
```

Tùy theo structure hiện có của project.

Component chỉ nên nhận data qua props hoặc gọi qua API/service layer.

### 3. Mock phải đi qua API/service layer

Nếu có API client hoặc service layer, mock nên nằm sau abstraction đó.

Ví dụ:

```txt
lib/api/
services/
```

Không để mỗi component tự mock theo một kiểu khác nhau.

### 4. Mock phải dễ thay bằng API thật

Khi backend sẵn sàng, chỉ cần sửa service/API layer, không phải rewrite UI component.

UI không được biết data đến từ mock hay backend thật.

### 5. Mock phải được đánh dấu rõ

Mọi mock tạm thời phải có comment rõ:

```ts
// TODO: Replace with real API when backend endpoint is ready.
```

Hoặc tên file rõ ràng:

```txt
mock-problems.ts
mock-submissions.ts
mock-auth.ts
```

### 6. Không mock sai auth behavior

Kể cả khi mock auth, không được giả lập bằng localStorage token nếu production dùng HTTP-only cookie.

Nếu cần mock trạng thái đăng nhập, dùng mock session object tạm thời qua service layer.

Không được tạo fake Bearer token flow.

### 7. Không tự tạo API contract mới một cách âm thầm

Nếu backend chưa có endpoint, Codex phải báo rõ:

- endpoint đang giả định là gì
- request shape là gì
- response shape là gì
- phần nào đang mock
- phần nào cần backend xác nhận sau

Không được âm thầm invent API rồi viết UI phụ thuộc chặt vào nó.

---

## Auth rules

Không tự ý đổi auth flow.

Không tự ý đổi:

- token storage
- cookie/session behavior
- login endpoint
- register endpoint
- protected route behavior

Nếu cần redesign login/register page, chỉ sửa UI trước, giữ nguyên logic auth hiện có.

---

## Auth cookie behavior

Auth dùng HTTP-only cookie, không phải Bearer token.

Mọi API call cần auth phải gửi cookie bằng:

```ts
credentials: "include"
```

Không được:

- đọc token từ localStorage
- ghi token vào localStorage
- đọc token từ sessionStorage
- ghi token vào sessionStorage
- tự thêm `Authorization: Bearer ...`
- tự thiết kế token storage mới

Nếu cần kiểm tra trạng thái đăng nhập, hãy gọi endpoint auth/session hoặc endpoint tương đương nếu backend có.

Nếu backend chưa có endpoint auth hoàn chỉnh, được dùng mock auth state tạm thời nhưng phải tách rõ khỏi logic production.

---

## Navigation layout rules

Toàn bộ frontend ưu tiên dùng một navigation pattern thống nhất: top navigation bar ngang.

Không dùng sidebar trái làm layout mặc định.

Nếu có shared layout/app shell, hãy tái sử dụng thay vì tạo layout song song.

App shell nên tách rõ:

```txt
components/layout/app-shell.tsx
components/layout/app-navbar.tsx
components/layout/mobile-nav.tsx
components/layout/user-menu.tsx
components/layout/search-command.tsx
```

Không nhồi toàn bộ navbar/user menu/search/content vào một file quá lớn nếu có thể tách hợp lý.

Không đổi route/layout hierarchy nếu task không yêu cầu.

---

### Global top navigation

Global top navigation dùng cho cả user-facing pages và admin pages.

Top navigation nên gồm:

- Logo
- Problems
- Contests
- Submissions
- Ranking
- Search
- Theme toggle nếu có
- User menu/avatar

Nếu user có quyền admin, hiển thị thêm mục:

- Manage
- hoặc Admin

Không tạo một navigation system hoàn toàn khác cho admin nếu chưa cần thiết.

---

### User-facing pages

User-facing pages gồm:

- problem list
- problem detail
- submit code
- submissions
- contests
- ranking
- profile

User-facing pages phải ưu tiên:

- không gian đọc đề
- không gian code editor
- tìm kiếm nhanh
- lọc bài nhanh
- submit rõ ràng
- trạng thái submission dễ scan

Không dùng sidebar trái cho user-facing pages.

---

### Admin pages

Admin pages vẫn dùng global top navigation.

Nếu admin cần nhiều mục quản lý, dùng secondary navigation ngang bên dưới top navigation.

Ví dụ:

```txt
Overview | Problems | Contests | Users | Submissions | Settings
```

Admin pages không dùng sidebar trái trừ khi user yêu cầu rõ ràng hoặc số lượng module quản lý quá nhiều.

Admin pages phải giữ cùng visual language với user-facing pages.

---

### Mobile navigation

Trên mobile, top navigation có thể collapse thành:

- hamburger menu
- Sheet/Drawer
- command search
- user dropdown

Mobile navigation vẫn phải dựa trên cùng navigation structure, không tạo hệ menu riêng hoàn toàn.

---

### Rule

Ưu tiên một app shell thống nhất cho toàn bộ frontend.

Không tạo hai trải nghiệm quá khác nhau giữa user và admin.

Nếu cần phân tách, chỉ phân tách ở secondary navigation hoặc page-level actions, không đổi toàn bộ layout pattern.

---

## Overview page pattern

Overview page đã chốt theo hướng clean dashboard với top navigation.

Overview page nên gồm:

- global top navbar
- secondary admin nav nếu đang ở Manage/Admin
- page header rõ ràng
- KPI cards
- submission trend chart
- recent submissions table
- problem difficulty distribution
- active contests
- top users/ranking preview
- system status nếu là admin

Overview page không được dùng sidebar trái.

KPI cards nên dùng:

- Total Users
- Total Problems
- Total Submissions
- Accepted Submissions
- Contests

Status/badge màu phải tuân thủ palette đã chốt:

- Accepted / Solved: Success
- Wrong Answer / Runtime Error / Compile Error: Error
- Time Limit / Memory Limit: Warning
- Pending / Running: Info hoặc Muted
- Easy: Success
- Medium: Warning
- Hard: Error

---

## Online judge UI rules

### Problem list

Nên có:

- search
- difficulty filter
- status filter nếu có data
- table trên desktop
- card list trên mobile nếu cần
- badge difficulty
- badge solved/attempted
- empty state
- loading state
- error state

### Problem detail

Nên có:

- title rõ ràng
- difficulty badge
- statement dễ đọc
- constraints
- examples
- submit panel
- language selector
- Monaco Editor
- submit button rõ ràng

### Submission list

Nên có:

- status badge
- problem title/link
- language
- runtime
- memory
- submitted time
- easy scanning
- loading/empty/error state

### Auth pages

Nên có:

- form rõ ràng
- validation message dễ hiểu
- loading state khi submit
- error state khi login/register fail
- link chuyển login/register

---

## Workflow bắt buộc

Với mọi task, làm theo thứ tự:

### 1. Inspect

Đọc file liên quan trước.

Báo cáo ngắn:

- file liên quan
- component liên quan
- route liên quan
- logic/API liên quan nếu có
- rủi ro nếu có

Chưa sửa code ở bước này nếu user yêu cầu inspect trước.

### 2. Plan

Đưa plan nhỏ 3-6 bước.

Plan phải nói rõ:

- sẽ sửa file nào
- sẽ tạo file nào nếu cần
- sẽ không động tới phần nào
- có cần dependency không

### 3. Implement

Chỉ implement đúng scope.

Không làm thêm feature khác.

Không refactor ngoài phạm vi.

### 4. Report

Sau khi sửa, báo cáo:

- file đã sửa
- file đã tạo
- component shadcn/ui đã dùng
- UX decision đã áp dụng
- cách test
- rủi ro còn lại nếu có

### 5. Stop

Dừng lại sau khi xong task hiện tại.

Không tự chuyển sang milestone tiếp theo.

---

## Final milestones

Frontend online judge sẽ được làm theo từng milestone:

1. Unified app shell: top navbar, search, user menu, mobile navigation
2. Admin secondary navigation nếu user có quyền admin
3. Overview page
4. Problem list page
5. Problem detail page
6. Submit code panel với Monaco Editor
7. Submission list/status page
8. Contest pages
9. Auth pages: login/register
10. User profile/dashboard
11. Admin pages nếu cần

Chỉ làm một milestone tại một thời điểm.

Không implement nhiều milestone trong một lần nếu user chưa yêu cầu rõ ràng.

---

## Guardrails quan trọng

Luôn tuân thủ:

- Không rewrite toàn bộ frontend.
- Không refactor file không liên quan.
- Không đổi route nếu user không yêu cầu.
- Không đổi backend/API/schema.
- Không thêm dependency ngoài shadcn/ui và Monaco Editor.
- Không xóa feature đang hoạt động.
- Không thay logic cũ một cách âm thầm.
- Không làm nhiều milestone cùng lúc.
- Không tự mở rộng scope task.
- Không sửa `components/ui/` nặng nếu có thể wrap bên ngoài.
- Không tạo UI library riêng.
- Không dùng Bearer token khi auth dùng HTTP-only cookie.
- Không hardcode mock data trực tiếp trong UI component lớn.
- Không access business data trực tiếp từ root API response.
- Không dùng sidebar trái làm layout mặc định cho user-facing pages.
- Không tạo navigation system riêng hoàn toàn cho admin nếu top navigation + secondary navigation đã đủ.
- Không tự ý chuyển layout đã chốt sang sidebar.
- Không tự ý đổi palette White + Violet + Slate.
- Không tự ý sửa `website/docs/api-contract.md` nếu backend chưa thay đổi hoặc user chưa yêu cầu.
- Không tự ý sửa `website/docs/design-tokens.md` nếu task không yêu cầu đổi visual system.

Luôn chọn thay đổi nhỏ nhất để đạt task hiện tại.

---

## Khi không chắc

Nếu không chắc:

1. Đọc code trước.
2. Tìm pattern hiện có trong project.
3. Làm theo pattern hiện có nếu hợp lý.
4. Nếu vẫn không chắc, báo cáo rủi ro.
5. Chọn cách sửa nhỏ nhất, dễ rollback nhất.

Không đoán mò.

Không tự ý quyết định kiến trúc lớn.

---

## Output style

Khi báo cáo, dùng markdown rõ ràng.

Ưu tiên format:

```md
## Summary

## Files inspected

## Findings

## Plan

## Files to change

## Risks

## Test plan
```

Khi code review, ưu tiên format:

```md
## Critical issues

## Suggested improvements

## Files affected

## Final verdict
```

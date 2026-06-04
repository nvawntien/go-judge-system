# Frontend Roadmap

## Purpose

File này dùng để tracking tiến độ frontend.

Codex phải đọc file này trước khi implement milestone mới.

File này giúp tránh:

- làm nhảy milestone
- làm quá scope
- quên trạng thái hiện tại
- tự ý chuyển sang feature chưa được review
- tự ý đánh dấu task là Done khi chưa được xác nhận

File này không thay thế:

- `website/AGENTS.md`
- `website/docs/api-contract.md`
- `website/docs/design-tokens.md`

Thứ tự đọc khi làm frontend:

1. `website/AGENTS.md`
2. `website/docs/api-contract.md`
3. `website/docs/design-tokens.md`
4. `website/docs/frontend-roadmap.md`

---

## Current status

Current milestone:

```txt
Milestone 2 - Problem list page
```

Status:

```txt
Not started
```

Summary:

- Milestone 0 đã xong.
- Milestone 1 đã xong và đã review xong.
- Milestone 2 chưa bắt đầu.
- Các milestone sau chưa bắt đầu.
- Không implement Milestone 2 trong task này. Chỉ cập nhật roadmap.

---

## Current next action

Current task:

```txt
Start Milestone 2: Problem list page
```

Do next:

- prepare to start Milestone 2: Problem list page
- do not implement it in this task
- only update roadmap status in this task

Do not implement Milestone 2 yet.

Milestone 2 is the next milestone, but this task only updates roadmap state.

---

## Milestone status

| # | Milestone | Status | Notes |
|---|---|---|---|
| 0 | Bootstrap frontend foundation | Done | Next.js App Router, TypeScript, Tailwind, shadcn/ui foundation, light/dark tokens |
| 1 | Unified app shell | Done | Top navbar, search UI, theme toggle, user placeholder, active nav, and mobile nav polish đã xong; review passed |
| 2 | Problem list page | Not started | Chưa làm |
| 3 | Problem detail page | Not started | Chưa làm |
| 4 | Submit code panel with Monaco Editor | Not started | Chưa làm |
| 5 | Submission list/status page | Not started | Chưa làm |
| 6 | Contest pages | Not started | Backend chưa có contest endpoint; chỉ mock nếu user yêu cầu |
| 7 | Auth pages | Not started | Auth dùng HTTP-only cookie |
| 8 | User profile/dashboard | Not started | Profile endpoints đang ở `Unknown / needs confirmation` |
| 9 | Admin pages | Not started | Dùng secondary top navigation nếu cần, không dùng sidebar mặc định |

---

## Milestone 0: Bootstrap frontend foundation

Status:

```txt
Done
```

Completed:

- Created frontend app inside `website/`.
- Added Next.js App Router foundation.
- Added TypeScript foundation.
- Added Tailwind/shadcn styling foundation.
- Added light/dark theme tokens foundation.
- Added placeholder landing page.
- Confirmed no default sidebar layout.

Rules:

- Do not redo this milestone unless setup is broken.
- Do not recreate the Next.js app from scratch.
- If setup issue exists, fix the smallest broken part.

---

## Milestone 1: Unified app shell

Status:

```txt
Done
```

Goal:

Create a unified top navigation shell for the entire app.

Requirements:

- top navbar ngang
- logo
- nav items:
  - Problems
  - Contests
  - Submissions
  - Ranking
  - Manage
- global search UI
- theme toggle
- user menu/avatar placeholder
- mobile navigation
- content container
- light/dark theme support
- no sidebar trái

Current progress:

- top navbar exists
- logo exists
- nav items exist
- search UI exists
- theme toggle icon exists
- user placeholder exists
- basic content container exists
- no sidebar
- desktop active nav state đã được polish rõ hơn
- mobile nav active state đã có
- user dropdown placeholder đã được polish
- theme toggle icon alignment đã được sửa
- typecheck passed
- build passed
- no API/backend changes
- no Milestone 2 work started

Still needed:

- none

Definition of done:

- desktop navbar works visually
- mobile navigation works
- active nav state is clear
- theme toggle works or is clearly scoped as placeholder
- user dropdown exists as UI placeholder
- no backend/API logic added
- no sidebar layout
- no feature pages implemented
- no problem list/detail/submission/auth/admin logic added
- review completed

Review result:

- typecheck passed
- build passed
- no sidebar
- no API/backend changes
- no Milestone 2 work started

Do not mark this milestone as `Done` until review passes.

If implementation is complete but review has not passed, set status to:

```txt
Needs review
```

---

## Milestone 2: Problem list page

Status:

```txt
Not started
```

Do not start until Milestone 1 is `Done`.

Goal:

Implement the problem list page.

API contract:

```txt
GET /api/v1/problems
```

Endpoint status:

- public endpoint
- response wrapper `{ status, code, msg, data }`

Query params:

- `page`
- `limit`
- `difficulty`
- `search`

Requirements:

- page header
- search
- difficulty filter
- status filter if data exists
- table on desktop
- card list on mobile if needed
- difficulty badges
- empty/loading/error state
- pagination
- no direct raw API response handling in UI

Rules:

- UI component must not handle raw `ApiResponse<T>` directly.
- API/service layer must unwrap response.
- Mock only through service layer if needed.
- Do not hardcode large mock data inside UI components.
- Do not start this milestone until Milestone 1 is `Done`.

---

## Milestone 3: Problem detail page

Status:

```txt
Not started
```

Do not start until Milestone 2 is `Done`.

Goal:

Implement problem detail page.

API contract:

```txt
GET /api/v1/problems/{slug}
```

Requirements:

- problem header
- difficulty badge
- statement
- examples
- constraints
- hints if available
- split layout prepared for submit panel
- no Monaco integration yet unless Milestone 4 starts

Rules:

- Use slug for user-facing problem detail.
- Do not use admin problem ID route for public problem detail.
- Do not implement submit logic in this milestone.
- Do not start this milestone until Milestone 2 is `Done`.

---

## Milestone 4: Submit code panel with Monaco Editor

Status:

```txt
Not started
```

Do not start until Milestone 3 is `Done`.

Goal:

Implement code submit panel using Monaco Editor.

API contract:

```txt
POST /api/v1/submissions
```

Requirements:

- Monaco Editor
- language selector
- submit button
- submit loading state
- submit error state
- result placeholder if needed
- must send `credentials: "include"`
- do not use Bearer token
- do not store token in `localStorage` or `sessionStorage`

Important API note:

Request body uses:

- `problem_id`
- `problem_name`
- `language`
- `source_code`

`problem_name` may actually be problem slug based on contract notes.

Do not silently assume this; follow `api-contract.md` and mark uncertainty if needed.

Rules:

- Do not start this milestone until Milestone 3 is `Done`.
- Do not add editor library other than approved Monaco Editor.
- Do not implement auth token storage.

---

## Milestone 5: Submission list/status page

Status:

```txt
Not started
```

Do not start until Milestone 4 is `Done`, unless user explicitly asks to build a read-only public submissions page first.

Goal:

Implement submission list/status pages.

API contract:

```txt
GET /api/v1/submissions
GET /api/v1/my/submissions
GET /api/v1/my/submissions/{id}
```

Requirements:

- status badges
- problem link
- language
- runtime/memory
- submitted time
- loading/empty/error state
- detail page or detail panel if needed

Rules:

- Public/global submissions use `GET /api/v1/submissions`.
- My submissions require auth and `credentials: "include"`.
- Status colors must follow `design-tokens.md`.
- UI must not parse raw `ApiResponse<T>` directly.

---

## Milestone 6: Contest pages

Status:

```txt
Not started
```

Backend status:

No contest route found in backend contract.

Rule:

Only mock contest pages if user explicitly asks.

Mock must:

- go through service layer
- be marked clearly
- use the same response wrapper
- not create a fake permanent API contract silently

Do not invent real contest API contract silently.

Possible future pages:

- contest list
- contest detail
- contest ranking
- contest problem set

If user asks for contest UI before backend exists, mark this milestone as:

```txt
Blocked
```

or implement clearly marked mock UI only when explicitly approved.

---

## Milestone 7: Auth pages

Status:

```txt
Not started
```

Goal:

Implement auth pages.

API contract:

```txt
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/logout
POST /api/v1/auth/refresh-token
POST /api/v1/auth/email/verify
POST /api/v1/auth/email/resend-verification
POST /api/v1/auth/password/forgot
POST /api/v1/auth/password/reset
PUT /api/v1/auth/password/change
```

Rules:

- Auth uses HTTP-only cookies.
- Must use `credentials: "include"`.
- Do not use Bearer token.
- Do not store token in `localStorage` or `sessionStorage`.
- Cookie is auth source of truth.
- Login response may include token data, but frontend must not persist it as source of truth.

Pages likely needed:

- login
- register
- verify email
- forgot password
- reset password
- change password

Do not implement this milestone until the user explicitly asks, because auth touches app behavior and protected routing.

---

## Milestone 8: User profile/dashboard

Status:

```txt
Not started
```

Backend status:

Profile endpoints are currently in `Unknown / needs confirmation`.

Unknown endpoints include:

```txt
GET /api/v1/auth/profile
GET /api/v1/auth/profile/{username}
```

Rules:

- Do not implement real profile integration until endpoint is confirmed.
- Mock only if user asks.
- Mock must go through service layer.
- Do not invent new profile contract silently.

If profile endpoint is still unknown when this milestone starts, mark milestone as:

```txt
Blocked
```

or implement mock-only UI if explicitly approved.

---

## Milestone 9: Admin pages

Status:

```txt
Not started
```

Goal:

Implement admin/manage pages.

Requirements:

- use global top navbar
- use secondary horizontal navigation if needed
- do not use sidebar by default
- preserve same visual language as user-facing pages

Possible sections:

- Overview
- Problems
- Contests
- Users
- Submissions
- Settings

Admin API areas:

- admin problems
- admin submissions
- testcase upload
- rejudge submission

Rules:

- Do not use sidebar unless user explicitly requests.
- Do not create a separate admin visual system.
- Admin uses top nav + secondary nav.
- Do not start admin pages until the user explicitly asks or roadmap status is updated.

---

## Working rules

- Codex must read this file before implementing a new milestone.
- Codex must also read:
  - `website/AGENTS.md`
  - `website/docs/api-contract.md`
  - `website/docs/design-tokens.md`
- Do not start a milestone marked `Not started` unless user explicitly asks.
- Do not jump to the next milestone if the current milestone is not `Done`.
- Do not mark a milestone as `Done` without review.
- If implementation is complete but review has not passed, set status to `Needs review`.
- Only update status to `Done` when user asks or review has passed.
- Do not implement multiple milestones in one task.
- Do not update `api-contract.md` when only tracking frontend progress.
- Do not update `design-tokens.md` when only tracking frontend progress.
- Do not update `AGENTS.md` when only tracking frontend progress.
- Keep notes honest and specific.
- Update `Current next action` after each milestone status change.

---

## Status values

Only use these statuses:

```txt
Not started
In progress
Blocked
Needs review
Done
```

Meaning:

- `Not started`: no implementation yet
- `In progress`: currently being worked on
- `Blocked`: cannot continue because dependency, contract, or design decision is missing
- `Needs review`: implementation done but review not completed
- `Done`: implementation completed and reviewed

---

## Change policy

Only update this file when:

- a milestone starts
- a milestone is partially completed
- a milestone needs review
- a milestone is reviewed and done
- a milestone becomes blocked
- user asks to change roadmap
- the current next action changes

Do not use this file as a dumping ground for random notes.

Do not rewrite the full roadmap if only one milestone status changed.

Prefer the smallest accurate update.

---

## Prompting rules

When asking Codex to work on frontend, include:

```md
Đọc và tuân thủ:

- `website/AGENTS.md`
- `website/docs/api-contract.md`
- `website/docs/design-tokens.md`
- `website/docs/frontend-roadmap.md`

Task hiện tại: [task cụ thể]

Chỉ làm đúng milestone hiện tại.
Không nhảy sang milestone tiếp theo.
```

For current state, the next task should be:

```md
Polish Milestone 1: Unified app shell.

Chỉ làm:
- active nav state
- user dropdown placeholder
- mobile navigation
- verify dark mode
- verify responsive behavior
- fix UI typo if any

Không bắt đầu Milestone 2.
```

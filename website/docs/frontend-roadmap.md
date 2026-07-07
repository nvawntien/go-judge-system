# JudgeHub Frontend Roadmap

## Goal

This roadmap breaks the frontend into small, reviewable phases so implementation stays practical, aligned, and easy to validate.

## Phase 0: Rules And Documentation

Goal:
Establish the project rules, architecture boundaries, and collaboration workflow.

Scope:

- Create `website/AGENTS.md`
- Create planning and design documents in `website/docs`
- Lock routing, layout, and agent responsibilities

Files likely touched:

- `website/AGENTS.md`
- `website/docs/product-scope.md`
- `website/docs/frontend-roadmap.md`
- `website/docs/design-system.md`
- `website/docs/codex-workflow.md`
- `website/docs/agent-collaboration.md`

Done criteria:

- Source-of-truth docs exist
- User and admin app boundaries are explicit
- UI work can start without architectural ambiguity

What not to do:

- Do not implement pages
- Do not create components
- Do not add dependencies

## Phase 1: Frontend Skeleton

Goal:
Create the minimum frontend structure needed for implementation.

Scope:

- Create Next.js App Router skeleton
- Create separate user layout and admin layout
- Create route placeholders
- Create a basic shared component structure
- Create the mock data folder structure

Files likely touched:

- `website/app/*`
- `website/components/*`
- `website/lib/mocks/*`
- `website/package.json`
- `website/tsconfig.json`
- `website/tailwind.config.*`

Done criteria:

- User and admin shells are separated
- Placeholder routes exist for planned pages
- Shared folders are ready for future phases

What not to do:

- Do not connect APIs
- Do not build heavy page content
- Do not mix user and admin navigation

## Phase 2: User Problems Page

Goal:
Implement the user-facing Problems page.

Scope:

- Build `/problems`
- Use mock data for filters, cards, tables, tags, or difficulty states
- Keep the page inside the user layout

Files likely touched:

- `website/app/problems/*`
- `website/components/problems/*`
- `website/lib/mocks/*`

Done criteria:

- Problems page is visually coherent
- Data comes from mocks, not inline arrays
- Navbar remains user-only

What not to do:

- Do not add real backend data
- Do not build contest or admin features in the same phase

## Phase 3: User Contests Page

Goal:
Implement the user-facing Contests page.

Scope:

- Build `/contests`
- Support mock contest states and overview cards
- Preserve the user app shell and visual system

Files likely touched:

- `website/app/contests/*`
- `website/components/contests/*`
- `website/lib/mocks/*`

Done criteria:

- Contests page matches JudgeHub direction
- Information hierarchy is clear
- Mock data remains centralized

What not to do:

- Do not add real-time countdown infrastructure beyond static or mock display
- Do not build admin contest management here

## Phase 4: User Submissions Page

Goal:
Implement the user-facing Submissions page.

Scope:

- Build `/submissions`
- Create mock submission list or table
- Support status badges and filtering UI

Files likely touched:

- `website/app/submissions/*`
- `website/components/submissions/*`
- `website/lib/mocks/*`

Done criteria:

- Submission states are readable
- Table or list design remains uncluttered
- Filtering patterns are reusable

What not to do:

- Do not connect live judge results
- Do not implement admin moderation tools

## Phase 5: User Ranking, Discuss, And Profile

Goal:
Implement the remaining user app core pages.

Scope:

- Build `/ranking`
- Build `/discuss`
- Build `/profile`

Files likely touched:

- `website/app/ranking/*`
- `website/app/discuss/*`
- `website/app/profile/*`
- `website/components/ranking/*`
- `website/components/discuss/*`
- `website/components/profile/*`
- `website/lib/mocks/*`

Done criteria:

- User app route set is complete
- Shared user navigation remains consistent
- Page patterns feel related but not repetitive

What not to do:

- Do not add real social or profile persistence features
- Do not blend admin analytics into ranking pages

## Phase 6: Admin Dashboard

Goal:
Implement the main admin landing experience.

Scope:

- Build `/admin`
- Add overview cards, summary widgets, and operational placeholders using mocks
- Use the admin sidebar shell only

Files likely touched:

- `website/app/admin/*`
- `website/components/admin/dashboard/*`
- `website/lib/mocks/*`

Done criteria:

- Admin dashboard is structurally separate from the user app
- Cards and summaries feel operational but not crowded
- No user navbar appears in admin

What not to do:

- Do not implement all admin management pages in the same phase
- Do not connect production monitoring services

## Phase 7: Admin Management Pages

Goal:
Implement the remaining admin management screens.

Scope:

- Build `/admin/users`
- Build `/admin/problems`
- Build `/admin/contests`
- Build `/admin/submissions`
- Build `/admin/leaderboard`
- Build `/admin/groups`
- Build `/admin/system-logs`
- Build `/admin/settings`

Files likely touched:

- `website/app/admin/**/*`
- `website/components/admin/**/*`
- `website/lib/mocks/*`

Done criteria:

- Admin route map is complete
- Sidebar information architecture is consistent
- Management UI remains usable and visually clean

What not to do:

- Do not reuse user page shells
- Do not introduce real permission systems or backend wiring

## Phase 8: Refactor, Accessibility, Responsive Polish, And Testing

Goal:
Stabilize the frontend after core screens exist.

Scope:

- Refactor oversized files
- Improve accessibility
- Tune responsive behavior
- Add or run lint, typecheck, build, and any available tests

Files likely touched:

- Any frontend files affected by cleanup
- Validation or test-related configuration if already part of the project

Done criteria:

- Core screens are cleaner and more maintainable
- Responsive and accessibility issues are reduced
- Available checks pass

What not to do:

- Do not start new feature work
- Do not add speculative abstractions that are not required by discovered issues

## Phase Workflow For Every Phase

1. Codex reads `website/AGENTS.md`, `website/docs/*`, and `website/.agent/skills/ui-ux-pro-max/SKILL.md`.
2. Codex writes a small implementation plan.
3. Codex writes a focused prompt for Gemini Antigravity.
4. Gemini Antigravity reads the same documents and the UI skill.
5. Gemini Antigravity implements only the approved scope.
6. Gemini Antigravity runs available checks.
7. Codex reviews for architecture, design consistency, code quality, and security.
8. Refactor if needed.
9. Commit when stable.

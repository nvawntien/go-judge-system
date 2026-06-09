# JudgeHub Frontend Agents Guide

## Project Identity

- Product name: JudgeHub
- Area covered by this guide: frontend workspace under `website/`
- Product type: modern Online Judge platform
- Frontend split: User App and Admin App

## Hard Rules

- Do not implement backend integrations in early frontend phases.
- Do not add dependencies unless the change request clearly requires them and the reason is documented.
- Do not build large UI features unless the active phase explicitly asks for implementation.
- Keep User App and Admin App structurally separate.
- Never place admin navigation into the user-facing navbar.
- Never reuse the user top navbar inside the admin area.
- Admin routes only live under `/admin`.
- After meaningful tasks, run lint, typecheck, and build if available.
- If a command does not exist, report that clearly and never pretend it succeeded.

## Communication And Documentation Language Rules

- All project documentation in `website/AGENTS.md` and `website/docs/*` must be written in English.
- When communicating with the user in chat, always use Vietnamese.
- Code identifiers, component names, route names, and technical terms may remain in English.

## AI Agent Roles

- Codex: Planner, Architect, Reviewer.
- Gemini Antigravity: Frontend Implementer, UI Builder.
- Codex owns scope control, architecture review, prompts, and implementation review.
- Gemini owns page building, components, spacing, typography, responsive behavior, accessibility details, and polish.

## UI/UX Skill Requirement

- Dedicated UI/UX skill path: `website/.agent/skills/ui-ux-pro-max/SKILL.md`
- Any UI, layout, component, spacing, typography, responsive, accessibility, or polish task must start by reading that skill.
- Every implementation prompt for Gemini Antigravity must explicitly instruct it to read this skill before coding.

## Preferred Stack

- Next.js with App Router
- TypeScript
- Tailwind CSS
- shadcn/ui-style component patterns when appropriate
- Lucide React icons only if already available or clearly justified

## Routing Rules

- User App routes:
  - `/`
  - `/problems`
  - `/contests`
  - `/submissions`
  - `/ranking`
  - `/discuss`
  - `/profile`
- Admin App routes:
  - `/admin`
  - `/admin/users`
  - `/admin/problems`
  - `/admin/contests`
  - `/admin/submissions`
  - `/admin/leaderboard`
  - `/admin/groups`
  - `/admin/system-logs`
  - `/admin/settings`
- `/admin` must behave as a separate application shell.

## Layout Rules

- User App uses a horizontal top navigation.
- User navbar items: Problems, Contests, Submissions, Ranking, Discuss.
- User navbar must never include Manage, Admin, or any admin-only entry.
- User navbar right section may include search, theme toggle, notifications, and user avatar menu.
- Admin App uses a vertical left sidebar.
- Admin sidebar items: Dashboard, Users, Problems, Contests, Submissions, Leaderboard, Groups, System Logs, Settings.

## Design System Summary

- Brand: JudgeHub
- Primary color: purple
- Background: white or very light lavender
- Surface style: white cards, rounded corners, thin borders, subtle shadows
- Product tone: modern SaaS dashboard
- Density: spacious and clean, not crowded
- Typography: modern, readable, light visual feel
- Default approach: desktop-first while remaining mobile-safe
- Do not change the brand direction without explicit instruction.

## Component Rules

- Prefer small, focused, reusable components.
- Prefer named exports.
- Page files should mostly compose layouts and sections, not hold large data or heavy logic.
- Avoid monolithic files.
- Reuse primitives and layout shells before creating new patterns.

## Mock Data Rules

- Store mock data in `website/lib/mocks`.
- Do not place large mock arrays directly inside page components.
- Use mock data only for early frontend phases.
- Keep mock shapes close to the intended product model, but do not over-engineer API layers yet.

## Code Quality Rules

- Keep files readable and reasonably short.
- Avoid duplication across user and admin features.
- Refactor when components or pages start carrying mixed responsibilities.
- Leave clear architecture for future API integration, but do not build unused abstractions early.

## Testing And Validation

- Run available lint, typecheck, and build checks for implementation phases.
- Review responsive behavior, empty states, and navigation consistency.
- Validate accessibility basics such as semantic structure, visible focus, color contrast, and keyboard reachability.

## Security Rules

- Never hardcode secrets, tokens, API keys, or credentials.
- Do not expose admin controls in user-facing navigation.
- Avoid fake authentication logic that could be mistaken for real security.
- Do not connect backend APIs unless explicitly requested.

## Git Workflow

- Work in small, phase-based increments.
- Keep prompts, implementation, review, and refactor steps explicit.
- Commit only when the phase is stable and reviewed.
- Do not mix unrelated cleanup into feature-phase changes.

## Codex Behavior

- Read `website/AGENTS.md`, `website/docs/*`, and the UI/UX skill before planning a frontend phase.
- Break work into small phases with clear outcomes.
- Write implementation prompts for Gemini Antigravity.
- Review architecture, routing, design consistency, code size, and mock data placement after implementation.
- Request refactors when files become too large, duplicated, or structurally mixed.
- Avoid implementing large UI tasks unless explicitly asked.

## Gemini Antigravity Behavior

- Read `website/AGENTS.md`, `website/docs/*`, and `website/.agent/skills/ui-ux-pro-max/SKILL.md` before each UI task.
- Implement only the requested phase scope.
- Build layouts and components with attention to spacing, hierarchy, polish, and responsive behavior.
- Run available checks and fix issues before handoff.
- Keep code aligned with routing, layout, and mock-data rules.

## Current Phase Priorities

- Establish the frontend product scope and architecture.
- Lock the route map and layout separation between User App and Admin App.
- Define the shared design direction and component rules.
- Define the Codex and Gemini collaboration workflow.
- Prepare future implementation prompts without building real UI yet.

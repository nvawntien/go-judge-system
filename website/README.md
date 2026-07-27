# AstraCode — website

Front-end for the go-judge-system, implementing the `AstraCode.dc.html` Claude Design
prototype against the real services behind the KrakenD gateway.

- **Next.js 15** (App Router) + **React 19** + **TypeScript**
- No CSS framework: the prototype is inline styles over CSS custom properties, so the
  design tokens are ported verbatim into [`src/app/globals.css`](src/app/globals.css)
- Session lives in the **HttpOnly cookies** issued by the auth service; every request is
  sent with `credentials: 'include'` and a 401 triggers one `refresh-token` retry

## Running

```bash
# 1. Backend (from the repository root)
docker compose up -d

# 2. Front-end
cd website
cp .env.example .env.local
npm install
npm run dev            # http://localhost:3000
```

> The gateway's CORS policy only allows `http://localhost:3000`
> (`gateway/settings/service.json`), so the dev server must stay on port 3000.

| Script | Purpose |
| --- | --- |
| `npm run dev` | Dev server on :3000 |
| `npm run build` | Production build |
| `npm run start` | Serve the production build (run `npm run build` first) |
| `npm run typecheck` | `tsc --noEmit` |

## Environment

| Variable | Default | Notes |
| --- | --- | --- |
| `NEXT_PUBLIC_API_BASE_URL` | `http://localhost:8080` | KrakenD gateway |
| `NEXT_PUBLIC_RUN_ENDPOINT` | `/api/v1/submissions/run` | Custom-test execution; **not implemented yet** — see below |

## Screens → API

| Route | Endpoints |
| --- | --- |
| `/` (dashboard) | `GET /api/v1/me`, `GET /api/v1/me/submissions`, `GET /api/v1/problems` |
| `/problems` | `GET /api/v1/problems`, `GET /api/v1/tags` |
| `/problems/[slug]` (workspace) | `GET /api/v1/problems/{slug}`, `POST /api/v1/submissions`, `GET /api/v1/submissions/{id}` (polled), `GET /api/v1/me/submissions?problem_id=` |
| `/submissions` | `GET /api/v1/me/submissions`, `GET /api/v1/submissions/{id}` (row expansion) |
| `/profile` | `GET /api/v1/me`, `GET /api/v1/me/submissions`, `GET /api/v1/problems` |
| `/u/[username]` | `GET /api/v1/users/{username}/profile` |
| `/settings` | `PATCH /api/v1/me/profile`, `POST /api/v1/me/avatar`, `PUT /api/v1/auth/password/change` |
| `/login` | `POST /api/v1/auth/{login,register}`, `POST /api/v1/auth/password/forgot` |
| `/contests`, `/leaderboard`, `/discuss` | none — no service exists |
| `/design-system` | none — static design notes |

Types in [`src/lib/types.ts`](src/lib/types.ts) mirror the Go DTOs one-for-one; the
gateway's `{status, code, msg, data}` envelope is unwrapped in
[`src/lib/api.ts`](src/lib/api.ts).

## Derived data

The backend has no progress/statistics endpoints, so anything the prototype showed as a
stat is computed client-side from the caller's own submission history plus the problem
catalog ([`src/lib/progress.ts`](src/lib/progress.ts)):

- solved / attempted state on problem rows
- solved-by-difficulty bars, language usage, top topics
- streak, weekly activity, 52-week contribution calendar, milestones

Both feeds are cached in-module (60 s for submissions, 5 min for the catalog) and paged
at 100 items × 10 pages; past that the profile labels the count "last 1000".

## Known gaps vs. the prototype

These are backend limitations, not missing UI work. Nothing is mocked.

| Prototype feature | Status |
| --- | --- |
| **Run** (custom test execution) | No endpoint. The button is wired to `NEXT_PUBLIC_RUN_ENDPOINT` and expects `RunResponse` from `src/lib/types.ts`; until that route exists it reports the gap in the Console tab. **Submit** runs the full judge and works today. |
| Runtime / memory per submission | `GetSubmissionResponse` does not return `execution_time` / `memory_used` even though `entity.Submission` stores them — those columns are omitted rather than faked. |
| Per-test-case verdict breakdown | The judge only returns a single status; the verdict panel shows the status, not a passed/total ladder. |
| Acceptance rate & solver counts on problems | Not in `ProblemResponse`; the table shows time/memory limits instead. |
| Leaderboard, contests, discussions, notifications, editorials | No services — the routes render an explicit "not available" state. |
| Rating history graph | Only the current `rating` scalar is exposed. |
| GitHub / Google sign-in | No OAuth in the auth service; the buttons were dropped. |
| Bookmarks | No endpoint. |

`time_limit` has no documented unit — the admin DTO caps it at 30 (seconds) while seeded
rows store `1000` (milliseconds). `formatTimeLimit()` reads values ≥ 50 as milliseconds.

## Gateway change

`PATCH` was missing from `allow_methods` in `gateway/settings/service.json`, so the
browser preflight for `PATCH /api/v1/me/profile` failed even though the endpoint works.
It has been added; restart the gateway after pulling (`docker compose restart gateway`).

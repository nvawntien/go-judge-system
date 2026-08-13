# Services

## Auth Service

* **Entrypoint:** `services/auth/cmd/server/main.go`; Wire composition in `cmd/server/wire.go`.
* **Responsibility:** registration, email verification/resend, login/refresh/logout/logout-all, password lifecycle, public profile, authenticated profile/avatar, and Auth-owned admin user listing/detail, role assignment and suspension.
* **Internal layout:** domain `User`, email/password value objects and domain errors; auth/user/admin use cases; Gin and internal gRPC handlers; adapters for Postgres, Redis token invalidation, JWT, bcrypt, SMTP and MinIO avatar storage.
* **HTTP:** router is `services/auth/internal/adapter/inbound/http/router.go`; public auth endpoints under `/api/v1/auth`, protected `/api/v1/me`, public username profile and `GET /api/v1/users/search`, and admin `GET /api/v1/admin/users`, `GET /api/v1/admin/users/:user_id`, `PATCH /api/v1/admin/users/:user_id/suspension`, and role assignment routes. Public discovery searches only username/full name and returns only active, unsuspended accounts.
* **gRPC:** internal `PublicUserService.ResolvePublicUser` (`proto/auth/v1/user_internal.proto`) listens on port 9091. It returns only a stable ID and canonical username for active, non-suspended accounts; absent, unverified and suspended accounts are all `NotFound`.
* **Data/dependencies:** owns `auth_db.users`; Redis tracks logout-all token issued-at. Suspending a user, a successful authenticated password change, and a successful password reset record a cutoff before the corresponding database write, so already-issued access and refresh tokens are rejected; unsuspending deliberately leaves that cutoff intact. Password reset consumes its latest one-time Redis token atomically before invalidation/persistence, so a downstream failure does not leave a replayable token. Profile, avatar, and password mutations use column-scoped persistence operations to avoid overwriting unrelated user fields. Avatars go to MinIO bucket `avatars`; SMTP is configured in `config/config.yaml`.
* **Configuration:** server, database, Redis, SMTP, JWT, app frontend URL, MinIO, logger. Environment keys use uppercase dotted-path replacement, e.g. `DATABASE_PASSWORD`, `JWT_ACCESS_SECRET` (`pkg/config/config.go`).

## Problem Service

* **Entrypoint:** `services/problem/cmd/server/main.go`; Wire in `cmd/server/wire.go`.
* **Responsibility:** problem catalogue, tags, hidden/published state, admin authoring, and testcase ZIP metadata/storage.
* **Internal layout:** `domain/entity` contains Problem/TestCase/Tag; application ports and user/admin/worker use cases; Gin and gRPC inbound adapters; GORM and MinIO outbound adapters.
* **HTTP:** public `GET /api/v1/problems`, `GET /api/v1/problems/:slug`, `GET /api/v1/tags`; contributor/moderator protected `/api/v1/my` and `/api/v1/admin` routes; exact service routes in its router.
* **gRPC:** `ProblemService.GetTestCase` and `GetProblem` (`proto/problem/v1/problem.proto`) on configured port 9092. `GetTestCase` is worker-facing and returns a presigned ZIP URL with count/version.
* **Data/dependencies:** owns `problem_db` tables and MinIO bucket `testcases`. Problem create requests require separate description/input/output prose; legacy statement splitting is an explicit dry-run-first `cmd/backfill-problem-formats` command, not service startup. Redis is constructed by the container configuration but no Problem cache usage was found in the application/adapters.

## Submission Service

* **Entrypoint:** `services/submission/cmd/server/main.go`; Wire in `cmd/server/wire.go`.
* **Responsibility:** durable official submission intake, rejudge attempts, submission/result query APIs, interactive run-code coordination, Kafka bridging, and submission SSE.
* **Internal layout:** domain Submission/Attempt/Result/Outbox/Stream types; user/admin/result use cases; Gin/SSE and Kafka inbound adapters; Postgres, outbox relay, gRPC and Kafka outbound adapters.
* **HTTP:** protected create, run, ticket, detail, own list, self-only `GET /api/v1/me/profile-stats`, public `GET /api/v1/users/:username/profile-stats`, and admin list/detail/rejudge routes; unauthenticated service SSE route `/events/submissions/:submission_id` validates a short-lived signed ticket. Profile stats are authoritative PostgreSQL aggregates from `submission_db`, not a paginated-history/frontend calculation. See router for exact paths.
* **gRPC clients:** Auth `ResolvePublicUser` for public profile-stat visibility/identity; Problem `GetProblem`; Judge Worker `RunCode` for interactive execution.
* **Background work:** outbox polling and result consumer start in `internal/container/app.go`.
* **Data/dependencies:** owns `submission_db` tables; produces jobs and consumes results; list/detail result summaries are scoped to each submission's `current_attempt_id`, while historical attempt rows remain retained. The local `EventHub` is an in-process SSE fan-out, not a cross-instance bus. Redis is configured but no Submission Redis adapter was found.

## Judge Worker

* **Entrypoint:** `workers/judge/cmd/server/main.go`; Wire in `cmd/server/wire.go`.
* **Responsibility:** consume official judge jobs, obtain official testcases, invoke go-judge, publish results; expose synchronous `RunCode` for Submission Service.
* **Internal layout:** judge use cases; Kafka and gRPC inbound adapters; gRPC Problem metadata client, HTTP go-judge client, Kafka result producer, and filesystem testcase loader outbound adapters.
* **Kafka:** consumes `judge.submission.jobs` as group `judge-worker-v1`; publishes results and DLT messages.
* **gRPC:** `JudgeService.RunCode` from `proto/judge/v1/run.proto`, configured port 9093.
* **Storage:** no database. It keeps a Docker-volume cache at `/cache/testcases`, validates ZIP SHA-256, then uses a read-only mounted cache from the sandbox.
* **Configuration:** Kafka, Problem gRPC, judging timeout/memory/concurrency and `run_code` output limits. `WORKER_POOL_SIZE` overrides the Kafka claim pool; `GO_JUDGE_URL` is read by the provider configuration (verify when changing deployment settings).

## Edge and frontend

`envoy/envoy.yaml` exposes Envoy on 8080 in front of KrakenD. Gateway definitions in `gateway/settings/` do JWT validation and pass `X-User-ID`, `X-Username`, `X-Role`, and `X-Token-Iat` downstream. `website/` is a Next.js application with API/SSE client code under `website/src/lib/`; it is not part of the Go workspace and its `.next`/`node_modules` are build/generated directories.

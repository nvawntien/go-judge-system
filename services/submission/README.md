# Go Judge System - Submission Service

The Submission Service owns authenticated submission creation and detail reads,
along with the infrastructure required to deliver judge jobs reliably.

## Active runtime components

- Gin HTTP server on port `8083`
- `GET /health`
- authenticated `POST /api/v1/submissions`
- authenticated `GET /api/v1/submissions/{submission_id}`
- authenticated `GET /api/v1/me/submissions`
- authenticated `GET /api/v1/admin/submissions`
- PostgreSQL connection
- shared Zap logger and HTTP logging/recovery middleware
- shared authentication middleware wiring, including Redis-backed logout-all
  token invalidation, ready for future protected routes
- transactional outbox repository and Kafka relay

The judge-result consumer is intentionally disconnected until its application
use case is rebuilt. Existing domain entities, PostgreSQL repositories,
transaction manager, outbox publisher, and shared judge contracts remain
available for future submission flows.

## Submission detail

`GET /api/v1/submissions/{submission_id}` requires authentication.
`submission_id` is the unique identifier of the Submission. Users and
contributors may read their own submissions; moderators and administrators may
read any submission. Requests for another user's inaccessible submission return
the same `404 Not Found` response as a missing submission.

The response includes the complete stored source code and `problem_title`, the
canonical Problem title captured as a snapshot when the Submission was created.
The endpoint reads only from Submission Service storage and does not call
Problem Service.

## My submissions

`GET /api/v1/me/submissions` returns only the authenticated actor's own
Submission history. Users, contributors, moderators, and administrators share
the same owner-only behavior on this endpoint.

Supported query parameters:

- `page`, default `1`;
- `limit`, default `20`, maximum `100`;
- `status`;
- `language`;
- `problem_id`.

Results are ordered by `created_at DESC, id DESC`. Each item is compact and
contains `id`, `problem_id`, `problem_title`, `language`, `status`, and
`created_at`; source code is not included. `problem_title` is the historical
Problem title snapshot stored with the Submission. Full source code remains
available through `GET /api/v1/submissions/{submission_id}`.

## Admin submissions

`GET /api/v1/admin/submissions` lists Submissions across the whole system.
Authentication is required, and only moderators and administrators are allowed.
Users and contributors receive `403 Forbidden`.

Supported query parameters:

- `page`, default `1`;
- `limit`, default `20`, maximum `100`;
- `status`;
- `language`;
- `problem_id`;
- `user_id`.

Results are ordered by `created_at DESC, id DESC`. Each item contains `id`,
`problem_id`, `problem_title`, `user_id`, `username`, `language`, `status`, and
`created_at`; source code is not included. `user_id` is the canonical stable
identity for backend operations. `username` is the Submission-time snapshot for
display and may differ from the user's current username after an account
rename.

The list endpoint reads stored snapshots only: `problem_title` comes from
`Submission.ProblemName`, and `username` comes from `Submission.Username`.
It does not call Problem Service or Auth Service. Full source code remains
available through `GET /api/v1/submissions/{submission_id}`. Username filtering
is intentionally deferred; this first admin list filters users by canonical
`user_id` only.

## Configuration

The service loads `/app/config/config.yaml` through `pkg/config`. Sensitive
database and Redis values are supplied through environment overrides.

## Validation

From this module:

```bash
go test ./...
go vet ./...
```

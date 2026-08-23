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
- Kafka judge-result consumer for `judge.submission.results`

## Judge attempt correlation

Every newly created Submission stores an internal `current_attempt_id`. The same
value is written into the judge job outbox payload before the create transaction
commits. The outbox relay publishes that persisted payload unchanged, so outbox
retries preserve the exact attempt ID instead of generating a new one.

Judge results also carry an internal `attempt_id`. The Submission Service result
consumer locks the Submission row, compares the incoming attempt against
`current_attempt_id`, and only applies matching results. Matching results update
the Submission terminal status and replace testcase result rows atomically in the
same database transaction.

Stale results whose attempt ID no longer matches are acknowledged and ignored:
they are not retried, not sent to DLT, and do not write to the database. Legacy
Submissions with an empty `current_attempt_id` are intentionally treated as
unverifiable and also ignored; backfill or cleanup for those rows should be done
separately before relying on historical result replay.

## Rejudge single-flight

Only moderators and administrators may initiate a rejudge. The rejudge
transaction locks the target Submission row and permits a new attempt only when
its current status is terminal. A request while that Submission is `PENDING` or
`JUDGING` returns `409 Conflict`; it creates no replacement attempt or outbox
job. This is scoped to the Submission itself, so it applies across moderators
and service replicas without delaying rejudge requests for different
Submissions.

Duplicate results for the current attempt are safe: processing repeats the same
deterministic replacement for testcase rows and converges on the same Submission
status/result snapshot. Invalid/malformed result messages are non-retryable and
are forwarded to the DLT/drop policy by the Kafka adapter.

Attempt IDs are internal transport/storage fields only. Public HTTP request and
response DTOs do not expose them. Rejudge APIs are protected separately and are
documented below.

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

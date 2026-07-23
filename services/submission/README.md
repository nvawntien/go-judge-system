# Go Judge System - Submission Service

The Submission Service owns authenticated submission creation and detail reads,
along with the infrastructure required to deliver judge jobs reliably.

## Active runtime components

- Gin HTTP server on port `8083`
- `GET /health`
- authenticated `POST /api/v1/submissions`
- authenticated `GET /api/v1/submissions/{submission_id}`
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

## Configuration

The service loads `/app/config/config.yaml` through `pkg/config`. Sensitive
database and Redis values are supplied through environment overrides.

## Validation

From this module:

```bash
go test ./...
go vet ./...
```

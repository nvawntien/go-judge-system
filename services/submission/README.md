# Go Judge System - Submission Service

The Submission Service is currently a minimal infrastructure baseline. Its public
HTTP surface contains only the health check while submission APIs are rebuilt in
later phases.

## Active runtime components

- Gin HTTP server on port `8083`
- `GET /health`
- PostgreSQL connection
- shared Zap logger and HTTP logging/recovery middleware
- shared authentication middleware wiring, including Redis-backed logout-all
  token invalidation, ready for future protected routes
- transactional outbox repository and Kafka relay

The judge-result consumer is intentionally disconnected until its application
use case is rebuilt. Existing domain entities, PostgreSQL repositories,
transaction manager, outbox publisher, and shared judge contracts remain
available for future submission flows.

## Configuration

The service loads `/app/config/config.yaml` through `pkg/config`. Sensitive
database and Redis values are supplied through environment overrides.

## Validation

From this module:

```bash
go test ./...
go vet ./...
```

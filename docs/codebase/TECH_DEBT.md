# Architecture risks and technical debt

## Confirmed from code/configuration

| Area | Finding | Evidence / impact |
| --- | --- | --- |
| Schema change management | No versioned SQL migrations or migration command; repositories call `AutoMigrate` at runtime. | `infra/postgres/init.sql`; repository constructors. Production schema rollout/reversibility is not explicit. |
| Worker capacity scaling | Compose intentionally limits each worker to one concurrent official job and 512 MiB after executor memory-pressure remediation. | `docker-compose.yml`, `judge_job_consumer.go`; any throughput increase needs a fresh memory/concurrency measurement rather than raising `WORKER_POOL_SIZE` independently. |
| SSE scale boundary | Event delivery uses an in-memory `EventHub`; no shared pub/sub adapter is present. | `services/submission/internal/adapter/outbound/stream/event_hub.go`; multi-replica client delivery is not guaranteed. |
| DLT recovery | Worker produces a DLT topic but no DLT consumer/replay tool was found. | `infra/kafka/kafka-init.sh`, `dlt_publisher.go`. Failed jobs need manual/operator recovery. |
| Gateway/API drift risk | Gateway route JSON and service routers are separate configuration/code surfaces. | `gateway/settings/`, all three router files. A local handler route can be unavailable publicly or gateway config can drift. |
| Redis ownership | Problem and Submission receive Redis configuration, but no active cache adapters were found. | Wire/config versus adapter tree. This increases operational surface without demonstrated use. |
| User-suspension audit trail | Administrative suspension is a current boolean state with session invalidation, but there is no persisted reason, actor, expiry, or state-change history. | `auth_db.users`, Auth admin user use case. Operational moderation review and reversibility need a dedicated audit model if required. |
| Session cutoff resolution | Logout-all, suspension, authenticated password-change, and password-reset invalidation use Unix-second JWT `iat`; a fresh login in the same second waits cancelably for the next second. | `session_invalidation.go`; correct but can add nearly one second of authentication latency. |
| Local developer ergonomics | Auth/Problem/Submission main programs hard-code `/app/config`; no root task runner exists. | their `cmd/server/main.go`, absence of Makefile. |
| Legacy Problem statement normalization | The explicit backfill only recognizes unambiguous standalone Input/Output headings. Rows using a different authoring pattern are safely skipped and need editorial review. | `services/problem/cmd/backfill-problem-formats`, `internal/application/backfill/problem_formats.go` |
| Sandbox privilege | Compose runs the go-judge container with `privileged: true`. | `docker-compose.yml`; this is a high-value host isolation boundary requiring deployment-specific review. |

## Potential concerns — require verification before remediation

* **At-least-once duplicate behavior:** Job producer, consumer, and PostgreSQL are not a single transaction. Attempt IDs and outbox/result guards exist, but audit live failure/rebalance behavior before asserting exactly-once guarantees.
* **Testcase cache concurrency:** cache promotion has no cross-process lock. It is likely safe for one Compose worker but needs testing before horizontally scaling shared cache volumes.
* **Official data at rest:** worker sanitizes result messages, but `submission_results` has input/expected-output fields and interactive calls can populate output. Confirm data-retention/access-control policy before changing schemas or APIs.
* **Transport security:** internal gRPC clients are configured with explicit insecure credentials in provider wiring for the Compose network. Verify TLS/mTLS requirements outside local Docker before production deployment.
* **Official-testcase gRPC boundary:** `ProblemService.GetTestCase` returns a presigned official-testcase ZIP URL to an unauthenticated internal gRPC caller. It is not exposed through the public gateway, but therefore relies entirely on network isolation; add authenticated service identity/authorization before allowing untrusted workloads on that network.
* **Auth trust boundary:** Envoy removes client-supplied `X-User-ID`, `X-Username`, `X-Role`, and `X-Token-Iat`; KrakenD validates the access JWT and recreates them before services install typed claims. Services must remain inaccessible to untrusted networks because this is a network-level trust boundary, not cryptographic header authentication. Direct service tests must not be treated as production authentication. Consider mTLS or signed service identity only if this network boundary can no longer be maintained.
* **Executor hardening:** input/output limits, process limits, tempfs and language environments are configured, but container/kernel isolation policy depends on the executor image/runtime. Security review of the actual deployment host is needed; do not infer safety from application code alone.
* **Observability:** Zap request logs and Kafka logs are present, but no tracing provider, metrics endpoint, or distributed correlation ID propagation was found. Validate any platform-level telemetry not stored in this repository.
* **Readme staleness:** root README describes folders such as `api-tests/` and test fixtures that are absent from the checked-out tree. Treat code and Compose as authoritative.

## Testing gaps observed

Unit tests cover many use cases/adapters, including submission result application, Kafka consumers, gRPC clients and testcase loading. No checked-in end-to-end Compose test, DLT replay test, multi-replica SSE test, schema migration test, or Kubernetes deployment test was found. The absence is based on repository discovery, not a claim that external CI/system tests do not exist.

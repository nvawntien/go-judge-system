# Data and ownership

## PostgreSQL

Compose starts one PostgreSQL server but initialization creates three isolated databases: `auth_db`, `problem_db`, and `submission_db` (`infra/postgres/init.sql`). Each service config points at its own database. No service code was found directly accessing another service's database; cross-service reads are gRPC calls.

There are no checked-in SQL schema migrations or migration command. Repositories call GORM `AutoMigrate` at construction time. This is a confirmed schema-management design fact, not an assertion that production migration succeeds safely.

| Owner | Tables/models | Notable indexes/relationships | Evidence |
| --- | --- | --- | --- |
| Auth | `users` | primary UUID; unique username/email | `services/auth/.../postgres/user.go` |
| Problem | `problems`, `tags`, `problem_tags`, `test_cases` | unique problem slug; author/deleted indexes; unique tag slug; join uniqueness; one testcase row per `problem_id` | `services/problem/.../postgres/{problem,tag,testcase}.go` |
| Submission | `submissions`, `submission_attempts`, `submission_results`, `outbox_messages` | submission filters; unique attempt ID; attempt compound lookup/order indexes; outbox aggregate/status indexes | `services/submission/.../postgres/` |

Problem persistence stores examples, constraints and hints as PostgreSQL JSONB. Problem/Tag deletion is soft deletion by GORM model fields; testcase storage uses a physical row keyed by problem. Submission source code and optional testcase outputs are text columns; this has storage and sensitive-data implications described in [TECH_DEBT.md](TECH_DEBT.md).

### Transaction boundaries

* Problem create/update wraps the problem write and `problem_tags` synchronization in a GORM transaction (`problem.go`).
* Submission creation/rejudge use the transaction manager that injects a GORM transaction into context. The submission/attempt/outbox writes are intended to commit together (`tx_manager.go`, relevant use cases).
* Result application uses transaction-backed repositories to update the terminal submission/attempt and replace matching attempt result rows (`apply_judge_result.go`, `submission_result.go`).
* The outbox relay is intentionally outside the creation transaction: it polls committed pending records, Kafka-publishes, then marks publication/failure (`outbox/relay.go`).

## Redis

Redis is configured globally in Compose with AOF and an allkeys-LRU 128 MB policy. The confirmed application use is Auth's `LogoutAllIATStore`, which lets downstream shared auth middleware reject access tokens issued on or before a logout-all timestamp. Problem and Submission Wire configs receive Redis settings, but no active cache adapter was found under those services.

## MinIO and filesystem cache

Auth owns MinIO avatar objects. Problem owns testcase ZIP objects and creates presigned URLs for workers. The Worker downloads these using a 30-second HTTP client, verifies/cache-checks SHA-256, extracts only numeric `.in`/`.out` files with ZIP size/path/symlink protections, and atomically promotes a per-problem local cache directory. Docker shares that cache volume read-only with the go-judge container (`docker-compose.yml`, `official_loader.go`).

## Kafka

Kafka runs in single-node KRaft mode. `infra/kafka/kafka-init.sh` explicitly creates three topics: jobs (three partitions), results (three), DLT (one). Producers use all acknowledgements and idempotent Sarama settings (`pkg/kafka/sarama.go`). This does not create exactly-once processing across Kafka and PostgreSQL; the outbox and attempt ID mitigate distinct portions of that problem.

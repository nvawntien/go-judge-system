# Concurrency and distributed behavior

## Goroutines and shutdown

* Auth launches the HTTP server in a goroutine and waits on server error or SIGINT/SIGTERM, then uses a 10-second HTTP shutdown context (`services/auth/internal/container/app.go`).
* Problem concurrently launches HTTP and gRPC, stops HTTP on a signal or gRPC error (`services/problem/internal/container/app.go`). The observed code does not call gRPC graceful-stop in the signal path; treat shutdown semantics as a change-sensitive area.
* Submission concurrently runs HTTP, outbox relay and Kafka result consumption from a cancelable worker context. It cancels workers, shuts down HTTP, and closes connections/resources in `Close`.
* Worker concurrently runs Kafka consumption and gRPC. It stops gRPC and waits up to 10 seconds for consumer completion during signal shutdown.

## Kafka processing

The Judge Worker creates a worker pool per Sarama consumer claim, size four by default or `WORKER_POOL_SIZE`; messages are only marked after success or after DLT handling. Processing failures retry three times with 500 ms, 1 s and 2 s waits, then send to DLT (`judge_job_consumer.go`). Cancellation skips retry/commit to permit redelivery.

Submission's result consumer is another consumer-group loop. Broker delivery remains at-least-once: use `AttemptID` and result application guards as the cross-service deduplication boundary; do not treat an idempotent Kafka producer as end-to-end exactly once.

## Timeouts and limits

* gRPC caller deadlines come from Problem/Judge configuration and adapters.
* Worker testcase download uses 30 seconds, limits ZIP to 64 MiB and extracted content to 128 MiB (`official_loader.go`).
* go-judge REST client has a 120-second client timeout; requests set CPU, wall clock, memory, process and output limits. Requests are batched in groups of 50 and official submission execution stops after first failure.
* Interactive run-code has explicit source/stdin/expected-output/testcase/concurrency/timeout limits in Submission config and use case.

## In-process SSE

Submission `EventHub` manages subscriptions/channels in process. It supports live fan-out and heartbeat stream handling but is not backed by Redis or Kafka. Therefore an SSE connection and a result consumer in different service replicas need verification; horizontal scale requires a shared event transport or routing affinity.

## Race/resource observations

The Worker `Close` method writes the shared `closeErr` from multiple goroutines without synchronization (`workers/judge/internal/container/app.go`); this is a confirmed race-risk in the cleanup path. The testcase cache update deletes and renames a per-problem directory without an inter-process lock; concurrent worker replicas sharing one volume are a potential race. Current Compose has one worker container, so multi-replica behavior is **NEEDS VERIFICATION**.

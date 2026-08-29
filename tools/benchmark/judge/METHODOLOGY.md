# AstraCode Judge benchmark methodology

## KPI #2: massive submission admission resilience

The 100K massive-burst mode is a **cardinality** workload: one real public
Submission API POST from each of 100,000 distinct benchmark identities. It is
not 100,000 RPS and it is not a sustained arrival-rate test. The benchmark
records each actual POST start and response completion, then reports
POST-start spread, client launch attempted throughput, effective accepted
intake, Submission API POST latency, and raw 429/other-4xx/5xx/transport/
ambiguous-POST counts.

The primary KPI #2 mode is `--observation-mode realistic`. Each accepted POST
does exactly one public event-ticket POST followed by exactly one public SSE
GET. A stream is established only after HTTP 200 plus `text/event-stream`; it
is held for `--sse-hold-duration` or until a terminal event/stream close. The
mode has no SSE reconnect, GET reconciliation, terminal drain, E2E collection,
or Judge Core claim. It reports Submission, ticket, and SSE stages with their
own denominators. Compilation is outside this KPI.

`--observation-mode admission-only` remains the front-door diagnostic. It ends
each logical submission when its POST returns and never starts event-ticket,
SSE, reconciliation, drain, or E2E collection. Its terminal, Judge Core, E2E,
and pipeline-terminal metrics are **NOT MEASURED**, not zero. Existing full
observation retains its SSE-first terminal behavior.

Both 100K modes need client qualification evidence: RLIMIT soft/hard limits,
mode-specific nofile estimate, observed open-FD samples, runtime CPU/goroutine
counts, active POST/ticket/SSE peaks, connection-reuse/new-connection counters,
HTTP protocol counts, and actual scheduling spread. The realistic estimate is
not a socket count: HTTP/2 may multiplex requests and streams. Local `EMFILE`,
`ENFILE`, ephemeral-port exhaustion, or material scheduling delay emits the
`LOAD_GENERATOR_LIMITED` quality flag and yields `CLIENT_LIMITED` as the
run-level classification; this is not server saturation. Generic TLS,
connection, and deadline failures remain transport failures unless corroborated
by direct load-generator evidence such as FD exhaustion, ephemeral-port
exhaustion, or measured local scheduling/resource collapse. App-node
container/restart and Kafka-lag collectors are external evidence; missing
collector input is unavailable, never zero.

## Canonical latency domains

**Compile overhead** is the wall time spent compiling a submission. It is
reported separately and is never included in Judge Core latency or Judge Core
throughput.

**Judge Core service time** begins only after a successful compile completes
and ends when all required testcase-execution batches complete. It includes
Worker-to-sandbox execution RPCs, testcase stdin setup, sandbox setup,
executable reference handling, execution, output capture, resource accounting,
execution cleanup, batching, and serialization. It excludes compilation, the
Submission API, outbox/Kafka queueing, Problem Service, testcase download,
terminal persistence/SSE, and client observation delay.

**Pipeline terminal observation** is a different measure: a terminal result
observed by this harness crosses queueing, compilation, Judge Core, result
persistence, and observation. `pipeline_terminal_throughput_per_sec` must
never be described as Judge Core throughput.

**E2E** is client submission through observed terminal verdict. **Contestant
CPU time** is sandbox-reported diagnostic CPU time; it is not Judge Core wall
time and is not a reciprocal throughput metric.

## Evidence rules

`judge-bench` raw CSV records client submit/accept/terminal timestamps, but
does not currently record compile-complete and testcase-execution-complete for
each submission. Therefore ordinary harness artifacts emit Judge Core metrics
as `UNAVAILABLE`, rather than estimating them from pipeline latency.

A controlled compile-excluded benchmark with direct phase timestamps may
report observed `judge_core_throughput_per_sec`. At concurrency one,
`1 / median(judge_core_wall_seconds)` is permitted only as
`isolated_judge_core_service_rate_estimate_per_sec`; it is not demonstrated,
production, or sustainable throughput.

If terminal observations are fewer than accepted submissions, E2E samples are
right-censored. Their p95/p99 are tails of the observed sample only, not a
complete population distribution.

Burst sizes (for example, 1K accounts each submitting once) are cardinalities,
not requests per second. Intake rates are derived only from actual POST-start
or accepted timestamps.

## Historical interpretation

Historical v0.1.8/v0.1.9 1K burst raw intake measurements remain valid as
intake evidence. Their terminal rates are valid observed **pipeline** terminal
rates and are retained unchanged under the corrected name. Judge Core
throughput is `N/A — not measured` unless raw artifacts include the required
compile-excluded phase timestamps.

In particular, the approximately 1.043 terminal completions/s observed in the
v0.1.9 1K run is a right-censorable pipeline terminal observation, not Judge
throughput. The known submission #3596 trace separately records approximately
1640.6 ms compile overhead and approximately 175.2 ms testcase-batch RPC wall;
only the latter is relevant Judge Core execution evidence. Its approximately
73 ms contestant CPU sum is diagnostic only.

The HTTP-vs-gRPC and MemoryFile-vs-CachedFile local A/B artifacts already
measure their testcase execution phase after compilation; their compile
duration is diagnostic and excluded from the distributions.

## Single-node Judge Core capacity benchmark

`TestGoJudgeCoreCapacity` is an opt-in integration benchmark for one Judge
Worker client and one executorserver sandbox. It compiles one deterministic
14-case C++ Missing Number-style program before measurement, warms the
production testcase-input cache, and then measures only the four official
testcase batches (`4 + 4 + 4 + 2`). Compile and cache population are reported
as setup diagnostics and are excluded from every Judge Core latency and
throughput value.

Each tested level uses a **closed-loop** model: exactly `C` workers begin at a
barrier and repeatedly execute complete logical submissions without think
time. The headline throughput is the directly observed count of correct,
complete submissions inside the fixed wall-clock window divided by that
window. It is never a reciprocal median-latency estimate. An optional
concurrency-one reciprocal is diagnostic only and is not the headline number.

The default sweep is `1,2,4,8`, with two 20-second repetitions per level and
five warmup submissions per level. A level with a transport error, malformed
response, incorrect verdict, or unexpected `FileAdd` during hot measurement
is invalid capacity evidence. The report separately names the measured
production configured level (`C=2`) and the highest error-free tested level.
It classifies a saturation point only when a successive level plateaus or
degrades together with a material p95 queueing signal; otherwise it reports
`MAX_TESTED_NO_SATURATION` rather than guessing a hardware maximum.

Any multi-node table is labelled **IDEAL LINEAR PROJECTION — NOT
DEMONSTRATED**. It is a multiplication of the measured one-node Judge Core
result, not evidence of Kafka, network, scheduling, or cluster capacity.

Build the opt-in standalone test binary from the Judge Worker module:

```bash
cd workers/judge
go test -c -tags=integration \
  -o /tmp/astracode-judge-core-bench.test \
  ./internal/adapter/outbound/execute
```

Run it from the Worker container/network context so
`judge_sandbox:5051` resolves privately. Required environment is
`ASTRACODE_JUDGE_CORE_GRPC_ADDR` and `ASTRACODE_JUDGE_CORE_RESULTS_DIR`.
Optional overrides are `ASTRACODE_JUDGE_CORE_CONCURRENCY` (default
`1,2,4,8`), `ASTRACODE_JUDGE_CORE_DURATION` (default `20s`),
`ASTRACODE_JUDGE_CORE_REPETITIONS` (default `2`),
`ASTRACODE_JUDGE_CORE_WARMUP` (default `5`),
`ASTRACODE_JUDGE_CORE_RELEASE`, and
`ASTRACODE_JUDGE_CORE_SANDBOX_CONTAINER` for optional one-second Docker
resource sampling. The result directory is a parent directory; the test
creates a timestamped run subdirectory containing `raw.csv`, `summary.json`,
`report.md`, and (when sampling is requested) `resources.csv`.

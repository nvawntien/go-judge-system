# AstraCode Judge benchmark methodology

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

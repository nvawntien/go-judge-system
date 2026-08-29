# Judge benchmark harness

See [benchmark methodology](METHODOLOGY.md) for the strict separation between
compile overhead, Judge Core, pipeline terminal observations, E2E, and
contestant CPU diagnostics.

`judge-bench` is a controlled, external HTTP client for evaluating AstraCode's
submission and Judge capacity. It uses already-issued cookie sessions only. It
does not register users, automate password login, create accounts, access a
database, change Redis, or retry a submission POST.

## Safety

- Keep `users.local.json` local with mode `0600`; it is ignored by Git. The file must
  contain unique aliases and access cookies, as shown in
  [`users.example.json`](users.example.json).
- Non-loopback targets require HTTPS, `--allow-remote`, the exact
  `--confirm-target-host`, and an explicit `--max-submissions` cap.
- `preflight` never creates submissions or event tickets. It validates local
  inputs and makes authenticated GET requests for `/api/v1/me` and the
  selected public problem. With the explicit session-refresh policy it may use
  the normal Auth refresh endpoint before identity validation, but never during
  measured load or drain.
- The tool never writes tokens, cookies, source text, real user IDs, or SSE
  tickets to its result artifacts.

## Local session bootstrap

`judge-bench` remains password-free. A separate local operator executable can
prepare its pre-issued cookie input through the normal public Login API:

```text
benchmark password -> bootstrap-sessions -> users.local.json -> judge-bench
```

It supports only `benchmark_judge_001` through `benchmark_judge_10000`, uses a
bounded (default `16`) Login + `/api/v1/me` worker pool, and writes the file
only if the full requested range succeeds. Output ordering remains canonical.
`users.local.json` contains
live credentials: keep it mode `0600`, do not share or upload it, and remove or
regenerate it after the benchmark window. Bootstrap sessions before warmup, not
during a measured run.

For a remote target, require HTTPS and explicitly confirm its host:

```bash
go run ./cmd/bootstrap-sessions \
  --base-url https://<public-host> --allow-remote \
  --confirm-target-host <public-host> --start 1 --count 50 \
  --concurrency 16 \
  --output ./users.local.json
```

The command prompts once for the shared password without echoing it. It never
accepts a password argument or environment variable. An optional password file
must be a local, regular, group/other-inaccessible file; interactive TTY input
is the recommended operator workflow.

## Commands

```bash
go run ./tools/benchmark/judge version

go run ./tools/benchmark/judge preflight --mode sustained \
  --base-url http://127.0.0.1:8080 --users-file ./users.local.json \
  --problem-id 1 --problem-slug two-sum --language GO \
  --source-file ./solution.go --expected-verdict ACCEPTED \
  --submit-cooldown 3s --rate 0.30 --duration 3m --max-submissions 60
```

Use `burst` with `--burst-size` or `sustained` with `--rate` and exactly one of
`--duration` or `--total-submissions`. Exact-volume sustained mode schedules
exactly that many measured logical arrivals; warmup is excluded from the total,
but included in the all-submissions `--max-submissions` safety cap. For example,
`--warmup-count 5 --total-submissions 1000` requires `--max-submissions` of at
least `1005`. Duration and exact-volume forms cannot be combined.

Warmup is the first stage permitted to create submissions. Sessions may refresh
only before warmup; no login or refresh traffic occurs during the measured load
or drain phase.

## Massive one-submit-per-user burst

`massive-burst` is a **cardinality** experiment, not an RPS experiment. It
selects the first `N` canonical aliases (`bench-001` through `bench-N`), assigns
each one exactly one submission, and releases the complete plan from a common
client origin. It does not claim literal simultaneity: each row records its
actual POST-start and acceptance timestamps, from which effective intake is
calculated.

For a full-observation 1K/5K/10K burst, first bootstrap the complete 10K pool
once, then select an exact prefix for each run. `--max-in-flight` must be at
least the burst size; set the explicit all-submission cap to that same value
(warmup is prohibited for this objective). Disable periodic safety
reconciliation explicitly—SSE remains primary and reconnect-triggered GET
reconciliation stays globally bounded—so the benchmark does not manufacture a
large GET polling storm.

```bash
go run . burst \
  --benchmark-objective massive-burst \
  --base-url https://<public-host> --allow-remote \
  --confirm-target-host <public-host> \
  --users-file ./users.local.json --user-count 1000 --burst-size 1000 \
  --max-in-flight 1000 --max-submissions 1000 \
  --submit-error-policy continue \
  --preflight-concurrency 256 \
  --safety-reconcile-interval=0 \
  --burst-start-timeout 2m --drain-timeout 30m \
  --problem-id <id> --problem-slug <slug> --language <language> \
  --source-file <source> --expected-verdict ACCEPTED --submit-cooldown 3s
```

Before any network traffic, preflight checks the local `RLIMIT_NOFILE` against
`2 × max-in-flight + 256`; it never changes the limit. A 1K burst therefore
needs at least `2256`, 5K needs `10256`, and 10K needs `20256` open-file
descriptors. If it fails, raise the shell limit (for example, `ulimit -n
20256`) before rerunning—do not treat a locally constrained run as server
saturation. Full observation can reach one active SSE stream per accepted
submission; keep the default only if this preflight passes. A terminal drain
timeout preserves raw data and reports terminal/accepted coverage; it never
invents E2E percentiles for unfinished work. Session validation/possible
refresh is bounded by `--preflight-concurrency` (default `256`) and completes
before the measured burst; it is never login traffic or a refresh loop during
load/drain. The access-token horizon includes that bounded final preflight plus
the configured `--burst-start-timeout`, not an arbitrary Judge drain.
The established SSE endpoint is authenticated by its one-time ticket; if a
long drain outlives the access cookie, an already-open stream can still receive
terminal events, but a later SSE reconnect or authenticated GET reconciliation
may fail and is recorded as partial observation coverage rather than hidden.

## Observation and output

For each accepted submission, the client obtains a one-time event ticket and
uses SSE as the primary completion observer. A terminal snapshot/event finishes
immediately. Authenticated GET is limited reconciliation after SSE trouble and
optional conservative safety checks, never a high-frequency polling loop.

Each run creates a private directory under `bench-results/` containing:

- `run.json` — immutable run configuration and timing metadata.
- `submissions.csv` — safe per-logical-submission outcomes.
- `windows.csv` — load/drain windows with client-side outstanding work.
- `summary.json` and `report.md` — aggregates and conservative classification.

`client_outstanding` means API-accepted submissions minus client-observed
terminal submissions. It is **not Kafka consumer lag**.

Optional external collectors write UTC RFC3339Nano CSV for correlation only;
they are not used to compute end-to-end latency:

```bash
tools/benchmark/judge/scripts/collect-container-stats.sh --help
tools/benchmark/judge/scripts/collect-kafka-lag.sh --help
go run ./tools/benchmark/judge analyze --run-dir bench-results/RUN-ID
```

## Data collection and analysis

Raw Go artifacts are immutable canonical measurements. The offline Python
pipeline turns them into `analysis/metrics.json`, CSV summaries, presentation
charts, and `analysis/report.html` without reading credentials, source files, or
secrets. It remains optional: normal benchmark execution does not require
Python.

```bash
cd tools/benchmark/judge/analysis
python -m venv /tmp/judge-analysis-venv
source /tmp/judge-analysis-venv/bin/activate
pip install -r requirements.txt

python -m judge_analysis analyze --run-dir ../../bench-results/RUN-ID \
  --container-stats /safe/path/container-stats.csv \
  --kafka-lag /safe/path/kafka-lag.csv

python -m judge_analysis compare \
  --results-root ../../bench-results --match 'pool1-rate*' \
  --output ../../bench-results/analysis-comparison-pool1
```

### Safe system-under-test metadata

Use a small local allowlisted configuration file to record the system that was
actually tested. Start from
[`../../system-config.example.json`](../system-config.example.json), fill only
the release label and node/resource values known to the operator, and save the
local copy as `tools/benchmark/system-config.json` (ignored by Git). The file
contains only a fixed schema: release/config label; App node count, cores and
RAM; and Judge node count, cores, RAM, worker-pool size, worker memory limit,
and sandbox memory limit. Unknown fields, zero/negative resources, and
environment dumps are rejected.

Pass it to each Judge run:

```bash
go run ./tools/benchmark/judge sustained ... \
  --system-config tools/benchmark/system-config.json
```

The validated structured values are copied into that run's `run.json`; the
original file is never copied. Comparisons keep different system configurations
in separate experiment groups. If it is omitted, reports say **unavailable**;
they do not guess configuration values.

### Capacity evidence reports

Each analyzed Judge run contains a two-level `analysis/report.html`: an
executive result card deck followed by configuration, data quality, latency,
load/queue, correctness, observer, Kafka, resource, and chart evidence. Its
raw `run.json`, `submissions.csv`, `windows.csv`, and `summary.json` remain the
canonical measurements.

The Judge comparison is the capacity evidence report. It reports the maximum
**demonstrated stable** rate, the first tested saturating rate, and the tested
interval between them; it never fabricates a precise capacity number. It also
retains every individual run behind each aggregate.

```bash
python -m judge_analysis compare \
  --results-root ../../bench-results --match 'single-node-*' \
  --output ../../bench-results/analysis-comparison-judge
```

Repeated experiments use run-level mean, standard deviation, coefficient of
variation where useful, and Student-t 95% confidence intervals. A configuration
with one repetition explicitly says `CI unavailable: N=1`; at least two
repetitions are required for a stable/saturating capacity boundary. One run is
useful for latency/correctness evidence, but cannot establish sustainable
capacity.

Massive-burst comparisons remain separate from sustained capacity comparisons.
They compare burst size with effective accepted intake, submit latency,
outstanding work, Kafka lag, and terminal/drain coverage. A burst size is never
rendered as a requested sustained arrival rate.

### Separate API capacity benchmark

[`../api/`](../api/) contains an optional k6 constant-arrival-rate,
GET-only benchmark for `GET /api/v1/problems/<slug>`, the same public
non-mutating problem read verified by Judge preflight. k6 measures
API/Gateway/service throughput; `judge-bench` measures the submission, Kafka,
Judge, sandbox, and terminal-verdict pipeline. They deliberately answer
different questions.

```bash
tools/benchmark/api/run.sh --base-url https://<public-host> --allow-remote \
  --confirm-target-host <public-host> --path /api/v1/problems/<slug> \
  --rate <rps> --duration <bounded-duration> \
  --preallocated-vus <n> --max-vus <n> --max-requests <cap> \
  --system-config tools/benchmark/system-config.json

python -m judge_analysis api-compare \
  --results-root ../../api/bench-results --match 'API-*' \
  --output ../../api/bench-results/analysis-comparison-api

python -m judge_analysis capacity-report \
  --judge-comparison ../../bench-results/analysis-comparison-judge/experiments.json \
  --api-comparison ../../api/bench-results/analysis-comparison-api/experiments.json \
  --output ../../bench-results/final-capacity-report
```

The final offline report shows the three primary KPIs with their evidence:
maximum demonstrated sustainable API requests/s, maximum demonstrated
sustainable Judge submissions/s, and Judge submit/E2E p50/p95/p99 at maximum
stable Judge load. Omit either comparison JSON when that benchmark family has
not been collected; the report shows it as unavailable rather than zero.

The collector scripts are run separately before the benchmark and stopped after
it. Missing collector files are shown as unavailable, never as zero. Their UTC
timestamps align resource/lag samples with benchmark windows; client-monotonic
latency remains authoritative.

Capacity work is a scientific workflow: define a hypothesis, control variables,
collect sustained measurements, repeat configurations, calculate statistics,
visualize, compare, then conclude. Repeat each rate/configuration before
claiming stability. One run is evidence about that run—not demonstrated capacity.
The comparison report separates the Go harness classification from conservative
analytical assessment. Analytical `STABLE` requires aligned accepted/completion
rates, no persistent client-outstanding growth, predictable drain, and no
material benchmark errors; `SATURATING` requires persistent evidence such as
outstanding growth plus a throughput gap, tail-latency rise, or remaining
backlog. Otherwise it is `INSUFFICIENT_DATA`. Missing Kafka or resource
collectors remain missing rather than being treated as zero.

Do not use this harness against production until benchmark accounts, a change
window, target confirmation, and an explicit submission budget are approved.

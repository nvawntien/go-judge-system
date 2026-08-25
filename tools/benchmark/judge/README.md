# Judge benchmark harness

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
- `preflight` is read-only: it only validates local inputs and makes GET
  requests for `/api/v1/me` and the selected public problem.
- The tool never writes tokens, cookies, source text, real user IDs, or SSE
  tickets to its result artifacts.

## Commands

```bash
go run ./tools/benchmark/judge version

go run ./tools/benchmark/judge preflight --mode sustained \
  --base-url http://127.0.0.1:8080 --users-file ./users.local.json \
  --problem-id 1 --problem-slug two-sum --language GO \
  --source-file ./solution.go --expected-verdict ACCEPTED \
  --submit-cooldown 3s --rate 0.30 --duration 3m --max-submissions 60
```

Use `burst` with `--burst-size` or `sustained` with `--rate` and `--duration`.
Warmup is the first stage permitted to create submissions. Sessions may refresh
only before warmup; no login or refresh traffic occurs during the measured load
or drain phase.

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

Do not use this harness against production until benchmark accounts, a change
window, target confirmation, and an explicit submission budget are approved.

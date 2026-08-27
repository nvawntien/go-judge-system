# AstraCode API benchmark

This is a separate, GET-only k6 workload for representative public problem reads:
`GET /api/v1/problems/<slug>`. It measures API/Gateway/service capacity, not
Judge execution. `judge-bench` remains the submission-to-terminal Judge
pipeline benchmark.

`k6` is an operator runtime dependency and is not needed by Go or Python tests.
The launcher rejects remote HTTP, unconfirmed remote hosts, unbounded duration,
missing request caps, and paths outside the verified public problem-read shape.

```bash
tools/benchmark/api/run.sh --base-url https://<host> --allow-remote \
  --confirm-target-host <host> --path /api/v1/problems/<slug> \
  --rate <rps> --duration <duration> --preallocated-vus <n> --max-vus <n> \
  --max-requests <cap> --system-config tools/benchmark/system-config.json
```

Each successful k6 run writes immutable `run.json` and `summary.json` under
`tools/benchmark/api/bench-results/`. They contain safe allowlisted metadata
only; no credentials or environment dumps are accepted.

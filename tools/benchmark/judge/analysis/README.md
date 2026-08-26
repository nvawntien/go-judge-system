# Judge benchmark analysis

This offline Python package analyzes the immutable, safe artifacts created by
`judge-bench`. It never reads `users.local.json`, passwords, cookies, or source
files. Go remains responsible for workload generation and canonical raw
measurement; Python only derives statistics, charts, and locally portable HTML.

```bash
cd tools/benchmark/judge/analysis
python -m venv /tmp/judge-analysis-venv
source /tmp/judge-analysis-venv/bin/activate
pip install -r requirements.txt

python -m judge_analysis analyze --run-dir ../../bench-results/RUN-ID
python -m judge_analysis compare --run-dir ../../bench-results/RUN-1 \
  --run-dir ../../bench-results/RUN-2 --output ../../bench-results/analysis-comparison-demo
```

Optional collector data is passed with `--container-stats` and `--kafka-lag`,
or placed as `container-stats.csv` / `kafka-lag.csv` in the run directory.
Missing collectors are reported as unavailable, never as zero. The analysis
uses UTC wall-clock timestamps only to align collector samples with benchmark
windows; it never recalculates client end-to-end latency from cross-host clocks.

For capacity work: define a hypothesis, control variables, collect sustained
measurements, repeat each configuration, analyze, compare, then conclude. One
run cannot establish capacity. Comparison groups runs from structured `run.json`
workload metadata, not directory-name guesses; a one-run group has no CI.

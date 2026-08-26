"""Group repeated runs by structured run.json workload metadata."""

from __future__ import annotations
import json
from pathlib import Path
import pandas as pd
from .loaders import load_run, DataError
from .metrics import calculate
from .statistics import mean_ci
from .plots import comparison as comparison_plots
from .report import render_comparison
from .capacity import evidence


def _key(data) -> tuple:
    workload, target = data.run.get("workload", {}), data.run.get("target", {})
    # Intentionally excludes run ID, repetition, timestamps, seed, and Git dirty state.
    return (data.run.get("mode"), workload.get("target_rate_per_second"), workload.get("burst_size"), workload.get("arrival_duration_ms"), workload.get("window_ms"), target.get("problem_id"), target.get("language"), workload.get("max_in_flight"), data.run.get("experiment", {}).get("node_count"))


def _judge_metric(metrics, container, metric, stat):
    return metrics["resources"].get("containers", {}).get(container, {}).get(metric, {}).get(stat)


def compare(run_dirs: list[str | Path], output: str | Path) -> pd.DataFrame:
    runs = []
    for directory in run_dirs:
        data = load_run(directory); metrics, _, _ = calculate(data); runs.append((data, metrics))
    groups = {}
    for data, metrics in runs: groups.setdefault(_key(data), []).append((data, metrics))
    rows = []
    for key, values in groups.items():
        first = values[0][0]; rates = [m["completion"]["completion_throughput_per_sec"] for _,m in values]
        p50 = [m["latency_ms"]["end_to_end"]["p50"] for _,m in values]; p95 = [m["latency_ms"]["end_to_end"]["p95"] for _,m in values]; p99 = [m["latency_ms"]["end_to_end"]["p99"] for _,m in values]
        cpu = [_judge_metric(m,"judge_worker","cpu_percent","p95") for _,m in values]; mem = [_judge_metric(m,"judge_worker","memory_mib","max") for _,m in values]; lag = [m["kafka"].get("lag_at_end") for _,m in values]
        ci, p50_ci, p95_ci, p99_ci = mean_ci(rates), mean_ci(p50), mean_ci(p95), mean_ci(p99)
        requested = first.run.get("workload",{}).get("target_rate_per_second")
        assessments = {m["analytical_assessment"]["state"] for _,m in values}
        row = {"experiment_label": f"{first.run.get('mode')} rate={requested}", "run_ids": [d.run.get("run_id") for d,_ in values], "repetitions": len(values), "requested_rate": requested,
               "throughput_mean": ci["mean"], "throughput_std": ci["std"], "throughput_ci95_low": ci["ci95_low"], "throughput_ci95_high": ci["ci95_high"], "throughput_ci_method": ci["method"],
               "p50_mean": p50_ci["mean"], "p50_ci95_low": p50_ci["ci95_low"], "p50_ci95_high": p50_ci["ci95_high"],
               "p95_mean": p95_ci["mean"], "p95_ci95_low": p95_ci["ci95_low"], "p95_ci95_high": p95_ci["ci95_high"],
               "p99_mean": p99_ci["mean"], "p99_ci95_low": p99_ci["ci95_low"], "p99_ci95_high": p99_ci["ci95_high"],
               "judge_cpu_p95_mean": mean_ci(cpu)["mean"], "judge_memory_max_mean": mean_ci(mem)["mean"], "kafka_lag_end_mean": mean_ci(lag)["mean"],
               "assessment": ("SATURATING" if "SATURATING" in assessments else "STABLE" if assessments == {"STABLE"} else "INSUFFICIENT_DATA")}
        rows.append(row)
    frame = pd.DataFrame(rows).sort_values(["requested_rate", "experiment_label"], na_position="last")
    out = Path(output); out.mkdir(parents=True, exist_ok=False); charts = comparison_plots(out / "charts", frame)
    frame.to_csv(out / "experiments.csv", index=False)
    capacity = evidence(frame)
    (out / "experiments.json").write_text(json.dumps({"experiments": rows, "capacity_evidence": capacity}, indent=2, default=str), encoding="utf-8")
    render_comparison(out / "comparison.html", rows, charts, capacity)
    return frame

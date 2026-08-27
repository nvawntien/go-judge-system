"""Group repeated Judge runs by workload and safe system configuration."""

from __future__ import annotations
import json
import hashlib
from pathlib import Path
import pandas as pd
from .loaders import DataError, load_run
from .metrics import calculate
from .statistics import mean_ci
from .plots import comparison as comparison_plots
from .report import render_comparison
from .capacity import evidence


def _config_key(system: dict) -> str:
    # `calculate` already copied the strict allowlist. Never use arbitrary
    # run.json content as an output/grouping fingerprint.
    if not system.get("available"):
        return "unavailable"
    value = {key: system[key] for key in ("label", "release", "app", "judge")}
    return json.dumps(value, sort_keys=True, separators=(",", ":"))


def _config_series(system: dict, label: str) -> str:
    return f"{label} [{hashlib.sha256(_config_key(system).encode()).hexdigest()[:8]}]"


def _key(data, metrics) -> tuple:
    workload, target = data.run.get("workload", {}), data.run.get("target", {})
    return (data.run.get("mode"), data.run.get("benchmark_objective"), workload.get("target_rate_per_second"), workload.get("burst_size"), workload.get("arrival_duration_ms"), workload.get("window_ms"), target.get("problem_id"), target.get("language"), workload.get("max_in_flight"), _config_key(metrics["system_config"]))


def _judge_metric(metrics, container, metric, stat):
    return metrics["resources"].get("containers", {}).get(container, {}).get(metric, {}).get(stat)


def _assessment(values):
    states = {m["analytical_assessment"]["state"] for _, m in values}
    return "SATURATING" if "SATURATING" in states else "STABLE" if states == {"STABLE"} else "INSUFFICIENT_DATA"


def compare(run_dirs: list[str | Path], output: str | Path) -> pd.DataFrame:
    runs = [(data, calculate(data)[0]) for directory in run_dirs for data in [load_run(directory)]]
    modes = {data.run.get("mode") for data, _ in runs}
    if "burst" in modes and "sustained" in modes:
        raise DataError("do not mix burst-cardinality and sustained-rate runs in one Judge comparison")
    groups = {}
    for data, metrics in runs:
        groups.setdefault(_key(data, metrics), []).append((data, metrics))
    rows, individual = [], []
    for values in groups.values():
        first, first_metrics = values[0]
        is_burst = first.run.get("mode") == "burst"
        requested = first.run.get("workload", {}).get("target_rate_per_second")
        burst_size = first.run.get("workload", {}).get("burst_size")
        if is_burst:
            # A burst cardinality is not an arrival rate. Intake is derived
            # from actual post/accept timestamps by the single-run analyzer.
            rates = [m.get("burst", {}).get("accepted", {}).get("throughput_per_sec") for _, m in values]
            accepted = rates
            p50 = [m["latency_ms"]["submit"]["p50"] for _, m in values]
            p95 = [m["latency_ms"]["submit"]["p95"] for _, m in values]
            p99 = [m["latency_ms"]["submit"]["p99"] for _, m in values]
        else:
            rates = [m["completion"]["completion_throughput_per_sec"] for _, m in values]
            accepted = [m["load"]["accepted_arrival_rate_per_sec"] for _, m in values]
            p50 = [m["latency_ms"]["end_to_end"]["p50"] for _, m in values]
            p95 = [m["latency_ms"]["end_to_end"]["p95"] for _, m in values]
            p99 = [m["latency_ms"]["end_to_end"]["p99"] for _, m in values]
        outstanding = [m["queue"]["max_client_outstanding"] for _, m in values]
        slope = [m["queue"].get("outstanding_slope_per_minute") for _, m in values]
        cpu = [_judge_metric(m, "judge_worker", "cpu_percent", "p95") for _, m in values]
        memory = [_judge_metric(m, "judge_worker", "memory_mib", "max") for _, m in values]
        lag_max = [m["kafka"].get("total_lag", {}).get("max") for _, m in values]
        lag_end = [m["kafka"].get("lag_at_end") for _, m in values]
        error_count = [sum(m["correctness"]["errors"].values()) for _, m in values]
        throughput_ci, accepted_ci, p50_ci, p95_ci, p99_ci = mean_ci(rates), mean_ci(accepted), mean_ci(p50), mean_ci(p95), mean_ci(p99)
        system = first_metrics["system_config"]
        label = system.get("label") if system.get("available") else "system-config-unavailable"
        subject = f"burst size={burst_size}" if is_burst else f"rate={requested}"
        row = {"comparison_kind": "burst" if is_burst else "sustained", "configuration_label": f"{label}; {first.run.get('mode')} {subject}", "system_config_label": label, "system_config_fingerprint": _config_key(system), "system_config_series": _config_series(system, label), "run_ids": [d.run.get("run_id") for d, _ in values], "repetitions": len(values), "requested_rate": requested, "burst_size": burst_size,
               "accepted_rate_mean": accepted_ci["mean"], "throughput_mean": throughput_ci["mean"], "throughput_std": throughput_ci["std"], "throughput_ci95_low": throughput_ci["ci95_low"], "throughput_ci95_high": throughput_ci["ci95_high"], "throughput_ci_method": throughput_ci["method"],
               "p50_mean": p50_ci["mean"], "p95_mean": p95_ci["mean"], "p99_mean": p99_ci["mean"], "outstanding_peak_mean": mean_ci(outstanding)["mean"], "outstanding_slope_mean": mean_ci(slope)["mean"], "kafka_lag_max_mean": mean_ci(lag_max)["mean"], "kafka_lag_end_mean": mean_ci(lag_end)["mean"], "judge_cpu_p95_mean": mean_ci(cpu)["mean"], "judge_memory_max_mean": mean_ci(memory)["mean"], "error_count_mean": mean_ci(error_count)["mean"], "assessment": _assessment(values), "system_config": system}
        rows.append(row)
        for data, metrics in values:
            individual.append({"run_id": data.run.get("run_id"), "comparison_kind": "burst" if is_burst else "sustained", "requested_rate": requested, "burst_size": burst_size, "completion_throughput": metrics["completion"]["completion_throughput_per_sec"], "effective_accepted_throughput": metrics.get("burst", {}).get("accepted", {}).get("throughput_per_sec") if is_burst else None, "e2e_p50": metrics["latency_ms"]["end_to_end"]["p50"], "e2e_p95": metrics["latency_ms"]["end_to_end"]["p95"], "e2e_p99": metrics["latency_ms"]["end_to_end"]["p99"], "submit_p50": metrics["latency_ms"]["submit"]["p50"], "submit_p95": metrics["latency_ms"]["submit"]["p95"], "submit_p99": metrics["latency_ms"]["submit"]["p99"], "error_count": sum(metrics["correctness"]["errors"].values()), "kafka_lag_end": metrics["kafka"].get("lag_at_end"), "peak_outstanding": metrics["queue"]["max_client_outstanding"], "judge_cpu_p95": _judge_metric(metrics, "judge_worker", "cpu_percent", "p95"), "judge_memory_max": _judge_metric(metrics, "judge_worker", "memory_mib", "max"), "assessment": metrics["analytical_assessment"]["state"]})
    frame = pd.DataFrame(rows)
    if len(frame):
        frame = frame.sort_values(["comparison_kind", "requested_rate", "burst_size", "configuration_label"], na_position="last")
    out = Path(output); out.mkdir(parents=True, exist_ok=False)
    charts = comparison_plots(out / "charts", frame)
    frame.to_csv(out / "experiments.csv", index=False)
    capacity = ({"state": "BURST_COMPARISON", "interval": "not applicable", "message": "Burst cardinalities are not sustained arrival rates; effective intake is measured from client timestamps."}
                if len(frame) and frame.comparison_kind.eq("burst").all() else evidence(frame[frame.comparison_kind == "sustained"]))
    (out / "experiments.json").write_text(json.dumps({"benchmark_type": "judge", "experiments": rows, "individual_runs": individual, "capacity_evidence": capacity}, indent=2, default=str), encoding="utf-8")
    render_comparison(out / "comparison.html", rows, charts, capacity, individual)
    return frame

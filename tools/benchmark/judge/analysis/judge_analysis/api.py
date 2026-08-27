"""Offline parser and comparison for k6 API capacity artifacts."""

from __future__ import annotations
import json
import hashlib
from pathlib import Path
import pandas as pd
from .loaders import DataError
from .statistics import mean_ci
from .capacity import evidence

THRESHOLDS = {"minimum_achieved_requested_ratio": 0.95, "maximum_error_rate": 0.0, "maximum_dropped_iterations": 0}
_APP_FIELDS = ("nodes", "cpu_cores_per_node", "memory_mib_per_node")
_JUDGE_FIELDS = (*_APP_FIELDS, "worker_pool_size", "worker_memory_limit_mib", "sandbox_memory_limit_mib")


def _read(path: Path) -> dict:
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        raise DataError(f"invalid API artifact {path.name}") from error
    if not isinstance(value, dict): raise DataError(f"invalid API artifact {path.name}")
    return value


def load_api_run(run_dir: str | Path) -> tuple[dict, dict]:
    path = Path(run_dir)
    run, summary = _read(path / "run.json"), _read(path / "summary.json")
    if run.get("benchmark_type") != "api" or summary.get("benchmark_type") != "api": raise DataError("artifacts are not an API benchmark run")
    if not isinstance(run.get("run_id"), str) or not run["run_id"]: raise DataError("API run has no valid run_id")
    workload, target = run.get("workload"), run.get("target")
    if not isinstance(workload, dict) or not isinstance(target, dict) or not isinstance(target.get("endpoint"), str): raise DataError("API run has invalid target/workload")
    for name in ("requested_rps", "achieved_rps", "total_requests", "successful_requests", "failed_requests", "error_rate", "dropped_iterations"):
        value = summary.get(name)
        if not isinstance(value, (int, float)) or value < 0: raise DataError(f"API summary has invalid {name}")
    latency = summary.get("latency_ms")
    if not isinstance(latency, dict) or any(name not in latency for name in ("p50", "p95", "p99", "max")): raise DataError("API summary has invalid latency")
    return run, summary


def assessment(summary: dict) -> str:
    requested, achieved = summary["requested_rps"], summary["achieved_rps"]
    if requested <= 0 or summary["total_requests"] <= 0: return "INSUFFICIENT_DATA"
    if summary["error_rate"] > THRESHOLDS["maximum_error_rate"] or summary["dropped_iterations"] > THRESHOLDS["maximum_dropped_iterations"]: return "SATURATING"
    if achieved / requested < THRESHOLDS["minimum_achieved_requested_ratio"]: return "SATURATING"
    return "STABLE"


def _safe_system_config(value) -> dict | None:
    if not isinstance(value, dict) or set(value) != {"label", "release", "app", "judge"}:
        return None
    if not isinstance(value["label"], str) or not value["label"] or not isinstance(value["release"], str) or not value["release"]:
        return None
    if not isinstance(value["app"], dict) or not isinstance(value["judge"], dict) or set(value["app"]) != set(_APP_FIELDS) or set(value["judge"]) != set(_JUDGE_FIELDS):
        return None
    try:
        clean = {"label": value["label"], "release": value["release"], "app": {key: int(value["app"][key]) for key in _APP_FIELDS}, "judge": {key: int(value["judge"][key]) for key in _JUDGE_FIELDS}}
    except (TypeError, ValueError):
        return None
    raw_numbers = [number for section in (value["app"], value["judge"]) for number in section.values()]
    if any(isinstance(number, bool) for number in raw_numbers):
        return None
    return clean if all(number > 0 for section in (clean["app"], clean["judge"]) for number in section.values()) else None


def _config_key(system: dict | None) -> str:
    return json.dumps(system, sort_keys=True, separators=(",", ":")) if system else "unavailable"


def _config_series(system: dict | None, label: str) -> str:
    return f"{label} [{hashlib.sha256(_config_key(system).encode()).hexdigest()[:8]}]"


def compare_api(run_dirs: list[str | Path], output: str | Path) -> pd.DataFrame:
    groups = {}
    for directory in run_dirs:
        run, summary = load_api_run(directory)
        groups.setdefault((summary["requested_rps"], run["target"]["endpoint"], _config_key(_safe_system_config(run.get("system_config")))), []).append((run, summary))
    rows, individual = [], []
    for values in groups.values():
        first, first_summary = values[0]
        achieved, p50, p95, p99 = ([summary["achieved_rps"] for _, summary in values], [summary["latency_ms"]["p50"] for _, summary in values], [summary["latency_ms"]["p95"] for _, summary in values], [summary["latency_ms"]["p99"] for _, summary in values])
        ci, p50_ci, p95_ci, p99_ci = mean_ci(achieved), mean_ci(p50), mean_ci(p95), mean_ci(p99)
        states = {assessment(summary) for _, summary in values}
        state = "SATURATING" if "SATURATING" in states else "STABLE" if states == {"STABLE"} else "INSUFFICIENT_DATA"
        system = _safe_system_config(first.get("system_config"))
        label = system.get("label") if system else "system-config-unavailable"
        rows.append({"configuration_label": f"{label}; API rate={first_summary['requested_rps']}", "system_config_label": label, "system_config_fingerprint": _config_key(system), "system_config_series": _config_series(system, label), "run_ids": [run["run_id"] for run, _ in values], "repetitions": len(values), "requested_rate": first_summary["requested_rps"], "achieved_mean": ci["mean"], "achieved_std": ci["std"], "achieved_ci95_low": ci["ci95_low"], "achieved_ci95_high": ci["ci95_high"], "achieved_ci_method": ci["method"], "p50_mean": p50_ci["mean"], "p95_mean": p95_ci["mean"], "p99_mean": p99_ci["mean"], "error_rate_mean": mean_ci([summary["error_rate"] for _, summary in values])["mean"], "dropped_iterations_mean": mean_ci([summary["dropped_iterations"] for _, summary in values])["mean"], "assessment": state, "endpoint": first["target"]["endpoint"], "system_config": system})
        for run, summary in values:
            individual.append({"run_id": run["run_id"], "requested_rate": summary["requested_rps"], "achieved_rps": summary["achieved_rps"], "p50": summary["latency_ms"]["p50"], "p95": summary["latency_ms"]["p95"], "p99": summary["latency_ms"]["p99"], "error_rate": summary["error_rate"], "dropped_iterations": summary["dropped_iterations"], "assessment": assessment(summary)})
    frame = pd.DataFrame(rows)
    if len(frame): frame = frame.sort_values(["requested_rate", "configuration_label"])
    out = Path(output); out.mkdir(parents=True, exist_ok=False); frame.to_csv(out / "experiments.csv", index=False)
    from .plots import api_comparison as api_plots
    charts = api_plots(out / "charts", frame)
    capacity_frame = frame.rename(columns={"achieved_mean": "throughput_mean"})
    capacity = evidence(capacity_frame, "req/s")
    payload = {"benchmark_type": "api", "thresholds": THRESHOLDS, "experiments": rows, "individual_runs": individual, "capacity_evidence": capacity}
    (out / "experiments.json").write_text(json.dumps(payload, indent=2, default=str), encoding="utf-8")
    from .report import render_api_comparison
    render_api_comparison(out / "comparison.html", rows, capacity, individual, THRESHOLDS, charts)
    return frame

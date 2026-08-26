"""Derive analysis-only aggregates without altering raw harness measurements."""

from __future__ import annotations
from collections import Counter
from datetime import datetime, timezone
import math
import numpy as np
import pandas as pd
from .loaders import RunData
from .statistics import distribution


def _float(value):
    return None if value is None or (isinstance(value, float) and math.isnan(value)) else float(value)


def _resource_metrics(containers: pd.DataFrame | None) -> dict:
    if containers is None:
        return {"available": False, "reason": "container statistics unavailable", "containers": {}}
    result = {}
    for name, group in containers.groupby("container"):
        result[str(name)] = {"cpu_percent": distribution(group["cpu_percent"]),
                             "memory_mib": distribution(group["memory_bytes"] / 1024**2),
                             "pids_max": _float(group["pids"].max()) if "pids" in group else None,
                             "samples": int(len(group))}
    return {"available": True, "containers": result}


def align_to_run(frame: pd.DataFrame | None, run: dict) -> pd.DataFrame | None:
    """Use UTC overlap only; never infer latency from collector wall clocks."""
    if frame is None:
        return None
    start, end = pd.to_datetime(run.get("started_at"), utc=True, errors="coerce"), pd.to_datetime(run.get("ended_at"), utc=True, errors="coerce")
    result = frame
    if not pd.isna(start): result = result[result.timestamp >= start]
    if not pd.isna(end): result = result[result.timestamp <= end]
    return result.copy()


def _kafka_metrics(kafka: pd.DataFrame | None, run: dict) -> dict:
    if kafka is None:
        return {"available": False, "reason": "Kafka lag data unavailable"}
    totals = kafka.groupby("timestamp", as_index=False)["lag"].sum().sort_values("timestamp")
    load = run.get("phases", {}).get("load", {})
    start, end = pd.to_datetime(load.get("started_at"), utc=True, errors="coerce"), pd.to_datetime(load.get("ended_at"), utc=True, errors="coerce")
    slope = None
    measured = totals
    if not pd.isna(start) and not pd.isna(end):
        measured = totals[(totals.timestamp >= start) & (totals.timestamp < end)]
    if len(measured) >= 2:
        seconds = (measured.timestamp - measured.timestamp.iloc[0]).dt.total_seconds().to_numpy()
        if seconds[-1] > 0:
            slope = float(np.polyfit(seconds, measured.lag.to_numpy(), 1)[0])
    baseline = float(totals.lag.iloc[0]) if len(totals) else None
    end_lag = float(totals.lag.iloc[-1]) if len(totals) else None
    drained = (baseline is not None and end_lag is not None and end_lag <= baseline)
    return {"available": True, "samples": int(len(totals)), "total_lag": distribution(totals.lag),
            "lag_at_end": end_lag, "lag_growth_slope_per_second": slope,
            "returns_to_baseline": drained,
            "per_partition_samples": int(len(kafka))}


def _quality(data: RunData) -> dict:
    submissions, windows = data.submissions, data.windows
    reasons = []
    duplicate_ids = int(submissions.loc[submissions.submission_id.notna(), "submission_id"].duplicated().sum())
    malformed = int((submissions[["submit_latency_ms", "end_to_end_latency_ms", "accepted_to_terminal_ms"]] < 0).any(axis=1).sum())
    valid_e2e = int(submissions.end_to_end_latency_ms.notna().sum())
    if duplicate_ids:
        reasons.append(f"duplicate accepted submission IDs: {duplicate_ids}")
    if malformed:
        reasons.append(f"malformed latency rows: {malformed}")
    if not len(windows):
        reasons.append("no benchmark windows")
    if reasons:
        state = "INVALID"
    elif data.kafka is None or data.containers is None:
        state = "PARTIAL"
        reasons.append("one or more optional collectors unavailable")
    else:
        state = "GOOD"
    def coverage(frame):
        if frame is None or not len(frame): return {"available": False}
        timestamps = frame.timestamp.sort_values()
        diffs = timestamps.drop_duplicates().diff().dt.total_seconds().dropna()
        start = pd.to_datetime(data.run.get("started_at"), utc=True, errors="coerce")
        end = pd.to_datetime(data.run.get("ended_at"), utc=True, errors="coerce")
        return {"available": True, "samples": int(len(frame)), "first_utc": timestamps.iloc[0].isoformat(), "last_utc": timestamps.iloc[-1].isoformat(),
                "median_sampling_interval_seconds": (float(diffs.median()) if len(diffs) else None),
                "clock_overlap_with_run": bool((pd.isna(start) or timestamps.iloc[-1] >= start) and (pd.isna(end) or timestamps.iloc[0] <= end))}
    return {"state": state, "reasons": reasons, "submission_rows": int(len(submissions)),
            "valid_e2e_latency_rows": valid_e2e, "missing_e2e_latency_rows": int(len(submissions) - valid_e2e),
            "duplicate_submission_ids": duplicate_ids, "malformed_rows": malformed,
            "collector_coverage": {"container_statistics": coverage(data.containers), "kafka_lag": coverage(data.kafka)}}


def analytical_assessment(data: RunData, metrics: dict) -> dict:
    """Conservative analysis-only heuristic; harness classification remains canonical."""
    if data.run.get("mode") != "sustained":
        return {"state": "INSUFFICIENT_DATA", "reasons": ["burst mode has no sustained-capacity assessment"]}
    quality = metrics["data_quality"]
    if quality["state"] == "INVALID":
        return {"state": "INSUFFICIENT_DATA", "reasons": ["invalid raw dataset"]}
    errors = metrics["correctness"]["errors"]
    if errors["rate_limited"] or errors["server_errors"] or errors["transport_failures"] or errors["completion_timeouts"]:
        return {"state": "INSUFFICIENT_DATA", "reasons": ["admission, transport, or completion failures distort capacity evidence"]}
    load = data.windows[data.windows.phase == "load"].sort_values("window_start")
    if len(load) < 3:
        return {"state": "INSUFFICIENT_DATA", "reasons": ["fewer than three load windows"]}
    recent = load.tail(3)
    outstanding_growth = bool(recent.client_outstanding.is_monotonic_increasing and recent.client_outstanding.iloc[-1] > recent.client_outstanding.iloc[0])
    throughput_gap = recent.completion_rate_per_sec.mean() < recent.accepted_rate_per_sec.mean() * 0.9
    latency = recent.e2e_latency_p95_ms.dropna()
    latency_growth = len(latency) >= 3 and latency.iloc[-1] > latency.iloc[0] * 1.25
    remaining = metrics["queue"]["ending_client_outstanding"]
    if outstanding_growth and (throughput_gap or latency_growth or remaining > 0):
        return {"state": "SATURATING", "reasons": ["client outstanding grew across recent load windows", "completion rate fell behind accepted rate or tail latency increased"]}
    if not outstanding_growth and not throughput_gap and remaining == 0:
        return {"state": "STABLE", "reasons": ["accepted and completion rates remained aligned; client outstanding drained"]}
    return {"state": "INSUFFICIENT_DATA", "reasons": ["mixed or insufficiently persistent evidence"]}


def calculate(data: RunData) -> tuple[dict, pd.DataFrame, pd.DataFrame]:
    submissions, windows = data.submissions.copy(), data.windows.copy()
    load = windows[windows.phase == "load"]
    accepted = submissions[submissions.accepted.astype(bool)]
    completed = submissions[submissions.terminal_status.fillna("").ne("")]
    verdicts = completed.terminal_status.value_counts(dropna=True).rename_axis("verdict").reset_index(name="count")
    verdicts["percent"] = verdicts["count"] / len(completed) * 100 if len(completed) else np.nan
    start = pd.to_datetime(data.run.get("phases", {}).get("load", {}).get("started_at"), utc=True, errors="coerce")
    timeseries = windows.copy()
    timeseries["elapsed_seconds"] = ((timeseries.window_start - start).dt.total_seconds() if not pd.isna(start) else np.nan)
    errors = {"rate_limited": int(submissions.rate_limited.astype(bool).sum()),
              "server_errors": int((submissions.outcome == "server_error").sum()),
              "transport_failures": int(submissions.error_class.fillna("").str.contains("transport", case=False).sum()),
              "completion_timeouts": int((submissions.outcome == "completion_timeout").sum()),
              "http_errors": int(((submissions.http_status.notna()) & ~submissions.http_status.isin([200, 201])).sum())}
    load_duration = float(load.window_duration_ms.sum() / 1000) if len(load) else 0.0
    containers, kafka = align_to_run(data.containers, data.run), align_to_run(data.kafka, data.run)
    metrics = {"analysis_schema_version": "judge-analysis/v1", "generated_at_utc": datetime.now(timezone.utc).isoformat(),
               "run_id": data.run.get("run_id"), "harness_classification": (data.summary or {}).get("classification"),
               "input_hashes": {name: __import__("hashlib").sha256((data.path / name).read_bytes()).hexdigest() for name in ["run.json", "submissions.csv", "windows.csv"]},
               "statistical_methods": {"percentiles": "numpy linear interpolation", "confidence_intervals": "Student-t across repetitions"},
               "load": {"intended": int(submissions.intended_at.notna().sum()), "attempted": int(submissions.attempted.astype(bool).sum()), "accepted": int(len(accepted)),
                        "accepted_arrival_rate_per_sec": (float(len(accepted) / load_duration) if load_duration else None)},
               "completion": {"completed": int(len(completed)), "completion_throughput_per_sec": (float(load.completed.sum() / load_duration) if load_duration else None),
                              "completion_ratio": (float(len(completed) / len(accepted)) if len(accepted) else None)},
               "latency_ms": {"submit": distribution(submissions.submit_latency_ms), "end_to_end": distribution(submissions.end_to_end_latency_ms), "accepted_to_terminal": distribution(submissions.accepted_to_terminal_ms)},
               "queue": {"max_client_outstanding": int(windows.client_outstanding_peak.max()) if len(windows) else 0,
                         "ending_client_outstanding": int(windows.client_outstanding.iloc[-1]) if len(windows) else 0,
                         "time_to_peak_seconds": (float(timeseries.loc[timeseries.client_outstanding_peak.idxmax(), "elapsed_seconds"]) if len(timeseries) and not pd.isna(start) else None),
                         "drain_duration_ms": (data.summary or {}).get("drain", {}).get("duration_ms")},
               "correctness": {"verdicts": dict(zip(verdicts["verdict"], verdicts["count"])), "accepted_ratio": (float((completed.terminal_status == "ACCEPTED").mean()) if len(completed) else None),
                               "system_error_count": int((completed.terminal_status == "SYSTEM_ERROR").sum()), "errors": errors},
               "observation": {"sse_completions": int(submissions.completion_source.fillna("").str.startswith("sse_").sum()),
                               "reconciliation_completions": int(submissions.completion_source.fillna("").str.startswith("get_").sum()),
                               "sse_failures": int(submissions.sse_failures.fillna(0).sum())},
               "kafka": _kafka_metrics(kafka, data.run), "resources": _resource_metrics(containers), "data_quality": _quality(data)}
    metrics["analytical_assessment"] = analytical_assessment(data, metrics)
    return metrics, timeseries, verdicts

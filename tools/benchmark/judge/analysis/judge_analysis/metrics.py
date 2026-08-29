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
        item = {"cpu_percent": distribution(group["cpu_percent"]),
                             "memory_mib": distribution(group["memory_bytes"] / 1024**2),
                             "pids_max": _float(group["pids"].max()) if "pids" in group else None,
                             "samples": int(len(group))}
        if "restart_count" in group:
            restarts = group["restart_count"].dropna()
            item["restart_count_start"] = _float(restarts.iloc[0]) if len(restarts) else None
            item["restart_count_end"] = _float(restarts.iloc[-1]) if len(restarts) else None
            item["restart_count_delta"] = (item["restart_count_end"] - item["restart_count_start"] if item["restart_count_end"] is not None and item["restart_count_start"] is not None else None)
        result[str(name)] = item
    return {"available": True, "containers": result}


def _system_config(run: dict) -> dict:
    value = run.get("system_config")
    required = {
        "label", "release", "app", "judge",
    }
    if not isinstance(value, dict) or not required.issubset(value):
        return {"available": False, "reason": "safe system configuration metadata unavailable"}
    # Copy only the strict allowlist emitted by judge-bench. This avoids ever
    # rendering arbitrary run.json data in a report.
    app, judge = value.get("app"), value.get("judge")
    if not isinstance(app, dict) or not isinstance(judge, dict):
        return {"available": False, "reason": "safe system configuration metadata invalid"}
    fields = {
        "app": ("nodes", "cpu_cores_per_node", "memory_mib_per_node"),
        "judge": ("nodes", "cpu_cores_per_node", "memory_mib_per_node", "worker_pool_size", "worker_memory_limit_mib", "sandbox_memory_limit_mib"),
    }
    try:
        clean = {"available": True, "label": str(value["label"]), "release": str(value["release"])}
        for section, names in fields.items():
            source = app if section == "app" else judge
            clean[section] = {name: int(source[name]) for name in names}
            if any(number <= 0 for number in clean[section].values()):
                raise ValueError
        if not clean["label"] or not clean["release"]:
            raise ValueError
        return clean
    except (KeyError, TypeError, ValueError):
        return {"available": False, "reason": "safe system configuration metadata invalid"}


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
    accepted_count = int(submissions.accepted.astype(bool).sum())
    terminal_expected = data.run.get("observation_mode") not in {"admission-only", "realistic"}
    right_censored = bool(terminal_expected and accepted_count and valid_e2e < accepted_count)
    if reasons:
        state = "INSUFFICIENT_DATA"
    elif data.kafka is None or data.containers is None or right_censored:
        state = "PARTIAL"
        if right_censored:
            reasons.append(f"terminal observation is right-censored: {valid_e2e}/{accepted_count} accepted submissions reached terminal observation")
        if data.kafka is None or data.containers is None:
            reasons.append("one or more optional collectors unavailable")
    else:
        state = "COMPLETE"
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


def _intake(values: pd.Series) -> dict:
    timestamps = pd.to_datetime(values.dropna(), utc=True, errors="coerce").dropna().sort_values()
    if len(timestamps) < 2:
        return {"count": int(len(timestamps)), "interval_ms": None, "throughput_per_sec": None}
    interval = (timestamps.iloc[-1] - timestamps.iloc[0]).total_seconds()
    if interval <= 0:
        return {"count": int(len(timestamps)), "interval_ms": None, "throughput_per_sec": None}
    return {"count": int(len(timestamps)), "interval_ms": interval * 1000, "throughput_per_sec": float(len(timestamps) / interval)}


def _burst_metrics(submissions: pd.DataFrame, summary: dict | None) -> dict:
    """Use actual POST/HTTP timestamps, never a synthetic zero load window."""
    measured = submissions[submissions.phase.eq("load")]
    attempted = _intake(measured.loc[measured.attempted.astype(bool), "post_started_at"])
    accepted = _intake(measured.loc[measured.accepted.astype(bool), "post_completed_at"])
    terminal = _intake(measured.loc[measured.terminal_status.fillna("").ne(""), "terminal_observed_at"])
    if summary:
        raw = summary.get("burst")
        if isinstance(raw, dict):
            return {"available": True, "attempted": attempted, "accepted": accepted, "terminal": terminal, "raw": raw,
                    "peak_logical_in_flight": raw.get("peak_logical_in_flight"), "peak_active_observers": raw.get("peak_active_observers")}
    return {"available": True, "attempted": attempted, "accepted": accepted, "terminal": terminal, "raw": {},
            "peak_logical_in_flight": None, "peak_active_observers": None}


def _realistic_metrics(submissions: pd.DataFrame, summary: dict | None) -> dict | None:
    raw = (summary or {}).get("realistic")
    if not isinstance(raw, dict) or raw.get("observation_mode") != "realistic":
        return None
    measured = submissions[submissions.phase.eq("load")]
    def truth(column):
        return measured[column].astype(bool) if column in measured else pd.Series(False, index=measured.index)
    tickets = truth("ticket_attempted")
    ticket_success = truth("ticket_succeeded")
    sse_attempted = truth("sse_attempted")
    sse_established = truth("sse_established")
    ticket_latency = measured["ticket_latency_ms"] if "ticket_latency_ms" in measured else []
    sse_latency = measured["sse_establishment_latency_ms"] if "sse_establishment_latency_ms" in measured else []
    ticket_start = measured.loc[tickets, "ticket_started_at"] if "ticket_started_at" in measured else pd.Series(dtype="datetime64[ns, UTC]")
    sse_start = measured.loc[sse_attempted, "sse_started_at"] if "sse_started_at" in measured else pd.Series(dtype="datetime64[ns, UTC]")
    sse_established_at = measured.loc[sse_established, "sse_established_at"] if "sse_established_at" in measured else pd.Series(dtype="datetime64[ns, UTC]")
    close_reasons = dict(measured.get("sse_close_reason", pd.Series(dtype=str)).dropna().value_counts())
    return {"observation_mode": "realistic",
            "submission": {"attempted": int(measured.attempted.astype(bool).sum()), "successful": int(measured.accepted.astype(bool).sum()), "success_percent": _float(raw.get("submission", {}).get("success_percent")), "throughput_per_sec": _intake(measured.loc[measured.accepted.astype(bool), "post_completed_at"])["throughput_per_sec"], "latency_ms": distribution(measured.submit_latency_ms)},
            "ticket": {"attempted": int(tickets.sum()), "successful": int(ticket_success.sum()), "success_percent": _float(raw.get("ticket", {}).get("success_percent")), "throughput_per_sec": _intake(ticket_start)["throughput_per_sec"], "latency_ms": distribution(ticket_latency)},
            "sse": {"attempted": int(sse_attempted.sum()), "established": int(sse_established.sum()), "failed": int(sse_attempted.sum() - sse_established.sum()), "establishment_percent": _float(raw.get("sse", {}).get("establishment_percent")), "establishment_rate_per_sec": _intake(sse_established_at)["throughput_per_sec"], "establishment_latency_ms": distribution(sse_latency), "peak_active_streams": raw.get("sse", {}).get("peak_active_streams"), "survived_full_hold": int(truth("sse_survived_full_hold").sum()), "closed_early": int((sse_established & ~truth("sse_survived_full_hold") & ~truth("sse_terminal_during_hold")).sum()), "terminal_during_hold": int(truth("sse_terminal_during_hold").sum()), "close_reasons": close_reasons, "start_spread_ms": _intake(sse_start)["interval_ms"]},
            "full_flow_success_percent": _float(raw.get("full_flow_success_percent")), "client_qualification": raw.get("client_qualification"), "system_survival": raw.get("system_survival"), "external_survival_evidence": raw.get("external_survival_evidence"), "health_probes": (summary or {}).get("health_probes", [])}


def analytical_assessment(data: RunData, metrics: dict) -> dict:
    """Conservative analysis-only heuristic; harness classification remains canonical."""
    if data.run.get("mode") != "sustained":
        return {"state": "INSUFFICIENT_DATA", "reasons": ["burst mode has no sustained-capacity assessment"]}
    quality = metrics["data_quality"]
    if quality["state"] == "INSUFFICIENT_DATA":
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
              "transport_failures": int((submissions.outcome == "ambiguous_post").sum()),
              "completion_timeouts": int((submissions.outcome == "completion_timeout").sum()),
              "http_errors": int(((submissions.http_status.notna()) & ~submissions.http_status.isin([200, 201])).sum())}
    load_duration = float(load.window_duration_ms.sum() / 1000) if len(load) else 0.0
    containers, kafka = align_to_run(data.containers, data.run), align_to_run(data.kafka, data.run)
    outstanding_slope = None
    if len(load) >= 2:
        elapsed = (load.window_start - load.window_start.iloc[0]).dt.total_seconds().to_numpy()
        if elapsed[-1] > 0:
            outstanding_slope = float(np.polyfit(elapsed, load.client_outstanding.to_numpy(), 1)[0] * 60)
    burst = _burst_metrics(submissions, data.summary) if data.run.get("mode") == "burst" else None
    accepted_rate = float(len(accepted) / load_duration) if load_duration else None
    if burst is not None:
        accepted_rate = burst["accepted"]["throughput_per_sec"]
    load_terminal_rate = float(load.completed.sum() / load_duration) if load_duration else None
    pipeline_terminal_rate = burst["terminal"]["throughput_per_sec"] if burst is not None else load_terminal_rate
    terminal_coverage = float(len(completed) / len(accepted)) if len(accepted) else None
    terminal_expected = data.run.get("observation_mode") not in {"admission-only", "realistic"}
    right_censored = bool(terminal_expected and len(accepted) and len(completed) < len(accepted))
    unavailable_reason = "raw harness artifacts do not contain compile completion and testcase-execution completion timestamps"
    admission_raw = (data.summary or {}).get("admission")
    admission = None
    if isinstance(admission_raw, dict) and admission_raw.get("observation_mode") == "admission-only":
        admission = {"observation_mode": "admission-only",
                     "effective_accepted_intake_per_sec": (burst or {}).get("accepted", {}).get("throughput_per_sec"),
                     "post_start_spread_ms": ((burst or {}).get("raw") or {}).get("post_start_offset_ms", {}).get("max"),
                     "system_survival": admission_raw.get("system_survival"),
                     "client_qualification": admission_raw.get("client_qualification"),
                     "external_survival_evidence": admission_raw.get("external_survival_evidence"),
                     "health_probes": (data.summary or {}).get("health_probes", [])}
    realistic = _realistic_metrics(submissions, data.summary)
    client_resources = align_to_run(data.client_resources, data.run)
    client_resource_metrics = {"available": False, "reason": "client resource samples unavailable"}
    if client_resources is not None:
        client_resource_metrics = {"available": True, "samples": int(len(client_resources)),
                                   "open_fds": distribution(client_resources.open_fds), "goroutines": distribution(client_resources.goroutines),
                                   "active_posts": distribution(client_resources.active_posts), "active_tickets": distribution(client_resources.active_tickets), "active_sse_streams": distribution(client_resources.active_sse_streams)}
    metrics = {"analysis_schema_version": "judge-analysis/v2", "generated_at_utc": datetime.now(timezone.utc).isoformat(),
               "run_id": data.run.get("run_id"), "harness_classification": (data.summary or {}).get("classification"),
               "input_hashes": {name: __import__("hashlib").sha256((data.path / name).read_bytes()).hexdigest() for name in ["run.json", "submissions.csv", "windows.csv"]},
               "statistical_methods": {"percentiles": "numpy linear interpolation", "confidence_intervals": "Student-t across repetitions"},
               "load": {"intended": int(submissions.intended_at.notna().sum()), "attempted": int(submissions.attempted.astype(bool).sum()), "accepted": int(len(accepted)),
                        "accepted_arrival_rate_per_sec": accepted_rate},
               "compile": {"included_in_judge_core": False, "availability": "UNAVAILABLE", "reason": "ordinary judge-bench artifacts do not contain per-submission compile wall timestamps"},
               "judge_core": {"definition": "compile_success_to_all_required_testcase_execution_batches_completed", "availability": "UNAVAILABLE", "reason": unavailable_reason, "completed": None, "throughput_per_sec": None, "wall_ms": distribution([])},
               "pipeline": {"terminal_completed": int(len(completed)), "terminal_throughput_per_sec": pipeline_terminal_rate, "terminal_observation_coverage": terminal_coverage, "right_censored": right_censored, "terminal_throughput_semantics": "observed terminal completion rate across the submission pipeline; not Judge Core throughput"},
               # Kept verbatim for old scripts. It is a pipeline terminal rate,
               # not a Judge Core measurement.
               "completion": {"completed": int(len(completed)), "completion_throughput_per_sec": load_terminal_rate,
                              "completion_ratio": terminal_coverage, "deprecated": True,
                              "semantics": "legacy alias for observed pipeline terminal completion rate; not Judge Core throughput"},
               "latency_ms": {"submit": distribution(submissions.submit_latency_ms), "end_to_end": distribution(submissions.end_to_end_latency_ms), "accepted_to_terminal": distribution(submissions.accepted_to_terminal_ms)},
               "contestant_execution": {"availability": "UNAVAILABLE", "reason": "judge-bench does not receive per-testcase sandbox CPU-time diagnostics; CPU time is not Judge Core wall time"},
               "system_config": _system_config(data.run),
               "queue": {"max_client_outstanding": int(windows.client_outstanding_peak.max()) if len(windows) else 0,
                         "ending_client_outstanding": int(windows.client_outstanding.iloc[-1]) if len(windows) else 0,
                         "time_to_peak_seconds": (float(timeseries.loc[timeseries.client_outstanding_peak.idxmax(), "elapsed_seconds"]) if len(timeseries) and not pd.isna(start) else None),
                         "drain_duration_ms": (data.summary or {}).get("drain", {}).get("duration_ms"),
                         "outstanding_slope_per_minute": outstanding_slope},
               "correctness": {"verdicts": dict(zip(verdicts["verdict"], verdicts["count"])), "accepted_ratio": (float((completed.terminal_status == "ACCEPTED").mean()) if len(completed) else None),
                               "system_error_count": int((completed.terminal_status == "SYSTEM_ERROR").sum()), "errors": errors},
               "observation": {"sse_completions": int(submissions.completion_source.fillna("").str.startswith("sse_").sum()),
                               "reconciliation_completions": int(submissions.completion_source.fillna("").str.startswith("get_").sum()),
                               "sse_failures": int(submissions.sse_failures.fillna(0).sum())},
               "kafka": _kafka_metrics(kafka, data.run), "resources": _resource_metrics(containers), "client_resources": client_resource_metrics, "data_quality": _quality(data), "burst": burst, "admission": admission, "realistic": realistic}
    metrics["analytical_assessment"] = analytical_assessment(data, metrics)
    return metrics, timeseries, verdicts

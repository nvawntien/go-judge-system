"""Headless matplotlib visualisations for one run and experiment comparisons."""

from __future__ import annotations
from pathlib import Path
import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
import numpy as np
import pandas as pd


def _save(path: Path) -> None:
    plt.grid(True, alpha=.25)
    plt.tight_layout()
    plt.savefig(path, dpi=150)
    plt.close()


def _line(value, label, color):
    if value is not None:
        plt.axvline(value, label=f"{label}: {value:.1f} ms", color=color, linestyle="--")


def single_run(charts: Path, submissions: pd.DataFrame, windows: pd.DataFrame, metrics: dict, containers: pd.DataFrame | None, kafka: pd.DataFrame | None, client_resources: pd.DataFrame | None = None) -> list[str]:
    charts.mkdir(parents=True, exist_ok=True)
    output = []
    if (metrics.get("realistic") or {}).get("observation_mode") == "realistic":
        measured = submissions[submissions.phase.eq("load")].copy()
        def timeline(column, title, name, color="#457b9d"):
            values = measured.dropna(subset=[column]).sort_values(column)
            if not len(values): return
            origin = values[column].iloc[0]
            plt.figure(figsize=(8, 4.5)); plt.step((values[column]-origin).dt.total_seconds(), np.arange(1, len(values)+1), where="post", color=color)
            plt.title(title); plt.xlabel("seconds from first event"); plt.ylabel("cumulative users")
            path=charts/name; _save(path); output.append(path.name)
        timeline("ticket_completed_at", "Ticket responses over time", "05_ticket_responses_over_time.png", "#2a9d8f")
        timeline("sse_established_at", "SSE establishments over time", "07_sse_establishments_over_time.png", "#6a5acd")
        for column, title, name, color in [("ticket_latency_ms", "Ticket latency distribution", "06_ticket_latency_distribution.png", "#2a9d8f"), ("sse_establishment_latency_ms", "SSE establishment latency distribution", "08_sse_establishment_latency_distribution.png", "#6a5acd")]:
            if column in measured:
                values=measured[column].dropna().to_numpy(dtype=float)
                if len(values):
                    plt.figure(figsize=(8,4.5)); plt.hist(values,bins=min(80,max(10,len(values)//100)),color=color); plt.title(title); plt.xlabel("milliseconds"); plt.ylabel("streams")
                    path=charts/name; _save(path); output.append(path.name)
        if "sse_close_reason" in measured:
            reasons=measured.sse_close_reason.dropna().value_counts()
            if len(reasons):
                plt.figure(figsize=(8,4.5)); plt.bar(reasons.index,reasons.values,color="#e76f51"); plt.title("SSE close reasons"); plt.xlabel("reason"); plt.ylabel("streams"); plt.xticks(rotation=20,ha="right")
                path=charts/"09_sse_close_reasons.png"; _save(path); output.append(path.name)
        if "sse_established_at" in measured and "sse_closed_at" in measured:
            established=measured.dropna(subset=["sse_established_at", "sse_closed_at"]).copy()
            if len(established):
                events=[]
                for _, row in established.iterrows(): events.extend([(row.sse_established_at,1),(row.sse_closed_at,-1)])
                events.sort(key=lambda value:value[0]); origin=events[0][0]; active=0; x=[]; y=[]
                for at,delta in events: active+=delta; x.append((at-origin).total_seconds()); y.append(active)
                plt.figure(figsize=(8,4.5)); plt.step(x,y,where="post",color="#264653"); plt.title("Active SSE streams"); plt.xlabel("seconds from first establishment"); plt.ylabel("active streams")
                path=charts/"10_active_sse_streams.png"; _save(path); output.append(path.name)
        if client_resources is not None and len(client_resources):
            origin=client_resources.timestamp.min(); x=(client_resources.timestamp-origin).dt.total_seconds()
            plt.figure(figsize=(8,4.5)); plt.plot(x,client_resources.open_fds,marker="o",label="open FDs"); plt.plot(x,client_resources.goroutines,marker="o",label="goroutines"); plt.legend(); plt.title("Load-generator FD and goroutine samples"); plt.xlabel("seconds from first sample"); plt.ylabel("count")
            path=charts/"11_client_fd_goroutines.png"; _save(path); output.append(path.name)
        # Retain the common POST/status and external-resource evidence.
        _admission_charts(charts, output, measured, containers, kafka)
        return output
    # Admission-only runs have intentionally no terminal/E2E observations.  Use
    # POST timing and status charts instead of presenting missing pipeline data.
    if (metrics.get("admission") or {}).get("observation_mode") == "admission-only":
        measured = submissions[submissions.phase.eq("load")].copy()
        _admission_charts(charts, output, measured, containers, kafka)
        return output

    latency = submissions.end_to_end_latency_ms.dropna().to_numpy(dtype=float)
    if len(latency):
        plt.figure(figsize=(8, 4.5)); plt.hist(latency, bins=min(30, max(5, len(latency))), color="#6a5acd", alpha=.8)
        for label, color in [("p50", "#2a9d8f"), ("p95", "#e76f51"), ("p99", "#d62828")]: _line(metrics["latency_ms"]["end_to_end"][label], label, color)
        plt.title(f"Pipeline E2E latency distribution (observed terminals, n={len(latency)})"); plt.xlabel("milliseconds"); plt.ylabel("submissions"); plt.legend()
        path = charts / "01_latency_distribution.png"; _save(path); output.append(path.name)
        values = np.sort(latency); plt.figure(figsize=(8, 4.5)); plt.step(values, np.arange(1, len(values)+1)/len(values), where="post")
        for label, color in [("p50", "#2a9d8f"), ("p95", "#e76f51"), ("p99", "#d62828")]: _line(metrics["latency_ms"]["end_to_end"][label], label, color)
        plt.title("Pipeline E2E latency ECDF (observed terminals)"); plt.xlabel("milliseconds"); plt.ylabel("fraction completed"); plt.ylim(0, 1.02); plt.legend()
        path = charts / "02_latency_ecdf.png"; _save(path); output.append(path.name)
        observed = submissions.dropna(subset=["terminal_observed_at", "end_to_end_latency_ms"]).copy()
        origin = observed.terminal_observed_at.min(); plt.figure(figsize=(8, 4.5)); plt.scatter((observed.terminal_observed_at-origin).dt.total_seconds(), observed.end_to_end_latency_ms, s=18, alpha=.7)
        plt.title("Pipeline E2E latency over time (observed terminals)"); plt.xlabel("seconds from first terminal observation"); plt.ylabel("milliseconds")
        path = charts / "03_latency_over_time.png"; _save(path); output.append(path.name)
    if len(windows):
        series = windows.sort_values("window_start").copy(); origin = series.window_start.min(); x = (series.window_start-origin).dt.total_seconds()
        plt.figure(figsize=(8, 4.5));
        for column, label in [("target_arrival_rate_per_sec", "target arrival"), ("accepted_rate_per_sec", "accepted"), ("completion_rate_per_sec", "pipeline terminal observed")]:
            if column in series: plt.plot(x, series[column], marker="o", label=label)
        plt.title("Intake and observed pipeline-terminal rate over time"); plt.xlabel("seconds from run windows start"); plt.ylabel("submissions / second"); plt.legend()
        path = charts / "04_throughput_over_time.png"; _save(path); output.append(path.name)
        plt.figure(figsize=(8, 4.5)); plt.step(x, series.client_outstanding, where="post", label="client outstanding"); plt.axhline(metrics["queue"]["max_client_outstanding"], color="#e76f51", linestyle="--", label="peak")
        plt.title("Client-observed outstanding work"); plt.xlabel("seconds from run windows start"); plt.ylabel("accepted minus terminal"); plt.legend()
        path = charts / "05_client_outstanding.png"; _save(path); output.append(path.name)
    if kafka is not None and len(kafka):
        total = kafka.groupby("timestamp", as_index=False).lag.sum().sort_values("timestamp"); x = (total.timestamp-total.timestamp.min()).dt.total_seconds()
        plt.figure(figsize=(8, 4.5)); plt.plot(x, total.lag, marker="o", label="total consumer-group lag"); plt.title("Kafka consumer-group lag"); plt.xlabel("seconds from first sample"); plt.ylabel("messages"); plt.legend()
        path = charts / "06_kafka_lag.png"; _save(path); output.append(path.name)
    return output


def _admission_charts(charts: Path, output: list[str], measured: pd.DataFrame, containers: pd.DataFrame | None, kafka: pd.DataFrame | None) -> None:
    started = measured.dropna(subset=["post_started_at"]).sort_values("post_started_at")
    if len(started):
        origin = started.post_started_at.iloc[0]
        plt.figure(figsize=(8, 4.5)); plt.hist((started.post_started_at-origin).dt.total_seconds() * 1000, bins=min(80, max(10, len(started)//100)), color="#457b9d")
        plt.title("POST-start distribution (actual client timestamps)"); plt.xlabel("milliseconds from first POST start"); plt.ylabel("submissions")
        path = charts / "01_post_start_distribution.png"; _save(path); output.append(path.name)
    latency = measured.submit_latency_ms.dropna().to_numpy(dtype=float)
    if len(latency):
        plt.figure(figsize=(8, 4.5)); plt.hist(latency, bins=min(80, max(10, len(latency)//100)), color="#2a9d8f")
        plt.title("Submission API POST latency distribution"); plt.xlabel("milliseconds"); plt.ylabel("submissions")
        path = charts / "02_submit_latency_distribution.png"; _save(path); output.append(path.name)
    completed = measured.dropna(subset=["post_completed_at"]).sort_values("post_completed_at")
    if len(completed):
        origin = completed.post_completed_at.iloc[0]
        elapsed = (completed.post_completed_at-origin).dt.total_seconds()
        plt.figure(figsize=(8, 4.5)); plt.step(elapsed, np.arange(1, len(completed)+1), where="post", label="HTTP responses"); plt.step(elapsed, completed.accepted.astype(int).cumsum(), where="post", label="accepted"); plt.legend(); plt.title("POST responses over time"); plt.xlabel("seconds from first response"); plt.ylabel("cumulative submissions")
        path = charts / "03_accepted_responses_over_time.png"; _save(path); output.append(path.name)
    statuses = measured.http_status.fillna(0).astype(int).astype(str).value_counts()
    if len(statuses):
        plt.figure(figsize=(8, 4.5)); plt.bar(statuses.index, statuses.values, color="#e76f51"); plt.title("HTTP status counts"); plt.xlabel("status"); plt.ylabel("submissions")
        path = charts / "04_http_status_counts.png"; _save(path); output.append(path.name)
    if kafka is not None and len(kafka):
        total = kafka.groupby("timestamp", as_index=False).lag.sum().sort_values("timestamp"); x = (total.timestamp-total.timestamp.min()).dt.total_seconds(); plt.figure(figsize=(8,4.5)); plt.plot(x,total.lag,marker="o"); plt.title("Kafka consumer-group lag"); plt.xlabel("seconds from first sample"); plt.ylabel("messages"); path=charts/"07_kafka_lag.png"; _save(path); output.append(path.name)
    if containers is not None and len(containers):
        for number, metric, label, unit in [("05", "cpu_percent", "App container CPU", "%"), ("06", "memory_bytes", "App container memory", "MiB")]:
            plt.figure(figsize=(8,4.5))
            for container, group in containers.groupby("container"):
                x=(group.timestamp-group.timestamp.min()).dt.total_seconds(); values=group[metric]/1024**2 if metric=="memory_bytes" else group[metric]; plt.plot(x,values,marker="o",label=container)
            plt.title(label); plt.xlabel("seconds from first sample"); plt.ylabel(unit); plt.legend(); path=charts/f"{number}_app_{'cpu' if metric=='cpu_percent' else 'memory'}.png"; _save(path); output.append(path.name)


def comparison(charts: Path, experiments: pd.DataFrame) -> list[str]:
    charts.mkdir(parents=True, exist_ok=True); output = []
    if len(experiments) and "comparison_kind" in experiments and experiments.comparison_kind.eq("burst").all():
        burst_plots = [
            ("burst_size_vs_effective_accepted_throughput.png", "burst_size", "throughput_mean", "Burst size vs effective accepted intake", "distinct submissions", "accepted sub/s"),
            ("burst_size_vs_submit_p95.png", "burst_size", "p95_mean", "Burst size vs submit p95", "distinct submissions", "milliseconds"),
            ("burst_size_vs_outstanding.png", "burst_size", "outstanding_peak_mean", "Burst size vs peak client outstanding", "distinct submissions", "accepted minus terminal"),
        ]
        for filename, x, y, title, xlabel, ylabel in burst_plots:
            view = experiments.dropna(subset=[x, y])
            if not len(view):
                continue
            plt.figure(figsize=(7, 4.5))
            for label, group in view.groupby("system_config_series"):
                plt.plot(group[x], group[y], marker="o", label=label)
            plt.title(title); plt.xlabel(xlabel); plt.ylabel(ylabel); plt.legend()
            path = charts / filename; _save(path); output.append(path.name)
        lag = experiments.dropna(subset=["burst_size", "kafka_lag_end_mean"])
        if len(lag):
            plt.figure(figsize=(7, 4.5)); plt.plot(lag.burst_size, lag.kafka_lag_end_mean, marker="o")
            plt.title("Burst size vs Kafka lag at end"); plt.xlabel("distinct submissions"); plt.ylabel("messages")
            path = charts / "burst_size_vs_kafka_lag.png"; _save(path); output.append(path.name)
        return output
    plots = [("throughput_vs_rate.png", "requested_rate", "throughput_mean", "Requested rate vs observed pipeline terminal rate", "requested sub/s", "pipeline terminal sub/s"),
             ("p95_latency_vs_rate.png", "requested_rate", "p95_mean", "Requested rate vs E2E p95", "requested sub/s", "milliseconds"),
             ("p99_latency_vs_rate.png", "requested_rate", "p99_mean", "Requested rate vs E2E p99", "requested sub/s", "milliseconds"),
             ("cpu_vs_throughput.png", "throughput_mean", "judge_cpu_p95_mean", "Pipeline terminal rate vs Judge CPU", "pipeline terminal sub/s", "CPU %"),
             ("memory_vs_throughput.png", "throughput_mean", "judge_memory_max_mean", "Pipeline terminal rate vs Judge memory", "pipeline terminal sub/s", "MiB")]
    for filename, x, y, title, xlabel, ylabel in plots:
        view = experiments.dropna(subset=[x, y])
        if not len(view): continue
        plt.figure(figsize=(7, 4.5));
        for label, group in view.groupby("system_config_series"):
            plt.plot(group[x], group[y], marker="o", label=label)
        plt.title(title); plt.xlabel(xlabel); plt.ylabel(ylabel); plt.legend()
        path = charts / filename; _save(path); output.append(path.name)
    lag = experiments.dropna(subset=["requested_rate", "kafka_lag_end_mean"])
    if len(lag):
        plt.figure(figsize=(7, 4.5)); plt.plot(lag.requested_rate, lag.kafka_lag_end_mean, marker="o"); plt.title("Requested rate vs Kafka lag at end"); plt.xlabel("requested sub/s"); plt.ylabel("messages")
        path = charts / "kafka_lag_vs_rate.png"; _save(path); output.append(path.name)
    outstanding = experiments.dropna(subset=["requested_rate", "outstanding_peak_mean"])
    if len(outstanding):
        plt.figure(figsize=(7, 4.5)); plt.plot(outstanding.requested_rate, outstanding.outstanding_peak_mean, marker="o"); plt.title("Requested rate vs peak client outstanding"); plt.xlabel("requested sub/s"); plt.ylabel("accepted minus terminal")
        path = charts / "outstanding_vs_rate.png"; _save(path); output.append(path.name)
    efficiency = experiments.dropna(subset=["requested_rate", "throughput_mean"]).copy()
    if len(efficiency):
        efficiency["efficiency"] = efficiency.throughput_mean / efficiency.requested_rate.replace(0, np.nan)
        plt.figure(figsize=(7, 4.5)); plt.plot(efficiency.requested_rate, efficiency.efficiency, marker="o"); plt.title("Observed pipeline-terminal efficiency"); plt.xlabel("requested sub/s"); plt.ylabel("pipeline terminal / requested")
        path = charts / "throughput_efficiency.png"; _save(path); output.append(path.name)
        plt.figure(figsize=(7, 4.5)); plt.plot(efficiency.requested_rate, efficiency.throughput_mean, marker="o", label="observed pipeline terminal"); plt.title("Pipeline evidence frontier from tested configurations"); plt.xlabel("requested sub/s"); plt.ylabel("pipeline terminal sub/s"); plt.legend()
        path = charts / "capacity_frontier.png"; _save(path); output.append(path.name)
    return output


def api_comparison(charts: Path, experiments: pd.DataFrame) -> list[str]:
    """API-only capacity charts; absent measurements remain absent."""
    charts.mkdir(parents=True, exist_ok=True)
    output = []
    for filename, y, title, ylabel in [
        ("throughput_vs_rate.png", "achieved_mean", "Requested API rate vs achieved throughput", "achieved req/s"),
        ("p95_latency_vs_rate.png", "p95_mean", "Requested API rate vs p95 latency", "milliseconds"),
        ("p99_latency_vs_rate.png", "p99_mean", "Requested API rate vs p99 latency", "milliseconds"),
        ("error_rate_vs_rate.png", "error_rate_mean", "Requested API rate vs error rate", "error fraction"),
    ]:
        view = experiments.dropna(subset=["requested_rate", y])
        if not len(view):
            continue
        plt.figure(figsize=(7, 4.5))
        for label, group in view.groupby("system_config_series"):
            plt.plot(group.requested_rate, group[y], marker="o", label=label)
        plt.title(title); plt.xlabel("requested req/s"); plt.ylabel(ylabel); plt.legend()
        path = charts / filename; _save(path); output.append(path.name)
    return output

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


def single_run(charts: Path, submissions: pd.DataFrame, windows: pd.DataFrame, metrics: dict, containers: pd.DataFrame | None, kafka: pd.DataFrame | None) -> list[str]:
    charts.mkdir(parents=True, exist_ok=True)
    output = []
    latency = submissions.end_to_end_latency_ms.dropna().to_numpy(dtype=float)
    if len(latency):
        plt.figure(figsize=(8, 4.5)); plt.hist(latency, bins=min(30, max(5, len(latency))), color="#6a5acd", alpha=.8)
        for label, color in [("p50", "#2a9d8f"), ("p95", "#e76f51"), ("p99", "#d62828")]: _line(metrics["latency_ms"]["end_to_end"][label], label, color)
        plt.title(f"End-to-end latency distribution (n={len(latency)})"); plt.xlabel("milliseconds"); plt.ylabel("submissions"); plt.legend()
        path = charts / "01_latency_distribution.png"; _save(path); output.append(path.name)
        values = np.sort(latency); plt.figure(figsize=(8, 4.5)); plt.step(values, np.arange(1, len(values)+1)/len(values), where="post")
        for label, color in [("p50", "#2a9d8f"), ("p95", "#e76f51"), ("p99", "#d62828")]: _line(metrics["latency_ms"]["end_to_end"][label], label, color)
        plt.title("End-to-end latency ECDF"); plt.xlabel("milliseconds"); plt.ylabel("fraction completed"); plt.ylim(0, 1.02); plt.legend()
        path = charts / "02_latency_ecdf.png"; _save(path); output.append(path.name)
        observed = submissions.dropna(subset=["terminal_observed_at", "end_to_end_latency_ms"]).copy()
        origin = observed.terminal_observed_at.min(); plt.figure(figsize=(8, 4.5)); plt.scatter((observed.terminal_observed_at-origin).dt.total_seconds(), observed.end_to_end_latency_ms, s=18, alpha=.7)
        plt.title("End-to-end latency over time"); plt.xlabel("seconds from first terminal observation"); plt.ylabel("milliseconds")
        path = charts / "03_latency_over_time.png"; _save(path); output.append(path.name)
    if len(windows):
        series = windows.sort_values("window_start").copy(); origin = series.window_start.min(); x = (series.window_start-origin).dt.total_seconds()
        plt.figure(figsize=(8, 4.5));
        for column, label in [("target_arrival_rate_per_sec", "target arrival"), ("accepted_rate_per_sec", "accepted"), ("completion_rate_per_sec", "completed")]:
            if column in series: plt.plot(x, series[column], marker="o", label=label)
        plt.title("Throughput over time"); plt.xlabel("seconds from run windows start"); plt.ylabel("submissions / second"); plt.legend()
        path = charts / "04_throughput_over_time.png"; _save(path); output.append(path.name)
        plt.figure(figsize=(8, 4.5)); plt.step(x, series.client_outstanding, where="post", label="client outstanding"); plt.axhline(metrics["queue"]["max_client_outstanding"], color="#e76f51", linestyle="--", label="peak")
        plt.title("Client-observed outstanding work"); plt.xlabel("seconds from run windows start"); plt.ylabel("accepted minus terminal"); plt.legend()
        path = charts / "05_client_outstanding.png"; _save(path); output.append(path.name)
    if kafka is not None and len(kafka):
        total = kafka.groupby("timestamp", as_index=False).lag.sum().sort_values("timestamp"); x = (total.timestamp-total.timestamp.min()).dt.total_seconds()
        plt.figure(figsize=(8, 4.5)); plt.plot(x, total.lag, marker="o", label="total consumer-group lag"); plt.title("Kafka consumer-group lag"); plt.xlabel("seconds from first sample"); plt.ylabel("messages"); plt.legend()
        path = charts / "06_kafka_lag.png"; _save(path); output.append(path.name)
    if containers is not None and len(containers):
        for number, metric, label, unit in [("07", "cpu_percent", "Judge CPU", "%"), ("08", "memory_bytes", "Judge memory", "MiB")]:
            judge = containers[containers.container.isin(["judge_worker", "judge_sandbox"])]
            if len(judge):
                plt.figure(figsize=(8, 4.5))
                for container, group in judge.groupby("container"):
                    x = (group.timestamp-group.timestamp.min()).dt.total_seconds(); values = group[metric] / 1024**2 if metric == "memory_bytes" else group[metric]
                    plt.plot(x, values, marker="o", label=container)
                plt.title(label); plt.xlabel("seconds from first sample"); plt.ylabel(unit); plt.legend()
                path = charts / f"{number}_judge_{'cpu' if metric == 'cpu_percent' else 'memory'}.png"; _save(path); output.append(path.name)
        judge = containers[containers.container.isin(["judge_worker", "judge_sandbox"])]
        if len(judge) and len(windows):
            window = windows.sort_values("window_start"); cpu = []
            for _, row in window.iterrows():
                samples = judge[(judge.timestamp >= row.window_start) & (judge.timestamp < row.window_end)]
                cpu.append(samples.cpu_percent.mean() if len(samples) else np.nan)
            plt.figure(figsize=(8, 4.5)); plt.scatter(cpu, window.completion_rate_per_sec, label="window mean CPU"); plt.title("Completion throughput vs Judge CPU (correlation only)"); plt.xlabel("Judge CPU %"); plt.ylabel("completed / second"); plt.legend()
            path = charts / "09_resource_overlay.png"; _save(path); output.append(path.name)
    verdicts = submissions[submissions.terminal_status.fillna("").ne("")].terminal_status.value_counts()
    if len(verdicts):
        plt.figure(figsize=(8, 4.5)); plt.bar(verdicts.index, verdicts.values, color="#457b9d"); plt.title("Terminal verdict distribution"); plt.xlabel("verdict"); plt.ylabel("submissions"); plt.xticks(rotation=30, ha="right")
        path = charts / "10_verdict_distribution.png"; _save(path); output.append(path.name)
    return output


def comparison(charts: Path, experiments: pd.DataFrame) -> list[str]:
    charts.mkdir(parents=True, exist_ok=True); output = []
    plots = [("throughput_vs_rate.png", "requested_rate", "throughput_mean", "Requested rate vs completed throughput", "requested sub/s", "completed sub/s"),
             ("p95_latency_vs_rate.png", "requested_rate", "p95_mean", "Requested rate vs E2E p95", "requested sub/s", "milliseconds"),
             ("p99_latency_vs_rate.png", "requested_rate", "p99_mean", "Requested rate vs E2E p99", "requested sub/s", "milliseconds"),
             ("cpu_vs_throughput.png", "throughput_mean", "judge_cpu_p95_mean", "Completed throughput vs Judge CPU", "completed sub/s", "CPU %"),
             ("memory_vs_throughput.png", "throughput_mean", "judge_memory_max_mean", "Completed throughput vs Judge memory", "completed sub/s", "MiB")]
    for filename, x, y, title, xlabel, ylabel in plots:
        view = experiments.dropna(subset=[x, y])
        if not len(view): continue
        plt.figure(figsize=(7, 4.5));
        for label, group in view.groupby("experiment_label"):
            plt.plot(group[x], group[y], marker="o", label=label)
        plt.title(title); plt.xlabel(xlabel); plt.ylabel(ylabel); plt.legend()
        path = charts / filename; _save(path); output.append(path.name)
    lag = experiments.dropna(subset=["requested_rate", "kafka_lag_end_mean"])
    if len(lag):
        plt.figure(figsize=(7, 4.5)); plt.plot(lag.requested_rate, lag.kafka_lag_end_mean, marker="o"); plt.title("Requested rate vs Kafka lag at end"); plt.xlabel("requested sub/s"); plt.ylabel("messages")
        path = charts / "kafka_lag_vs_rate.png"; _save(path); output.append(path.name)
    efficiency = experiments.dropna(subset=["requested_rate", "throughput_mean"]).copy()
    if len(efficiency):
        efficiency["efficiency"] = efficiency.throughput_mean / efficiency.requested_rate.replace(0, np.nan)
        plt.figure(figsize=(7, 4.5)); plt.plot(efficiency.requested_rate, efficiency.efficiency, marker="o"); plt.title("Throughput efficiency"); plt.xlabel("requested sub/s"); plt.ylabel("completed / requested")
        path = charts / "throughput_efficiency.png"; _save(path); output.append(path.name)
        plt.figure(figsize=(7, 4.5)); plt.plot(efficiency.requested_rate, efficiency.throughput_mean, marker="o", label="observed"); plt.title("Capacity frontier from tested configurations"); plt.xlabel("requested sub/s"); plt.ylabel("completed sub/s"); plt.legend()
        path = charts / "capacity_frontier.png"; _save(path); output.append(path.name)
    return output

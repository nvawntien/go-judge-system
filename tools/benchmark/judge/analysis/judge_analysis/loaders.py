"""Load immutable Go-harness artifacts and optional collector CSVs."""

from __future__ import annotations
import hashlib
import json
from dataclasses import dataclass, field
from pathlib import Path
import pandas as pd
from .schemas import REQUIRED_SUBMISSION_COLUMNS, REQUIRED_WINDOW_COLUMNS, CONTAINER_COLUMNS, KAFKA_COLUMNS, CLIENT_RESOURCE_COLUMNS


class DataError(ValueError):
    pass


@dataclass
class RunData:
    path: Path
    run: dict
    submissions: pd.DataFrame
    windows: pd.DataFrame
    summary: dict | None
    containers: pd.DataFrame | None = None
    kafka: pd.DataFrame | None = None
    client_resources: pd.DataFrame | None = None
    quality: list[str] = field(default_factory=list)


def sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def _json(path: Path) -> dict:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        raise DataError(f"invalid required JSON artifact {path.name}") from error


def _csv(path: Path, required: set[str], kind: str) -> pd.DataFrame:
    try:
        frame = pd.read_csv(path)
    except (OSError, pd.errors.ParserError) as error:
        raise DataError(f"invalid {kind} CSV") from error
    missing = required - set(frame.columns)
    if missing:
        raise DataError(f"{kind} CSV missing required columns: {', '.join(sorted(missing))}")
    return frame


def _time(frame: pd.DataFrame, columns: list[str], kind: str) -> None:
    for column in columns:
        if column not in frame:
            continue
        # Harness CSVs can mix empty optional fields with RFC3339 values that
        # carry different fractional-second precision; parse each ISO value.
        parsed = pd.to_datetime(frame[column], utc=True, errors="coerce", format="mixed")
        nonempty = frame[column].notna() & frame[column].astype(str).ne("")
        if parsed[nonempty].isna().any():
            raise DataError(f"{kind} contains invalid UTC timestamp in {column}")
        frame[column] = parsed


def _number(frame: pd.DataFrame, columns: list[str], kind: str) -> None:
    for column in columns:
        if column in frame:
            converted = pd.to_numeric(frame[column], errors="coerce")
            if frame[column].notna().any() and converted[frame[column].notna()].isna().any():
                raise DataError(f"{kind} contains invalid numeric value in {column}")
            frame[column] = converted


def _boolean(frame: pd.DataFrame, columns: list[str], kind: str) -> None:
    for column in columns:
        if column not in frame:
            continue
        values = frame[column].astype(str).str.lower()
        allowed = values.isin({"true", "false"})
        if not allowed.all():
            raise DataError(f"{kind} contains invalid boolean value in {column}")
        frame[column] = values.eq("true")


def _optional(run_dir: Path, explicit: str | None, default: str, columns: set[str], kind: str) -> pd.DataFrame | None:
    path = Path(explicit) if explicit else run_dir / default
    if not path.exists():
        return None
    frame = _csv(path, columns, kind)
    _time(frame, ["timestamp"], kind)
    return frame


def load_run(run_dir: str | Path, container_stats: str | None = None, kafka_lag: str | None = None) -> RunData:
    path = Path(run_dir).resolve()
    if not path.is_dir():
        raise DataError("run directory does not exist")
    run, submissions_path, windows_path = _json(path / "run.json"), path / "submissions.csv", path / "windows.csv"
    if not isinstance(run.get("run_id"), str) or not run["run_id"]:
        raise DataError("run.json has no valid run_id")
    for name in ["started_at", "ended_at"]:
        value = run.get(name)
        if value is not None and pd.isna(pd.to_datetime(value, utc=True, errors="coerce", format="mixed")):
            raise DataError(f"run.json has invalid UTC {name}")
    submissions = _csv(submissions_path, REQUIRED_SUBMISSION_COLUMNS, "submissions")
    windows = _csv(windows_path, REQUIRED_WINDOW_COLUMNS, "windows")
    _time(submissions, ["intended_at", "post_started_at", "post_completed_at", "terminal_observed_at", "ticket_started_at", "ticket_completed_at", "sse_started_at", "sse_established_at", "sse_closed_at"], "submissions")
    _time(windows, ["window_start", "window_end"], "windows")
    _boolean(submissions, ["attempted", "accepted", "rate_limited", "ticket_attempted", "ticket_succeeded", "sse_attempted", "sse_established", "sse_terminal_during_hold", "sse_survived_full_hold"], "submissions")
    _number(submissions, ["sequence", "submission_id", "submit_latency_ms", "end_to_end_latency_ms", "accepted_to_terminal_ms", "ticket_latency_ms", "sse_establishment_latency_ms"], "submissions")
    _number(windows, ["window_index", "window_duration_ms", "intended", "attempted", "accepted", "completed", "accepted_cumulative", "completed_cumulative", "client_outstanding", "client_outstanding_peak", "attempted_rate_per_sec", "accepted_rate_per_sec", "completion_rate_per_sec"], "windows")
    if submissions.duplicated(["sequence"], keep=False).any():
        raise DataError("duplicate submission sequence records")
    if (submissions[["submit_latency_ms", "end_to_end_latency_ms", "accepted_to_terminal_ms"]] < 0).any().any():
        raise DataError("negative latency in submissions")
    # The Go harness restarts the window index for the drain phase, so monotonic
    # indexing is required within each phase rather than globally.
    for phase, group in windows.groupby("phase"):
        ordered = group.sort_values("window_start")
        if ordered["window_index"].duplicated().any() or not ordered["window_index"].is_monotonic_increasing:
            raise DataError(f"windows have non-monotonic or duplicate indexes in {phase}")
    for field in ["accepted_cumulative", "completed_cumulative", "client_outstanding"]:
        if (windows[field] < 0).any():
            raise DataError(f"impossible negative {field}")
    summary_path = path / "summary.json"
    summary = _json(summary_path) if summary_path.exists() else None
    containers = _optional(path, container_stats, "container-stats.csv", CONTAINER_COLUMNS, "container statistics")
    kafka = _optional(path, kafka_lag, "kafka-lag.csv", KAFKA_COLUMNS, "Kafka lag")
    client_resources = _optional(path, None, "client_resources.csv", CLIENT_RESOURCE_COLUMNS, "client resource")
    if containers is not None:
        _number(containers, ["cpu_percent", "memory_bytes", "memory_limit_bytes", "memory_percent", "pids", "restart_count"], "container statistics")
    if kafka is not None:
        _number(kafka, ["partition", "current_offset", "log_end_offset", "lag"], "Kafka lag")
    if client_resources is not None:
        _number(client_resources, ["open_fds", "goroutines", "active_posts", "active_tickets", "active_sse_streams"], "client resource")
    return RunData(path, run, submissions, windows, summary, containers, kafka, client_resources)

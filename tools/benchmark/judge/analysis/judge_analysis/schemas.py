"""Schema constants shared by loaders and validation."""

RUN_SCHEMA_VERSION = "judge-analysis/v1"
REQUIRED_SUBMISSION_COLUMNS = {
    "run_id", "phase", "sequence", "intended_at", "post_started_at",
    "post_completed_at", "attempted", "accepted", "submission_id",
    "terminal_status", "terminal_observed_at", "submit_latency_ms",
    "end_to_end_latency_ms", "accepted_to_terminal_ms", "completion_source",
    "rate_limited", "outcome", "error_class",
}
REQUIRED_WINDOW_COLUMNS = {
    "run_id", "phase", "window_index", "window_start", "window_end",
    "window_duration_ms", "intended", "attempted", "accepted", "completed",
    "accepted_cumulative", "completed_cumulative", "client_outstanding",
    "client_outstanding_peak", "attempted_rate_per_sec", "accepted_rate_per_sec",
    "completion_rate_per_sec",
}
CONTAINER_COLUMNS = {"timestamp", "node", "container", "cpu_percent", "memory_bytes", "pids"}
KAFKA_COLUMNS = {"timestamp", "consumer_group", "topic", "partition", "current_offset", "log_end_offset", "lag"}

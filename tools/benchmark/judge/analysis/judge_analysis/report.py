"""Portable evidence-first HTML reports with no network dependencies."""

from __future__ import annotations
import html
from pathlib import Path


def _text(value, fallback="unavailable"):
    return fallback if value is None or value == "" else str(value)


def _number(value, suffix="", precision=3):
    return "unavailable" if value is None else f"{float(value):.{precision}f}{suffix}"


def _latency(value):
    if value is None:
        return "unavailable"
    return _number(value, " ms", 1) if abs(float(value)) < 1000 else _number(float(value) / 1000, " s", 3)


def _percent(value):
    return "unavailable" if value is None else _number(float(value) * 100, "%", 1)


def _table(rows, headers=("Metric", "Value")):
    head = "".join(f"<th>{html.escape(str(value))}</th>" for value in headers)
    body = "".join("<tr>" + "".join(f"<td>{html.escape(_text(value))}</td>" for value in row) + "</tr>" for row in rows)
    return f"<table><thead><tr>{head}</tr></thead><tbody>{body}</tbody></table>"


def _cards(values):
    return "<div class=cards>" + "".join(f"<article><small>{html.escape(label)}</small><strong>{html.escape(_text(value))}</strong></article>" for label, value in values) + "</div>"


def _page(title, body):
    return f'''<!doctype html><html><head><meta charset="utf-8"><title>{html.escape(title)}</title><style>
body{{font-family:system-ui,sans-serif;margin:2rem;color:#17202a;background:#f7fafc;max-width:1400px}}h1,h2,h3{{color:#243b53}}.cards{{display:grid;grid-template-columns:repeat(auto-fit,minmax(155px,1fr));gap:.8rem}}article,section,figure{{background:#fff;border:1px solid #d9e2ec;border-radius:8px;padding:1rem;margin:1rem 0}}article strong{{display:block;font-size:1.15rem;margin-top:.3rem}}table{{border-collapse:collapse;width:100%}}td,th{{padding:.4rem .65rem;border:1px solid #d9e2ec;text-align:left}}th{{background:#edf2f7}}figure{{display:inline-block;vertical-align:top;width:46%}}img{{max-width:100%;height:auto}}.warning{{color:#9c2f00}}</style></head><body>{body}</body></html>'''


def _latency_table(metrics):
    headers = ["Latency family", "count", "min", "mean", "std", "CV", "p50", "p90", "p95", "p99", "max"]
    rows = []
    for label, key in (("Submit / API", "submit"), ("Judge end-to-end", "end_to_end"), ("Accepted to terminal", "accepted_to_terminal")):
        data = metrics["latency_ms"][key]
        row = [label, data.get("count"), *[_latency(data.get(field)) for field in ("min", "mean", "std")], _number(data.get("cv"), "", 3), *[_latency(data.get(field)) for field in ("p50", "p90", "p95", "p99", "max")]]
        rows.append(row)
    return _table(rows, headers)


def _system_config(metrics, run):
    cfg = metrics["system_config"]
    repository = run.get("repository", {})
    if not cfg.get("available"):
        system = [("System config", "unavailable"), ("Reason", cfg.get("reason"))]
    else:
        app, judge = cfg["app"], cfg["judge"]
        system = [("Config label", cfg["label"]), ("Release", cfg["release"]), ("App", f"{app['nodes']} node(s), {app['cpu_cores_per_node']} cores/node, {app['memory_mib_per_node']} MiB/node"), ("Judge", f"{judge['nodes']} node(s), {judge['cpu_cores_per_node']} cores/node, {judge['memory_mib_per_node']} MiB/node"), ("WORKER_POOL_SIZE", judge["worker_pool_size"]), ("Judge worker memory limit", f"{judge['worker_memory_limit_mib']} MiB"), ("Sandbox memory limit", f"{judge['sandbox_memory_limit_mib']} MiB")]
    target, workload, users = run.get("target", {}), run.get("workload", {}), run.get("users", {})
    reproducibility = [("Git SHA", repository.get("git_sha")), ("Working tree", "dirty" if repository.get("dirty") else "clean"), ("Target", target.get("base_url")), ("Problem", f"{target.get('problem_id', 'unavailable')} / {target.get('problem_slug', 'unavailable')}"), ("Language", target.get("language")), ("Expected verdict", target.get("expected_verdict")), ("Mode", run.get("mode")), ("Requested rate", _number(workload.get("target_rate_per_second"), " sub/s")), ("Measured volume", _text(workload.get("total_submissions"), "duration-based")), ("Burst size", workload.get("burst_size")), ("Duration", _number((workload.get("arrival_duration_ms") or 0) / 1000, " s")), ("Users selected", users.get("selected")), ("Max submissions", workload.get("max_submissions")), ("Max in flight", workload.get("max_in_flight")), ("Warmup count", workload.get("warmup_count")), ("Repetition", run.get("repetition"))]
    return _table(reproducibility), _table(system)


def render_run(path: Path, metrics: dict, charts: list[str], run: dict | None = None) -> None:
    run = run or {}
    e2e, load, completion = metrics["latency_ms"]["end_to_end"], metrics["load"], metrics["completion"]
    mode = run.get("mode", "unavailable")
    burst = metrics.get("burst") if mode == "burst" else None
    massive_burst = bool(burst) and run.get("benchmark_objective") == "massive-burst"
    throughput = "not applicable for burst" if mode == "burst" else _number(completion.get("completion_throughput_per_sec"), " sub/s")
    attempted_intake = _number((burst or {}).get("attempted", {}).get("throughput_per_sec"), " sub/s")
    accepted_intake = _number((burst or {}).get("accepted", {}).get("throughput_per_sec"), " sub/s")
    terminal_intake = _number((burst or {}).get("terminal", {}).get("throughput_per_sec"), " sub/s")
    cards = [("Run ID", metrics.get("run_id")), ("Mode", mode), ("Run state", run.get("state")), ("Harness classification", metrics.get("harness_classification")), ("Analytical assessment", metrics["analytical_assessment"]["state"]), ("Requested arrival", _number(run.get("workload", {}).get("target_rate_per_second"), " sub/s")), ("Accepted arrival", _number(load.get("accepted_arrival_rate_per_sec"), " sub/s")), ("Judge throughput", throughput), ("Submit p95", _latency(metrics["latency_ms"]["submit"].get("p95"))), ("E2E p50", _latency(e2e.get("p50"))), ("E2E p95", _latency(e2e.get("p95"))), ("E2E p99", _latency(e2e.get("p99"))), ("AC ratio", _percent(metrics["correctness"].get("accepted_ratio"))), ("Kafka lag end", _number(metrics["kafka"].get("lag_at_end"))), ("Data quality", metrics["data_quality"]["state"])]
    if massive_burst:
        cards = [("Burst size", run.get("workload", {}).get("burst_size")), ("Distinct accounts", run.get("users", {}).get("selected")), ("Effective attempted intake", attempted_intake), ("Effective accepted intake", accepted_intake), ("Judge terminal throughput", terminal_intake)] + cards
    config_table, system_table = _system_config(metrics, run)
    queue = metrics["queue"]; errors = metrics["correctness"]["errors"]
    load_rows = [("Intended", load["intended"]), ("Attempted", load["attempted"]), ("Accepted", load["accepted"]), ("Completed", completion["completed"]), ("Actual accepted rate", _number(load["accepted_arrival_rate_per_sec"], " sub/s")), ("Completed throughput", throughput), ("Completion ratio", _percent(completion["completion_ratio"]))]
    if burst:
        load_rows += [("Actual POST-start interval", _number(burst["attempted"].get("interval_ms"), " ms")), ("Actual accepted intake interval", _number(burst["accepted"].get("interval_ms"), " ms")), ("Effective attempted throughput", attempted_intake), ("Effective accepted throughput", accepted_intake), ("Effective terminal throughput", terminal_intake), ("Peak logical in-flight", burst.get("peak_logical_in_flight")), ("Peak active SSE observers", burst.get("peak_active_observers"))]
    load_table = _table(load_rows)
    queue_table = _table([("Peak client outstanding", queue["max_client_outstanding"]), ("Outstanding at load end", queue["ending_client_outstanding"]), ("Outstanding slope", _number(queue.get("outstanding_slope_per_minute"), " /min")), ("Drain duration", _number((queue.get("drain_duration_ms") or 0) / 1000, " s"))])
    error_table = _table([("429", errors["rate_limited"]), ("Other HTTP errors", errors.get("http_errors")), ("5xx", errors["server_errors"]), ("Transport failures", errors["transport_failures"]), ("Completion timeouts", errors["completion_timeouts"]), ("SSE completions", metrics["observation"]["sse_completions"]), ("SSE failures", metrics["observation"]["sse_failures"]), ("GET reconciliations", metrics["observation"]["reconciliation_completions"])])
    verdicts = _table([(name, count) for name, count in metrics["correctness"]["verdicts"].items()], ("Verdict", "Count"))
    kafka = metrics["kafka"]; kafka_table = _table([("Availability", "available" if kafka.get("available") else kafka.get("reason")), ("Lag max", _number(kafka.get("total_lag", {}).get("max"))), ("Lag end", _number(kafka.get("lag_at_end"))), ("Lag slope", _number(kafka.get("lag_growth_slope_per_second"), " msg/s")), ("Returns to baseline", kafka.get("returns_to_baseline"))])
    resources = metrics["resources"]
    resource_rows = [("Availability", "available" if resources.get("available") else resources.get("reason"))]
    for name in ("judge_worker", "judge_sandbox", "judge_submission", "judge_kafka", "judge_postgres", "judge_redis"):
        item = resources.get("containers", {}).get(name)
        if item: resource_rows.append((name, f"CPU mean/p95/max {_number(item['cpu_percent'].get('mean'),'%',1)} / {_number(item['cpu_percent'].get('p95'),'%',1)} / {_number(item['cpu_percent'].get('max'),'%',1)}; memory mean/p95/max {_number(item['memory_mib'].get('mean'),' MiB',1)} / {_number(item['memory_mib'].get('p95'),' MiB',1)} / {_number(item['memory_mib'].get('max'),' MiB',1)}"))
    captions = {"01_latency_distribution.png": "Distribution shape with p50, p95, and p99 markers.", "02_latency_ecdf.png": "Fraction of completions at or below each latency.", "03_latency_over_time.png": "Individual end-to-end latency observations over time.", "04_throughput_over_time.png": "Target, accepted, and completion rates by benchmark window.", "05_client_outstanding.png": "Client-observed accepted work not yet terminal; this is not Kafka lag.", "06_kafka_lag.png": "Kafka consumer-group lag when collector data is available."}
    chart_html = "".join(f'<figure><img src="charts/{html.escape(name)}" alt="{html.escape(name)}"><figcaption>{html.escape(captions.get(name, name))}</figcaption></figure>' for name in charts)
    quality = metrics["data_quality"]
    heading = "ASTRACODE MASSIVE SUBMISSION BURST" if massive_burst else "AstraCode Judge Benchmark"
    body = f"<h1>{heading}</h1><p>Executive result first; detailed evidence below. Raw CSV/JSON artifacts remain authoritative.</p>{_cards(cards)}<section><h2>Test configuration</h2><h3>Benchmark workload and target</h3>{config_table}<h3>System under test</h3>{system_table}</section><section><h2>Data quality: {html.escape(quality['state'])}</h2><p>{html.escape('; '.join(quality['reasons']) or 'Required artifacts and available collectors were valid.')}</p></section><section><h2>Latency detail</h2><p>p95 means 95% of observations completed no slower than the displayed value; p99 exposes tail latency. Max is not a percentile.</p>{_latency_table(metrics)}</section><section><h2>Load, queue, and correctness</h2><h3>Load</h3>{load_table}<h3>Queue</h3>{queue_table}<h3>Errors and observation</h3>{error_table}<h3>Verdicts</h3>{verdicts}</section><section><h2>Kafka and resources</h2><h3>Kafka</h3>{kafka_table}<h3>Important containers</h3>{_table(resource_rows)}</section><section><h2>Charts</h2>{chart_html or '<p>Charts unavailable because the associated valid data was unavailable.</p>'}</section>"
    path.write_text(_page(f"Judge benchmark {metrics['run_id']}", body), encoding="utf-8")


def render_comparison(path: Path, experiments: list[dict], charts: list[str], capacity: dict | None = None, individual_runs: list[dict] | None = None) -> None:
    capacity = capacity or {}; individual_runs = individual_runs or []
    is_burst = bool(experiments) and all(item.get("comparison_kind") == "burst" for item in experiments)
    if is_burst:
        headers = ["Configuration", "Burst size", "N", "Effective accepted", "Submit p50", "Submit p95", "Submit p99", "Peak outstanding", "Kafka max/end", "Judge CPU p95", "Judge RAM max", "Errors"]
        rows = [[item.get("configuration_label"), item.get("burst_size"), item.get("repetitions"), _number(item.get("throughput_mean"), " sub/s"), _latency(item.get("p50_mean")), _latency(item.get("p95_mean")), _latency(item.get("p99_mean")), _number(item.get("outstanding_peak_mean")), f"{_number(item.get('kafka_lag_max_mean'))} / {_number(item.get('kafka_lag_end_mean'))}", _number(item.get("judge_cpu_p95_mean"), "%", 1), _number(item.get("judge_memory_max_mean"), " MiB", 1), _number(item.get("error_count_mean"))] for item in experiments]
        individual = _table([[item.get(key) for key in ("run_id", "burst_size", "effective_accepted_throughput", "submit_p50", "submit_p95", "submit_p99", "error_count", "kafka_lag_end", "peak_outstanding", "judge_cpu_p95", "judge_memory_max", "assessment")] for item in individual_runs], ("Run ID", "Burst", "Effective accepted", "Submit p50", "Submit p95", "Submit p99", "Errors", "Kafka end", "Peak outstanding", "Judge CPU p95", "Judge RAM max", "E2E assessment"))
        images = "".join(f'<figure><img src="charts/{html.escape(chart)}" alt="{html.escape(chart)}"><figcaption>{html.escape(chart.replace("_", " ").replace(".png", ""))}</figcaption></figure>' for chart in charts)
        body = f"<h1>ASTRACODE MASSIVE SUBMISSION BURST</h1><p>{html.escape(capacity.get('message', 'Burst intake is measured from actual client timestamps.'))}</p><section><h2>Burst evidence</h2>{_table(rows, headers)}</section><section><h2>Individual-run evidence</h2>{individual}</section><section><h2>Burst charts</h2>{images or '<p>Charts unavailable because required data was unavailable.</p>'}</section>"
        path.write_text(_page("AstraCode Massive Submission Burst", body), encoding="utf-8")
        return
    cards = [("Maximum demonstrated stable", _number(capacity.get("demonstrated_stable_rate"), " sub/s")), ("First tested saturation", _number(capacity.get("first_saturating_rate"), " sub/s")), ("Saturation interval", capacity.get("interval")), ("Data quality", capacity.get("state"))]
    headers = ["Configuration", "Requested", "N", "Accepted mean", "Completed mean", "Std", "95% CI", "E2E p50", "E2E p95", "E2E p99", "Peak outstanding", "Slope", "Kafka max/end", "Judge CPU p95", "Judge RAM max", "Errors", "Assessment"]
    rows = []
    for item in experiments:
        ci = "CI unavailable: N=1" if item.get("repetitions") == 1 else f"[{_number(item.get('throughput_ci95_low'))}, {_number(item.get('throughput_ci95_high'))}]"
        rows.append([item.get("configuration_label"), _number(item.get("requested_rate"), " sub/s"), item.get("repetitions"), _number(item.get("accepted_rate_mean"), " sub/s"), _number(item.get("throughput_mean"), " sub/s"), _number(item.get("throughput_std")), ci, _latency(item.get("p50_mean")), _latency(item.get("p95_mean")), _latency(item.get("p99_mean")), _number(item.get("outstanding_peak_mean")), _number(item.get("outstanding_slope_mean"), " /min"), f"{_number(item.get('kafka_lag_max_mean'))} / {_number(item.get('kafka_lag_end_mean'))}", _number(item.get("judge_cpu_p95_mean"), "%", 1), _number(item.get("judge_memory_max_mean"), " MiB", 1), _number(item.get("error_count_mean")), item.get("assessment")])
    individual = _table([[item.get(key) for key in ("run_id", "requested_rate", "completion_throughput", "e2e_p50", "e2e_p95", "e2e_p99", "error_count", "kafka_lag_end", "peak_outstanding", "judge_cpu_p95", "judge_memory_max", "assessment")] for item in individual_runs], ("Run ID", "Requested", "Completed", "E2E p50", "E2E p95", "E2E p99", "Errors", "Kafka end", "Peak outstanding", "Judge CPU p95", "Judge RAM max", "Assessment"))
    images = "".join(f'<figure><img src="charts/{html.escape(chart)}" alt="{html.escape(chart)}"><figcaption>{html.escape(chart.replace("_", " ").replace(".png", ""))}</figcaption></figure>' for chart in charts)
    body = f"<h1>AstraCode Judge Capacity</h1><p>{html.escape(capacity.get('message', 'Repeated sustained experiments are required for capacity evidence.'))}</p>{_cards(cards)}<section><h2>Judge capacity evidence</h2>{_table(rows, headers)}</section><section><h2>Individual-run evidence</h2>{individual}</section><section><h2>Capacity charts</h2>{images or '<p>Charts unavailable because required data was unavailable.</p>'}</section>"
    path.write_text(_page("AstraCode Judge Capacity", body), encoding="utf-8")


def render_api_comparison(path: Path, experiments: list[dict], capacity: dict, individual_runs: list[dict], thresholds: dict, charts: list[str]) -> None:
    cards = [("Maximum demonstrated stable API", _number(capacity.get("demonstrated_stable_rate"), " req/s")), ("First tested saturation", _number(capacity.get("first_saturating_rate"), " req/s")), ("Saturation interval", capacity.get("interval")), ("Data quality", capacity.get("state"))]
    rows = []
    for item in experiments:
        ci = "CI unavailable: N=1" if item.get("repetitions") == 1 else f"[{_number(item.get('achieved_ci95_low'))}, {_number(item.get('achieved_ci95_high'))}]"
        rows.append([item.get("configuration_label"), item.get("endpoint"), _number(item.get("requested_rate"), " req/s"), item.get("repetitions"), _number(item.get("achieved_mean"), " req/s"), _number(item.get("achieved_std")), ci, _latency(item.get("p50_mean")), _latency(item.get("p95_mean")), _latency(item.get("p99_mean")), _percent(item.get("error_rate_mean")), _number(item.get("dropped_iterations_mean")), item.get("assessment")])
    individual = _table([[item.get(key) for key in ("run_id", "requested_rate", "achieved_rps", "p50", "p95", "p99", "error_rate", "dropped_iterations", "assessment")] for item in individual_runs], ("Run ID", "Requested", "Achieved", "p50", "p95", "p99", "Error rate", "Dropped", "Assessment"))
    images = "".join(f'<figure><img src="charts/{html.escape(chart)}" alt="{html.escape(chart)}"><figcaption>{html.escape(chart.replace("_", " ").replace(".png", ""))}</figcaption></figure>' for chart in charts)
    body = f"<h1>AstraCode API Capacity</h1><p>{html.escape(capacity.get('message', 'Repeated API experiments are required for capacity evidence.'))}</p>{_cards(cards)}<section><h2>API capacity evidence</h2><p>Stable requires achieved/requested ≥ {thresholds['minimum_achieved_requested_ratio']}, error rate ≤ {thresholds['maximum_error_rate']}, and dropped iterations ≤ {thresholds['maximum_dropped_iterations']}.</p>{_table(rows, ('Configuration','Endpoint','Requested','N','Achieved mean','Std','95% CI','p50','p95','p99','Error rate','Dropped','Assessment'))}</section><section><h2>Individual API runs</h2>{individual}</section><section><h2>API capacity charts</h2>{images or '<p>Charts unavailable because valid source series were unavailable.</p>'}</section>"
    path.write_text(_page("AstraCode API Capacity", body), encoding="utf-8")


def render_unified(path: Path, judge: dict | None, api: dict | None) -> None:
    judge_capacity = (judge or {}).get("capacity_evidence", {})
    api_capacity = (api or {}).get("capacity_evidence", {})
    stable_judge = _number(judge_capacity.get("demonstrated_stable_rate"), " submissions/s") if judge else "unavailable"
    stable_api = _number(api_capacity.get("demonstrated_stable_rate"), " req/s") if api else "unavailable"
    judge_experiments = (judge or {}).get("experiments", [])
    api_experiments = (api or {}).get("experiments", [])
    stable_row = next((row for row in judge_experiments if row.get("assessment") == "STABLE" and row.get("requested_rate") == judge_capacity.get("demonstrated_stable_rate")), None)
    judge_rows = [[row.get("configuration_label"), row.get("requested_rate"), row.get("throughput_mean"), row.get("p50_mean"), row.get("p95_mean"), row.get("p99_mean"), row.get("assessment")] for row in judge_experiments]
    api_rows = [[row.get("configuration_label"), row.get("requested_rate"), row.get("achieved_mean"), row.get("p50_mean"), row.get("p95_mean"), row.get("p99_mean"), row.get("assessment")] for row in api_experiments]
    configs = []
    for row in judge_experiments + api_experiments:
        config = row.get("system_config")
        if isinstance(config, dict) and config not in configs:
            configs.append(config)
    config_rows = []
    for config in configs:
        app, judge_config = config.get("app", {}), config.get("judge", {})
        config_rows.append([config.get("label"), config.get("release"), f"{app.get('nodes')} nodes; {app.get('cpu_cores_per_node')} cores/node; {app.get('memory_mib_per_node')} MiB/node", f"{judge_config.get('nodes')} nodes; {judge_config.get('cpu_cores_per_node')} cores/node; {judge_config.get('memory_mib_per_node')} MiB/node; pool {judge_config.get('worker_pool_size')}; worker/sandbox {judge_config.get('worker_memory_limit_mib')}/{judge_config.get('sandbox_memory_limit_mib')} MiB"])
    final_cards = [('API maximum demonstrated sustainable throughput', stable_api), ('Judge maximum demonstrated sustainable throughput', stable_judge), ('Judge E2E p50 at max stable', _latency((stable_row or {}).get('p50_mean'))), ('Judge E2E p95 at max stable', _latency((stable_row or {}).get('p95_mean'))), ('Judge E2E p99 at max stable', _latency((stable_row or {}).get('p99_mean'))), ('Judge saturation interval', judge_capacity.get('interval')), ('API saturation interval', api_capacity.get('interval'))]
    body = f"<h1>AstraCode Performance Capacity</h1><p>Evidence-oriented capacity report. Missing benchmark families are unavailable, never zero.</p>{_cards(final_cards)}<section><h2>System under test</h2>{_table(config_rows, ('Configuration','Release','App','Judge')) if config_rows else '<p>Safe system configuration metadata unavailable.</p>'}<p>Each evidence row retains its safe configuration label. Runs with different system metadata are not grouped together.</p></section><section><h2>API capacity evidence</h2>{_table(api_rows, ('Configuration','Requested req/s','Achieved req/s','p50','p95','p99','Assessment')) if api else '<p>API capacity: unavailable.</p>'}</section><section><h2>Judge capacity evidence</h2>{_table(judge_rows, ('Configuration','Requested sub/s','Completed sub/s','E2E p50','E2E p95','E2E p99','Assessment')) if judge else '<p>Judge capacity: unavailable.</p>'}</section><section><h2>Conclusion</h2><p>Only tested stable and saturating boundaries are reported. This report does not manufacture a precise capacity number.</p></section>"
    path.write_text(_page("AstraCode Performance Capacity", body), encoding="utf-8")

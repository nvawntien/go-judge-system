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
    for label, key in (("Submit / API", "submit"), ("Pipeline E2E (observed terminals)", "end_to_end"), ("Accepted to terminal (observed)", "accepted_to_terminal")):
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
    if (metrics.get("admission") or {}).get("observation_mode") == "admission-only":
        _render_admission_run(path, metrics, charts, run)
        return
    if (metrics.get("realistic") or {}).get("observation_mode") == "realistic":
        _render_realistic_run(path, metrics, charts, run)
        return
    e2e, load, pipeline = metrics["latency_ms"]["end_to_end"], metrics["load"], metrics["pipeline"]
    mode = run.get("mode", "unavailable")
    burst = metrics.get("burst") if mode == "burst" else None
    massive_burst = bool(burst) and run.get("benchmark_objective") == "massive-burst"
    pipeline_rate = _number(pipeline.get("terminal_throughput_per_sec"), " sub/s")
    attempted_intake = _number((burst or {}).get("attempted", {}).get("throughput_per_sec"), " sub/s")
    accepted_intake = _number((burst or {}).get("accepted", {}).get("throughput_per_sec"), " sub/s")
    terminal_intake = _number((burst or {}).get("terminal", {}).get("throughput_per_sec"), " sub/s")
    core = metrics["judge_core"]
    cards = [("Run ID", metrics.get("run_id")), ("Mode", mode), ("Run state", run.get("state")), ("Harness classification", metrics.get("harness_classification")), ("Analytical assessment", metrics["analytical_assessment"]["state"]), ("Requested arrival", _number(run.get("workload", {}).get("target_rate_per_second"), " sub/s")), ("Accepted arrival", _number(load.get("accepted_arrival_rate_per_sec"), " sub/s")), ("Judge Core throughput", f"N/A — {core.get('reason')}" if core.get("availability") != "AVAILABLE" else _number(core.get("throughput_per_sec"), " sub/s")), ("Pipeline terminal rate", pipeline_rate), ("Submit p95", _latency(metrics["latency_ms"]["submit"].get("p95"))), ("E2E p50", _latency(e2e.get("p50"))), ("E2E p95", _latency(e2e.get("p95"))), ("E2E p99", _latency(e2e.get("p99"))), ("AC ratio", _percent(metrics["correctness"].get("accepted_ratio"))), ("Kafka lag end", _number(metrics["kafka"].get("lag_at_end"))), ("Data quality", metrics["data_quality"]["state"])]
    if massive_burst:
        cards = [("Burst size", run.get("workload", {}).get("burst_size")), ("Distinct accounts", run.get("users", {}).get("selected")), ("Effective attempted intake", attempted_intake), ("Effective accepted intake", accepted_intake), ("Observed pipeline terminal rate", terminal_intake)] + cards
    config_table, system_table = _system_config(metrics, run)
    queue = metrics["queue"]; errors = metrics["correctness"]["errors"]
    load_rows = [("Intended", load["intended"]), ("Attempted", load["attempted"]), ("Accepted", load["accepted"]), ("Terminal observed", pipeline["terminal_completed"]), ("Actual accepted rate", _number(load["accepted_arrival_rate_per_sec"], " sub/s")), ("Observed pipeline-terminal rate", pipeline_rate), ("Terminal observation coverage", _percent(pipeline["terminal_observation_coverage"]))]
    if burst:
        load_rows += [("Actual POST-start interval", _number(burst["attempted"].get("interval_ms"), " ms")), ("Actual accepted intake interval", _number(burst["accepted"].get("interval_ms"), " ms")), ("Effective attempted throughput", attempted_intake), ("Effective accepted throughput", accepted_intake), ("Observed pipeline terminal rate", terminal_intake), ("Peak logical in-flight", burst.get("peak_logical_in_flight")), ("Peak active SSE observers", burst.get("peak_active_observers"))]
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
    censor_note = "Terminal/E2E observations are right-censored; displayed E2E tails cover observed terminal samples only." if pipeline.get("right_censored") else "Terminal/E2E observations cover all accepted submissions observed by this run."
    core_table = _table([("Compile included in Judge Core", metrics["compile"].get("included_in_judge_core")), ("Compile overhead", f"{metrics['compile'].get('availability')} — {metrics['compile'].get('reason')}"), ("Judge Core definition", core.get("definition")), ("Judge Core", f"{core.get('availability')} — {core.get('reason')}")])
    body = f"<h1>{heading}</h1><p>Executive result first; detailed evidence below. Raw CSV/JSON artifacts remain authoritative.</p>{_cards(cards)}<section><h2>Test configuration</h2><h3>Benchmark workload and target</h3>{config_table}<h3>System under test</h3>{system_table}</section><section><h2>Compile and Judge Core</h2>{core_table}<p>Compile is never included in Judge Core latency or throughput. This harness has no phase timestamps to calculate Judge Core from terminal observations.</p></section><section><h2>Data quality: {html.escape(quality['state'])}</h2><p>{html.escape('; '.join(quality['reasons']) or 'Required artifacts and available collectors were valid.')}</p><p class=warning>{html.escape(censor_note)}</p></section><section><h2>Pipeline E2E latency detail</h2><p>p95 means 95% of observed terminal samples completed no slower than the displayed value; p99 exposes their tail. Max is not a percentile.</p>{_latency_table(metrics)}</section><section><h2>Load, queue, and correctness</h2><h3>Load</h3>{load_table}<h3>Queue</h3>{queue_table}<h3>Errors and observation</h3>{error_table}<h3>Verdicts</h3>{verdicts}</section><section><h2>Kafka and resources</h2><h3>Kafka</h3>{kafka_table}<h3>Important containers</h3>{_table(resource_rows)}</section><section><h2>Charts</h2>{chart_html or '<p>Charts unavailable because the associated valid data was unavailable.</p>'}</section>"
    path.write_text(_page(f"Judge benchmark {metrics['run_id']}", body), encoding="utf-8")


def _render_admission_run(path: Path, metrics: dict, charts: list[str], run: dict) -> None:
    admission, load, errors = metrics["admission"], metrics["load"], metrics["correctness"]["errors"]
    burst = metrics.get("burst") or {}
    attempted, accepted = load.get("attempted"), load.get("accepted")
    acceptance = accepted / attempted if attempted else None
    cards = [("Intended", load.get("intended")), ("Attempted", attempted), ("Accepted", accepted), ("Acceptance", _percent(acceptance)), ("Effective accepted intake", _number(admission.get("effective_accepted_intake_per_sec"), " sub/s")), ("Submit p95", _latency(metrics["latency_ms"]["submit"].get("p95"))), ("POST-start spread", _number(admission.get("post_start_spread_ms"), " ms")), ("System survival", admission.get("system_survival")), ("Client qualification", admission.get("client_qualification"))]
    intake = _table([("Acceptance (accepted / attempted POSTs)", _percent(acceptance)), ("Attempted launch interval", _number((burst.get("attempted") or {}).get("interval_ms"), " ms")), ("Accepted response interval", _number((burst.get("accepted") or {}).get("interval_ms"), " ms")), ("Client launch attempted throughput", _number((burst.get("attempted") or {}).get("throughput_per_sec"), " sub/s")), ("Effective accepted intake", _number(admission.get("effective_accepted_intake_per_sec"), " sub/s")), ("429 / other 4xx / 5xx / transport", f"{errors['rate_limited']} / {errors['http_errors']} / {errors['server_errors']} / {errors['transport_failures']}"), ("Terminal/Judge Core/E2E", "NOT MEASURED — admission-only")])
    probes = _table([(item.get("name"), item.get("status"), _latency(item.get("latency_ms"))) for item in admission.get("health_probes", [])], ("Probe", "Status", "Latency")) if admission.get("health_probes") else "<p>Post-burst probes unavailable.</p>"
    resources = metrics["resources"]
    resource_rows = [("Availability", "available" if resources.get("available") else resources.get("reason"))]
    for name, item in resources.get("containers", {}).items(): resource_rows.append((name, f"CPU p95 {_number(item['cpu_percent'].get('p95'),'%',1)}; memory max {_number(item['memory_mib'].get('max'),' MiB',1)}; restart delta {_number(item.get('restart_count_delta'))}"))
    images = "".join(f'<figure><img src="charts/{html.escape(chart)}" alt="{html.escape(chart)}"><figcaption>{html.escape(chart.replace("_", " ").replace(".png", ""))}</figcaption></figure>' for chart in charts)
    body = f"<h1>ASTRACODE 100K SUBMISSION BURST — ADMISSION ONLY</h1><p>This is a cardinality burst, not a 100K RPS claim. Actual POST timestamps remain authoritative.</p>{_cards(cards)}<section><h2>Submission API intake</h2>{intake}</section><section><h2>Scope</h2><p>Every identity issued one POST with no POST retry. The harness deliberately did not request tickets, open SSE, reconcile GETs, wait for terminal verdicts, or calculate Judge Core/E2E metrics.</p></section><section><h2>Client classification</h2><p><code>CLIENT_LIMITED</code> is the run classification when direct local evidence emits <code>LOAD_GENERATOR_LIMITED</code> (or reaches the configured in-flight bound). Generic TLS, connection, and deadline failures remain transport failures unless explicit FD, ephemeral-port, scheduling, or other measured load-generator resource evidence corroborates them.</p></section><section><h2>Post-burst public probes</h2>{probes}</section><section><h2>Kafka and App-node resources</h2>{_table(resource_rows)}<p>Collector absence is unavailable, never zero. Kafka backlog is expected for this intake-only KPI.</p></section><section><h2>Charts</h2>{images or '<p>Charts unavailable because collector or client data was unavailable.</p>'}</section><section><h2>CV-safe wording</h2><p>Populate only from the actual run: Load-tested AstraCode under a 100,000-user near-simultaneous submission burst, achieving the recorded acceptance percentage and effective intake rate with the recorded POST-start spread. This is not a 100K RPS claim.</p></section>"
    path.write_text(_page(f"Admission benchmark {metrics['run_id']}", body), encoding="utf-8")


def _render_realistic_run(path: Path, metrics: dict, charts: list[str], run: dict) -> None:
    realistic, load, errors = metrics["realistic"], metrics["load"], metrics["correctness"]["errors"]
    submit, ticket, sse = realistic["submission"], realistic["ticket"], realistic["sse"]
    client = metrics.get("client_resources", {})
    cards = [("Intended", load.get("intended")), ("Accepted", submit.get("successful")), ("Effective accepted intake", _number(submit.get("throughput_per_sec"), " sub/s")), ("Ticket success", _percent(ticket.get("success_percent"))), ("SSE establishment", _percent(sse.get("establishment_percent"))), ("Peak active SSE", sse.get("peak_active_streams")), ("System survival", realistic.get("system_survival")), ("Client qualification", realistic.get("client_qualification"))]
    submission_table = _table([("Attempted / accepted", f"{submit.get('attempted')} / {submit.get('successful')}"), ("Acceptance", _percent(submit.get("success_percent"))), ("Effective accepted intake", _number(submit.get("throughput_per_sec"), " sub/s")), ("POST p50 / p95 / p99", f"{_latency(submit['latency_ms'].get('p50'))} / {_latency(submit['latency_ms'].get('p95'))} / {_latency(submit['latency_ms'].get('p99'))}"), ("429 / other 4xx / 5xx / transport", f"{errors['rate_limited']} / {errors['http_errors']} / {errors['server_errors']} / {errors['transport_failures']}")])
    ticket_table = _table([("Attempted / successful", f"{ticket.get('attempted')} / {ticket.get('successful')}"), ("Success", _percent(ticket.get("success_percent"))), ("Request rate", _number(ticket.get("throughput_per_sec"), " req/s")), ("p50 / p95 / p99", f"{_latency(ticket['latency_ms'].get('p50'))} / {_latency(ticket['latency_ms'].get('p95'))} / {_latency(ticket['latency_ms'].get('p99'))}")])
    sse_table = _table([("Attempted / established / failed", f"{sse.get('attempted')} / {sse.get('established')} / {sse.get('failed')}"), ("Establishment", _percent(sse.get("establishment_percent"))), ("Establishment rate", _number(sse.get("establishment_rate_per_sec"), " streams/s")), ("p50 / p95 / p99 establishment", f"{_latency(sse['establishment_latency_ms'].get('p50'))} / {_latency(sse['establishment_latency_ms'].get('p95'))} / {_latency(sse['establishment_latency_ms'].get('p99'))}"), ("Peak active / full hold / early close / terminal", f"{sse.get('peak_active_streams')} / {sse.get('survived_full_hold')} / {sse.get('closed_early')} / {sse.get('terminal_during_hold')}")])
    client_table = _table([("Client sample availability", "available" if client.get("available") else client.get("reason")), ("FD samples / peak", f"{client.get('samples')} / {_number((client.get('open_fds') or {}).get('max'))}"), ("Goroutine peak", _number((client.get("goroutines") or {}).get("max"))), ("HTTP protocol evidence", f"H1/H2 are recorded in run metadata; response protocol counts are transport observations, not socket counts.")])
    probes = _table([(item.get("name"), item.get("status"), _latency(item.get("latency_ms"))) for item in realistic.get("health_probes", [])], ("Probe", "Status", "Latency")) if realistic.get("health_probes") else "<p>Post-burst probes unavailable.</p>"
    images = "".join(f'<figure><img src="charts/{html.escape(chart)}" alt="{html.escape(chart)}"><figcaption>{html.escape(chart.replace("_", " ").replace(".png", ""))}</figcaption></figure>' for chart in charts)
    body = f"<h1>ASTRACODE 100K-USER REALISTIC SUBMISSION BURST</h1><p>This is a cardinality experiment, not a 100K RPS claim. Each user independently follows POST → ticket → one SSE establishment/hold. Actual timestamps are authoritative.</p>{_cards(cards)}<section><h2>Submission</h2>{submission_table}</section><section><h2>Ticket</h2>{ticket_table}</section><section><h2>SSE</h2>{sse_table}<p>Established streams are held only for the configured bounded interval, or until a terminal event/stream close. No automatic SSE retry, reconciliation GET, or Judge drain occurs.</p></section><section><h2>Client qualification</h2>{client_table}<p>A recommended nofile limit is a starting point only. <code>CLIENT_LIMITED</code> is used only when direct local evidence emits <code>LOAD_GENERATOR_LIMITED</code> (or reaches the configured in-flight bound). Generic TLS, connection, and deadline failures remain transport failures unless corroborated by explicit local resource evidence.</p></section><section><h2>Post-burst health and external evidence</h2>{probes}<p>Container and Kafka collector absence is unavailable, never zero. Kafka backlog is expected and Judge completion is outside this KPI.</p></section><section><h2>Charts</h2>{images or '<p>Charts unavailable because supporting raw data was unavailable.</p>'}</section><section><h2>CV-safe wording</h2><p>Populate only from actual run values: Load-tested AstraCode under a 100,000-user near-simultaneous realistic submission burst, reporting distinct submission, ticket, and SSE establishment outcomes with the recorded POST-start spread. This is not a 100K RPS or Judge-Core claim.</p></section>"
    path.write_text(_page(f"Realistic admission benchmark {metrics['run_id']}", body), encoding="utf-8")


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
    cards = [("Judge Core throughput", "N/A — not directly measured"), ("Maximum stable requested arrival", _number(capacity.get("demonstrated_stable_rate"), " sub/s")), ("First tested saturation", _number(capacity.get("first_saturating_rate"), " sub/s")), ("Data quality", capacity.get("state"))]
    headers = ["Configuration", "Requested", "N", "Accepted mean", "Pipeline terminal mean", "Std", "95% CI", "E2E p50", "E2E p95", "E2E p99", "Peak outstanding", "Slope", "Kafka max/end", "Judge CPU p95", "Judge RAM max", "Errors", "Assessment"]
    rows = []
    for item in experiments:
        ci = "CI unavailable: N=1" if item.get("repetitions") == 1 else f"[{_number(item.get('throughput_ci95_low'))}, {_number(item.get('throughput_ci95_high'))}]"
        rows.append([item.get("configuration_label"), _number(item.get("requested_rate"), " sub/s"), item.get("repetitions"), _number(item.get("accepted_rate_mean"), " sub/s"), _number(item.get("pipeline_terminal_throughput_mean", item.get("throughput_mean")), " sub/s"), _number(item.get("pipeline_terminal_throughput_std", item.get("throughput_std"))), ci, _latency(item.get("p50_mean")), _latency(item.get("p95_mean")), _latency(item.get("p99_mean")), _number(item.get("outstanding_peak_mean")), _number(item.get("outstanding_slope_mean"), " /min"), f"{_number(item.get('kafka_lag_max_mean'))} / {_number(item.get('kafka_lag_end_mean'))}", _number(item.get("judge_cpu_p95_mean"), "%", 1), _number(item.get("judge_memory_max_mean"), " MiB", 1), _number(item.get("error_count_mean")), item.get("assessment")])
    individual = _table([[item.get(key) for key in ("run_id", "requested_rate", "pipeline_terminal_throughput", "e2e_p50", "e2e_p95", "e2e_p99", "error_count", "kafka_lag_end", "peak_outstanding", "judge_cpu_p95", "judge_memory_max", "assessment")] for item in individual_runs], ("Run ID", "Requested", "Pipeline terminal", "E2E p50", "E2E p95", "E2E p99", "Errors", "Kafka end", "Peak outstanding", "Judge CPU p95", "Judge RAM max", "Assessment"))
    images = "".join(f'<figure><img src="charts/{html.escape(chart)}" alt="{html.escape(chart)}"><figcaption>{html.escape(chart.replace("_", " ").replace(".png", ""))}</figcaption></figure>' for chart in charts)
    body = f"<h1>AstraCode Judge Pipeline Capacity Evidence</h1><p>{html.escape(capacity.get('message', 'Repeated sustained experiments are required for capacity evidence.'))}</p><p class=warning>These artifacts measure pipeline terminal observation and E2E latency. Judge Core compile-excluded service time and throughput are unavailable without explicit phase timestamps.</p>{_cards(cards)}<section><h2>Pipeline capacity evidence</h2>{_table(rows, headers)}</section><section><h2>Individual-run evidence</h2>{individual}</section><section><h2>Capacity charts</h2>{images or '<p>Charts unavailable because required data was unavailable.</p>'}</section>"
    path.write_text(_page("AstraCode Judge Pipeline Capacity Evidence", body), encoding="utf-8")


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
    stable_judge = "N/A — not directly measured" if judge else "unavailable"
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
    final_cards = [('API maximum demonstrated sustainable throughput', stable_api), ('Judge Core maximum demonstrated throughput', stable_judge), ('Pipeline E2E p50 at stable requested load', _latency((stable_row or {}).get('p50_mean'))), ('Pipeline E2E p95 at stable requested load', _latency((stable_row or {}).get('p95_mean'))), ('Pipeline E2E p99 at stable requested load', _latency((stable_row or {}).get('p99_mean'))), ('Pipeline saturation interval', judge_capacity.get('interval')), ('API saturation interval', api_capacity.get('interval'))]
    body = f"<h1>AstraCode Performance Capacity</h1><p>Evidence-oriented capacity report. Missing benchmark families are unavailable, never zero.</p>{_cards(final_cards)}<section><h2>System under test</h2>{_table(config_rows, ('Configuration','Release','App','Judge')) if config_rows else '<p>Safe system configuration metadata unavailable.</p>'}<p>Each evidence row retains its safe configuration label. Runs with different system metadata are not grouped together.</p></section><section><h2>API capacity evidence</h2>{_table(api_rows, ('Configuration','Requested req/s','Achieved req/s','p50','p95','p99','Assessment')) if api else '<p>API capacity: unavailable.</p>'}</section><section><h2>Pipeline terminal evidence</h2>{_table(judge_rows, ('Configuration','Requested sub/s','Pipeline terminal sub/s','E2E p50','E2E p95','E2E p99','Assessment')) if judge else '<p>Pipeline evidence: unavailable.</p>'}</section><section><h2>Conclusion</h2><p>Judge Core throughput is only shown when a compile-excluded measurement provides direct phase timestamps; pipeline terminal rates never substitute for it.</p></section>"
    path.write_text(_page("AstraCode Performance Capacity", body), encoding="utf-8")

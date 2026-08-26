"""Portable HTML reports with local PNG references and no network dependencies."""

from __future__ import annotations
import html
import json
from pathlib import Path


def _value(value, unit=""):
    if value is None: return "unavailable"
    if isinstance(value, float): return f"{value:.3f}{unit}"
    return f"{value}{unit}"


def render_run(path: Path, metrics: dict, charts: list[str]) -> None:
    e2e, resources, kafka = metrics["latency_ms"]["end_to_end"], metrics["resources"], metrics["kafka"]
    judge = resources.get("containers", {}).get("judge_worker", {})
    cards = [("Status", metrics.get("harness_classification") or "unavailable"), ("Analytical assessment", metrics["analytical_assessment"]["state"]),
             ("Accepted arrival", _value(metrics["load"]["accepted_arrival_rate_per_sec"], " sub/s")), ("Completed throughput", _value(metrics["completion"]["completion_throughput_per_sec"], " sub/s")),
             ("E2E p50", _value(e2e["p50"], " ms")), ("E2E p95", _value(e2e["p95"], " ms")), ("E2E p99", _value(e2e["p99"], " ms")),
             ("Kafka lag max", _value(kafka.get("total_lag", {}).get("max"))), ("Kafka lag end", _value(kafka.get("lag_at_end"))),
             ("AC ratio", _value(metrics["correctness"]["accepted_ratio"] * 100 if metrics["correctness"]["accepted_ratio"] is not None else None, "%"))]
    card_html = "".join(f"<article><small>{html.escape(k)}</small><strong>{html.escape(str(v))}</strong></article>" for k,v in cards)
    chart_html = "".join(f'<figure><img src="charts/{html.escape(name)}" alt="{html.escape(name)}"><figcaption>{html.escape(name)}</figcaption></figure>' for name in charts)
    quality = metrics["data_quality"]
    body = f"""<!doctype html><html><head><meta charset=\"utf-8\"><title>Judge benchmark {html.escape(str(metrics['run_id']))}</title><style>
body{{font-family:system-ui,sans-serif;margin:2rem;color:#17202a;background:#fafafa}}h1,h2{{color:#243b53}}.cards{{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:1rem}}article,section,figure{{background:white;border:1px solid #d9e2ec;border-radius:8px;padding:1rem;margin:1rem 0}}article strong{{display:block;font-size:1.3rem;margin-top:.3rem}}figure{{display:inline-block;vertical-align:top;width:46%}}img{{max-width:100%;height:auto}}table{{border-collapse:collapse}}td,th{{padding:.35rem .7rem;border:1px solid #d9e2ec;text-align:left}}.warning{{color:#9c2f00}}</style></head><body>
<h1>AstraCode Judge Benchmark</h1><p>Run: <b>{html.escape(str(metrics['run_id']))}</b>. Raw harness files remain canonical; this report is regenerated analysis.</p>
<div class=\"cards\">{card_html}</div>
<section><h2>Data quality: {html.escape(quality['state'])}</h2><p>{html.escape('; '.join(quality['reasons']) or 'All required artifacts were valid and optional collectors were available.')}</p><pre>{html.escape(json.dumps(quality, indent=2))}</pre></section>
<section><h2>Latency interpretation</h2><p>Mean is the central average. p95 means 95% of observed completions were no slower than this value. p99 exposes tail latency. Max is not a percentile and is sensitive to outliers; no normal distribution is assumed.</p><pre>{html.escape(json.dumps(metrics['latency_ms'], indent=2))}</pre></section>
<section><h2>Load, queue, correctness, and observation</h2><pre>{html.escape(json.dumps({k: metrics[k] for k in ['load','completion','queue','correctness','observation','kafka','resources','analytical_assessment']}, indent=2))}</pre></section>
<section><h2>Charts</h2>{chart_html or '<p>Charts unavailable because the corresponding valid data was absent.</p>'}</section></body></html>"""
    path.write_text(body, encoding="utf-8")


def render_comparison(path: Path, experiments: list[dict], charts: list[str], capacity: dict | None = None) -> None:
    rows = "".join("<tr>" + "".join(f"<td>{html.escape(str(value))}</td>" for value in [row.get("experiment_label"), row.get("repetitions"), row.get("requested_rate"), row.get("throughput_mean"), row.get("throughput_ci95_low"), row.get("throughput_ci95_high"), row.get("assessment")]) + "</tr>" for row in experiments)
    images = "".join(f'<figure><img src="charts/{html.escape(chart)}" alt="{html.escape(chart)}"><figcaption>{html.escape(chart)}</figcaption></figure>' for chart in charts)
    capacity_text = html.escape(json.dumps(capacity or {}, indent=2))
    path.write_text(f"""<!doctype html><html><head><meta charset=\"utf-8\"><title>Judge benchmark comparison</title><style>body{{font-family:system-ui;margin:2rem}}td,th{{border:1px solid #ccc;padding:.4rem}}table{{border-collapse:collapse}}figure{{display:inline-block;width:46%}}img{{max-width:100%}}</style></head><body><h1>AstraCode Judge Benchmark Comparison</h1><p>Confidence intervals are Student-t intervals across run-level values. One repetition has no CI. Conclusions are evidence from tested rates, not a precise capacity claim.</p><h2>Capacity evidence</h2><pre>{capacity_text}</pre><table><tr><th>Experiment</th><th>N</th><th>Requested rate</th><th>Completed throughput</th><th>CI low</th><th>CI high</th><th>Assessment</th></tr>{rows}</table><section>{images}</section></body></html>""", encoding="utf-8")

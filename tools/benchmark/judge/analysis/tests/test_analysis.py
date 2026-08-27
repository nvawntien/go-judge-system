import json
from pathlib import Path
import pytest
from judge_analysis.cli import main
from judge_analysis.loaders import DataError, load_run
from judge_analysis.metrics import calculate
from judge_analysis.statistics import distribution, mean_ci
from judge_analysis.api import assessment, load_api_run
from conftest import SYSTEM_CONFIG, make_api_run, make_massive_burst_run, make_run


def test_known_percentiles_and_ci_are_honest():
    d=distribution([1,2,3,4,5,6,7,8,9,10])
    assert d["p50"] == 5.5 and d["p90"] == 9.1 and d["p99"] == 9.91
    assert mean_ci([1])["ci95_low"] is None
    assert mean_ci([1,2,3])["ci95_low"] < 2 < mean_ci([1,2,3])["ci95_high"]


def test_healthy_run_analysis_and_html(tmp_path):
    run=make_run(tmp_path/"healthy")
    assert main(["analyze","--run-dir",str(run)]) == 0
    analysis=run/"analysis"; metrics=json.loads((analysis/"metrics.json").read_text())
    assert metrics["data_quality"]["state"] == "COMPLETE"
    assert metrics["analytical_assessment"]["state"] == "STABLE"
    assert (analysis/"charts"/"01_latency_distribution.png").stat().st_size > 100
    html=(analysis/"report.html").read_text()
    assert "healthy-r1" in html and "Latency detail" in html and "test-pool-1" in html and "password-sentinel" not in html


def test_saturated_and_missing_collectors_are_not_zero(tmp_path):
    run=make_run(tmp_path/"saturated", saturated=True, collectors=False)
    data=load_run(run); metrics,_,_=calculate(data)
    assert metrics["kafka"]["available"] is False
    assert metrics["resources"]["available"] is False
    assert metrics["data_quality"]["state"] == "PARTIAL"
    assert metrics["analytical_assessment"]["state"] == "SATURATING"


def test_corrupt_required_artifacts_fail_clearly(tmp_path):
    run=make_run(tmp_path/"bad")
    (run/"submissions.csv").write_text("run_id\nwrong\n")
    with pytest.raises(DataError, match="missing required columns"):
        load_run(run)


@pytest.mark.parametrize("column,value", [("terminal_observed_at","not-a-time"), ("end_to_end_latency_ms","-1")])
def test_invalid_timestamp_and_negative_latency_are_rejected(tmp_path, column, value):
    run=make_run(tmp_path/"bad-value")
    rows=(run/"submissions.csv").read_text().splitlines()
    header=rows[0].split(","); index=header.index(column); first=rows[1].split(","); first[index]=value; rows[1]=",".join(first)
    (run/"submissions.csv").write_text("\n".join(rows)+"\n")
    with pytest.raises(DataError): load_run(run)


def test_missing_kafka_suppresses_only_kafka_chart(tmp_path):
    run=make_run(tmp_path/"partial", collectors=False)
    assert main(["analyze","--run-dir",str(run)]) == 0
    names={path.name for path in (run/"analysis"/"charts").iterdir()}
    assert "01_latency_distribution.png" in names and "06_kafka_lag.png" not in names


def test_external_samples_align_by_utc_not_file_order(tmp_path):
    run=make_run(tmp_path/"aligned")
    stats=run/"container-stats.csv"
    lines=stats.read_text().splitlines()
    lines.append("2026-08-27T00:00:00Z,judge,judge_worker,99,999999999,1000000000,99,99")
    stats.write_text("\n".join(lines)+"\n")
    metrics,_,_=calculate(load_run(run))
    assert metrics["resources"]["containers"]["judge_worker"]["samples"] == 4


def test_compare_groups_structured_repetitions_and_generates_report(tmp_path):
    runs=[make_run(tmp_path/f"r{i}", run_id=f"healthy-r{i}", repetition=i) for i in range(1,4)]
    output=tmp_path/"comparison"
    assert main(["compare", *sum((["--run-dir",str(run)] for run in runs), []), "--output",str(output)]) == 0
    rows=(output/"experiments.csv").read_text().splitlines()
    assert len(rows)==2 and (output/"comparison.html").exists()


def test_reports_escape_metadata_and_keep_unavailable_config_explicit(tmp_path):
    config = {**SYSTEM_CONFIG, "label": "<untrusted & label>"}
    run = make_run(tmp_path / "escaped", system_config=config)
    assert main(["analyze", "--run-dir", str(run)]) == 0
    html = (run / "analysis" / "report.html").read_text()
    assert "&lt;untrusted &amp; label&gt;" in html
    older = make_run(tmp_path / "older", system_config=None)
    metrics, _, _ = calculate(load_run(older))
    assert metrics["system_config"]["available"] is False


def test_single_report_shows_exact_measured_volume_and_requested_rate(tmp_path):
    run = make_run(tmp_path / "exact-volume", rate=1.0, total_submissions=12)
    assert main(["analyze", "--run-dir", str(run)]) == 0
    html = (run / "analysis" / "report.html").read_text()
    assert "Measured volume" in html and ">12<" in html and "1.000 sub/s" in html


def test_compare_never_groups_different_system_configurations(tmp_path):
    other = {**SYSTEM_CONFIG, "label": "test-pool-2", "judge": {**SYSTEM_CONFIG["judge"], "worker_pool_size": 2}}
    first = make_run(tmp_path / "first", run_id="same-rate-a", system_config=SYSTEM_CONFIG)
    second = make_run(tmp_path / "second", run_id="same-rate-b", system_config=other)
    output = tmp_path / "comparison"
    assert main(["compare", "--run-dir", str(first), "--run-dir", str(second), "--output", str(output)]) == 0
    payload = json.loads((output / "experiments.json").read_text())
    assert len(payload["experiments"]) == 2
    assert all(row["repetitions"] == 1 for row in payload["experiments"])
    assert "CI unavailable: N=1" in (output / "comparison.html").read_text()


def test_judge_capacity_boundary_requires_repeated_stable_and_saturating_runs(tmp_path):
    stable = [make_run(tmp_path / f"stable-{i}", run_id=f"stable-{i}", rate=.4, repetition=i) for i in range(1, 4)]
    saturated = [make_run(tmp_path / f"saturated-{i}", run_id=f"saturated-{i}", rate=.8, saturated=True, repetition=i) for i in range(1, 4)]
    output = tmp_path / "comparison"
    assert main(["compare", *sum((["--run-dir", str(run)] for run in stable + saturated), []), "--output", str(output)]) == 0
    capacity = json.loads((output / "experiments.json").read_text())["capacity_evidence"]
    assert capacity["demonstrated_stable_rate"] == .4
    assert capacity["first_saturating_rate"] == .8
    assert capacity["interval"] == "(0.4, 0.8] sub/s"


def test_capacity_boundary_is_unavailable_for_one_repetition_per_rate(tmp_path):
    stable = make_run(tmp_path / "stable", rate=.4)
    saturated = make_run(tmp_path / "saturated", rate=.8, saturated=True)
    output = tmp_path / "comparison"
    assert main(["compare", "--run-dir", str(stable), "--run-dir", str(saturated), "--output", str(output)]) == 0
    assert json.loads((output / "experiments.json").read_text())["capacity_evidence"]["state"] == "INSUFFICIENT_DATA"


def test_api_artifacts_compare_and_unified_reports(tmp_path):
    api_runs = [make_api_run(tmp_path / f"api-{i}", run_id=f"api-{i}", repetition=i) for i in range(1, 4)]
    judge_runs = [make_run(tmp_path / f"judge-{i}", run_id=f"judge-{i}", repetition=i) for i in range(1, 4)]
    api_output, judge_output = tmp_path / "api-comparison", tmp_path / "judge-comparison"
    assert main(["api-compare", *sum((["--run-dir", str(run)] for run in api_runs), []), "--output", str(api_output)]) == 0
    assert main(["compare", *sum((["--run-dir", str(run)] for run in judge_runs), []), "--output", str(judge_output)]) == 0
    unified = tmp_path / "unified"
    assert main(["capacity-report", "--api-comparison", str(api_output / "experiments.json"), "--judge-comparison", str(judge_output / "experiments.json"), "--output", str(unified)]) == 0
    assert "AstraCode Performance Capacity" in (unified / "capacity-report.html").read_text()
    assert (api_output / "comparison.html").exists()
    assert (api_output / "charts" / "throughput_vs_rate.png").stat().st_size > 100


def test_api_parser_rejects_malformed_and_api_assessment_flags_drops(tmp_path):
    broken = make_api_run(tmp_path / "broken")
    summary = json.loads((broken / "summary.json").read_text()); del summary["latency_ms"]["p95"]
    (broken / "summary.json").write_text(json.dumps(summary))
    with pytest.raises(DataError): load_api_run(broken)
    saturating = make_api_run(tmp_path / "drops", dropped=1)
    output = tmp_path / "api-output"
    assert main(["api-compare", "--run-dir", str(saturating), "--output", str(output)]) == 0
    assert json.loads((output / "experiments.json").read_text())["experiments"][0]["assessment"] == "SATURATING"


def test_api_repeated_boundary_and_zero_requests_are_honest(tmp_path):
    stable = [make_api_run(tmp_path / f"api-stable-{i}", run_id=f"api-stable-{i}", rate=10, repetition=i) for i in range(1, 4)]
    saturated = [make_api_run(tmp_path / f"api-saturated-{i}", run_id=f"api-saturated-{i}", rate=20, achieved=15, repetition=i) for i in range(1, 4)]
    output = tmp_path / "api-boundary"
    assert main(["api-compare", *sum((["--run-dir", str(run)] for run in stable + saturated), []), "--output", str(output)]) == 0
    capacity = json.loads((output / "experiments.json").read_text())["capacity_evidence"]
    assert capacity["interval"] == "(10.0, 20.0] req/s"
    assert assessment({"requested_rps": 1, "achieved_rps": 0, "total_requests": 0, "error_rate": 0, "dropped_iterations": 0}) == "INSUFFICIENT_DATA"


def test_unified_reports_one_benchmark_family_as_unavailable(tmp_path):
    api = make_api_run(tmp_path / "api", system_config=None)
    api_output = tmp_path / "api-output"
    assert main(["api-compare", "--run-dir", str(api), "--output", str(api_output)]) == 0
    unified = tmp_path / "unified"
    assert main(["capacity-report", "--api-comparison", str(api_output / "experiments.json"), "--output", str(unified)]) == 0
    html = (unified / "capacity-report.html").read_text()
    assert "Judge capacity: unavailable." in html and "API capacity evidence" in html


def test_comparisons_never_copy_arbitrary_system_config_fields(tmp_path):
    unsafe = {**SYSTEM_CONFIG, "DATABASE_PASSWORD": "sentinel-secret"}
    judge = make_run(tmp_path / "judge", system_config=unsafe)
    api = make_api_run(tmp_path / "api", system_config=unsafe)
    judge_out, api_out = tmp_path / "judge-out", tmp_path / "api-out"
    assert main(["compare", "--run-dir", str(judge), "--output", str(judge_out)]) == 0
    assert main(["api-compare", "--run-dir", str(api), "--output", str(api_out)]) == 0
    assert "sentinel-secret" not in (judge_out / "experiments.json").read_text()
    assert "sentinel-secret" not in (api_out / "experiments.json").read_text()


def test_massive_burst_uses_actual_intake_not_sustained_rate_and_renders_partial_e2e(tmp_path):
    run = make_massive_burst_run(tmp_path / "burst", burst_size=10, completed=False)
    assert main(["analyze", "--run-dir", str(run)]) == 0
    metrics = json.loads((run / "analysis" / "metrics.json").read_text())
    assert metrics["burst"]["accepted"]["interval_ms"] == 90
    assert metrics["load"]["accepted_arrival_rate_per_sec"] == metrics["burst"]["accepted"]["throughput_per_sec"]
    html = (run / "analysis" / "report.html").read_text()
    assert "ASTRACODE MASSIVE SUBMISSION BURST" in html
    assert "Burst size" in html and "Effective accepted intake" in html
    assert "Requested arrival" in html and "unavailable" in html


def test_massive_burst_comparison_is_cardinality_evidence_not_capacity(tmp_path):
    runs = [make_massive_burst_run(tmp_path / f"b{i}", run_id=f"b{i}", burst_size=size, repetition=i) for i, size in enumerate((10, 50, 100), 1)]
    output = tmp_path / "burst-comparison"
    assert main(["compare", *sum((["--run-dir", str(run)] for run in runs), []), "--output", str(output)]) == 0
    payload = json.loads((output / "experiments.json").read_text())
    assert payload["capacity_evidence"]["state"] == "BURST_COMPARISON"
    assert [row["burst_size"] for row in payload["experiments"]] == [10, 50, 100]
    html = (output / "comparison.html").read_text()
    assert "ASTRACODE MASSIVE SUBMISSION BURST" in html
    assert (output / "charts" / "burst_size_vs_effective_accepted_throughput.png").stat().st_size > 100

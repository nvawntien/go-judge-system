import json
from pathlib import Path
import pytest
from judge_analysis.cli import main
from judge_analysis.loaders import DataError, load_run
from judge_analysis.metrics import calculate
from judge_analysis.statistics import distribution, mean_ci
from conftest import make_run


def test_known_percentiles_and_ci_are_honest():
    d=distribution([1,2,3,4,5,6,7,8,9,10])
    assert d["p50"] == 5.5 and d["p90"] == 9.1 and d["p99"] == 9.91
    assert mean_ci([1])["ci95_low"] is None
    assert mean_ci([1,2,3])["ci95_low"] < 2 < mean_ci([1,2,3])["ci95_high"]


def test_healthy_run_analysis_and_html(tmp_path):
    run=make_run(tmp_path/"healthy")
    assert main(["analyze","--run-dir",str(run)]) == 0
    analysis=run/"analysis"; metrics=json.loads((analysis/"metrics.json").read_text())
    assert metrics["data_quality"]["state"] == "GOOD"
    assert metrics["analytical_assessment"]["state"] == "STABLE"
    assert (analysis/"charts"/"01_latency_distribution.png").stat().st_size > 100
    html=(analysis/"report.html").read_text()
    assert "healthy-r1" in html and "p95" in html and "password-sentinel" not in html


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

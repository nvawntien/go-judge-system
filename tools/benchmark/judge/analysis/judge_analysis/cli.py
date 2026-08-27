from __future__ import annotations
import argparse
import json
from pathlib import Path
import sys
from . import __version__
from .compare import compare
from .loaders import DataError, load_run
from .metrics import align_to_run, calculate
from .plots import single_run
from .report import render_run
from .api import compare_api
from .unified import capacity_report


def analyze(args) -> int:
    data = load_run(args.run_dir, args.container_stats, args.kafka_lag)
    output = Path(args.run_dir) / "analysis"
    if output.exists():
        raise DataError("analysis output already exists; remove it only after preserving any prior analysis")
    output.mkdir(); metrics, timeseries, verdicts = calculate(data)
    charts = single_run(output / "charts", data.submissions, data.windows, metrics, align_to_run(data.containers, data.run), align_to_run(data.kafka, data.run))
    (output / "metrics.json").write_text(json.dumps(metrics, indent=2, default=str), encoding="utf-8")
    # Flatten core metrics as rows so spreadsheet use does not need JSON parsing.
    import pandas as pd
    pd.DataFrame([{"name": key, "value": value} for key, value in _flatten(metrics)]).to_csv(output / "metrics.csv", index=False)
    timeseries.to_csv(output / "timeseries.csv", index=False); verdicts.to_csv(output / "verdicts.csv", index=False)
    render_run(output / "report.html", metrics, charts, data.run)
    print(f"Analysis complete\nRun: {metrics['run_id']}\nReport: {output / 'report.html'}\nData quality: {metrics['data_quality']['state']}\nHarness classification: {metrics.get('harness_classification')}\nAnalytical assessment: {metrics['analytical_assessment']['state']}")
    return 0


def _flatten(value, prefix=""):
    if isinstance(value, dict):
        for key, child in value.items(): yield from _flatten(child, f"{prefix}.{key}" if prefix else key)
    elif isinstance(value, list):
        yield prefix, json.dumps(value)
    else: yield prefix, value


def collect_runs(args) -> list[Path]:
    paths = [Path(value) for value in args.run_dir]
    if args.results_root:
        root = Path(args.results_root)
        paths.extend(path for path in sorted(root.glob(args.match)) if path.is_dir())
    unique = list(dict.fromkeys(path.resolve() for path in paths))
    if not unique: raise DataError("specify one or more --run-dir values, or --results-root with --match")
    return unique


def main(argv=None) -> int:
    parser = argparse.ArgumentParser(prog="judge_analysis")
    parser.add_argument("--version", action="version", version=__version__)
    commands = parser.add_subparsers(dest="command", required=True)
    single = commands.add_parser("analyze"); single.add_argument("--run-dir", required=True); single.add_argument("--container-stats"); single.add_argument("--kafka-lag")
    multiple = commands.add_parser("compare"); multiple.add_argument("--run-dir", action="append", default=[]); multiple.add_argument("--results-root"); multiple.add_argument("--match", default="*"); multiple.add_argument("--output", required=True)
    api = commands.add_parser("api-compare"); api.add_argument("--run-dir", action="append", default=[]); api.add_argument("--results-root"); api.add_argument("--match", default="*"); api.add_argument("--output", required=True)
    capacity = commands.add_parser("capacity-report"); capacity.add_argument("--judge-comparison"); capacity.add_argument("--api-comparison"); capacity.add_argument("--output", required=True)
    args = parser.parse_args(argv)
    try:
        if args.command == "analyze": return analyze(args)
        if args.command == "capacity-report":
            capacity_report(args.output, args.judge_comparison, args.api_comparison)
            print(f"Capacity report complete\nReport: {Path(args.output) / 'capacity-report.html'}")
            return 0
        output = Path(args.output)
        frame = compare_api(collect_runs(args), output) if args.command == "api-compare" else compare(collect_runs(args), output)
        print(f"Comparison complete\nExperiments: {len(frame)}\nReport: {output / 'comparison.html'}")
        return 0
    except DataError as error:
        print(f"judge-analysis: {error}", file=sys.stderr); return 2


if __name__ == "__main__":
    raise SystemExit(main())

"""Compose already-generated API and Judge comparison evidence offline."""

from __future__ import annotations
import json
from pathlib import Path
from .loaders import DataError


def _read(path: str | Path | None):
    if not path: return None
    try: return json.loads(Path(path).read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error: raise DataError("invalid comparison evidence JSON") from error


def capacity_report(output: str | Path, judge_json: str | Path | None, api_json: str | Path | None):
    judge, api = _read(judge_json), _read(api_json)
    if judge is None and api is None: raise DataError("supply Judge and/or API comparison evidence")
    out = Path(output); out.mkdir(parents=True, exist_ok=False)
    from .report import render_unified
    render_unified(out / "capacity-report.html", judge, api)
    (out / "capacity-report.json").write_text(json.dumps({"judge": judge, "api": api}, indent=2), encoding="utf-8")

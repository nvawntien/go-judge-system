"""Evidence-oriented capacity wording; deliberately avoids false precision."""

from __future__ import annotations
import pandas as pd


def evidence(experiments: pd.DataFrame) -> dict:
    stable = experiments[experiments.assessment == "STABLE"].sort_values("requested_rate")
    unstable = experiments[experiments.assessment == "SATURATING"].sort_values("requested_rate")
    if not len(stable) or not len(unstable):
        return {"state": "INSUFFICIENT_EVIDENCE", "message": "Repeated stable and clearly saturating sustained experiments are both required."}
    low, high = float(stable.requested_rate.max()), float(unstable.requested_rate.min())
    return {"state": "TESTED_INTERVAL", "demonstrated_stable_rate": low, "first_saturating_rate": high,
            "message": f"Saturation lies within the tested interval ({low}, {high}] sub/s; this is not a precise capacity number."}

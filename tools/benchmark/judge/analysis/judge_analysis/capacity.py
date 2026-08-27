"""Evidence-oriented capacity wording; deliberately avoids false precision."""

from __future__ import annotations
import pandas as pd


def evidence(experiments: pd.DataFrame, unit: str = "sub/s") -> dict:
    # A capacity boundary has meaning only for one explicitly described system
    # under test. A caller may still render a mixed comparison, but must not
    # turn it into one synthetic capacity figure.
    system_key = "system_config_fingerprint" if "system_config_fingerprint" in experiments else "system_config_label"
    if system_key in experiments and experiments[system_key].nunique(dropna=False) > 1:
        return {"state": "INSUFFICIENT_DATA", "interval": "unavailable", "message": "Multiple system configurations selected; compare each configuration separately for capacity evidence."}
    stable = experiments[experiments.assessment == "STABLE"].sort_values("requested_rate")
    unstable = experiments[experiments.assessment == "SATURATING"].sort_values("requested_rate")
    if not len(stable) or not len(unstable):
        return {"state": "INSUFFICIENT_DATA", "interval": "unavailable", "message": "Repeated stable and clearly saturating sustained experiments are both required."}
    if "repetitions" in experiments and (stable.repetitions.min() < 2 or unstable.repetitions.min() < 2):
        return {"state": "INSUFFICIENT_DATA", "interval": "unavailable", "message": "At least two repetitions are required for each stable and saturating capacity boundary."}
    low, high = float(stable.requested_rate.max()), float(unstable.requested_rate.min())
    return {"state": "TESTED_INTERVAL", "demonstrated_stable_rate": low, "first_saturating_rate": high, "interval": f"({low}, {high}] {unit}",
            "message": f"Saturation lies within the tested interval ({low}, {high}] {unit}; this is not a precise capacity number."}

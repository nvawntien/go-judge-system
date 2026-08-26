"""Explicit statistical methods; no normal-distribution claim for latencies."""

from __future__ import annotations
import math
from typing import Iterable
import numpy as np
from scipy import stats


def finite(values: Iterable[float]) -> np.ndarray:
    data = np.asarray(list(values), dtype=float)
    return data[np.isfinite(data)]


def distribution(values: Iterable[float]) -> dict:
    data = finite(values)
    if not len(data):
        return {"count": 0, "mean": None, "std": None, "cv": None, "min": None,
                "p50": None, "p90": None, "p95": None, "p99": None, "max": None}
    quantiles = np.percentile(data, [50, 90, 95, 99], method="linear")
    mean = float(data.mean())
    std = float(data.std(ddof=1)) if len(data) > 1 else 0.0
    return {"count": int(len(data)), "mean": mean, "std": std,
            "cv": (std / mean if mean else None), "min": float(data.min()),
            "p50": float(quantiles[0]), "p90": float(quantiles[1]),
            "p95": float(quantiles[2]), "p99": float(quantiles[3]), "max": float(data.max())}


def mean_ci(values: Iterable[float], confidence: float = 0.95) -> dict:
    data = finite(values)
    if len(data) < 2:
        return {"n": int(len(data)), "mean": (float(data[0]) if len(data) else None), "std": None,
                "ci95_low": None, "ci95_high": None, "method": "unavailable: fewer than two repetitions"}
    mean, sem = float(data.mean()), stats.sem(data)
    margin = float(stats.t.ppf((1 + confidence) / 2, len(data) - 1) * sem)
    return {"n": int(len(data)), "mean": mean, "std": float(data.std(ddof=1)),
            "ci95_low": mean - margin, "ci95_high": mean + margin,
            "method": "Student-t mean confidence interval"}

// Package report renders a human-readable view of the machine-readable run
// artifacts. It intentionally contains only aggregate, non-secret data.
package report

import (
	"fmt"
	"strconv"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/model"
)

func Markdown(summary model.RunSummary) string {
	if summary.Burst != nil && summary.Burst.Massive {
		return massiveBurstMarkdown(summary)
	}
	return fmt.Sprintf(`# Judge benchmark %s

Classification: **%s**

## Load window

- Measured volume (intended): %d submissions
- Intended / attempted / accepted: %d / %d / %d
- HTTP-201 accepted after load stop: %d
- Terminal completions during load: %d
- Client outstanding at load end: %d
- Burst POST-start spread: %v ms
- Attempted / accepted / observed pipeline-terminal rate: %.3f / %.3f / %.3f per second

## Compile and Judge Core

- Compile overhead: %s
- Judge Core service time: %s
- Judge Core throughput: %s

## Drain

- Outstanding at start: %d
- Terminal completions: %d
- Remaining: %d
- Duration: %d ms

## Quality

%v
`, summary.RunID, summary.Classification,
		summary.Counts.Intended, summary.Counts.Intended, summary.Counts.Attempted, summary.LoadWindow.Accepted, summary.LoadWindow.BoundaryAcceptedAfterLoad,
		summary.LoadWindow.Completed, summary.LoadWindow.OutstandingAtEnd, optionalInt(summary.LoadWindow.BurstSpreadMS),
		summary.Rates.LoadAttemptedPerSecond, summary.Rates.LoadAcceptedPerSecond, summary.Rates.LoadPipelineTerminalCompletionSecond,
		availability(summary.Compile.Availability, summary.Compile.Reason), availability(summary.JudgeCore.Availability, summary.JudgeCore.Reason), availability(summary.JudgeCore.Availability, summary.JudgeCore.Reason),
		summary.Drain.OutstandingAtStart, summary.Drain.Completed, summary.Drain.Remaining, summary.Drain.DurationMS,
		summary.QualityFlags)
}

func massiveBurstMarkdown(summary model.RunSummary) string {
	burst := summary.Burst
	return fmt.Sprintf(`# ASTRACODE MASSIVE SUBMISSION BURST %s

Classification: **%s**

## Submission API

- Attempted / accepted: %d / %d
- Effective attempted throughput: %s submissions/s
- Effective accepted throughput: %s submissions/s
- Submit p50 / p95 / p99: %s / %s / %s ms
- HTTP 429 / other 4xx / 5xx / transport: %d / %d / %d / %d

## Burst quality

- POST-start spread: %v ms
- POST-start p50 / p95 / p99 offset: %s / %s / %s ms
- Schedule delay p50 / p95 / p99: %s / %s / %s ms
- Peak logical in-flight: %d
- Peak active SSE observers: %d

## Compile and Judge Core

- Compile overhead: %s
- Judge Core service time / throughput: %s / %s

## Pipeline terminal and E2E

- Terminal completed / accepted: %d / %d
- Observed pipeline-terminal throughput: %s submissions/s
- Terminal observation coverage: %s
- E2E sample status: %s
- Remaining backlog after drain: %d
- Drain duration: %d ms
- E2E p50 / p95 / p99: %s / %s / %s ms
- SSE completions / failures / GET reconciliations: %d / %d / %d

## Quality

%v
`, summary.RunID, summary.Classification,
		summary.Counts.Attempted, summary.Counts.Accepted,
		optionalFloat(burst.AttemptedThroughputPerSec), optionalFloat(burst.AcceptedThroughputPerSec),
		optionalFloat(summary.Latencies.SubmitMS.P50), optionalFloat(summary.Latencies.SubmitMS.P95), optionalFloat(summary.Latencies.SubmitMS.P99),
		summary.Counts.RateLimited, summary.Counts.Other4xx, summary.Counts.ServerErrors, summary.Counts.TransportFailures,
		optionalInt(summary.LoadWindow.BurstSpreadMS), optionalFloat(burst.PostStartOffsetMS.P50), optionalFloat(burst.PostStartOffsetMS.P95), optionalFloat(burst.PostStartOffsetMS.P99),
		optionalFloat(summary.Latencies.ScheduleDelayMS.P50), optionalFloat(summary.Latencies.ScheduleDelayMS.P95), optionalFloat(summary.Latencies.ScheduleDelayMS.P99),
		burst.PeakLogicalInFlight, burst.PeakActiveObservers,
		availability(summary.Compile.Availability, summary.Compile.Reason), availability(summary.JudgeCore.Availability, summary.JudgeCore.Reason), availability(summary.JudgeCore.Availability, summary.JudgeCore.Reason),
		summary.Counts.Terminal, summary.Counts.Accepted, optionalFloat(burst.PipelineTerminalThroughputPerSec), optionalPercent(summary.Pipeline.TerminalObservationCoverage), e2eSampleStatus(summary.Pipeline.RightCensored), summary.Drain.Remaining, summary.Drain.DurationMS,
		optionalFloat(summary.Latencies.EndToEndMS.P50), optionalFloat(summary.Latencies.EndToEndMS.P95), optionalFloat(summary.Latencies.EndToEndMS.P99),
		summary.Observer.SSECompletions, summary.Observer.SSEFailures, summary.Observer.GETReconciliations,
		summary.QualityFlags)
}

func optionalInt(value *int64) string {
	if value == nil {
		return "n/a"
	}
	return strconv.FormatInt(*value, 10)
}

func optionalFloat(value *float64) string {
	if value == nil {
		return "unavailable"
	}
	return fmt.Sprintf("%.3f", *value)
}

func availability(state, reason string) string {
	if state == "" || state == "AVAILABLE" {
		return "available"
	}
	if reason == "" {
		return state
	}
	return state + " — " + reason
}

func optionalPercent(value *float64) string {
	if value == nil {
		return "unavailable"
	}
	return fmt.Sprintf("%.1f%%", *value*100)
}

func e2eSampleStatus(rightCensored bool) string {
	if rightCensored {
		return "PARTIAL / right-censored; p95/p99 cover observed terminal samples only"
	}
	return "complete for accepted submissions observed by this run"
}

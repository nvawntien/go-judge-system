// Package report renders a human-readable view of the machine-readable run
// artifacts. It intentionally contains only aggregate, non-secret data.
package report

import (
	"fmt"
	"strconv"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/model"
)

func Markdown(summary model.RunSummary) string {
	if summary.Admission != nil && summary.Admission.ObservationMode == "admission-only" {
		return admissionOnlyMarkdown(summary)
	}
	if summary.Realistic != nil && summary.Realistic.ObservationMode == "realistic" {
		return realisticMarkdown(summary)
	}
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

func realisticMarkdown(summary model.RunSummary) string {
	realistic := summary.Realistic
	return fmt.Sprintf(`# ASTRACODE REALISTIC MASSIVE SUBMISSION BURST %s

Classification: **%s**

## Submission → ticket → SSE hold

- Intended / attempted / accepted: %d / %d / %d
- Submission acceptance: %s
- Effective accepted intake: %s submissions/s
- Submit p50 / p95 / p99: %s / %s / %s ms
- Ticket attempted / successful: %d / %d (%s), p95 %s ms
- SSE attempted / established: %d / %d (%s), p95 establishment %s ms
- Full-flow success (SSE established / intended): %s
- Peak active SSE streams: %d
- Streams surviving full hold / closed early / terminal during hold: %d / %d / %d
- POST-start / SSE-start spread: %v / %v ms

## Scope boundary

- Each accepted POST makes exactly one ticket request and exactly one SSE attempt.
- Established streams stay open for the configured hold duration unless a terminal event arrives or the stream closes/fails.
- No SSE reconnect, GET reconciliation, Judge drain, E2E, or Judge Core metric is performed.
- Compile is excluded from this API/SSE resilience KPI.

## Client qualification

- Status: %s
- FD estimate / observed peak: %d / %d (starting operator limit recommendation: %d; not a guarantee)
- Active POST / ticket / SSE peaks: %d / %d / %d
- HTTP response protocols H1 / H2 / other: %d / %d / %d

## Post-burst probes

%s

## Quality

%v
`, summary.RunID, summary.Classification,
		summary.Counts.Intended, summary.Counts.Attempted, summary.Counts.Accepted, percent(summary.Counts.Accepted, summary.Counts.Attempted),
		optionalFloat(realistic.Submission.ThroughputPerSec), optionalFloat(summary.Latencies.SubmitMS.P50), optionalFloat(summary.Latencies.SubmitMS.P95), optionalFloat(summary.Latencies.SubmitMS.P99),
		realistic.Ticket.Attempted, realistic.Ticket.Successful, optionalPercent(realistic.Ticket.SuccessPercent), optionalFloat(realistic.Ticket.LatencyMS.P95),
		realistic.SSE.Attempted, realistic.SSE.Established, optionalPercent(realistic.SSE.EstablishmentPercent), optionalFloat(realistic.SSE.EstablishmentLatencyMS.P95), optionalPercent(realistic.FullFlowSuccessPercent),
		realistic.SSE.PeakActiveStreams, realistic.SSE.SurvivedFullHold, realistic.SSE.ClosedEarly, realistic.SSE.TerminalDuringHold,
		optionalInt(summary.LoadWindow.BurstSpreadMS), optionalInt(realistic.SSE.StartSpreadMS),
		realistic.ClientQualification, summary.ClientDiagnostics.NoFileRequired, summary.ClientDiagnostics.PeakOpenFDs, summary.ClientDiagnostics.NoFileRecommended,
		summary.ClientDiagnostics.PeakActivePosts, summary.ClientDiagnostics.PeakActiveTickets, summary.ClientDiagnostics.PeakActiveSSE, summary.ClientDiagnostics.HTTP1Responses, summary.ClientDiagnostics.HTTP2Responses, summary.ClientDiagnostics.OtherProtocolResponses,
		healthMarkdown(summary.HealthProbes), summary.QualityFlags)
}

func admissionOnlyMarkdown(summary model.RunSummary) string {
	burst := summary.Burst
	admission := summary.Admission
	return fmt.Sprintf(`# ASTRACODE ADMISSION-ONLY MASSIVE SUBMISSION BURST %s

Classification: **%s**

## Submission API intake (POST responses only)

- Intended / attempted / accepted: %d / %d / %d
- Acceptance: %s
- Client launch attempted throughput: %s submissions/s
- Effective accepted intake: %s submissions/s
- POST-start spread: %v ms
- Submit p50 / p95 / p99: %s / %s / %s ms
- HTTP 429 / other 4xx / 5xx / transport / ambiguous: %d / %d / %d / %d / %d

## Scope boundary

- Observation mode: admission-only
- SSE tickets / streams / GET reconciliation: **not started**
- Terminal verdict, pipeline drain, E2E, and Judge Core: **not measured**
- Compile is excluded from this intake KPI.

## Post-burst probes

%s

## Client qualification

- Status: %s
- External container/Kafka survival evidence: %s (collect externally; unavailable is not zero)
- CLIENT_LIMITED is the run classification when direct local evidence emits a LOAD_GENERATOR_LIMITED quality flag (or reaches the configured in-flight bound). Generic TLS, connect, and deadline failures remain transport failures unless corroborating local resource evidence exists.

## Quality

%v
	`, summary.RunID, summary.Classification,
		summary.Counts.Intended, summary.Counts.Attempted, summary.Counts.Accepted,
		percent(summary.Counts.Accepted, summary.Counts.Attempted),
		optionalFloat(burst.AttemptedThroughputPerSec), optionalFloat(admission.EffectiveAcceptedIntakePerSec), optionalInt(admission.PostStartSpreadMS),
		optionalFloat(summary.Latencies.SubmitMS.P50), optionalFloat(summary.Latencies.SubmitMS.P95), optionalFloat(summary.Latencies.SubmitMS.P99),
		summary.Counts.RateLimited, summary.Counts.Other4xx, summary.Counts.ServerErrors, summary.Counts.TransportFailures, summary.Counts.AmbiguousPosts,
		healthMarkdown(summary.HealthProbes), admission.ClientQualification, admission.ExternalSurvivalEvidence, summary.QualityFlags)
}

func percent(value, total int) string {
	if total == 0 {
		return "N/A"
	}
	return fmt.Sprintf("%.3f%%", float64(value)*100/float64(total))
}

func healthMarkdown(probes []model.HealthProbe) string {
	if len(probes) == 0 {
		return "- UNAVAILABLE"
	}
	text := ""
	for _, probe := range probes {
		text += fmt.Sprintf("- %s: %s (%.3f ms)\n", probe.Name, probe.Status, probe.LatencyMS)
	}
	return text
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

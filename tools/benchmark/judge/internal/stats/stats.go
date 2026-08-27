// Package stats provides deterministic benchmark-only aggregates. It never
// interprets client outstanding as Kafka lag.
package stats

import (
	"math"
	"sort"
	"time"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/config"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/model"
)

func Distribution(values []float64) model.Distribution {
	if len(values) == 0 {
		return model.Distribution{}
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	sum := 0.0
	for _, value := range sorted {
		sum += value
	}
	min, mean, max := sorted[0], sum/float64(len(sorted)), sorted[len(sorted)-1]
	variance := 0.0
	for _, value := range sorted {
		delta := value - mean
		variance += delta * delta
	}
	std := math.Sqrt(variance / float64(len(sorted)))
	var cv *float64
	if mean != 0 {
		value := std / mean
		cv = &value
	}
	return model.Distribution{Count: len(sorted), Min: &min, Mean: &mean, Std: &std, CV: cv, P50: percentile(sorted, .50), P90: percentile(sorted, .90), P95: percentile(sorted, .95), P99: percentile(sorted, .99), Max: &max}
}

// percentile uses the documented nearest-rank method.
func percentile(sorted []float64, p float64) *float64 {
	if len(sorted) == 0 {
		return nil
	}
	index := int(math.Ceil(p*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	value := sorted[index]
	return &value
}

type Ledger struct {
	accepted map[int64]struct{}
	terminal map[int64]struct{}
}

func NewLedger() *Ledger {
	return &Ledger{accepted: map[int64]struct{}{}, terminal: map[int64]struct{}{}}
}

func (l *Ledger) Accept(id int64) bool {
	if id <= 0 {
		return false
	}
	if _, exists := l.accepted[id]; exists {
		return false
	}
	l.accepted[id] = struct{}{}
	return true
}

func (l *Ledger) Complete(id int64) bool {
	if _, accepted := l.accepted[id]; !accepted {
		return false
	}
	if _, exists := l.terminal[id]; exists {
		return false
	}
	l.terminal[id] = struct{}{}
	return true
}

func (l *Ledger) Outstanding() int { return len(l.accepted) - len(l.terminal) }

type WindowInput struct {
	RunID           string
	Phase           model.Phase
	FilterPhase     *model.Phase
	Start           time.Time
	End             time.Time
	Origin          time.Time
	Window          time.Duration
	TargetRate      *float64
	Records         []model.SubmissionRecord
	PeakInFlight    int
	ActiveObservers int
}

type ledgerEvent struct {
	at       time.Time
	accepted bool
	id       int64
}

func BuildWindows(input WindowInput) []model.WindowRecord {
	if input.Window <= 0 || !input.End.After(input.Start) {
		return nil
	}
	ledger := NewLedger()
	var prior []ledgerEvent
	for _, record := range input.Records {
		if input.FilterPhase != nil && record.Phase != *input.FilterPhase {
			continue
		}
		if record.Accepted && record.SubmissionID != nil && record.PostCompletedAt != nil && record.PostCompletedAt.Before(input.Start) {
			prior = append(prior, ledgerEvent{at: *record.PostCompletedAt, accepted: true, id: *record.SubmissionID})
		}
		if record.TerminalObservedAt != nil && record.SubmissionID != nil && record.TerminalObservedAt.Before(input.Start) {
			prior = append(prior, ledgerEvent{at: *record.TerminalObservedAt, id: *record.SubmissionID})
		}
	}
	applyLedgerEvents(ledger, prior, nil)
	var windows []model.WindowRecord
	intendedCum, attemptedCum, acceptedCum, completedCum := 0, 0, 0, 0
	for windowStart, index := input.Start, 0; windowStart.Before(input.End); windowStart, index = windowStart.Add(input.Window), index+1 {
		windowEnd := windowStart.Add(input.Window)
		if windowEnd.After(input.End) {
			windowEnd = input.End
		}
		record := model.WindowRecord{RunID: input.RunID, Phase: input.Phase, WindowIndex: index, WindowStart: windowStart.UTC(), WindowEnd: windowEnd.UTC(), WindowStartOffsetMS: windowStart.Sub(input.Origin).Milliseconds(), WindowEndOffsetMS: windowEnd.Sub(input.Origin).Milliseconds(), WindowDurationMS: windowEnd.Sub(windowStart).Milliseconds(), ClientOutstandingStart: ledger.Outstanding(), TargetRatePerSecond: input.TargetRate, PeakInFlight: input.PeakInFlight}
		peak := ledger.Outstanding()
		var events []ledgerEvent
		var schedule, submit, e2e []float64
		for _, submission := range input.Records {
			if input.FilterPhase != nil && submission.Phase != *input.FilterPhase {
				continue
			}
			if inWindow(submission.IntendedAt, windowStart, windowEnd) {
				record.Intended++
				intendedCum++
			}
			if inWindow(submission.PostStartedAt, windowStart, windowEnd) {
				record.Attempted++
				attemptedCum++
				if submission.ScheduleDelayMS != nil {
					schedule = append(schedule, *submission.ScheduleDelayMS)
				}
			}
			if inWindow(submission.PostCompletedAt, windowStart, windowEnd) && submission.Accepted {
				record.Accepted++
				acceptedCum++
				if submission.SubmissionID != nil {
					events = append(events, ledgerEvent{at: *submission.PostCompletedAt, accepted: true, id: *submission.SubmissionID})
				}
				if submission.SubmitLatencyMS != nil {
					submit = append(submit, *submission.SubmitLatencyMS)
				}
			}
			if inWindow(submission.TerminalObservedAt, windowStart, windowEnd) && submission.TerminalStatus != "" {
				record.Completed++
				completedCum++
				if submission.SubmissionID != nil {
					events = append(events, ledgerEvent{at: *submission.TerminalObservedAt, id: *submission.SubmissionID})
				}
				if submission.EndToEndLatencyMS != nil {
					e2e = append(e2e, *submission.EndToEndLatencyMS)
				}
			}
			if inWindow(submission.PostCompletedAt, windowStart, windowEnd) && submission.RateLimited {
				record.RateLimited++
			}
			if inWindow(submission.PostCompletedAt, windowStart, windowEnd) && submission.Outcome == model.OutcomeRejected4xx {
				record.Other4xx++
			}
			if inWindow(submission.PostCompletedAt, windowStart, windowEnd) && submission.Outcome == model.OutcomeServerError {
				record.ServerErrors++
			}
			if inWindow(submission.PostCompletedAt, windowStart, windowEnd) && submission.Outcome == model.OutcomeAmbiguousPost {
				record.TransportFailures++
			}
			if inWindow(submission.TerminalObservedAt, windowStart, windowEnd) && submission.Outcome == model.OutcomeCompletionTimeout {
				record.CompletionTimeouts++
			}
		}
		applyLedgerEvents(ledger, events, &peak)
		record.ActiveObserversEnd = activeObserversAt(input.Records, input.FilterPhase, windowEnd)
		record.IntendedCumulative, record.AttemptedCumulative, record.AcceptedCumulative, record.CompletedCumulative = intendedCum, attemptedCum, acceptedCum, completedCum
		record.ClientOutstanding, record.ClientOutstandingPeak = ledger.Outstanding(), peak
		seconds := windowEnd.Sub(windowStart).Seconds()
		record.AttemptedRatePerSecond, record.AcceptedRatePerSecond, record.CompletionRatePerSecond = float64(record.Attempted)/seconds, float64(record.Accepted)/seconds, float64(record.Completed)/seconds
		scheduleDist, submitDist, e2eDist := Distribution(schedule), Distribution(submit), Distribution(e2e)
		record.ScheduleDelayP50MS, record.ScheduleDelayP95MS, record.ScheduleDelayP99MS, record.ScheduleDelayMaxMS = scheduleDist.P50, scheduleDist.P95, scheduleDist.P99, scheduleDist.Max
		record.SubmitLatencyP50MS, record.SubmitLatencyP95MS = submitDist.P50, submitDist.P95
		record.E2ELatencyP50MS, record.E2ELatencyP95MS = e2eDist.P50, e2eDist.P95
		windows = append(windows, record)
	}
	return windows
}

func activeObserversAt(records []model.SubmissionRecord, phase *model.Phase, at time.Time) int {
	count := 0
	for _, record := range records {
		if phase != nil && record.Phase != *phase {
			continue
		}
		if record.ObserverStartedAt == nil || record.ObserverStartedAt.After(at) {
			continue
		}
		if record.ObserverEndedAt == nil || record.ObserverEndedAt.After(at) {
			count++
		}
	}
	return count
}

func applyLedgerEvents(ledger *Ledger, events []ledgerEvent, peak *int) {
	sort.SliceStable(events, func(left, right int) bool {
		if events[left].at.Equal(events[right].at) {
			// An accepted response precedes a terminal observation at the same
			// client timestamp, ensuring a valid accepted-before-terminal ledger.
			return events[left].accepted && !events[right].accepted
		}
		return events[left].at.Before(events[right].at)
	})
	for _, event := range events {
		if event.accepted {
			ledger.Accept(event.id)
		} else {
			ledger.Complete(event.id)
		}
		if peak != nil && ledger.Outstanding() > *peak {
			*peak = ledger.Outstanding()
		}
	}
}

func inWindow(value *time.Time, start, end time.Time) bool {
	return value != nil && !value.Before(start) && value.Before(end)
}

type ClassificationInput struct {
	Mode                 model.Mode
	Objective            config.Objective
	TargetRate           float64
	Windows              []model.WindowRecord
	Summary              model.RunSummary
	HasExternalLagGrowth bool
}

func Classify(input ClassificationInput) (model.Classification, []string) {
	if input.Mode == model.ModeBurst {
		return model.ClassificationNA, []string{"BURST_MODE"}
	}
	if input.Summary.Counts.Accepted < 20 || len(input.Windows) < 3 {
		return model.ClassificationInconclusive, []string{"INSUFFICIENT_SAMPLE"}
	}
	if len(input.Summary.QualityFlags) > 0 {
		return model.ClassificationInconclusive, input.Summary.QualityFlags
	}
	if input.Objective != config.ObjectiveAdmissionControl && input.Summary.Counts.RateLimited > 0 {
		return model.ClassificationInconclusive, []string{"UNEXPECTED_429"}
	}
	if input.Summary.Counts.AmbiguousPosts > 0 || input.Summary.Counts.UserPoolExhaustions > 0 || input.Summary.Counts.CompletionTimeouts > 0 || input.Summary.Counts.TransportFailures > 0 || input.Summary.Counts.Other4xx > 0 || input.Summary.Counts.ServerErrors > 0 {
		return model.ClassificationInconclusive, []string{"REQUEST_OR_OBSERVER_FAILURE"}
	}
	last := lastWindows(input.Windows, 3)
	growing := strictlyGrowingOutstanding(last)
	accepted, completed := totalAccepted(last), totalCompleted(last)
	if accepted > 0 && float64(completed) < .90*float64(accepted) && growing && (input.Summary.Drain.TimedOut || input.HasExternalLagGrowth || latencyGrowth(input.Summary.Latencies.EndToEndMS)) {
		return model.ClassificationSaturated, []string{"CLIENT_OUTSTANDING_GROWTH", "COMPLETION_DEFICIT"}
	}
	if input.TargetRate > 0 && within(input.Summary.Rates.LoadAttemptedPerSecond, input.TargetRate, .05) && balanced(last) && !growing && !latencyGrowth(input.Summary.Latencies.EndToEndMS) && !input.Summary.Drain.TimedOut && input.Summary.Drain.Remaining == 0 {
		return model.ClassificationStable, []string{"BALANCED_WINDOWS", "PREDICTABLE_DRAIN"}
	}
	return model.ClassificationInconclusive, []string{"INSUFFICIENT_STEADY_STATE_EVIDENCE"}
}

func lastWindows(values []model.WindowRecord, count int) []model.WindowRecord {
	if len(values) <= count {
		return values
	}
	return values[len(values)-count:]
}
func totalAccepted(values []model.WindowRecord) int {
	total := 0
	for _, value := range values {
		total += value.Accepted
	}
	return total
}
func totalCompleted(values []model.WindowRecord) int {
	total := 0
	for _, value := range values {
		total += value.Completed
	}
	return total
}
func strictlyGrowingOutstanding(values []model.WindowRecord) bool {
	if len(values) < 3 {
		return false
	}
	for i := 1; i < len(values); i++ {
		if values[i].ClientOutstanding <= values[i-1].ClientOutstanding {
			return false
		}
	}
	return true
}
func balanced(values []model.WindowRecord) bool {
	accepted, completed := totalAccepted(values), totalCompleted(values)
	return accepted > 0 && math.Abs(float64(accepted-completed)) <= math.Max(2, .10*float64(accepted))
}
func latencyGrowth(distribution model.Distribution) bool {
	return distribution.Count > 0 && distribution.P50 != nil && distribution.P95 != nil && *distribution.P95 > *distribution.P50*1.25
}
func within(value, target, tolerance float64) bool { return math.Abs(value-target) <= target*tolerance }

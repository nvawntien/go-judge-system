package stats

import (
	"testing"
	"time"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/config"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/model"
)

func TestDistributionUsesNearestRank(t *testing.T) {
	d := Distribution([]float64{1, 2, 3, 4})
	if d.P50 == nil || *d.P50 != 2 || d.P95 == nil || *d.P95 != 4 {
		t.Fatalf("distribution=%+v", d)
	}
	if Distribution(nil).P50 != nil {
		t.Fatal("empty distribution fabricated percentile")
	}
}

func TestLedgerDeduplicatesAndNeverGoesNegative(t *testing.T) {
	l := NewLedger()
	if l.Complete(1) || l.Outstanding() != 0 {
		t.Fatal("completion before acceptance changed ledger")
	}
	if !l.Accept(1) || !l.Complete(1) || l.Complete(1) || l.Outstanding() != 0 {
		t.Fatal("ledger did not deduplicate")
	}
}

func TestBuildWindowsUsesHalfOpenBoundary(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	id := int64(1)
	boundary := start.Add(time.Second)
	records := []model.SubmissionRecord{{Phase: model.PhaseLoad, IntendedAt: &start, PostStartedAt: &start, PostCompletedAt: &start, Accepted: true, SubmissionID: &id}, {Phase: model.PhaseLoad, TerminalObservedAt: &boundary, SubmissionID: &id, TerminalStatus: "ACCEPTED"}}
	windows := BuildWindows(WindowInput{Phase: model.PhaseLoad, Start: start, End: start.Add(2 * time.Second), Origin: start, Window: time.Second, Records: records})
	if len(windows) != 2 || windows[0].Completed != 0 || windows[1].Completed != 1 {
		t.Fatalf("windows=%+v", windows)
	}
}

func TestClassifyTreatsUnexpectedRateLimitAsInconclusive(t *testing.T) {
	summary := model.RunSummary{Counts: model.Counts{Accepted: 50, RateLimited: 1}}
	classification, reasons := Classify(ClassificationInput{Mode: model.ModeSustained, Objective: config.ObjectiveJudgeCapacity, TargetRate: 1, Windows: make([]model.WindowRecord, 4), Summary: summary})
	if classification != model.ClassificationInconclusive || len(reasons) != 1 || reasons[0] != "UNEXPECTED_429" {
		t.Fatalf("classification=%s reasons=%v", classification, reasons)
	}
}

func TestBuildWindowsTracksOutstandingAcrossLoadAndDrain(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	id := int64(1)
	accepted := start.Add(100 * time.Millisecond)
	terminal := start.Add(1500 * time.Millisecond)
	observerStart := accepted
	observerEnd := terminal
	phase := model.PhaseLoad
	records := []model.SubmissionRecord{{Phase: phase, IntendedAt: &start, PostStartedAt: &start, PostCompletedAt: &accepted, Accepted: true, SubmissionID: &id, ObserverStartedAt: &observerStart, ObserverEndedAt: &observerEnd, TerminalObservedAt: &terminal, TerminalStatus: "ACCEPTED"}}
	load := BuildWindows(WindowInput{Phase: model.PhaseLoad, FilterPhase: &phase, Start: start, End: start.Add(time.Second), Origin: start, Window: time.Second, Records: records})
	drain := BuildWindows(WindowInput{Phase: model.PhaseDrain, FilterPhase: &phase, Start: start.Add(time.Second), End: start.Add(2 * time.Second), Origin: start, Window: time.Second, Records: records})
	if load[0].Accepted != 1 || load[0].Completed != 0 || load[0].ClientOutstanding != 1 || load[0].ActiveObserversEnd != 1 {
		t.Fatalf("load=%+v", load[0])
	}
	if drain[0].Completed != 1 || drain[0].ClientOutstanding != 0 || drain[0].ActiveObserversEnd != 0 {
		t.Fatalf("drain=%+v", drain[0])
	}
}

func TestBuildWindowsUsesPartialFinalDurationAndLoadEndIsExcluded(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(2500 * time.Millisecond)
	first := start
	last := start.Add(2 * time.Second)
	records := []model.SubmissionRecord{{Phase: model.PhaseLoad, IntendedAt: &first, PostStartedAt: &first}, {Phase: model.PhaseLoad, IntendedAt: &last, PostStartedAt: &last}}
	windows := BuildWindows(WindowInput{Phase: model.PhaseLoad, Start: start, End: end, Origin: start, Window: time.Second, Records: records})
	if len(windows) != 3 || windows[2].WindowDurationMS != 500 || windows[2].AttemptedRatePerSecond != 2 {
		t.Fatalf("windows=%+v", windows)
	}
	boundary := end
	records = append(records, model.SubmissionRecord{Phase: model.PhaseLoad, PostStartedAt: &boundary})
	windows = BuildWindows(WindowInput{Phase: model.PhaseLoad, Start: start, End: end, Origin: start, Window: time.Second, Records: records})
	if windows[2].Attempted != 1 {
		t.Fatalf("load-end event leaked into half-open final window: %+v", windows[2])
	}
}

func TestClassifyStableAndSaturatedFixtures(t *testing.T) {
	stableWindows := []model.WindowRecord{
		{Accepted: 10, Completed: 10, ClientOutstanding: 0},
		{Accepted: 10, Completed: 10, ClientOutstanding: 0},
		{Accepted: 10, Completed: 10, ClientOutstanding: 0},
		{Accepted: 10, Completed: 10, ClientOutstanding: 0},
	}
	stable := model.RunSummary{Counts: model.Counts{Accepted: 40}, Rates: model.Rates{LoadAttemptedPerSecond: 1}, Drain: model.Drain{Remaining: 0}}
	if got, _ := Classify(ClassificationInput{Mode: model.ModeSustained, Objective: config.ObjectiveJudgeCapacity, TargetRate: 1, Windows: stableWindows, Summary: stable}); got != model.ClassificationStable {
		t.Fatalf("stable fixture classified %s", got)
	}
	saturatedWindows := []model.WindowRecord{
		{Accepted: 10, Completed: 1, ClientOutstanding: 9},
		{Accepted: 10, Completed: 1, ClientOutstanding: 18},
		{Accepted: 10, Completed: 1, ClientOutstanding: 27},
		{Accepted: 10, Completed: 1, ClientOutstanding: 36},
	}
	saturated := model.RunSummary{Counts: model.Counts{Accepted: 40}, Drain: model.Drain{TimedOut: true}}
	if got, _ := Classify(ClassificationInput{Mode: model.ModeSustained, Objective: config.ObjectiveJudgeCapacity, TargetRate: 1, Windows: saturatedWindows, Summary: saturated}); got != model.ClassificationSaturated {
		t.Fatalf("saturated fixture classified %s", got)
	}
}

func TestClassifyTwoLoadWindowsIsInconclusive(t *testing.T) {
	summary := model.RunSummary{Counts: model.Counts{Accepted: 20}}
	classification, reasons := Classify(ClassificationInput{Mode: model.ModeSustained, Objective: config.ObjectiveJudgeCapacity, TargetRate: 1, Windows: []model.WindowRecord{{Accepted: 10, Completed: 1, ClientOutstanding: 9}, {Accepted: 10, Completed: 1, ClientOutstanding: 18}}, Summary: summary})
	if classification != model.ClassificationInconclusive || len(reasons) != 1 || reasons[0] != "INSUFFICIENT_SAMPLE" {
		t.Fatalf("classification=%s reasons=%v", classification, reasons)
	}
}

package report

import (
	"strings"
	"testing"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/model"
)

func TestMassiveBurstReportSeparatesJudgeCoreFromPipelineTerminalRate(t *testing.T) {
	rate, coverage := 1.043, 0.311
	text := Markdown(model.RunSummary{
		RunID: "B-R1", Counts: model.Counts{Accepted: 1000, Terminal: 311},
		Compile:   model.CompileMetrics{Availability: "UNAVAILABLE", Reason: "compile timestamps unavailable"},
		JudgeCore: model.JudgeCoreMetrics{Availability: "UNAVAILABLE", Reason: "phase timestamps unavailable"},
		Pipeline:  model.PipelineMetrics{TerminalObservationCoverage: &coverage, RightCensored: true},
		Burst:     &model.BurstMetrics{Massive: true, PipelineTerminalThroughputPerSec: &rate},
	})
	for _, want := range []string{"Compile and Judge Core", "Pipeline terminal and E2E", "Observed pipeline-terminal throughput", "UNAVAILABLE"} {
		if !strings.Contains(text, want) {
			t.Fatalf("report missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Effective terminal throughput") || strings.Contains(text, "Judge / E2E") {
		t.Fatalf("ambiguous historical terminology remained:\n%s", text)
	}
}

func TestAdmissionOnlyReportUsesAttemptedPostsForAcceptance(t *testing.T) {
	text := Markdown(model.RunSummary{
		RunID: "admission-denominator",
		Counts: model.Counts{
			Intended:  100,
			Attempted: 80,
			Accepted:  72,
		},
		Burst:     &model.BurstMetrics{},
		Admission: &model.AdmissionMetrics{ObservationMode: "admission-only"},
	})
	if !strings.Contains(text, "- Acceptance: 90.000%") {
		t.Fatalf("report did not use accepted/attempted:\n%s", text)
	}
	if strings.Contains(text, "- Acceptance: 72.000%") {
		t.Fatalf("report still used accepted/intended:\n%s", text)
	}
}

func TestAdmissionOnlyReportMakesZeroAttemptAcceptanceUnavailable(t *testing.T) {
	text := Markdown(model.RunSummary{
		RunID:     "admission-zero-attempts",
		Counts:    model.Counts{Intended: 100},
		Burst:     &model.BurstMetrics{},
		Admission: &model.AdmissionMetrics{ObservationMode: "admission-only"},
	})
	if !strings.Contains(text, "- Acceptance: N/A") {
		t.Fatalf("report did not make zero-attempt acceptance unavailable:\n%s", text)
	}
}

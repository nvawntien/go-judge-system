package backfill

import (
	"bytes"
	"context"
	"testing"

	"go-judge-system/services/problem/internal/domain/entity"
)

func TestSplitLegacyDescription(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		description string
		wantReason  string
		wantInput   string
		wantOutput  string
	}{
		{
			name:        "standalone headers with source attribution",
			description: "Story.\n\nInput\n\nRead n.\n\nOutput Format:\n\nPrint n.\n\nSource: example",
			wantInput:   "Read n.", wantOutput: "Print n.",
		},
		{name: "markdown headers", description: "Story\n\n## Input Format\nRead n\n\n### Output\nPrint n", wantInput: "Read n", wantOutput: "Print n"},
		{name: "inline words are not headers", description: "Input is described here.\n\nOutput is described here.", wantReason: "no standalone Input and Output headings"},
		{name: "duplicate is ambiguous", description: "Story\n\nInput\na\n\nInput\nb\n\nOutput\nc", wantReason: "ambiguous: multiple Input headings"},
		{name: "inverted is ambiguous", description: "Story\n\nOutput\na\n\nInput\nb", wantReason: "ambiguous: Input must precede Output"},
		{name: "empty segment is skipped", description: "Story\n\nInput\n\nOutput\nvalue", wantReason: "empty description, input format, or output format"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, input, output, reason := SplitLegacyDescription(test.description)
			if reason != test.wantReason || input != test.wantInput || output != test.wantOutput {
				t.Fatalf("SplitLegacyDescription() = input %q, output %q, reason %q", input, output, reason)
			}
		})
	}
}

func TestRunIsIdempotentAndReportsSkips(t *testing.T) {
	t.Parallel()
	repo := &fakeRepository{problems: []*entity.Problem{
		{ID: 1, TitleSlug: "ready", Description: "Story\n\nInput\na\n\nOutput\nb"},
		{ID: 2, TitleSlug: "existing", Description: "old", InputFormat: "already"},
		{ID: 3, TitleSlug: "unknown", Description: "plain prose"},
	}}
	var out bytes.Buffer
	result, err := Run(context.Background(), repo, true, &out)
	if err != nil {
		t.Fatal(err)
	}
	if result.Scanned != 3 || result.Migrated != 1 || result.SkippedPopulated != 1 || result.SkippedNoHeaders != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if repo.updates != 1 || repo.problems[0].InputFormat != "a" || repo.problems[0].OutputFormat != "b" {
		t.Fatalf("unexpected update: %+v", repo.problems[0])
	}

	result, err = Run(context.Background(), repo, true, &out)
	if err != nil {
		t.Fatal(err)
	}
	if result.Migrated != 0 || result.SkippedPopulated != 2 || repo.updates != 1 {
		t.Fatalf("second run was not idempotent: %+v", result)
	}
}

type fakeRepository struct {
	problems []*entity.Problem
	updates  int
}

func (r *fakeRepository) ListForFormatBackfill(context.Context) ([]*entity.Problem, error) {
	return r.problems, nil
}

func (r *fakeRepository) UpdateFormatsForBackfill(_ context.Context, id int64, description, inputFormat, outputFormat string) (bool, error) {
	for _, problem := range r.problems {
		if problem.ID == id && problem.InputFormat == "" && problem.OutputFormat == "" {
			problem.Description = description
			problem.InputFormat = inputFormat
			problem.OutputFormat = outputFormat
			r.updates++
			return true, nil
		}
	}
	return false, nil
}

package judge

import (
	"testing"

	judgev1 "go-judge-system/pkg/pb/judge/v1"
	"go-judge-system/services/submission/internal/application/dto"
)

func TestMapRunCodeStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "completed", in: dto.RunCodeStatusCompleted, want: dto.RunCodeStatusCompleted},
		{name: "compile error", in: dto.RunCodeStatusCompilationError, want: dto.RunCodeStatusCompilationError},
		{name: "system error", in: dto.RunCodeStatusSystemError, want: dto.RunCodeStatusSystemError},
		{name: "unknown falls back to system error", in: "COMPILATION_ERROR", want: dto.RunCodeStatusSystemError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := mapRunCodeStatus(tt.in); got != tt.want {
				t.Fatalf("mapRunCodeStatus(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMapCodeDiagnostics(t *testing.T) {
	t.Parallel()

	endLine := int32(7)
	endColumn := int32(12)
	testcaseID := "custom-1"
	got := mapCodeDiagnostics([]*judgev1.CodeDiagnostic{{
		Kind:       "runtime",
		Severity:   "error",
		Message:    "panic: boom",
		Line:       7,
		Column:     3,
		EndLine:    &endLine,
		EndColumn:  &endColumn,
		TestcaseId: &testcaseID,
	}})

	if len(got) != 1 {
		t.Fatalf("diagnostics length = %d, want 1", len(got))
	}
	if got[0].Kind != "runtime" || got[0].Severity != "error" || got[0].Message != "panic: boom" ||
		got[0].Line != 7 || got[0].Column != 3 ||
		got[0].EndLine == nil || *got[0].EndLine != 7 ||
		got[0].EndColumn == nil || *got[0].EndColumn != 12 ||
		got[0].TestCaseID == nil || *got[0].TestCaseID != testcaseID {
		t.Fatalf("diagnostic = %+v", got[0])
	}
}

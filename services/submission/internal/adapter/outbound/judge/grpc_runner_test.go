package judge

import (
	"testing"

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

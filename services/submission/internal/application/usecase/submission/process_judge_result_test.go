package submission

import (
	"testing"

	"go-judge-system/services/submission/internal/domain/entity"
)

func TestParseSubmissionStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "accepted", input: "ACCEPTED", wantErr: false},
		{name: "wrong answer", input: "WRONG_ANSWER", wantErr: false},
		{name: "tle", input: "TLE", wantErr: false},
		{name: "mle", input: "MLE", wantErr: false},
		{name: "runtime", input: "RUNTIME_ERROR", wantErr: false},
		{name: "compile", input: "COMPILATION_ERROR", wantErr: false},
		{name: "invalid", input: "UNKNOWN", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseSubmissionStatus(tc.input)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for %q", tc.input)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.input, err)
			}
		})
	}
}

func TestParseResultStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    entity.ResultStatus
		wantErr bool
	}{
		{name: "accepted", input: "ACCEPTED", want: entity.ResultAccepted, wantErr: false},
		{name: "wrong answer", input: "WRONG_ANSWER", want: entity.ResultWrongAnswer, wantErr: false},
		{name: "tle", input: "TLE", want: entity.ResultTimeLimit, wantErr: false},
		{name: "mle", input: "MLE", want: entity.ResultMemoryLimit, wantErr: false},
		{name: "runtime", input: "RUNTIME_ERROR", want: entity.ResultRuntimeError, wantErr: false},
		{name: "invalid", input: "COMPILATION_ERROR", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseResultStatus(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("parseResultStatus(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

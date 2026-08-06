package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
)

func TestRunCodeUseCaseValidation(t *testing.T) {
	claims := auth.Claims{UserID: "u1", Username: "alice", Role: rbac.RoleUser}

	tests := []struct {
		name string
		req  dto.RunCodeRequest
		want error
	}{
		{
			name: "duplicate testcase IDs",
			req:  validRunRequest([]dto.RunTestCaseInput{{ID: "case-1", Kind: "sample", Stdin: "1\n", ExpectedOutput: stringPtr("1\n")}, {ID: "case-1", Kind: "custom", Stdin: "2\n"}}),
			want: domain.ErrInvalidRunTestCase,
		},
		{
			name: "empty testcase array",
			req:  validRunRequest(nil),
			want: domain.ErrInvalidRunTestCase,
		},
		{
			name: "sample requires expected output",
			req:  validRunRequest([]dto.RunTestCaseInput{{ID: "sample-1", Kind: "sample", Stdin: "1\n"}}),
			want: domain.ErrInvalidRunTestCase,
		},
		{
			name: "unsupported language",
			req:  dto.RunCodeRequest{ProblemID: 1, Language: "JAVASCRIPT", SourceCode: "code", TestCases: []dto.RunTestCaseInput{{ID: "custom-1", Kind: "custom", Stdin: "1\n"}}},
			want: domain.ErrUnsupportedRunLanguage,
		},
		{
			name: "custom expected output optional",
			req:  validRunRequest([]dto.RunTestCaseInput{{ID: "custom-1", Kind: "custom", Stdin: "1\n"}}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewRunCodeUseCase(&stubProblemReader{}, &stubJudgeRunner{}, dto.RunCodeLimits{RequestTimeout: time.Second})
			_, err := uc.Execute(context.Background(), claims, tt.req)
			if tt.want == nil {
				if err != nil {
					t.Fatalf("Execute() error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestRunCodeUseCasePreservesResultOrder(t *testing.T) {
	uc := NewRunCodeUseCase(&stubProblemReader{}, &stubJudgeRunner{}, dto.RunCodeLimits{RequestTimeout: time.Second})
	res, err := uc.Execute(
		context.Background(),
		auth.Claims{UserID: "u1", Username: "alice", Role: rbac.RoleUser},
		validRunRequest([]dto.RunTestCaseInput{
			{ID: "sample-1", Kind: "sample", Stdin: "1\n", ExpectedOutput: stringPtr("1\n")},
			{ID: "custom-1", Kind: "custom", Stdin: "2\n"},
		}),
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := []string{res.Tests[0].ID, res.Tests[1].ID}; got[0] != "sample-1" || got[1] != "custom-1" {
		t.Fatalf("result order = %v, want [sample-1 custom-1]", got)
	}
}

func validRunRequest(testCases []dto.RunTestCaseInput) dto.RunCodeRequest {
	return dto.RunCodeRequest{
		ProblemID:  1,
		Language:   "GO",
		SourceCode: "package main\nfunc main() {}\n",
		TestCases:  testCases,
	}
}

type stubProblemReader struct{}

func (s *stubProblemReader) GetProblem(context.Context, int64, dto.ProblemActor) (dto.ProblemMetadata, error) {
	return dto.ProblemMetadata{ID: 1, Title: "Two Sum", Slug: "two-sum", TimeLimit: 2, MemoryLimit: 256}, nil
}

func (s *stubProblemReader) GetTestCaseMetadata(context.Context, int64) (dto.ProblemTestCaseMetadata, error) {
	return dto.ProblemTestCaseMetadata{}, nil
}

type stubJudgeRunner struct{}

func (s *stubJudgeRunner) RunCode(_ context.Context, req outbound.JudgeRunRequest) (dto.RunCodeResponse, error) {
	tests := make([]dto.RunTestCaseResult, 0, len(req.TestCases))
	for _, tc := range req.TestCases {
		tests = append(tests, dto.RunTestCaseResult{ID: tc.ID, Kind: tc.Kind, Status: "executed"})
	}
	return dto.RunCodeResponse{Status: "completed", Tests: tests}, nil
}

func stringPtr(value string) *string {
	return &value
}

package judge

import (
	"context"
	"testing"

	"go-judge-system/workers/judge/internal/application/port/outbound"
)

type fakeRunExecutor struct {
	req    outbound.ExecutionRequest
	result *outbound.ExecutionResult
}

func (f *fakeRunExecutor) Execute(_ context.Context, req outbound.ExecutionRequest) (*outbound.ExecutionResult, error) {
	f.req = req
	return f.result, nil
}

func TestRunCodeUseCaseUsesSharedExecutorAndMapsResult(t *testing.T) {
	t.Parallel()

	expected := "1\n"
	executor := &fakeRunExecutor{
		result: &outbound.ExecutionResult{
			Status: "ACCEPTED",
			TestCases: []outbound.TestCaseResult{
				{Index: 1, ID: "sample-1", Kind: "sample", Status: "ACCEPTED", ExpectedOutput: &expected},
				{Index: 2, ID: "custom-1", Kind: "custom", Status: "ACCEPTED"},
			},
		},
	}
	useCase := NewRunCodeUseCase(executor)

	got, err := useCase.Execute(context.Background(), outbound.RunRequest{
		Language:   "GO",
		SourceCode: "package main\nfunc main(){}",
		TestCases: []outbound.RunTestCase{
			{ID: "sample-1", Kind: "sample", Stdin: "1\n", ExpectedOutput: &expected},
			{ID: "custom-1", Kind: "custom", Stdin: "2\n"},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if executor.req.StopOnFirstFailure {
		t.Fatal("StopOnFirstFailure = true, want false for run code")
	}
	if got.Status != "completed" || got.TestCases[0].Status != "accepted" || got.TestCases[1].Status != "executed" {
		t.Fatalf("run result = %#v", got)
	}
}

func TestRunCodeUseCaseMapsCompileErrorDiagnostics(t *testing.T) {
	t.Parallel()

	compileOutput := "main.go:19:9: make (built-in) must be called"
	executor := &fakeRunExecutor{
		result: &outbound.ExecutionResult{
			Status:        "COMPILATION_ERROR",
			CompileOutput: &compileOutput,
			Diagnostics: []outbound.CodeDiagnostic{{
				Kind:     "compile",
				Severity: "error",
				Message:  "make (built-in) must be called",
				Line:     19,
				Column:   9,
			}},
		},
	}

	got, err := NewRunCodeUseCase(executor).Execute(context.Background(), outbound.RunRequest{
		Language:   "GO",
		SourceCode: "bad",
		TestCases:  []outbound.RunTestCase{{ID: "sample-1", Kind: "sample"}},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.Status != "compile_error" || got.CompileOutput != compileOutput || len(got.Diagnostics) != 1 {
		t.Fatalf("run result = %#v, want compile_error with diagnostics", got)
	}
}

func TestRunCodeUseCaseMapsRuntimeDiagnostics(t *testing.T) {
	t.Parallel()

	testCaseID := "custom-1"
	executor := &fakeRunExecutor{
		result: &outbound.ExecutionResult{
			Status: "RUNTIME_ERROR",
			TestCases: []outbound.TestCaseResult{{
				ID:     testCaseID,
				Kind:   "custom",
				Status: "RUNTIME_ERROR",
				Diagnostics: []outbound.CodeDiagnostic{{
					TestCaseID: &testCaseID,
					Kind:       "runtime",
					Severity:   "error",
					Message:    "panic: boom",
					Line:       7,
					Column:     1,
				}},
			}},
		},
	}

	got, err := NewRunCodeUseCase(executor).Execute(context.Background(), outbound.RunRequest{
		Language:   "GO",
		SourceCode: "package main",
		TestCases:  []outbound.RunTestCase{{ID: testCaseID, Kind: "custom"}},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.Status != "completed" || got.TestCases[0].Status != "runtime_error" || len(got.Diagnostics) != 1 ||
		len(got.TestCases[0].Diagnostics) != 1 {
		t.Fatalf("run result = %#v, want runtime test diagnostic", got)
	}
}

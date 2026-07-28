package judge

import (
	"context"

	"go-judge-system/workers/judge/internal/application/port/outbound"
)

type RunCodeUseCase struct {
	executor outbound.CodeExecutor
}

const (
	runCodeStatusCompleted        = "completed"
	runCodeStatusCompilationError = "compile_error"
	runCodeStatusSystemError      = "system_error"
)

func NewRunCodeUseCase(executor outbound.CodeExecutor) *RunCodeUseCase {
	return &RunCodeUseCase{executor: executor}
}

func (u *RunCodeUseCase) Execute(ctx context.Context, req outbound.RunRequest) (*outbound.RunResult, error) {
	testCases := make([]outbound.ExecutionTestCase, 0, len(req.TestCases))
	for index, testCase := range req.TestCases {
		testCases = append(testCases, outbound.ExecutionTestCase{
			Index:          index + 1,
			ID:             testCase.ID,
			Kind:           testCase.Kind,
			Stdin:          testCase.Stdin,
			ExpectedOutput: testCase.ExpectedOutput,
		})
	}

	result, err := u.executor.Execute(ctx, outbound.ExecutionRequest{
		Language:           req.Language,
		SourceCode:         req.SourceCode,
		TestCases:          testCases,
		Limits:             req.Limits,
		StopOnFirstFailure: false,
	})
	if err != nil {
		return nil, err
	}

	return mapExecutionResultToRunResult(result), nil
}

func mapExecutionResultToRunResult(result *outbound.ExecutionResult) *outbound.RunResult {
	if result == nil {
		return &outbound.RunResult{Status: runCodeStatusSystemError, TestCases: []outbound.RunTestCaseResult{}}
	}

	status := runCodeStatusCompleted
	if result.Status == "COMPILATION_ERROR" {
		status = runCodeStatusCompilationError
	}
	compileOutput := ""
	if result.CompileOutput != nil {
		compileOutput = *result.CompileOutput
	}

	testCases := make([]outbound.RunTestCaseResult, 0, len(result.TestCases))
	for _, testCase := range result.TestCases {
		stdout := ""
		if testCase.ActualOutput != nil {
			stdout = *testCase.ActualOutput
		}
		stderr := ""
		if testCase.Stderr != nil {
			stderr = *testCase.Stderr
		}
		testCases = append(testCases, outbound.RunTestCaseResult{
			ID:              testCase.ID,
			Kind:            testCase.Kind,
			Status:          mapExecutionStatusToRunStatus(testCase.Status, testCase.ExpectedOutput),
			Stdout:          stdout,
			Stderr:          stderr,
			ExpectedOutput:  testCase.ExpectedOutput,
			ExecutionTimeMS: int64(testCase.ExecutionTime),
			MemoryUsedKB:    int64(testCase.MemoryUsed),
		})
	}

	return &outbound.RunResult{
		Status:        status,
		CompileOutput: compileOutput,
		TestCases:     testCases,
	}
}

func mapExecutionStatusToRunStatus(status string, expected *string) string {
	switch status {
	case "ACCEPTED":
		if expected == nil {
			return "executed"
		}
		return "accepted"
	case "WRONG_ANSWER":
		return "wrong_answer"
	case "TIME_LIMIT_EXCEEDED":
		return "time_limit_exceeded"
	case "MEMORY_LIMIT_EXCEEDED":
		return "memory_limit_exceeded"
	case "OUTPUT_LIMIT_EXCEEDED":
		return "output_limit_exceeded"
	case "RUNTIME_ERROR":
		return "runtime_error"
	default:
		return runCodeStatusSystemError
	}
}

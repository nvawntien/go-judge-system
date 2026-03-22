package outbound

import (
	"context"
)

// ExecutionResult represents the result of code execution
type ExecutionResult struct {
	Status        string // "ACCEPTED", "WRONG_ANSWER", "TLE", "MLE", "RUNTIME_ERROR", "COMPILATION_ERROR"
	CompileOutput *string
	TestCases     []TestCaseResult
	ExecutionTime int // milliseconds
	MemoryUsed    int // kilobytes
	Error         *string
}

type TestCaseResult struct {
	TestCaseID    int64
	Status        string
	ActualOutput  *string
	ExecutionTime int // milliseconds
	MemoryUsed    int // kilobytes
	Order         int
}

// CodeExecutor executes submitted code against test cases
type CodeExecutor interface {
	Execute(ctx context.Context, language, sourceCode string, testCases []TestCase) (*ExecutionResult, error)
}

type TestCase struct {
	ID     int64
	Input  string
	Output string
	Order  int
}

// ResultPublisher publishes judge results back to submission service.
// attemptID is forwarded from the original job for idempotency tracking.
type ResultPublisher interface {
	PublishResult(ctx context.Context, submissionID int64, attemptID string, result *ExecutionResult) error
}

// TestCaseFetcher retrieves test cases for a problem from an external source
// (e.g., Problem Service internal API).
type TestCaseFetcher interface {
	FetchTestCases(ctx context.Context, problemID int64) ([]TestCase, error)
}

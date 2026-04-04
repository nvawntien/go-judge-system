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
	Index          int
	Status         string
	ActualOutput   *string
	Input          *string // populated for failed tests only
	ExpectedOutput *string // populated for failed tests only
	ExecutionTime  int     // milliseconds
	MemoryUsed     int     // kilobytes
}

// TestCaseBundle represents a set of testcases cached on local disk.
// Dir points to the extracted directory containing {N}.in / {N}.out files.
// Worker does NOT call Cleanup — cache is kept for future requests.
type TestCaseBundle struct {
	Dir       string // e.g. /cache/testcases/problem_42/
	TestCount int
}

// TestCaseFetcher handles downloading & caching testcases.
// Uses local disk cache — only downloads from MinIO on cache miss or version change.
type TestCaseFetcher interface {
	FetchTestCases(ctx context.Context, problemID int64) (*TestCaseBundle, error)
}

// CodeExecutor executes submitted code against test cases.
// Receives a TestCaseBundle (disk path) instead of []TestCase (in-memory).
type CodeExecutor interface {
	Execute(ctx context.Context, language, sourceCode string, bundle *TestCaseBundle) (*ExecutionResult, error)
}

// ResultPublisher publishes judge results back to submission service.
// attemptID is forwarded from the original job for idempotency tracking.
type ResultPublisher interface {
	PublishResult(ctx context.Context, submissionID int64, attemptID string, result *ExecutionResult) error
}

package outbound

import (
	"context"
)

type ProblemTestCaseMetadata struct {
	ProblemID      int64
	ZipDownloadURL string
	TestCount      int
	Version        int
}

type ProblemTestCaseMetadataReader interface {
	GetTestCaseMetadata(ctx context.Context, problemID int64) (ProblemTestCaseMetadata, error)
}

type OfficialTestCaseLoader interface {
	Load(ctx context.Context, metadata ProblemTestCaseMetadata) (OfficialTestCaseBundle, error)
}

type OfficialTestCaseBundle struct {
	TestCases       []ExecutionTestCase
	TestCount       int
	Version         int
	DatasetChecksum string
}

// TestcaseDatasetIdentity is authoritative Worker-side dataset provenance for
// sandbox testcase-input cache entries. It deliberately contains no expected
// output and no sandbox-specific FileID.
type TestcaseDatasetIdentity struct {
	ProblemID       int64
	Version         int
	DatasetChecksum string
}

type ExecutionTestCase struct {
	Index          int
	ID             string
	Kind           string
	Stdin          string
	ExpectedOutput *string
}

type ExecutionRequest struct {
	Language           string
	SourceCode         string
	TestCases          []ExecutionTestCase
	Limits             ExecutionLimits
	StopOnFirstFailure bool
	TestcaseDataset    *TestcaseDatasetIdentity
}

// ExecutionResult represents the result of code execution
type ExecutionResult struct {
	Status          string // "ACCEPTED", "WRONG_ANSWER", "TLE", "MLE", "RUNTIME_ERROR", "COMPILATION_ERROR"
	CompileOutput   *string
	ErrorMessage    *string
	Diagnostics     []CodeDiagnostic
	TestCases       []TestCaseResult
	ExecutionTime   int // milliseconds
	MemoryUsed      int // kilobytes
	Error           *string
	TestcaseVersion *int
	TestCount       *int
	DatasetChecksum *string
}

type TestCaseResult struct {
	Index          int
	ID             string
	Kind           string
	Status         string
	ActualOutput   *string
	Stderr         *string
	Input          *string // populated for failed tests only
	ExpectedOutput *string // populated for failed tests only
	ExecutionTime  int     // milliseconds
	MemoryUsed     int     // kilobytes
	Diagnostics    []CodeDiagnostic
}

type CodeDiagnostic struct {
	TestCaseID *string
	Kind       string
	Severity   string
	Message    string
	Line       int
	Column     int
	EndLine    *int
	EndColumn  *int
}

type ExecutionLimits struct {
	TimeLimitMS      int64
	MemoryLimitKB    int64
	OutputLimitBytes int64
}

type RunTestCase struct {
	ID             string
	Kind           string
	Stdin          string
	ExpectedOutput *string
}

type RunRequest struct {
	Language   string
	SourceCode string
	TestCases  []RunTestCase
	Limits     ExecutionLimits
}

type RunResult struct {
	Status        string
	CompileOutput string
	Diagnostics   []CodeDiagnostic
	TestCases     []RunTestCaseResult
}

type RunTestCaseResult struct {
	ID              string
	Kind            string
	Status          string
	Stdout          string
	Stderr          string
	ExpectedOutput  *string
	ExecutionTimeMS int64
	MemoryUsedKB    int64
	Diagnostics     []CodeDiagnostic
}

type CodeExecutor interface {
	Execute(ctx context.Context, req ExecutionRequest) (*ExecutionResult, error)
}

// ResultPublisher publishes judge results back to submission service.
// attemptID is forwarded from the original job for idempotency tracking.
type ResultPublisher interface {
	PublishResult(ctx context.Context, submissionID int64, attemptID string, result *ExecutionResult) error
}

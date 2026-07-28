package judge

import (
	"context"
	"errors"
	"testing"

	pkgjudge "go-judge-system/pkg/judge"
	"go-judge-system/workers/judge/internal/application/port/outbound"
	workerdomain "go-judge-system/workers/judge/internal/domain"

	"go.uber.org/zap"
)

type fakeMetadataReader struct {
	metadata outbound.ProblemTestCaseMetadata
	err      error
}

func (f *fakeMetadataReader) GetTestCaseMetadata(context.Context, int64) (outbound.ProblemTestCaseMetadata, error) {
	return f.metadata, f.err
}

type fakeOfficialLoader struct {
	testCases []outbound.ExecutionTestCase
	err       error
}

func (f *fakeOfficialLoader) Load(context.Context, outbound.ProblemTestCaseMetadata) ([]outbound.ExecutionTestCase, error) {
	return f.testCases, f.err
}

type fakeExecutor struct {
	req    outbound.ExecutionRequest
	result *outbound.ExecutionResult
	err    error
}

func (f *fakeExecutor) Execute(_ context.Context, req outbound.ExecutionRequest) (*outbound.ExecutionResult, error) {
	f.req = req
	return f.result, f.err
}

type fakeResultPublisher struct {
	submissionID int64
	attemptID    string
	result       *outbound.ExecutionResult
}

func (f *fakeResultPublisher) PublishResult(
	_ context.Context,
	submissionID int64,
	attemptID string,
	result *outbound.ExecutionResult,
) error {
	f.submissionID = submissionID
	f.attemptID = attemptID
	f.result = result
	return nil
}

func TestProcessJudgeJobUsesOfficialLoaderSharedExecutorAndSanitizesResult(t *testing.T) {
	t.Parallel()

	expected := "2\n"
	input := "1\n"
	publisher := &fakeResultPublisher{}
	executor := &fakeExecutor{
		result: &outbound.ExecutionResult{
			Status: "WRONG_ANSWER",
			TestCases: []outbound.TestCaseResult{
				{Index: 1, Status: "WRONG_ANSWER", Input: &input, ExpectedOutput: &expected},
			},
		},
	}
	useCase := NewProcessJudgeJobUseCase(
		executor,
		publisher,
		&fakeMetadataReader{metadata: outbound.ProblemTestCaseMetadata{ProblemID: 42, TestCount: 1, Version: 1}},
		&fakeOfficialLoader{testCases: []outbound.ExecutionTestCase{{Index: 1, Stdin: input, ExpectedOutput: &expected}}},
		zap.NewNop(),
	)

	err := useCase.Execute(context.Background(), &pkgjudge.JobMessage{
		SubmissionID: 99,
		ProblemID:    42,
		AttemptID:    "attempt-a",
		Language:     "GO",
		SourceCode:   "package main\nfunc main(){}",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if publisher.submissionID != 99 || publisher.attemptID != "attempt-a" {
		t.Fatalf("published submission/attempt = %d/%q", publisher.submissionID, publisher.attemptID)
	}
	if !executor.req.StopOnFirstFailure {
		t.Fatal("StopOnFirstFailure = false, want true for submission")
	}
	if publisher.result.TestCases[0].Input != nil || publisher.result.TestCases[0].ExpectedOutput != nil {
		t.Fatalf("hidden testcase data leaked in result: %#v", publisher.result.TestCases[0])
	}
}

func TestProcessJudgeJobReturnsRetryableErrorWithoutPublishing(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("problem-service unavailable")
	publisher := &fakeResultPublisher{}
	useCase := NewProcessJudgeJobUseCase(
		&fakeExecutor{},
		publisher,
		&fakeMetadataReader{err: wantErr},
		&fakeOfficialLoader{},
		zap.NewNop(),
	)

	err := useCase.Execute(context.Background(), &pkgjudge.JobMessage{SubmissionID: 99, ProblemID: 42, AttemptID: "attempt-a"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
	if publisher.result != nil {
		t.Fatalf("published result = %#v, want nil", publisher.result)
	}
}

func TestProcessJudgeJobPublishesUserCodeVerdictsWithoutSystemError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		result *outbound.ExecutionResult
		want   string
	}{
		{
			name:   "compilation error",
			result: &outbound.ExecutionResult{Status: "COMPILATION_ERROR", CompileOutput: stringPtr("main.go:3:2: undefined: x")},
			want:   "COMPILATION_ERROR",
		},
		{
			name: "runtime error",
			result: &outbound.ExecutionResult{
				Status:    "RUNTIME_ERROR",
				TestCases: []outbound.TestCaseResult{{Index: 1, Status: "RUNTIME_ERROR"}},
			},
			want: "RUNTIME_ERROR",
		},
		{
			name: "time limit",
			result: &outbound.ExecutionResult{
				Status:    "TIME_LIMIT_EXCEEDED",
				TestCases: []outbound.TestCaseResult{{Index: 1, Status: "TIME_LIMIT_EXCEEDED"}},
			},
			want: "TIME_LIMIT_EXCEEDED",
		},
		{
			name: "memory limit",
			result: &outbound.ExecutionResult{
				Status:    "MEMORY_LIMIT_EXCEEDED",
				TestCases: []outbound.TestCaseResult{{Index: 1, Status: "MEMORY_LIMIT_EXCEEDED"}},
			},
			want: "MEMORY_LIMIT_EXCEEDED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publisher := &fakeResultPublisher{}
			useCase := NewProcessJudgeJobUseCase(
				&fakeExecutor{result: tt.result},
				publisher,
				&fakeMetadataReader{metadata: outbound.ProblemTestCaseMetadata{ProblemID: 42, TestCount: 1, Version: 1}},
				&fakeOfficialLoader{testCases: []outbound.ExecutionTestCase{{Index: 1, Stdin: "1\n"}}},
				zap.NewNop(),
			)

			err := useCase.Execute(context.Background(), &pkgjudge.JobMessage{
				SubmissionID: 99,
				ProblemID:    42,
				AttemptID:    "attempt-a",
				Language:     "GO",
				SourceCode:   "package main\nfunc main(){}",
			})
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if publisher.result == nil || publisher.result.Status != tt.want {
				t.Fatalf("published result = %#v, want %s", publisher.result, tt.want)
			}
		})
	}
}

func TestProcessJudgeJobPublishesNonRetryableSystemError(t *testing.T) {
	t.Parallel()

	publisher := &fakeResultPublisher{}
	useCase := NewProcessJudgeJobUseCase(
		&fakeExecutor{},
		publisher,
		&fakeMetadataReader{err: workerdomain.MarkNonRetryable(errors.New("testcase not found"))},
		&fakeOfficialLoader{},
		zap.NewNop(),
	)

	err := useCase.Execute(context.Background(), &pkgjudge.JobMessage{SubmissionID: 99, ProblemID: 42, AttemptID: "attempt-a"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if publisher.result == nil || publisher.result.Status != "SYSTEM_ERROR" {
		t.Fatalf("published result = %#v, want SYSTEM_ERROR", publisher.result)
	}
}

func stringPtr(value string) *string {
	return &value
}

package judge

import (
	"context"

	judgev1 "go-judge-system/pkg/pb/judge/v1"
	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcRunner struct {
	client judgev1.JudgeServiceClient
}

func NewGRPCRunner(client judgev1.JudgeServiceClient) outbound.JudgeRunner {
	return &grpcRunner{client: client}
}

func (r *grpcRunner) RunCode(ctx context.Context, req outbound.JudgeRunRequest) (dto.RunCodeResponse, error) {
	testCases := make([]*judgev1.RunTestCaseInput, 0, len(req.TestCases))
	for _, tc := range req.TestCases {
		testCases = append(testCases, &judgev1.RunTestCaseInput{
			Id:             tc.ID,
			Kind:           tc.Kind,
			Stdin:          tc.Stdin,
			ExpectedOutput: tc.ExpectedOutput,
		})
	}

	res, err := r.client.RunCode(ctx, &judgev1.RunCodeRequest{
		Language:         req.Language,
		SourceCode:       req.SourceCode,
		Testcases:        testCases,
		TimeLimitMs:      req.TimeLimitMS,
		MemoryLimitKb:    req.MemoryLimitKB,
		OutputLimitBytes: req.OutputLimitBytes,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.DeadlineExceeded:
			return dto.RunCodeResponse{}, domain.ErrJudgeWorkerTimeout.Wrap(err)
		case codes.Unavailable:
			return dto.RunCodeResponse{}, domain.ErrJudgeWorkerUnavailable.Wrap(err)
		default:
			return dto.RunCodeResponse{}, domain.ErrJudgeWorkerUnavailable.Wrap(err)
		}
	}

	out := dto.RunCodeResponse{
		Status:        res.GetStatus(),
		CompileOutput: res.GetCompileOutput(),
		Tests:         make([]dto.RunTestCaseResult, 0, len(res.GetTests())),
	}
	for _, tc := range res.GetTests() {
		out.Tests = append(out.Tests, dto.RunTestCaseResult{
			ID:              tc.GetId(),
			Kind:            tc.GetKind(),
			Status:          tc.GetStatus(),
			Stdout:          tc.GetStdout(),
			Stderr:          tc.GetStderr(),
			ExpectedOutput:  tc.ExpectedOutput,
			ExecutionTimeMS: tc.GetExecutionTimeMs(),
			MemoryUsedKB:    tc.GetMemoryUsedKb(),
		})
	}

	return out, nil
}

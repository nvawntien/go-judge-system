package handler

import (
	"context"

	judgev1 "go-judge-system/pkg/pb/judge/v1"
	"go-judge-system/workers/judge/internal/application/port/inbound"
	"go-judge-system/workers/judge/internal/application/port/outbound"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RunCodeHandler struct {
	useCase inbound.RunCodeUseCase
}

func NewRunCodeHandler(useCase inbound.RunCodeUseCase) *RunCodeHandler {
	return &RunCodeHandler{useCase: useCase}
}

func (h *RunCodeHandler) Handle(ctx context.Context, req *judgev1.RunCodeRequest) (*judgev1.RunCodeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	testCases := make([]outbound.RunTestCase, 0, len(req.GetTestcases()))
	for _, tc := range req.GetTestcases() {
		testCases = append(testCases, outbound.RunTestCase{
			ID:             tc.GetId(),
			Kind:           tc.GetKind(),
			Stdin:          tc.GetStdin(),
			ExpectedOutput: tc.ExpectedOutput,
		})
	}

	result, err := h.useCase.Execute(ctx, outbound.RunRequest{
		Language:   req.GetLanguage(),
		SourceCode: req.GetSourceCode(),
		TestCases:  testCases,
		Limits: outbound.ExecutionLimits{
			TimeLimitMS:      req.GetTimeLimitMs(),
			MemoryLimitKB:    req.GetMemoryLimitKb(),
			OutputLimitBytes: req.GetOutputLimitBytes(),
		},
	})
	if err != nil {
		if ctx.Err() != nil {
			return nil, status.Error(codes.DeadlineExceeded, "run code deadline exceeded")
		}
		return nil, status.Error(codes.Internal, "run code failed")
	}

	response := &judgev1.RunCodeResponse{
		Status:        result.Status,
		CompileOutput: result.CompileOutput,
		Tests:         make([]*judgev1.RunTestCaseResult, 0, len(result.TestCases)),
	}
	for _, tc := range result.TestCases {
		response.Tests = append(response.Tests, &judgev1.RunTestCaseResult{
			Id:              tc.ID,
			Kind:            tc.Kind,
			Status:          tc.Status,
			Stdout:          tc.Stdout,
			Stderr:          tc.Stderr,
			ExpectedOutput:  tc.ExpectedOutput,
			ExecutionTimeMs: tc.ExecutionTimeMS,
			MemoryUsedKb:    tc.MemoryUsedKB,
		})
	}

	return response, nil
}

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
		Diagnostics:   mapDiagnostics(result.Diagnostics),
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
			Diagnostics:     mapDiagnostics(tc.Diagnostics),
		})
	}

	return response, nil
}

func mapDiagnostics(items []outbound.CodeDiagnostic) []*judgev1.CodeDiagnostic {
	diagnostics := make([]*judgev1.CodeDiagnostic, 0, len(items))
	for _, item := range items {
		diagnostic := &judgev1.CodeDiagnostic{
			Kind:     item.Kind,
			Severity: item.Severity,
			Message:  item.Message,
			Line:     int32(item.Line),
			Column:   int32(item.Column),
		}
		if item.EndLine != nil {
			value := int32(*item.EndLine)
			diagnostic.EndLine = &value
		}
		if item.EndColumn != nil {
			value := int32(*item.EndColumn)
			diagnostic.EndColumn = &value
		}
		if item.TestCaseID != nil {
			diagnostic.TestcaseId = item.TestCaseID
		}
		diagnostics = append(diagnostics, diagnostic)
	}
	return diagnostics
}

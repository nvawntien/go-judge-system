package outbound

import (
	"context"

	"go-judge-system/services/submission/internal/application/dto"
)

type JudgeRunRequest struct {
	Language         string
	SourceCode       string
	TestCases        []JudgeRunTestCase
	TimeLimitMS      int64
	MemoryLimitKB    int64
	OutputLimitBytes int64
}

type JudgeRunTestCase struct {
	ID             string
	Kind           string
	Stdin          string
	ExpectedOutput *string
}

type JudgeRunner interface {
	RunCode(ctx context.Context, req JudgeRunRequest) (dto.RunCodeResponse, error)
}

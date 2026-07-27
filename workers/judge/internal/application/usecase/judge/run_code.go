package judge

import (
	"context"

	"go-judge-system/workers/judge/internal/application/port/outbound"
)

type RunCodeUseCase struct {
	executor outbound.CodeExecutor
}

func NewRunCodeUseCase(executor outbound.CodeExecutor) *RunCodeUseCase {
	return &RunCodeUseCase{executor: executor}
}

func (u *RunCodeUseCase) Execute(ctx context.Context, req outbound.RunRequest) (*outbound.RunResult, error) {
	return u.executor.RunCode(ctx, req)
}

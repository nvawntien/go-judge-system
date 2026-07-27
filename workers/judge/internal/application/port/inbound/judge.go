package inbound

import (
	"context"

	"go-judge-system/pkg/judge"
	"go-judge-system/workers/judge/internal/application/port/outbound"
)

// ProcessJudgeJobUseCase processes a judge job from the queue
type ProcessJudgeJobUseCase interface {
	Execute(ctx context.Context, job *judge.JobMessage) error
}

type RunCodeUseCase interface {
	Execute(ctx context.Context, req outbound.RunRequest) (*outbound.RunResult, error)
}

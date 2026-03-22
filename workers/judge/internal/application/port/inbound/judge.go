package inbound

import (
	"context"

	"go-judge-system/pkg/judge"
)

// ProcessJudgeJobUseCase processes a judge job from the queue
type ProcessJudgeJobUseCase interface {
	Execute(ctx context.Context, job *judge.JobMessage) error
}

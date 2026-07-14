package worker

import (
	"context"
	"go-judge-system/services/problem/internal/application/dto"
)

type GetTestCaseUseCase interface {
	Execute(ctx context.Context, problemID int64) (*dto.InternalTestCaseResponse, error)
}

package outbound

import (
	"context"
	"go-judge-system/services/problem/internal/domain/entity"
)

type TestCaseRepository interface {
	Upsert(ctx context.Context, tc *entity.TestCase) error
	GetByProblemID(ctx context.Context, problemID int64) (*entity.TestCase, error)
	DeleteByProblemID(ctx context.Context, problemID int64) error
}

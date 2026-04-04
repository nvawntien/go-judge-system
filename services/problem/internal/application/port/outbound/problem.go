package outbound

import (
	"context"

	"go-judge-system/services/problem/internal/domain/entity"
)

type ProblemRepository interface {
	Create(ctx context.Context, problem *entity.Problem) error
	GetByID(ctx context.Context, id int64) (*entity.Problem, error)
	GetBySlug(ctx context.Context, slug string) (*entity.Problem, error)
	Update(ctx context.Context, problem *entity.Problem) error
	Delete(ctx context.Context, id int64) error // soft delete
	List(ctx context.Context, offset, limit int, difficulty, search string, includeHidden bool) ([]*entity.Problem, error)
	Count(ctx context.Context, difficulty, search string, includeHidden bool) (int64, error)
	ListByAuthor(ctx context.Context, authorID string, offset, limit int, difficulty, search string) ([]*entity.Problem, error)
	CountByAuthor(ctx context.Context, authorID string, difficulty, search string) (int64, error)
}
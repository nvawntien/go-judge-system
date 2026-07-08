package outbound

import (
	"context"

	"go-judge-system/services/problem/internal/domain/entity"
)

type TagRepository interface {
	Create(ctx context.Context, tag *entity.Tag) error
	GetByID(ctx context.Context, id uint) (*entity.Tag, error)
	GetBySlug(ctx context.Context, slug string) (*entity.Tag, error)
	Update(ctx context.Context, tag *entity.Tag) error
	Deactivate(ctx context.Context, id uint) error
	ListActive(ctx context.Context) ([]*entity.Tag, error)
	ListAll(ctx context.Context) ([]*entity.Tag, error)
	ListByIDs(ctx context.Context, ids []uint, activeOnly bool) ([]*entity.Tag, error)
	CountPublishedProblemsByTagID(ctx context.Context, tagID uint) (int64, error)
}

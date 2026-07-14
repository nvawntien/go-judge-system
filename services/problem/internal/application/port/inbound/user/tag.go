package user

import (
	"context"

	"go-judge-system/services/problem/internal/application/dto"
)

type ListTagsUseCase interface {
	Execute(ctx context.Context) (dto.ListTagsResponse, error)
}

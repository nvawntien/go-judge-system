package admin

import (
	"context"

	"go-judge-system/pkg/auth"
	"go-judge-system/services/problem/internal/application/dto"
)

type ListTagsUseCase interface {
	Execute(ctx context.Context, claims auth.Claims) (dto.AdminListTagsResponse, error)
}

type CreateTagUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, req dto.CreateTagRequest) (dto.AdminTagResponse, error)
}

type UpdateTagUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, params dto.TagIDRequest, req dto.UpdateTagRequest) (dto.AdminTagResponse, error)
}

type DeleteTagUseCase interface {
	Execute(ctx context.Context, claims auth.Claims, params dto.TagIDRequest) error
}

package tag

import (
	"context"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/problem/internal/application/dto"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/admin"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/application/usecase"
	"go-judge-system/services/problem/internal/domain"
)

type listTagsUseCase struct {
	tagRepo outbound.TagRepository
}

func NewListTagsUseCase(tagRepo outbound.TagRepository) inbound.ListTagsUseCase {
	return &listTagsUseCase{tagRepo: tagRepo}
}

func (uc *listTagsUseCase) Execute(ctx context.Context, claims auth.Claims) (dto.AdminListTagsResponse, error) {
	if !claims.Role.AtLeast(rbac.RoleModerator) {
		return dto.AdminListTagsResponse{}, domain.ErrForbidden
	}

	tags, err := uc.tagRepo.ListAll(ctx)
	if err != nil {
		return dto.AdminListTagsResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	items := make([]dto.AdminTagResponse, 0, len(tags))
	for _, tag := range tags {
		items = append(items, usecase.MapTagToAdminResponse(tag))
	}

	return dto.AdminListTagsResponse{Items: items}, nil
}

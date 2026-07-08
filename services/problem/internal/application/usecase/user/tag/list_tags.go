package tag

import (
	"context"

	"go-judge-system/services/problem/internal/application/dto"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/user"
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

func (uc *listTagsUseCase) Execute(ctx context.Context) (dto.ListTagsResponse, error) {
	tags, err := uc.tagRepo.ListActive(ctx)
	if err != nil {
		return dto.ListTagsResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	items := make([]dto.TagResponse, 0, len(tags))
	for _, tag := range tags {
		items = append(items, usecase.MapTagToResponse(tag))
	}

	return dto.ListTagsResponse{Items: items}, nil
}

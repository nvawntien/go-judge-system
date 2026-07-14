package tag

import (
	"context"
	"errors"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/pkg/response"
	"go-judge-system/services/problem/internal/application/dto"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/admin"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/application/usecase"
	"go-judge-system/services/problem/internal/domain"
	"go-judge-system/services/problem/internal/domain/entity"
)

type createTagUseCase struct {
	tagRepo outbound.TagRepository
}

func NewCreateTagUseCase(tagRepo outbound.TagRepository) inbound.CreateTagUseCase {
	return &createTagUseCase{tagRepo: tagRepo}
}

func (uc *createTagUseCase) Execute(ctx context.Context, claims auth.Claims, req dto.CreateTagRequest) (dto.AdminTagResponse, error) {
	if !claims.Role.AtLeast(rbac.RoleModerator) {
		return dto.AdminTagResponse{}, domain.ErrForbidden
	}

	name := normalizeTagName(req.Name)
	if name == "" {
		return dto.AdminTagResponse{}, response.NewAppError(response.CodeBadRequest, "name is required", nil)
	}

	slugSource := req.Slug
	if slugSource == "" {
		slugSource = name
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	tag := entity.NewTag(
		name,
		normalizeTagSlug(slugSource),
		normalizeTagDescription(req.Description),
		isActive,
	)

	if err := uc.tagRepo.Create(ctx, tag); err != nil {
		if errors.Is(err, domain.ErrTagAlreadyExists) {
			return dto.AdminTagResponse{}, domain.ErrTagAlreadyExists
		}
		return dto.AdminTagResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	return usecase.MapTagToAdminResponse(tag), nil
}

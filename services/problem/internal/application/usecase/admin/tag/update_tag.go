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
)

type updateTagUseCase struct {
	tagRepo outbound.TagRepository
}

func NewUpdateTagUseCase(tagRepo outbound.TagRepository) inbound.UpdateTagUseCase {
	return &updateTagUseCase{tagRepo: tagRepo}
}

func (uc *updateTagUseCase) Execute(ctx context.Context, claims auth.Claims, params dto.TagIDRequest, req dto.UpdateTagRequest) (dto.AdminTagResponse, error) {
	if !claims.Role.AtLeast(rbac.RoleModerator) {
		return dto.AdminTagResponse{}, domain.ErrForbidden
	}

	tag, err := uc.tagRepo.GetByID(ctx, params.ID)
	if err != nil {
		if errors.Is(err, domain.ErrTagNotFound) {
			return dto.AdminTagResponse{}, domain.ErrTagNotFound
		}
		return dto.AdminTagResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	if req.Name != nil {
		name := normalizeTagName(*req.Name)
		if name == "" {
			return dto.AdminTagResponse{}, response.NewAppError(response.CodeBadRequest, "name is required", nil)
		}
		tag.Name = name
	}

	if req.Slug != nil {
		tag.Slug = normalizeTagSlug(*req.Slug)
	}

	if req.Description != nil {
		tag.Description = normalizeTagDescription(*req.Description)
	}

	if req.IsActive != nil {
		if tag.IsActive && !*req.IsActive {
			if err := ensureTagCanBeDeactivated(ctx, uc.tagRepo, tag); err != nil {
				return dto.AdminTagResponse{}, err
			}
		}
		tag.IsActive = *req.IsActive
	}

	if err := uc.tagRepo.Update(ctx, tag); err != nil {
		if errors.Is(err, domain.ErrTagAlreadyExists) {
			return dto.AdminTagResponse{}, domain.ErrTagAlreadyExists
		}
		return dto.AdminTagResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	updatedTag, err := uc.tagRepo.GetByID(ctx, tag.ID)
	if err != nil {
		return dto.AdminTagResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	return usecase.MapTagToAdminResponse(updatedTag), nil
}

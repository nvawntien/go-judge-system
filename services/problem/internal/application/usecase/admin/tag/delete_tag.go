package tag

import (
	"context"
	"errors"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/problem/internal/application/dto"
	inbound "go-judge-system/services/problem/internal/application/port/inbound/admin"
	"go-judge-system/services/problem/internal/application/port/outbound"
	"go-judge-system/services/problem/internal/domain"
)

type deleteTagUseCase struct {
	tagRepo outbound.TagRepository
}

func NewDeleteTagUseCase(tagRepo outbound.TagRepository) inbound.DeleteTagUseCase {
	return &deleteTagUseCase{tagRepo: tagRepo}
}

func (uc *deleteTagUseCase) Execute(ctx context.Context, claims auth.Claims, params dto.TagIDRequest) error {
	if !claims.Role.AtLeast(rbac.RoleModerator) {
		return domain.ErrForbidden
	}

	tag, err := uc.tagRepo.GetByID(ctx, params.ID)
	if err != nil {
		if errors.Is(err, domain.ErrTagNotFound) {
			return domain.ErrTagNotFound
		}
		return domain.ErrInternalServer.Wrap(err)
	}

	if err := ensureTagCanBeDeactivated(ctx, uc.tagRepo, tag); err != nil {
		return err
	}

	if err := uc.tagRepo.Deactivate(ctx, tag.ID); err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	return nil
}

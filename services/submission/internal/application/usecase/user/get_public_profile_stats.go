package user

import (
	"context"
	"time"

	"go-judge-system/services/submission/internal/application/dto"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/user"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
)

type getPublicProfileStatsUseCase struct {
	publicUsers outbound.PublicUserResolver
	statsRepo   outbound.ProfileStatsRepository
	now         func() time.Time
}

func NewGetPublicProfileStatsUseCase(
	publicUsers outbound.PublicUserResolver,
	statsRepo outbound.ProfileStatsRepository,
) inbound.GetPublicProfileStatsUseCase {
	return &getPublicProfileStatsUseCase{publicUsers: publicUsers, statsRepo: statsRepo, now: time.Now}
}

func (uc *getPublicProfileStatsUseCase) Execute(
	ctx context.Context,
	req dto.GetPublicProfileStatsRequest,
) (dto.GetMyProfileStatsResponse, error) {
	user, err := uc.publicUsers.ResolvePublicUser(ctx, req.Username)
	if err != nil {
		return dto.GetMyProfileStatsResponse{}, err
	}
	if user.ID == "" {
		return dto.GetMyProfileStatsResponse{}, domain.ErrAuthServiceUnavailable
	}

	return getProfileStats(ctx, uc.statsRepo, uc.now, user.ID)
}

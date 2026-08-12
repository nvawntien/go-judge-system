package user

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/submission/internal/application/dto"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/user"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
)

const profileStatsActivityDays = 365

type getMyProfileStatsUseCase struct {
	statsRepo outbound.ProfileStatsRepository
	now       func() time.Time
}

func NewGetMyProfileStatsUseCase(statsRepo outbound.ProfileStatsRepository) inbound.GetMyProfileStatsUseCase {
	return &getMyProfileStatsUseCase{statsRepo: statsRepo, now: time.Now}
}

func (uc *getMyProfileStatsUseCase) Execute(ctx context.Context, claims auth.Claims) (dto.GetMyProfileStatsResponse, error) {
	if strings.TrimSpace(claims.UserID) == "" || claims.Role == "" {
		return dto.GetMyProfileStatsResponse{}, domain.ErrSubmissionUnauthenticated
	}
	switch claims.Role {
	case rbac.RoleUser, rbac.RoleContributor, rbac.RoleModerator, rbac.RoleAdmin:
	default:
		return dto.GetMyProfileStatsResponse{}, domain.ErrSubmissionForbidden
	}

	now := uc.now().UTC()
	activitySince := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).
		AddDate(0, 0, -(profileStatsActivityDays - 1))
	stats, err := uc.statsRepo.GetUserProfileStats(ctx, claims.UserID, activitySince)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return dto.GetMyProfileStatsResponse{}, err
		}
		return dto.GetMyProfileStatsResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	response := dto.GetMyProfileStatsResponse{
		TotalSubmissions:     stats.TotalSubmissions,
		AttemptedProblems:    stats.AttemptedProblems,
		AcceptedSubmissions:  stats.AcceptedSubmissions,
		SolvedProblems:       stats.SolvedProblems,
		AcceptanceRate:       acceptanceRate(stats.AcceptedSubmissions, stats.TotalSubmissions),
		VerdictDistribution:  make([]dto.ProfileStatsVerdictResponse, 0, len(stats.Verdicts)),
		LanguageDistribution: make([]dto.ProfileStatsLanguageResponse, 0, len(stats.Languages)),
		Activity:             make([]dto.ProfileStatsActivityResponse, 0, len(stats.Activity)),
	}
	for _, item := range stats.Verdicts {
		response.VerdictDistribution = append(response.VerdictDistribution, dto.ProfileStatsVerdictResponse{Verdict: item.Verdict, Count: item.Count})
	}
	for _, item := range stats.Languages {
		response.LanguageDistribution = append(response.LanguageDistribution, dto.ProfileStatsLanguageResponse{Language: item.Language, Count: item.Count})
	}
	for _, item := range stats.Activity {
		response.Activity = append(response.Activity, dto.ProfileStatsActivityResponse{Date: item.Date, Count: item.Count})
	}
	return response, nil
}

func acceptanceRate(accepted, total int64) float64 {
	if total == 0 {
		return 0
	}
	return math.Round((float64(accepted)/float64(total))*10000) / 100
}

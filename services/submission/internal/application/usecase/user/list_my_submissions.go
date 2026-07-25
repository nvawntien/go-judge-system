package user

import (
	"context"
	"errors"
	"strings"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/submission/internal/application/dto"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/user"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"
)

const (
	defaultSubmissionsPage  = 1
	defaultSubmissionsLimit = 20
	maxSubmissionsLimit     = 100
)

type listMySubmissionsUseCase struct {
	submissionRepo outbound.SubmissionRepository
}

func NewListMySubmissionsUseCase(
	submissionRepo outbound.SubmissionRepository,
) inbound.ListMySubmissionsUseCase {
	return &listMySubmissionsUseCase{submissionRepo: submissionRepo}
}

func (uc *listMySubmissionsUseCase) Execute(ctx context.Context, claims auth.Claims, req dto.ListMySubmissionsRequest) (dto.ListMySubmissionsResponse, error) {
	if strings.TrimSpace(claims.UserID) == "" || claims.Role == "" {
		return dto.ListMySubmissionsResponse{}, domain.ErrSubmissionUnauthenticated
	}
	switch claims.Role {
	case rbac.RoleUser, rbac.RoleContributor, rbac.RoleModerator, rbac.RoleAdmin:
	default:
		return dto.ListMySubmissionsResponse{}, domain.ErrSubmissionForbidden
	}

	page := defaultSubmissionsPage
	if req.Page != nil {
		if *req.Page <= 0 {
			return dto.ListMySubmissionsResponse{}, domain.ErrInvalidPage
		}
		page = *req.Page
	}

	limit := defaultSubmissionsLimit
	if req.Limit != nil {
		if *req.Limit <= 0 || *req.Limit > maxSubmissionsLimit {
			return dto.ListMySubmissionsResponse{}, domain.ErrInvalidLimit
		}
		limit = *req.Limit
	}

	status := strings.TrimSpace(req.Status)
	if status != "" {
		if _, ok := entity.ParseStatus(status); !ok {
			return dto.ListMySubmissionsResponse{}, domain.ErrInvalidSubmissionStatus
		}
	}

	language := strings.TrimSpace(req.Language)
	if language != "" {
		parsedLanguage, ok := entity.ParseLanguage(language)
		if !ok || !parsedLanguage.IsExecutable() {
			return dto.ListMySubmissionsResponse{}, domain.ErrInvalidLanguage
		}
	}

	if req.ProblemID != nil && *req.ProblemID <= 0 {
		return dto.ListMySubmissionsResponse{}, domain.ErrInvalidProblemID
	}

	result, err := uc.submissionRepo.ListByUser(ctx, outbound.ListSubmissionsFilter{
		UserID:    claims.UserID,
		Status:    status,
		Language:  language,
		ProblemID: req.ProblemID,
		Limit:     limit,
		Offset:    (page - 1) * limit,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return dto.ListMySubmissionsResponse{}, err
		}
		return dto.ListMySubmissionsResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	items := make([]dto.SubmissionListItem, 0, len(result.Items))
	for _, submission := range result.Items {
		if submission == nil {
			return dto.ListMySubmissionsResponse{}, domain.ErrInternalServer
		}
		items = append(items, dto.SubmissionListItem{
			ID:           submission.ID,
			ProblemID:    submission.ProblemID,
			ProblemTitle: submission.ProblemName,
			Language:     string(submission.Language),
			Status:       string(submission.Status),
			CreatedAt:    submission.CreatedAt,
		})
	}

	totalPages := 0
	if result.Total > 0 {
		totalPages = int((result.Total + int64(limit) - 1) / int64(limit))
	}

	return dto.ListMySubmissionsResponse{
		Items: items,
		Pagination: dto.PaginationResponse{
			Page:       page,
			Limit:      limit,
			Total:      result.Total,
			TotalPages: totalPages,
		},
	}, nil
}

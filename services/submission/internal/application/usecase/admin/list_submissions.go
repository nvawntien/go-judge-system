package admin

import (
	"context"
	"errors"
	"strings"

	"go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/submission/internal/application/dto"
	inbound "go-judge-system/services/submission/internal/application/port/inbound/admin"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"
)

const (
	defaultSubmissionsPage  = 1
	defaultSubmissionsLimit = 20
	maxSubmissionsLimit     = 100
)

type listAdminSubmissionsUseCase struct {
	submissionRepo outbound.SubmissionRepository
}

func NewListAdminSubmissionsUseCase(
	submissionRepo outbound.SubmissionRepository,
) inbound.ListAdminSubmissionsUseCase {
	return &listAdminSubmissionsUseCase{submissionRepo: submissionRepo}
}

func (uc *listAdminSubmissionsUseCase) Execute(
	ctx context.Context,
	claims auth.Claims,
	req dto.ListAdminSubmissionsRequest,
) (dto.ListAdminSubmissionsResponse, error) {
	if strings.TrimSpace(claims.UserID) == "" || claims.Role == "" {
		return dto.ListAdminSubmissionsResponse{}, domain.ErrSubmissionUnauthenticated
	}
	if !claims.Role.AtLeast(rbac.RoleModerator) {
		return dto.ListAdminSubmissionsResponse{}, domain.ErrSubmissionForbidden
	}

	page := defaultSubmissionsPage
	if req.Page != nil {
		if *req.Page <= 0 {
			return dto.ListAdminSubmissionsResponse{}, domain.ErrInvalidPage
		}
		page = *req.Page
	}

	limit := defaultSubmissionsLimit
	if req.Limit != nil {
		if *req.Limit <= 0 || *req.Limit > maxSubmissionsLimit {
			return dto.ListAdminSubmissionsResponse{}, domain.ErrInvalidLimit
		}
		limit = *req.Limit
	}

	status, err := parseOptionalStatus(req.Status)
	if err != nil {
		return dto.ListAdminSubmissionsResponse{}, err
	}
	language, err := parseOptionalLanguage(req.Language)
	if err != nil {
		return dto.ListAdminSubmissionsResponse{}, err
	}
	if req.ProblemID != nil && *req.ProblemID <= 0 {
		return dto.ListAdminSubmissionsResponse{}, domain.ErrInvalidProblemID
	}
	userID, err := parseOptionalUserID(req.UserID)
	if err != nil {
		return dto.ListAdminSubmissionsResponse{}, err
	}

	result, err := uc.submissionRepo.List(ctx, outbound.ListSubmissionsFilter{
		UserID:    userID,
		Status:    status,
		Language:  language,
		ProblemID: req.ProblemID,
		Limit:     limit,
		Offset:    (page - 1) * limit,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return dto.ListAdminSubmissionsResponse{}, err
		}
		return dto.ListAdminSubmissionsResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	items := make([]dto.AdminSubmissionListItem, 0, len(result.Items))
	for _, submission := range result.Items {
		if submission == nil {
			return dto.ListAdminSubmissionsResponse{}, domain.ErrInternalServer
		}
		items = append(items, dto.AdminSubmissionListItem{
			ID:           submission.ID,
			ProblemID:    submission.ProblemID,
			ProblemTitle: submission.ProblemName,
			UserID:       submission.UserID,
			Username:     submission.Username,
			Language:     string(submission.Language),
			Status:       string(submission.Status),
			CreatedAt:    submission.CreatedAt,
		})
	}

	totalPages := 0
	if result.Total > 0 {
		totalPages = int((result.Total + int64(limit) - 1) / int64(limit))
	}

	return dto.ListAdminSubmissionsResponse{
		Items: items,
		Pagination: dto.PaginationResponse{
			Page:       page,
			Limit:      limit,
			Total:      result.Total,
			TotalPages: totalPages,
		},
	}, nil
}

func parseOptionalStatus(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	status := strings.TrimSpace(*value)
	if status == "" {
		return nil, domain.ErrInvalidSubmissionStatus
	}
	if _, ok := entity.ParseStatus(status); !ok {
		return nil, domain.ErrInvalidSubmissionStatus
	}
	return &status, nil
}

func parseOptionalLanguage(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	language := strings.TrimSpace(*value)
	if language == "" {
		return nil, domain.ErrInvalidLanguage
	}
	if _, ok := entity.ParseLanguage(language); !ok {
		return nil, domain.ErrInvalidLanguage
	}
	return &language, nil
}

func parseOptionalUserID(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	userID := strings.TrimSpace(*value)
	if userID == "" {
		return nil, domain.ErrInvalidUserID
	}
	return &userID, nil
}

package user

import (
	"context"
	"strings"
	"unicode/utf8"

	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
)

const (
	defaultPublicUserSearchPage   = 1
	defaultPublicUserSearchLimit  = 10
	maxPublicUserSearchLimit      = 20
	maxPublicUserSearchQueryRunes = 100
)

type searchPublicUsersUseCase struct {
	userRepo outbound.UserRepository
}

func NewSearchPublicUsersUseCase(userRepo outbound.UserRepository) inbound.SearchPublicUsersUseCase {
	return &searchPublicUsersUseCase{userRepo: userRepo}
}

func (uc *searchPublicUsersUseCase) Execute(ctx context.Context, req dto.SearchPublicUsersRequest) (dto.SearchPublicUsersResponse, error) {
	page, limit, err := parsePublicUserSearchPagination(req.Page, req.Limit)
	if err != nil {
		return dto.SearchPublicUsersResponse{}, err
	}

	query := strings.TrimSpace(req.Query)
	if query == "" {
		return emptyPublicUserSearchResponse(page, limit), nil
	}
	if utf8.RuneCountInString(query) > maxPublicUserSearchQueryRunes {
		return dto.SearchPublicUsersResponse{}, response.NewAppError(response.CodeBadRequest, "search query must be at most 100 characters", nil)
	}

	result, err := uc.userRepo.SearchPublicUsers(ctx, outbound.SearchPublicUsersFilter{
		Query:  query,
		Limit:  limit,
		Offset: (page - 1) * limit,
	})
	if err != nil {
		return dto.SearchPublicUsersResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	items := make([]dto.PublicUserSearchItem, 0, len(result.Items))
	for _, user := range result.Items {
		if user == nil {
			return dto.SearchPublicUsersResponse{}, domain.ErrInternalServer
		}
		// The PostgreSQL query enforces this policy. Keep this guard so an
		// accidental future repository implementation cannot leak hidden users.
		if !user.IsActive || user.IsSuspended {
			continue
		}
		items = append(items, dto.PublicUserSearchItem{
			Username:  user.Username,
			FullName:  user.FullName,
			AvatarURL: user.AvatarURL,
			Rating:    user.Rating,
		})
	}

	totalPages := 0
	if result.Total > 0 {
		totalPages = int((result.Total + int64(limit) - 1) / int64(limit))
	}
	return dto.SearchPublicUsersResponse{
		Items: items,
		Pagination: dto.PublicUserSearchPagination{
			Page:       page,
			Limit:      limit,
			Total:      result.Total,
			TotalPages: totalPages,
		},
	}, nil
}

func parsePublicUserSearchPagination(pageValue, limitValue *int) (int, int, error) {
	page := defaultPublicUserSearchPage
	if pageValue != nil {
		if *pageValue <= 0 {
			return 0, 0, response.NewAppError(response.CodeBadRequest, "page must be greater than zero", nil)
		}
		page = *pageValue
	}

	limit := defaultPublicUserSearchLimit
	if limitValue != nil {
		if *limitValue <= 0 || *limitValue > maxPublicUserSearchLimit {
			return 0, 0, response.NewAppError(response.CodeBadRequest, "limit must be between 1 and 20", nil)
		}
		limit = *limitValue
	}
	return page, limit, nil
}

func emptyPublicUserSearchResponse(page, limit int) dto.SearchPublicUsersResponse {
	return dto.SearchPublicUsersResponse{
		Items: []dto.PublicUserSearchItem{},
		Pagination: dto.PublicUserSearchPagination{
			Page:  page,
			Limit: limit,
		},
	}
}

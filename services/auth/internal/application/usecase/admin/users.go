package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	pkgauth "go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/entity"

	"github.com/google/uuid"
)

const (
	defaultAdminUsersPage  = 1
	defaultAdminUsersLimit = 20
	maxAdminUsersLimit     = 100
)

type adminUsersUseCase struct {
	userRepo     outbound.UserRepository
	logoutAllIAT pkgauth.LogoutAllIATStore
}

func NewAdminUsersUseCase(userRepo outbound.UserRepository, logoutAllIAT pkgauth.LogoutAllIATStore) inbound.AdminUsersUseCase {
	return &adminUsersUseCase{userRepo: userRepo, logoutAllIAT: logoutAllIAT}
}

func (uc *adminUsersUseCase) List(ctx context.Context, claims pkgauth.Claims, req dto.ListAdminUsersRequest) (dto.ListAdminUsersResponse, error) {
	if !claims.Role.AtLeast(rbac.RoleAdmin) {
		return dto.ListAdminUsersResponse{}, domain.ErrForbidden
	}

	page, limit, err := parseAdminUsersPagination(req.Page, req.Limit)
	if err != nil {
		return dto.ListAdminUsersResponse{}, err
	}
	role, err := parseAdminUserRole(req.Role)
	if err != nil {
		return dto.ListAdminUsersResponse{}, err
	}
	isActive, isSuspended, err := parseAdminUserStatus(req.Status)
	if err != nil {
		return dto.ListAdminUsersResponse{}, err
	}

	result, err := uc.userRepo.ListUsers(ctx, outbound.ListUsersFilter{
		Search:      strings.TrimSpace(req.Search),
		Role:        role,
		IsActive:    isActive,
		IsSuspended: isSuspended,
		Limit:       limit,
		Offset:      (page - 1) * limit,
	})
	if err != nil {
		return dto.ListAdminUsersResponse{}, domain.ErrInternalServer.Wrap(err)
	}

	items := make([]dto.AdminUserResponse, 0, len(result.Items))
	for _, user := range result.Items {
		if user == nil {
			return dto.ListAdminUsersResponse{}, domain.ErrInternalServer
		}
		items = append(items, toAdminUserResponse(user))
	}

	totalPages := 0
	if result.Total > 0 {
		totalPages = int((result.Total + int64(limit) - 1) / int64(limit))
	}
	return dto.ListAdminUsersResponse{
		Items: items,
		Pagination: dto.AdminUsersPaginationResponse{
			Page:       page,
			Limit:      limit,
			Total:      result.Total,
			TotalPages: totalPages,
		},
	}, nil
}

func (uc *adminUsersUseCase) Get(ctx context.Context, claims pkgauth.Claims, params dto.UserIDRequest) (dto.AdminUserResponse, error) {
	if !claims.Role.AtLeast(rbac.RoleAdmin) {
		return dto.AdminUserResponse{}, domain.ErrForbidden
	}
	if err := validateAdminUserID(params.UserID); err != nil {
		return dto.AdminUserResponse{}, err
	}

	user, err := uc.userRepo.GetUserById(ctx, params.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return dto.AdminUserResponse{}, domain.ErrUserNotFound
		}
		return dto.AdminUserResponse{}, domain.ErrInternalServer.Wrap(err)
	}
	if user == nil {
		return dto.AdminUserResponse{}, domain.ErrInternalServer
	}
	return toAdminUserResponse(user), nil
}

func (uc *adminUsersUseCase) SetSuspension(
	ctx context.Context,
	claims pkgauth.Claims,
	params dto.UserIDRequest,
	req dto.SetUserSuspensionRequest,
) (dto.AdminUserResponse, error) {
	if !claims.Role.AtLeast(rbac.RoleAdmin) {
		return dto.AdminUserResponse{}, domain.ErrForbidden
	}
	if err := validateAdminUserID(params.UserID); err != nil {
		return dto.AdminUserResponse{}, err
	}
	if req.Suspended == nil {
		return dto.AdminUserResponse{}, response.NewAppError(response.CodeBadRequest, "suspended is required", nil)
	}

	user, err := uc.userRepo.GetUserById(ctx, params.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return dto.AdminUserResponse{}, domain.ErrUserNotFound
		}
		return dto.AdminUserResponse{}, domain.ErrInternalServer.Wrap(err)
	}
	if user == nil {
		return dto.AdminUserResponse{}, domain.ErrInternalServer
	}

	if *req.Suspended {
		// Revoke first: a database write failure must not leave an existing session usable.
		if err := uc.logoutAllIAT.SetLogoutAllIAT(ctx, user.ID, time.Now().Unix()); err != nil {
			return dto.AdminUserResponse{}, domain.ErrInternalServer.Wrap(err)
		}
		user.Suspend()
	} else {
		// Keep the cutoff so unsuspension never revives tokens issued before suspension.
		user.Unsuspend()
	}

	if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
		return dto.AdminUserResponse{}, domain.ErrInternalServer.Wrap(err)
	}
	return toAdminUserResponse(user), nil
}

func parseAdminUsersPagination(pageValue, limitValue *int) (int, int, error) {
	page := defaultAdminUsersPage
	if pageValue != nil {
		if *pageValue <= 0 {
			return 0, 0, response.NewAppError(response.CodeBadRequest, "page must be greater than zero", nil)
		}
		page = *pageValue
	}
	limit := defaultAdminUsersLimit
	if limitValue != nil {
		if *limitValue <= 0 || *limitValue > maxAdminUsersLimit {
			return 0, 0, response.NewAppError(response.CodeBadRequest, "limit must be between 1 and 100", nil)
		}
		limit = *limitValue
	}
	return page, limit, nil
}

func parseAdminUserRole(value string) (*rbac.Role, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	role, ok := rbac.ParseRole(value)
	if !ok {
		return nil, response.NewAppError(response.CodeBadRequest, "invalid role filter", nil)
	}
	return &role, nil
}

func parseAdminUserStatus(value string) (*bool, *bool, error) {
	switch strings.TrimSpace(value) {
	case "":
		return nil, nil, nil
	case "active":
		active, suspended := true, false
		return &active, &suspended, nil
	case "unverified":
		active, suspended := false, false
		return &active, &suspended, nil
	case "suspended":
		suspended := true
		return nil, &suspended, nil
	default:
		return nil, nil, response.NewAppError(response.CodeBadRequest, "invalid status filter", nil)
	}
}

func validateAdminUserID(userID string) error {
	if _, err := uuid.Parse(strings.TrimSpace(userID)); err != nil {
		return response.NewAppError(response.CodeParamInvalid, "invalid user id", nil)
	}
	return nil
}

func toAdminUserResponse(user *entity.User) dto.AdminUserResponse {
	return dto.AdminUserResponse{
		ID:          user.ID,
		FullName:    user.FullName,
		Username:    user.Username,
		Email:       user.Email,
		Role:        user.Role,
		Rating:      user.Rating,
		IsActive:    user.IsActive,
		IsSuspended: user.IsSuspended,
		AvatarURL:   user.AvatarURL,
		Bio:         user.Bio,
		Country:     user.Country,
		School:      user.School,
		Company:     user.Company,
		GithubURL:   user.GithubURL,
		WebsiteURL:  user.WebsiteURL,
		LinkedinURL: user.LinkedinURL,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   user.UpdatedAt.Format(time.RFC3339),
	}
}

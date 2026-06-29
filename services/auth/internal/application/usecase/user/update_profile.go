package user

import (
	"context"
	"errors"
	"net/url"
	"strings"

	pkgAuth "go-judge-system/pkg/auth"
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
)

type updateProfileUseCase struct {
	userRepo outbound.UserRepository
}

func NewUpdateProfileUseCase(userRepo outbound.UserRepository) inbound.UpdateProfileUseCase {
	return &updateProfileUseCase{userRepo: userRepo}
}

func (uc *updateProfileUseCase) Execute(ctx context.Context, claims pkgAuth.Claims, req dto.UpdateProfileRequest) (*dto.GetMeResponse, error) {
	user, err := uc.userRepo.GetUserById(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, domain.ErrInternalServer.Wrap(err)
	}

	fullName, err := validateOptionalText(req.FullName, "full_name", 100)
	if err != nil {
		return nil, err
	}
	avatarURL, err := validateOptionalURL(req.AvatarURL, "avatar_url", 500)
	if err != nil {
		return nil, err
	}
	bio, err := validateOptionalText(req.Bio, "bio", 500)
	if err != nil {
		return nil, err
	}
	country, err := validateOptionalText(req.Country, "country", 100)
	if err != nil {
		return nil, err
	}
	school, err := validateOptionalText(req.School, "school", 255)
	if err != nil {
		return nil, err
	}
	company, err := validateOptionalText(req.Company, "company", 255)
	if err != nil {
		return nil, err
	}
	githubURL, err := validateOptionalURL(req.GithubURL, "github_url", 500)
	if err != nil {
		return nil, err
	}
	websiteURL, err := validateOptionalURL(req.WebsiteURL, "website_url", 500)
	if err != nil {
		return nil, err
	}
	linkedinURL, err := validateOptionalURL(req.LinkedinURL, "linkedin_url", 500)
	if err != nil {
		return nil, err
	}

	user.UpdateProfile(fullName, avatarURL, bio, country, school, company, githubURL, websiteURL, linkedinURL)

	if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
		return nil, domain.ErrInternalServer.Wrap(err)
	}

	return toGetMeResponse(user), nil
}

func validateOptionalText(raw *string, field string, maxLen int) (*string, error) {
	if raw == nil {
		return nil, nil
	}

	trimmed := strings.TrimSpace(*raw)
	if len(trimmed) > maxLen {
		return nil, response.NewAppError(response.CodeBadRequest, field+" exceeds maximum length", nil)
	}
	if trimmed == "" {
		return stringPtr(""), nil
	}

	return &trimmed, nil
}

func validateOptionalURL(raw *string, field string, maxLen int) (*string, error) {
	value, err := validateOptionalText(raw, field, maxLen)
	if err != nil || value == nil {
		return value, err
	}
	if *value == "" {
		return value, nil
	}

	parsed, err := url.ParseRequestURI(*value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, response.NewAppError(response.CodeBadRequest, field+" must be a valid URL", nil)
	}

	return value, nil
}

func stringPtr(value string) *string {
	return &value
}

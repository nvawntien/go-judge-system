package user

import (
    "context"
    "errors"
    "net/url"
    "strings"
    "unicode/utf8"

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

    fullName, bio, country, school, company, githubURL, websiteURL, linkedinURL, err := validateProfileFields(req)
    if err != nil {
        return nil, err
    }

    user.UpdateProfile(fullName, bio, country, school, company, githubURL, websiteURL, linkedinURL)

    if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
        return nil, domain.ErrInternalServer.Wrap(err)
    }

    return toGetMeResponse(user), nil
}

func validateProfileFields(req dto.UpdateProfileRequest) (
    fullName, bio, country, school, company, githubURL, websiteURL, linkedinURL *string, err error,
) {
    if fullName, err = validateOptionalText(req.FullName, "full_name", 100); err != nil { return }
    if bio, err = validateOptionalText(req.Bio, "bio", 500); err != nil { return }
    if country, err = validateOptionalText(req.Country, "country", 100); err != nil { return }
    if school, err = validateOptionalText(req.School, "school", 255); err != nil { return }
    if company, err = validateOptionalText(req.Company, "company", 255); err != nil { return }
    if githubURL, err = validateOptionalURL(req.GithubURL, "github_url", 500); err != nil { return }
    if websiteURL, err = validateOptionalURL(req.WebsiteURL, "website_url", 500); err != nil { return }
    if linkedinURL, err = validateOptionalURL(req.LinkedinURL, "linkedin_url", 500); err != nil { return }
    return
}

func validateOptionalText(raw *string, field string, maxLen int) (*string, error) {
    if raw == nil {
        return nil, nil
    }

    trimmed := strings.TrimSpace(*raw)
    
    if utf8.RuneCountInString(trimmed) > maxLen {
        return nil, response.NewAppError(response.CodeBadRequest, field+" exceeds maximum length", nil)
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
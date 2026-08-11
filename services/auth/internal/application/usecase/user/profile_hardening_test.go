package user

import (
	"context"
	"errors"
	"testing"
	"time"

	pkgauth "go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/entity"
)

func TestGetProfileHidesNonPublicAccountStates(t *testing.T) {
	tests := []struct {
		name    string
		active  bool
		suspend bool
		wantErr error
	}{
		{name: "active", active: true},
		{name: "unverified", wantErr: domain.ErrUserNotFound},
		{name: "suspended", active: true, suspend: true, wantErr: domain.ErrUserNotFound},
		{name: "inactive and suspended", suspend: true, wantErr: domain.ErrUserNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := profileTestUser()
			user.IsActive = tt.active
			user.IsSuspended = tt.suspend
			repo := &profileHardeningRepository{user: user}
			got, err := NewGetProfileUseCase(repo).Execute(context.Background(), dto.GetProfileRequest{Username: user.Username})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && (got.Username != user.Username || got.AvatarURL == nil) {
				t.Fatalf("public response = %+v", got)
			}
		})
	}

	_, err := NewGetProfileUseCase(&profileHardeningRepository{getErr: domain.ErrUserNotFound}).Execute(
		context.Background(), dto.GetProfileRequest{Username: "missing"},
	)
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("unknown profile error = %v", err)
	}
}

func TestUpdateProfileUsesScopedWriteAndOnlyAllowsHTTPURLs(t *testing.T) {
	user := profileTestUser()
	repo := &profileHardeningRepository{user: user}
	fullName := "Updated Name"
	github := "https://github.com/example"
	got, err := NewUpdateProfileUseCase(repo).Execute(
		context.Background(),
		pkgauth.Claims{UserID: user.ID},
		dto.UpdateProfileRequest{FullName: &fullName, GithubURL: &github},
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repo.fullUpdateCalled || repo.profileUpdate.FullName == nil || *repo.profileUpdate.FullName != fullName ||
		repo.profileUpdate.GithubURL == nil || *repo.profileUpdate.GithubURL != github {
		t.Fatalf("profile update = %+v full=%t", repo.profileUpdate, repo.fullUpdateCalled)
	}
	if got.FullName != fullName || user.AvatarURL == nil || *user.AvatarURL != "https://avatar.example/old.png" {
		t.Fatalf("response/user = %+v/%+v", got, user)
	}

	for _, raw := range []string{
		"javascript://example.com", "ftp://example.com", "file://example.com", "data://example.com", "https://", "not a URL",
	} {
		raw := raw
		t.Run(raw, func(t *testing.T) {
			_, _, _, _, _, _, _, _, err := validateProfileFields(dto.UpdateProfileRequest{WebsiteURL: &raw})
			if err == nil {
				t.Fatalf("validateProfileFields(%q) error = nil", raw)
			}
		})
	}
	for _, raw := range []string{"http://example.com", "https://github.com/user", ""} {
		raw := raw
		t.Run("valid "+raw, func(t *testing.T) {
			_, _, _, _, _, _, _, _, err := validateProfileFields(dto.UpdateProfileRequest{WebsiteURL: &raw})
			if err != nil {
				t.Fatalf("validateProfileFields(%q) error = %v", raw, err)
			}
		})
	}
}

func TestScopedProfileAndAvatarUpdatesPreserveUnrelatedFields(t *testing.T) {
	user := profileTestUser()
	repo := &profileHardeningRepository{user: user}
	newBio := "New profile data"
	avatarURL := "https://avatar.example/new.png"
	avatarKey := "users/user-1/new.png"

	if err := repo.UpdateProfile(context.Background(), user.ID, outbound.ProfileUpdates{Bio: &newBio, UpdatedAt: time.Now()}); err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if err := repo.UpdateAvatar(context.Background(), user.ID, avatarURL, avatarKey, time.Now()); err != nil {
		t.Fatalf("UpdateAvatar() error = %v", err)
	}
	if user.Bio == nil || *user.Bio != newBio || user.AvatarURL == nil || *user.AvatarURL != avatarURL ||
		user.AvatarObjectKey == nil || *user.AvatarObjectKey != avatarKey {
		t.Fatalf("profile then avatar final user = %+v", user)
	}

	otherBio := "Newer profile data"
	if err := repo.UpdateAvatar(context.Background(), user.ID, "https://avatar.example/newer.png", "users/user-1/newer.png", time.Now()); err != nil {
		t.Fatalf("UpdateAvatar() error = %v", err)
	}
	if err := repo.UpdateProfile(context.Background(), user.ID, outbound.ProfileUpdates{Bio: &otherBio, UpdatedAt: time.Now()}); err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if user.Bio == nil || *user.Bio != otherBio || user.AvatarURL == nil || *user.AvatarURL != "https://avatar.example/newer.png" {
		t.Fatalf("avatar then profile final user = %+v", user)
	}
}

type profileHardeningRepository struct {
	user             *entity.User
	getErr           error
	profileUpdate    outbound.ProfileUpdates
	fullUpdateCalled bool
}

func (r *profileHardeningRepository) CreateUser(context.Context, *entity.User) error { return nil }
func (r *profileHardeningRepository) GetUserByEmail(context.Context, string) (*entity.User, error) {
	return r.user, r.getErr
}
func (r *profileHardeningRepository) GetUserByUsername(context.Context, string) (*entity.User, error) {
	return r.user, r.getErr
}
func (r *profileHardeningRepository) GetUserById(context.Context, string) (*entity.User, error) {
	return r.user, r.getErr
}
func (r *profileHardeningRepository) ListUsers(context.Context, outbound.ListUsersFilter) (outbound.ListUsersResult, error) {
	return outbound.ListUsersResult{}, nil
}
func (r *profileHardeningRepository) UpdateUser(context.Context, *entity.User) error {
	r.fullUpdateCalled = true
	return nil
}
func (r *profileHardeningRepository) UpdatePassword(context.Context, string, string, time.Time) error {
	return nil
}
func (r *profileHardeningRepository) UpdateProfile(_ context.Context, _ string, updates outbound.ProfileUpdates) error {
	r.profileUpdate = updates
	if updates.FullName != nil {
		r.user.FullName = *updates.FullName
	}
	if updates.Bio != nil {
		r.user.Bio = updates.Bio
	}
	if updates.Country != nil {
		r.user.Country = updates.Country
	}
	if updates.School != nil {
		r.user.School = updates.School
	}
	if updates.Company != nil {
		r.user.Company = updates.Company
	}
	if updates.GithubURL != nil {
		r.user.GithubURL = updates.GithubURL
	}
	if updates.WebsiteURL != nil {
		r.user.WebsiteURL = updates.WebsiteURL
	}
	if updates.LinkedinURL != nil {
		r.user.LinkedinURL = updates.LinkedinURL
	}
	return nil
}
func (r *profileHardeningRepository) UpdateAvatar(_ context.Context, _ string, avatarURL string, avatarObjectKey string, _ time.Time) error {
	r.user.AvatarURL = &avatarURL
	r.user.AvatarObjectKey = &avatarObjectKey
	return nil
}
func (r *profileHardeningRepository) DeleteUser(context.Context, string) error { return nil }

func profileTestUser() *entity.User {
	avatarURL := "https://avatar.example/old.png"
	avatarKey := "users/user-1/old.png"
	bio := "Old profile data"
	return &entity.User{
		ID:              "user-1",
		FullName:        "Profile User",
		Username:        "profile-user",
		Role:            rbac.RoleUser,
		IsActive:        true,
		AvatarURL:       &avatarURL,
		AvatarObjectKey: &avatarKey,
		Bio:             &bio,
		CreatedAt:       time.Now().UTC(),
	}
}

var _ outbound.UserRepository = (*profileHardeningRepository)(nil)

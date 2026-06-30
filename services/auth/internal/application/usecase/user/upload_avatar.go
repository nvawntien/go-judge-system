package user

import (
	"context"
	"errors"
	"io"
	"net/http"

	pkgAuth "go-judge-system/pkg/auth"
	"go-judge-system/pkg/response"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
)

const maxAvatarSize = 2 * 1024 * 1024

type uploadAvatarUseCase struct {
	userRepo      outbound.UserRepository
	avatarStorage outbound.AvatarStorage
}

func NewUploadAvatarUseCase(
	userRepo outbound.UserRepository,
	avatarStorage outbound.AvatarStorage,
) inbound.UploadAvatarUseCase {
	return &uploadAvatarUseCase{
		userRepo:      userRepo,
		avatarStorage: avatarStorage,
	}
}

func (uc *uploadAvatarUseCase) Execute(ctx context.Context, claims pkgAuth.Claims, req dto.UploadAvatarRequest) (*dto.UploadAvatarResponse, error) {
	if req.Avatar == nil {
		return nil, response.NewAppError(response.CodeBadRequest, "avatar file is required", nil)
	}
	if req.Avatar.Size <= 0 {
		return nil, response.NewAppError(response.CodeBadRequest, "avatar file must not be empty", nil)
	}
	if req.Avatar.Size > maxAvatarSize {
		return nil, response.NewAppError(response.CodeBadRequest, "avatar file size must not exceed 2MB", nil)
	}

	file, err := req.Avatar.Open()
	if err != nil {
		return nil, domain.ErrInternalServer.Wrap(err)
	}
	defer file.Close()

	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, domain.ErrInternalServer.Wrap(err)
	}

	contentType := http.DetectContentType(header[:n])
	if !isAllowedAvatarContentType(contentType) {
		return nil, response.NewAppError(response.CodeBadRequest, "avatar must be a jpeg, png, or webp image", nil)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, domain.ErrInternalServer.Wrap(err)
	}

	user, err := uc.userRepo.GetUserById(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, domain.ErrInternalServer.Wrap(err)
	}

	var oldAvatarObjectKey *string
	if user.AvatarObjectKey != nil && *user.AvatarObjectKey != "" {
		existing := *user.AvatarObjectKey
		oldAvatarObjectKey = &existing
	}

	avatarURL, objectKey, err := uc.avatarStorage.UploadAvatar(ctx, user.ID, contentType, file, req.Avatar.Size)
	if err != nil {
		return nil, domain.ErrInternalServer.Wrap(err)
	}

	user.UploadAvatar(avatarURL, objectKey)

	if err := uc.userRepo.UpdateUser(ctx, user); err != nil {
		_ = uc.avatarStorage.DeleteAvatar(ctx, objectKey)
		return nil, domain.ErrInternalServer.Wrap(err)
	}

	if oldAvatarObjectKey != nil {
		_ = uc.avatarStorage.DeleteAvatar(ctx, *oldAvatarObjectKey)
	}

	return &dto.UploadAvatarResponse{AvatarURL: avatarURL}, nil
}

func isAllowedAvatarContentType(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

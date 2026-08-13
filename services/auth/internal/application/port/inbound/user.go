package inbound

import (
	"context"
	pkgAuth "go-judge-system/pkg/auth"
	"go-judge-system/services/auth/internal/application/dto"
)

type GetMeUseCase interface {
	Execute(ctx context.Context, claims pkgAuth.Claims) (*dto.GetMeResponse, error)
}

type GetProfileUseCase interface {
	Execute(ctx context.Context, req dto.GetProfileRequest) (dto.GetProfileResponse, error)
}

type ResolvePublicUserUseCase interface {
	Execute(ctx context.Context, req dto.ResolvePublicUserRequest) (dto.ResolvePublicUserResponse, error)
}

type SearchPublicUsersUseCase interface {
	Execute(ctx context.Context, req dto.SearchPublicUsersRequest) (dto.SearchPublicUsersResponse, error)
}

type UpdateProfileUseCase interface {
	Execute(ctx context.Context, claims pkgAuth.Claims, req dto.UpdateProfileRequest) (*dto.GetMeResponse, error)
}

type UploadAvatarUseCase interface {
	Execute(ctx context.Context, claims pkgAuth.Claims, req dto.UploadAvatarRequest) (*dto.UploadAvatarResponse, error)
}

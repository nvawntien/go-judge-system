package auth

import (
	"context"
	"errors"
	pkgauth "go-judge-system/pkg/auth"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/domain"
	"time"
)

type logoutAllUseCase struct {
	logoutAllStore pkgauth.LogoutAllIATStore
}

func NewLogoutAllUseCase(logoutAllStore pkgauth.LogoutAllIATStore) inbound.LogoutAllUseCase {
	return &logoutAllUseCase{
		logoutAllStore: logoutAllStore,
	}
}

func (uc *logoutAllUseCase) Execute(ctx context.Context, userID string) error {
	if userID == "" {
		return domain.ErrForbidden.Wrap(errors.New("user ID is required"))
	}
	now := time.Now().Unix()

	if err := uc.logoutAllStore.SetLogoutAllIAT(ctx, userID, now); err != nil {
		return domain.ErrInternalServer.Wrap(err)
	}

	return nil
}

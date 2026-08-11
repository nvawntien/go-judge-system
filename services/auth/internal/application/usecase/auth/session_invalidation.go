package auth

import (
	"context"
	"time"

	pkgauth "go-judge-system/pkg/auth"
)

func waitForTokenIssuedAfterInvalidation(ctx context.Context, store pkgauth.LogoutAllIATStore, userID string) (bool, error) {
	cutoff, err := store.GetLogoutAllIAT(ctx, userID)
	if err != nil || cutoff == 0 || time.Now().Unix() > cutoff {
		return false, err
	}

	timer := time.NewTimer(time.Until(time.Unix(cutoff+1, 0)))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-timer.C:
		return true, nil
	}
}

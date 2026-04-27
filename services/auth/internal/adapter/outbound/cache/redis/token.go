package redis

import (
	"context"
	"time"

	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"

	"github.com/redis/go-redis/v9"
)

type tokenRepository struct {
	rdb *redis.Client
}

func verificationTokenKey(hashedToken string) string {
	return "token:" + hashedToken
}

func latestVerificationTokenKey(identifier string) string {
	return "token:latest:" + identifier
}

func resendCooldownKey(identifier string) string {
	return "token:cooldown:resend:" + identifier
}

func NewTokenRepository(rdb *redis.Client) outbound.TokenRepository {
	return &tokenRepository{rdb: rdb}
}

func (r *tokenRepository) Save(ctx context.Context, hashedToken string, identifier string, ttl time.Duration) error {
	tokenKey := verificationTokenKey(hashedToken)
	latestKey := latestVerificationTokenKey(identifier)

	_, err := r.rdb.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, tokenKey, identifier, ttl)
		pipe.Set(ctx, latestKey, hashedToken, ttl)
		return nil
	})

	return err
}

func (r *tokenRepository) FindByToken(ctx context.Context, hashedToken string) (string, error) {
	tokenKey := verificationTokenKey(hashedToken)

	identifier, err := r.rdb.Get(ctx, tokenKey).Result()
	if err == redis.Nil {
		return "", domain.ErrInvalidOrExpiredToken
	}
	if err != nil {
		return "", err
	}

	latestKey := latestVerificationTokenKey(identifier)
	latestHashedToken, err := r.rdb.Get(ctx, latestKey).Result()
	if err == redis.Nil {
		return "", domain.ErrInvalidOrExpiredToken
	}
	if err != nil {
		return "", err
	}

	if latestHashedToken != hashedToken {
		return "", domain.ErrInvalidOrExpiredToken
	}

	return identifier, nil
}

func (r *tokenRepository) Delete(ctx context.Context, hashedToken string) error {
    tokenKey := verificationTokenKey(hashedToken)

    identifier, err := r.rdb.Get(ctx, tokenKey).Result()
    if err == redis.Nil {
        return nil
    }
    if err != nil {
        return err
    }

    latestKey := latestVerificationTokenKey(identifier)

    latestHashedToken, err := r.rdb.Get(ctx, latestKey).Result()
    if err != nil && err != redis.Nil {
        return err
    }

    _, err = r.rdb.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
        pipe.Del(ctx, tokenKey)
        
        if latestHashedToken == hashedToken {
            pipe.Del(ctx, latestKey)
        }
        return nil
    })

    return err
}

func (r *tokenRepository) TryAcquireResendCooldown(ctx context.Context, identifier string, ttl time.Duration) (bool, error) {
	return r.rdb.SetNX(ctx, resendCooldownKey(identifier), "1", ttl).Result()
}

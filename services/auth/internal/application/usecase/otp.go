package usecase

import (
	"context"
	"crypto/rand"
	"fmt"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"time"

	"go.uber.org/zap"
)

const (
	PolicyRateLimitMax  = 1
	PolicyRateLimitTTL  = 30 * time.Second
	PolicyMaxUnverified = 5
	PolicyBlockDuration = 5 * time.Minute
	PolicyCountTTL      = 24 * time.Hour
	PolicyOTPExpiry     = 5 * time.Minute
)

type otpUseCase struct {
	cacheRepo outbound.CacheRepository
	mail      outbound.MailProvider
	logger    *zap.Logger
}

func NewOTPUseCase(cacheRepo outbound.CacheRepository, mail outbound.MailProvider, logger *zap.Logger) inbound.OTPUseCase {
	return &otpUseCase{
		cacheRepo: cacheRepo,
		mail:      mail,
		logger:    logger,
	}
}

func getBlockKey(id string) string { return fmt.Sprintf("otp:block:%s", id) }

func getRateKey(id string) string { return fmt.Sprintf("otp:rate:%s", id) }

func getCountKey(id string) string { return fmt.Sprintf("otp:count:%s", id) }

func getOTPKey(id string) string { return fmt.Sprintf("otp:val:%s", id) }

func (uc *otpUseCase) RequestOTP(ctx context.Context, identifier string) error {
	isBlocked, err := uc.cacheRepo.Exists(ctx, getBlockKey(identifier))
	if err != nil {
		return domain.ErrInternalServer
	}
	if isBlocked {
		return domain.ErrUserBlocked
	}

	rateReqs, err := uc.cacheRepo.IncrWithExpire(ctx, getRateKey(identifier), PolicyRateLimitTTL)
	if err != nil {
		return domain.ErrInternalServer
	}
	if rateReqs > int64(PolicyRateLimitMax) {
		return domain.ErrRateLimitExceeded
	}

	countReqs, err := uc.cacheRepo.IncrWithExpire(ctx, getCountKey(identifier), PolicyCountTTL)
	if err != nil {
		return domain.ErrInternalServer
	}
	if countReqs >= int64(PolicyMaxUnverified) {
		// Block user
		uc.cacheRepo.Set(ctx, getBlockKey(identifier), "1", PolicyBlockDuration)
		uc.cacheRepo.Delete(ctx, getCountKey(identifier))
		uc.logger.Warn("User blocked due to too many OTP requests", zap.String("identifier", identifier))
		return domain.ErrUserBlocked
	}

	otp := generateOTP()
	if err := uc.cacheRepo.Set(ctx, getOTPKey(identifier), otp, PolicyOTPExpiry); err != nil {
		return domain.ErrInternalServer
	}

	if err := uc.mail.SendOTP(ctx, identifier, otp); err != nil {
		return domain.ErrInternalServer
	}

	uc.logger.Info("OTP requested successfully", zap.String("identifier", identifier))
	return nil
}

func generateOTP() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%06d", int(b[0])<<16|int(b[1])<<8|int(b[2])%1000000)
}

package usecase

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"go-judge-system/services/auth/internal/application/port/inbound"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"math/big"
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
	PolicyVerifyFailMax = 5
	PolicyVerifyFailTTL = 10 * time.Minute
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

func getBlockKey(purpose, id string) string { return fmt.Sprintf("otp:block:%s:%s", purpose, id) }
func getRateKey(purpose, id string) string { return fmt.Sprintf("otp:rate:%s:%s", purpose, id) }
func getCountKey(purpose, id string) string { return fmt.Sprintf("otp:count:%s:%s", purpose, id) }
func getOTPKey(purpose, id string) string { return fmt.Sprintf("otp:val:%s:%s", purpose, id) }
func getVerifyFailKey(purpose, id string) string { return fmt.Sprintf("otp:verify_fail:%s:%s", purpose, id) }

func (uc *otpUseCase) RequestOTP(ctx context.Context, purpose, identifier string) error {
	isBlocked, err := uc.cacheRepo.Exists(ctx, getBlockKey(purpose, identifier))
	if err != nil {
		uc.logger.Error("failed to check block status in cache", zap.String("identifier", identifier), zap.Error(err))
		return domain.ErrInternalServer
	}
	if isBlocked {
		return domain.ErrUserBlocked
	}

	rateReqs, err := uc.cacheRepo.IncrWithExpire(ctx, getRateKey(purpose, identifier), PolicyRateLimitTTL)
	if err != nil {
		uc.logger.Error("failed to increment rate limit counter", zap.String("identifier", identifier), zap.Error(err))
		return domain.ErrInternalServer
	}
	if rateReqs > int64(PolicyRateLimitMax) {
		return domain.ErrRateLimitExceeded
	}

	countReqs, err := uc.cacheRepo.IncrWithExpire(ctx, getCountKey(purpose, identifier), PolicyCountTTL)
	if err != nil {
		uc.logger.Error("failed to increment count counter", zap.String("identifier", identifier), zap.Error(err))
		return domain.ErrInternalServer
	}
	if countReqs >= int64(PolicyMaxUnverified) {
		// Block user
		uc.cacheRepo.Set(ctx, getBlockKey(purpose, identifier), "1", PolicyBlockDuration)
		uc.cacheRepo.Delete(ctx, getCountKey(purpose, identifier))
		uc.logger.Warn("User blocked due to too many OTP requests", zap.String("identifier", identifier))
		return domain.ErrUserBlocked
	}

	otp, err := generateOTP()
	if err != nil {
		uc.logger.Error("failed to generate OTP", zap.String("identifier", identifier), zap.Error(err))
		return domain.ErrInternalServer
	}
	if err := uc.cacheRepo.Set(ctx, getOTPKey(purpose, identifier), otp, PolicyOTPExpiry); err != nil {
		uc.logger.Error("failed to store OTP in cache", zap.String("identifier", identifier), zap.Error(err))
		return domain.ErrInternalServer
	}

	if err := uc.mail.SendOTP(ctx, identifier, otp); err != nil {
		uc.logger.Error("failed to send OTP email", zap.String("identifier", identifier), zap.Error(err))
		return domain.ErrInternalServer
	}

	uc.logger.Info("OTP requested successfully", zap.String("identifier", identifier))
	return nil
}

func (uc *otpUseCase) VerifyOTP(ctx context.Context, purpose, identifier, otp string) error {
	blocked, err := uc.cacheRepo.Exists(ctx, getBlockKey(purpose, identifier))
	if err != nil {
		uc.logger.Error("failed to check block status in cache", zap.String("identifier", identifier), zap.Error(err))
		return domain.ErrInternalServer
	}
	if blocked {
		return domain.ErrUserBlocked
	}

	storedOTP, err := uc.cacheRepo.Get(ctx, getOTPKey(purpose, identifier))
	if err != nil {
		uc.logger.Warn("OTP not found or expired",
			zap.String("identifier", identifier),
		)
		return domain.ErrOTPInvalid
	}

	if subtle.ConstantTimeCompare([]byte(storedOTP), []byte(otp)) != 1 {
		failCount, err := uc.cacheRepo.IncrWithExpire(
			ctx,
			getVerifyFailKey(purpose, identifier),
			PolicyVerifyFailTTL,
		)
		if err != nil {
			return domain.ErrInternalServer
		}

		if failCount >= PolicyVerifyFailMax {
			uc.cacheRepo.Set(ctx, getBlockKey(purpose, identifier), "1", PolicyBlockDuration)
			uc.logger.Warn("User blocked due to OTP brute force",
				zap.String("identifier", identifier),
			)
			return domain.ErrUserBlocked
		}

		uc.logger.Warn("Invalid OTP",
			zap.String("identifier", identifier),
		)
		return domain.ErrOTPInvalid
	}

	uc.logger.Info("OTP verified successfully",
		zap.String("identifier", identifier),
	)
	return nil
}

func (uc *otpUseCase) Cleanup(ctx context.Context, purpose, identifier string) {
	keys := []string{
		getOTPKey(purpose, identifier),
		getCountKey(purpose, identifier),
		getRateKey(purpose, identifier),
		getVerifyFailKey(purpose, identifier),
	}

	for _, k := range keys {
		if err := uc.cacheRepo.Delete(ctx, k); err != nil {
			uc.logger.Warn("Failed to cleanup OTP key",
				zap.String("key", k),
				zap.Error(err),
			)
		}
	}
}

func generateOTP() (string, error) {
	const max = 1000000

	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%06d", n.Int64()), nil
}

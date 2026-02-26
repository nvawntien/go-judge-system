package container

import (
	"go-judge-system/pkg/cache"
	"go-judge-system/pkg/database"
	"go-judge-system/pkg/logger"
	"go-judge-system/services/auth/internal/adapter/inbound/http"
	"go-judge-system/services/auth/internal/adapter/inbound/http/handler"
	"go-judge-system/services/auth/internal/adapter/outbound/cache/redis"
	"go-judge-system/services/auth/internal/adapter/outbound/crypto"
	"go-judge-system/services/auth/internal/adapter/outbound/mail"
	"go-judge-system/services/auth/internal/adapter/outbound/persistence/postgres"
	"go-judge-system/services/auth/internal/application/usecase"

	"github.com/google/wire"
)

var OutboundProviderSet = wire.NewSet(
	postgres.NewUserRepository,
	redis.NewResetTokenRepository,
	crypto.NewTokenGenerator,
	redis.NewCacheRepository,
	mail.NewSMTPProvider,
)

var UseCaseProviderSet = wire.NewSet(
	usecase.NewOTPUseCase,
	usecase.NewRegisterUseCase,
	usecase.NewVerifyForgotPasswordUseCase,
	usecase.NewForgotPasswordUseCase,
	usecase.NewVerifyActivationUseCase,
	usecase.NewResendOTPUseCase,
	usecase.NewResetPasswordUseCase,
)

var InboundProviderSet = wire.NewSet(
	handler.NewRegisterHandler,
	handler.NewVerifyActivationHandler,
	handler.NewResetPasswordHandler,
	handler.NewResendOTPHandler,
	handler.NewVerifyForgotPasswordHandler,
	handler.NewForgotPasswordHandler,
	handler.NewAuthHandler,
	http.NewRouter,
)

var InfrastructureProviderSet = wire.NewSet(
	database.ConnectDatabase,
	cache.ConnectRedis,
	logger.NewLogger,
)

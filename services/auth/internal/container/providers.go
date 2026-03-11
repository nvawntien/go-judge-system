package container

import (
	"go-judge-system/pkg/cache"
	"go-judge-system/pkg/database"
	"go-judge-system/pkg/logger"
	"go-judge-system/services/auth/internal/adapter/inbound/http"
	"go-judge-system/services/auth/internal/adapter/inbound/http/handler"
	"go-judge-system/services/auth/internal/adapter/inbound/http/middleware"
	"go-judge-system/services/auth/internal/adapter/outbound/cache/redis"
	"go-judge-system/services/auth/internal/adapter/outbound/crypto"
	"go-judge-system/services/auth/internal/adapter/outbound/jwt"
	"go-judge-system/services/auth/internal/adapter/outbound/mail"
	"go-judge-system/services/auth/internal/adapter/outbound/otp"
	"go-judge-system/services/auth/internal/adapter/outbound/persistence/postgres"
	"go-judge-system/services/auth/internal/adapter/outbound/security"
	"go-judge-system/services/auth/internal/application/usecase"

	"github.com/google/wire"
)

var OutboundProviderSet = wire.NewSet(
	postgres.NewUserRepository,
	redis.NewResetTokenRepository,
	crypto.NewResetTokenGenerator,
	redis.NewCacheRepository,
	mail.NewSMTPProvider,
	security.NewBcryptHasher,
	jwt.NewJWTProvider,
	otp.NewOTPService,
)

var MiddlewareProviderSet = wire.NewSet(
	middleware.NewAuthMiddleware,
)

var UseCaseProviderSet = wire.NewSet(
	usecase.NewRegisterUseCase,
	usecase.NewVerifyForgotPasswordUseCase,
	usecase.NewForgotPasswordUseCase,
	usecase.NewVerifyActivationUseCase,
	usecase.NewResendOTPUseCase,
	usecase.NewResetPasswordUseCase,
	usecase.NewLoginUseCase,
	usecase.NewChangePasswordUseCase,
	usecase.NewRefreshTokenUseCase,
	usecase.NewGetProfileUseCase,
	usecase.NewUpdateUserRoleUseCase,
)

var InboundProviderSet = wire.NewSet(
	handler.NewLoginHandler,
	handler.NewRegisterHandler,
	handler.NewVerifyActivationHandler,
	handler.NewResetPasswordHandler,
	handler.NewResendOTPHandler,
	handler.NewVerifyForgotPasswordHandler,
	handler.NewForgotPasswordHandler,
	handler.NewChangePasswordHandler,
	handler.NewLogoutHandler,
	handler.NewRefreshTokenHandler,
	handler.NewGetProfileHandler,
	handler.NewUpdateUserRoleHandler,
	handler.NewAuthHandler,
	http.NewRouter,
)

var InfrastructureProviderSet = wire.NewSet(
	database.ConnectDatabase,
	cache.ConnectRedis,
	logger.NewLogger,
)

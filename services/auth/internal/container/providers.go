package container

import (
	auth "go-judge-system/pkg/auth"
	"go-judge-system/pkg/cache"
	"go-judge-system/pkg/database"
	"go-judge-system/pkg/logger"
	"go-judge-system/pkg/middleware"
	"go-judge-system/services/auth/internal/adapter/inbound/http"
	"go-judge-system/services/auth/internal/adapter/inbound/http/handler"
	authhandler "go-judge-system/services/auth/internal/adapter/inbound/http/handler/auth"
	"go-judge-system/services/auth/internal/adapter/outbound/cache/redis"
	"go-judge-system/services/auth/internal/adapter/outbound/crypto"
	"go-judge-system/services/auth/internal/adapter/outbound/jwt"
	"go-judge-system/services/auth/internal/adapter/outbound/mail"
	"go-judge-system/services/auth/internal/adapter/outbound/persistence/postgres"
	"go-judge-system/services/auth/internal/adapter/outbound/security"
	authusecase "go-judge-system/services/auth/internal/application/usecase/auth"

	"github.com/google/wire"
)

var InfrastructureProviderSet = wire.NewSet(
	database.ConnectDatabase,
	cache.ConnectRedis,
	logger.NewLogger,
)

var OutboundProviderSet = wire.NewSet(
	postgres.NewUserRepository,
	redis.NewTokenRepository,
	auth.NewRedisLogoutAllIATStore,
	jwt.NewJWTProvider,
	crypto.NewTokenGenerator,
	security.NewBcryptHasher,
	mail.NewSMTPProvider,
)

var MiddlewareProviderSet = wire.NewSet(
	middleware.NewAuthMiddleware,
)

var UseCaseProviderSet = wire.NewSet(
	authusecase.NewRegisterUseCase,
	authusecase.NewVerifyEmailUseCase,
	authusecase.NewResendVerificationUseCase,
	authusecase.NewLoginUseCase,
	authusecase.NewForgotPasswordUseCase,
	authusecase.NewResetPasswordUseCase,
	authusecase.NewChangePasswordUseCase,
	authusecase.NewLogoutAllUseCase,
	authusecase.NewRefreshTokenUseCase,
)

var InboundProviderSet = wire.NewSet(
	authhandler.NewRegisterHandler,
	authhandler.NewVerifyEmailHandler,
	authhandler.NewResendVerificationHandler,
	authhandler.NewLoginHandler,
	authhandler.NewLogoutHandler,
	authhandler.NewLogoutAllHandler,
	authhandler.NewForgotPasswordHandler,
	authhandler.NewResetPasswordHandler,
	authhandler.NewChangePasswordHandler,
	authhandler.NewRefreshTokenHandler,
	handler.NewAuthHandler,
	http.NewRouter,
)

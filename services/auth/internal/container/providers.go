package container

import (
	auth "go-judge-system/pkg/auth"
	"go-judge-system/pkg/cache"
	"go-judge-system/pkg/database"
	"go-judge-system/pkg/logger"
	"go-judge-system/pkg/middleware"
	minioclient "go-judge-system/pkg/minio"
	"go-judge-system/services/auth/internal/adapter/inbound/http"
	"go-judge-system/services/auth/internal/adapter/inbound/http/handler"
	adminhandler "go-judge-system/services/auth/internal/adapter/inbound/http/handler/admin"
	authhandler "go-judge-system/services/auth/internal/adapter/inbound/http/handler/auth"
	userhandler "go-judge-system/services/auth/internal/adapter/inbound/http/handler/user"
	"go-judge-system/services/auth/internal/adapter/outbound/cache/redis"
	"go-judge-system/services/auth/internal/adapter/outbound/crypto"
	"go-judge-system/services/auth/internal/adapter/outbound/jwt"
	"go-judge-system/services/auth/internal/adapter/outbound/mail"
	"go-judge-system/services/auth/internal/adapter/outbound/persistence/postgres"
	"go-judge-system/services/auth/internal/adapter/outbound/security"
	miniostorage "go-judge-system/services/auth/internal/adapter/outbound/storage/minio"
	adminusecase "go-judge-system/services/auth/internal/application/usecase/admin"
	authusecase "go-judge-system/services/auth/internal/application/usecase/auth"
	userusecase "go-judge-system/services/auth/internal/application/usecase/user"

	"github.com/google/wire"
)

var InfrastructureProviderSet = wire.NewSet(
	database.ConnectDatabase,
	cache.ConnectRedis,
	logger.NewLogger,
	minioclient.NewMinioClient,
)

var OutboundProviderSet = wire.NewSet(
	postgres.NewUserRepository,
	redis.NewTokenRepository,
	auth.NewRedisLogoutAllIATStore,
	jwt.NewJWTProvider,
	crypto.NewTokenGenerator,
	security.NewBcryptHasher,
	mail.NewSMTPProvider,
	miniostorage.NewAvatarStorage,
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

	userusecase.NewGetMeUseCase,
	userusecase.NewGetProfileUseCase,
	userusecase.NewUpdateProfileUseCase,
	userusecase.NewUploadAvatarUseCase,

	adminusecase.NewAssignRoleUseCase,
	adminusecase.NewAdminUsersUseCase,
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

	userhandler.NewGetMeHandler,
	userhandler.NewGetProfileHandler,
	userhandler.NewUpdateProfileHandler,
	userhandler.NewUploadAvatarHandler,

	adminhandler.NewAssignRoleHandler,
	adminhandler.NewAdminUsersHandler,

	handler.NewAuthHandler,
	handler.NewUserHandler,
	handler.NewAdminHandler,
	http.NewRouter,
)

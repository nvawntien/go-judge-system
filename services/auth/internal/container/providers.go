package container

import (
	"go-judge-system/pkg/database"
	"go-judge-system/pkg/logger"
	"go-judge-system/pkg/cache"
	"go-judge-system/services/auth/internal/adapter/inbound/http"
	"go-judge-system/services/auth/internal/adapter/inbound/http/handler"
	"go-judge-system/services/auth/internal/adapter/outbound/cache/redis"
	"go-judge-system/services/auth/internal/adapter/outbound/mail"
	"go-judge-system/services/auth/internal/adapter/outbound/persistence/postgres"
	"go-judge-system/services/auth/internal/application/usecase"

	"github.com/google/wire"
)

var OutboundProviderSet = wire.NewSet(
	postgres.NewUserRepository,
	redis.NewCacheRepository,
	mail.NewSMTPProvider,
)

var UseCaseProviderSet = wire.NewSet(
	usecase.NewOTPUseCase,
	usecase.NewRegisterUseCase,
)

var InboundProviderSet = wire.NewSet(
	handler.NewRegisterHandler,
	handler.NewAuthHandler,
	http.NewRouter,
)

var InfrastructureProviderSet = wire.NewSet(
	database.ConnectDatabase,
	cache.ConnectRedis,
	logger.NewLogger,
)

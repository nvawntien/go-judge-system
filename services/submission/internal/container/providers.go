package container

import (
	auth "go-judge-system/pkg/auth"
	"go-judge-system/pkg/cache"
	"go-judge-system/pkg/database"
	"go-judge-system/pkg/kafka"
	"go-judge-system/pkg/logger"
	"go-judge-system/pkg/middleware"
	"go-judge-system/services/submission/internal/adapter/inbound/http"
	"go-judge-system/services/submission/internal/adapter/outbound/outbox"
	"go-judge-system/services/submission/internal/adapter/outbound/persistence/postgres"

	"github.com/google/wire"
)

var InfrastructureProviderSet = wire.NewSet(
	database.ConnectDatabase,
	cache.ConnectRedis,
	logger.NewLogger,
	kafka.NewSyncProducer,
)

var MiddlewareProviderSet = wire.NewSet(
	middleware.NewAuthMiddleware,
)

var OutboundProviderSet = wire.NewSet(
	postgres.NewOutboxRepository,
	auth.NewRedisLogoutAllIATStore,
	outbox.NewOutboxRelay,
)

var InboundProviderSet = wire.NewSet(
	http.NewRouter,
)

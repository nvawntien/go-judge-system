package container

import (
	auth "go-judge-system/pkg/auth"
	"go-judge-system/pkg/cache"
	"go-judge-system/pkg/database"
	"go-judge-system/pkg/kafka"
	"go-judge-system/pkg/logger"
	"go-judge-system/pkg/middleware"
	"go-judge-system/services/submission/internal/adapter/inbound/http"
	"go-judge-system/services/submission/internal/adapter/inbound/http/handler"
	userhandler "go-judge-system/services/submission/internal/adapter/inbound/http/handler/user"
	judgepublisher "go-judge-system/services/submission/internal/adapter/outbound/judge"
	"go-judge-system/services/submission/internal/adapter/outbound/outbox"
	"go-judge-system/services/submission/internal/adapter/outbound/persistence/postgres"
	userusecase "go-judge-system/services/submission/internal/application/usecase/user"

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
	postgres.NewSubmissionRepository,
	postgres.NewTransactionManager,
	postgres.NewOutboxRepository,
	judgepublisher.NewOutboxJudgePublisher,
	auth.NewRedisLogoutAllIATStore,
	outbox.NewOutboxRelay,
)

var UseCaseProviderSet = wire.NewSet(
	userusecase.NewCreateSubmissionUseCase,
)

var InboundProviderSet = wire.NewSet(
	userhandler.NewCreateSubmissionHandler,
	handler.NewUserHandler,
	http.NewRouter,
)

package container

import (
	auth "go-judge-system/pkg/auth"
	"go-judge-system/pkg/cache"
	"go-judge-system/pkg/database"
	"go-judge-system/pkg/logger"
	"go-judge-system/pkg/middleware"
	"go-judge-system/services/problem/internal/adapter/inbound/http"
	"go-judge-system/services/problem/internal/adapter/inbound/http/handler"
	adminproblemhandler "go-judge-system/services/problem/internal/adapter/inbound/http/handler/admin/problem"
	"go-judge-system/services/problem/internal/adapter/outbound/persistence/postgres"
	testcasestorage "go-judge-system/services/problem/internal/adapter/outbound/storage/minio"
	adminproblemusecase "go-judge-system/services/problem/internal/application/usecase/admin/problem"

	"github.com/google/wire"
)

var InfrastructureProviderSet = wire.NewSet(
	database.ConnectDatabase,
	cache.ConnectRedis,
	logger.NewLogger,
	//minioclient.NewMinioClient,
)

var OutboundProviderSet = wire.NewSet(
	postgres.NewProblemRepository,
	postgres.NewTestCaseRepository,
	testcasestorage.NewTestCaseStorage,
	auth.NewRedisLogoutAllIATStore,
)

var MiddlewareProviderSet = wire.NewSet(
	middleware.NewAuthMiddleware,
)

var UseCaseProviderSet = wire.NewSet(
	adminproblemusecase.NewCreateProblemUseCase,
)

var InboundProviderSet = wire.NewSet(
	adminproblemhandler.NewCreateProblemHandler,

	handler.NewAdminHandler,
	http.NewRouter,
)

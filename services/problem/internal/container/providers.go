package container

import (
	auth "go-judge-system/pkg/auth"
	"go-judge-system/pkg/cache"
	"go-judge-system/pkg/database"
	"go-judge-system/pkg/logger"
	"go-judge-system/pkg/middleware"
	"go-judge-system/pkg/minio"
	"go-judge-system/services/problem/internal/adapter/inbound/http"
	"go-judge-system/services/problem/internal/adapter/inbound/http/handler"
	adminproblemhandler "go-judge-system/services/problem/internal/adapter/inbound/http/handler/admin/problem"
	admintestcasehandler "go-judge-system/services/problem/internal/adapter/inbound/http/handler/admin/testcase"
	userproblemhandler "go-judge-system/services/problem/internal/adapter/inbound/http/handler/user/problem"
	"go-judge-system/services/problem/internal/adapter/outbound/persistence/postgres"
	testcasestorage "go-judge-system/services/problem/internal/adapter/outbound/storage/minio"
	adminproblemusecase "go-judge-system/services/problem/internal/application/usecase/admin/problem"
	admintestcaseusecase "go-judge-system/services/problem/internal/application/usecase/admin/testcase"
	userproblemusecase "go-judge-system/services/problem/internal/application/usecase/user/problem"

	"github.com/google/wire"
)

var InfrastructureProviderSet = wire.NewSet(
	database.ConnectDatabase,
	cache.ConnectRedis,
	logger.NewLogger,
	minio.NewMinioClient,
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
	adminproblemusecase.NewListProblemsUseCase,
	adminproblemusecase.NewGetProblemUseCase,
	adminproblemusecase.NewPublishProblemUseCase,
	adminproblemusecase.NewHiddenProblemUseCase,

	admintestcaseusecase.NewUploadTestCaseUseCase,

	userproblemusecase.NewListProblemsUseCase,
	userproblemusecase.NewGetProblemUseCase,
)

var InboundProviderSet = wire.NewSet(
	adminproblemhandler.NewCreateProblemHandler,
	adminproblemhandler.NewListProblemsHandler,
	adminproblemhandler.NewGetProblemHandler,
	adminproblemhandler.NewPublishProblemHandler,
	adminproblemhandler.NewHiddenProblemHandler,

	admintestcasehandler.NewUploadTestCaseHandler,

	userproblemhandler.NewListProblemsHandler,
	userproblemhandler.NewGetProblemHandler,

	handler.NewAdminHandler,
	handler.NewUserHandler,
	http.NewRouter,
)

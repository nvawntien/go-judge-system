package container

import (
	auth "go-judge-system/pkg/auth"
	"go-judge-system/pkg/cache"
	"go-judge-system/pkg/database"
	"go-judge-system/pkg/logger"
	"go-judge-system/pkg/middleware"
	minioclient "go-judge-system/pkg/minio"
	"go-judge-system/services/problem/internal/adapter/inbound/http"
	"go-judge-system/services/problem/internal/adapter/inbound/http/handler"
	probhd "go-judge-system/services/problem/internal/adapter/inbound/http/handler/problem"
	testhd "go-judge-system/services/problem/internal/adapter/inbound/http/handler/test_case"
	"go-judge-system/services/problem/internal/adapter/outbound/persistence/postgres"
	testcasestorage "go-judge-system/services/problem/internal/adapter/outbound/storage/minio"
	probuc "go-judge-system/services/problem/internal/application/usecase/problem"
	testuc "go-judge-system/services/problem/internal/application/usecase/test_case"

	"github.com/google/wire"
)

var InfrastructureProviderSet = wire.NewSet(
	database.ConnectDatabase,
	cache.ConnectRedis,
	logger.NewLogger,
	minioclient.NewMinioClient,
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
	probuc.NewCreateProblemUseCase,
	probuc.NewUpdateProblemUseCase,
	probuc.NewDeleteProblemUseCase,
	probuc.NewGetProblemUseCase,
	probuc.NewListProblemsUseCase,
	probuc.NewPublishProblemUseCase,
	probuc.NewHideProblemUseCase,

	testuc.NewUploadTestCaseUseCase,
	testuc.NewGetTestCaseForWorkerUseCase,
	testuc.NewGCOrphanZipsUseCase,
	testuc.NewGCRunner,
)

var InboundProviderSet = wire.NewSet(
	probhd.NewCreateProblemHandler,
	probhd.NewUpdateProblemHandler,
	probhd.NewDeleteProblemHandler,
	probhd.NewGetProblemHandler,
	probhd.NewListProblemsHandler,
	probhd.NewPublishProblemHandler,
	probhd.NewHideProblemHandler,

	testhd.NewUploadTestCaseHandler,
	testhd.NewGetTestCaseForWorkerHandler,

	handler.NewProblemHandler,
	handler.NewTestCaseHandler,
	http.NewRouter,
)

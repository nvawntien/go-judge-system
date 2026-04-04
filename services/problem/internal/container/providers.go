package container

import (
	"go-judge-system/pkg/database"
	"go-judge-system/pkg/logger"
	"go-judge-system/pkg/minio"
	"go-judge-system/services/problem/internal/adapter/inbound/http"
	"go-judge-system/services/problem/internal/adapter/inbound/http/handler"
	probhd "go-judge-system/services/problem/internal/adapter/inbound/http/handler/problem"
	testhd "go-judge-system/services/problem/internal/adapter/inbound/http/handler/test_case"
	"go-judge-system/services/problem/internal/adapter/inbound/http/middleware"
	"go-judge-system/services/problem/internal/adapter/outbound/persistence/postgres"
	"go-judge-system/services/problem/internal/adapter/outbound/storage/minio"
	probuc "go-judge-system/services/problem/internal/application/usecase/problem"
	testuc "go-judge-system/services/problem/internal/application/usecase/test_case"

	"github.com/google/wire"
)

var InfrastructureProviderSet = wire.NewSet(
	database.ConnectDatabase,
	logger.NewLogger,
	minio.NewMinioClient,
)

var OutboundProviderSet = wire.NewSet(
	postgres.NewProblemRepository,
	postgres.NewTestCaseRepository,
	storage.NewMinioStorage,
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

	handler.NewProblemHandler,
	handler.NewTestCaseHandler,
	http.NewRouter,
)

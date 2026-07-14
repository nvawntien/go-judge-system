package container

import (
	auth "go-judge-system/pkg/auth"
	"go-judge-system/pkg/cache"
	"go-judge-system/pkg/database"
	"go-judge-system/pkg/logger"
	"go-judge-system/pkg/middleware"
	"go-judge-system/pkg/minio"
	"go-judge-system/services/problem/internal/adapter/inbound/grpc"
	grpchandler "go-judge-system/services/problem/internal/adapter/inbound/grpc/handler"
	grpctestcasehandler "go-judge-system/services/problem/internal/adapter/inbound/grpc/handler/worker/testcase"
	"go-judge-system/services/problem/internal/adapter/inbound/http"
	"go-judge-system/services/problem/internal/adapter/inbound/http/handler"
	adminproblemhandler "go-judge-system/services/problem/internal/adapter/inbound/http/handler/admin/problem"
	admintaghandler "go-judge-system/services/problem/internal/adapter/inbound/http/handler/admin/tag"
	admintestcasehandler "go-judge-system/services/problem/internal/adapter/inbound/http/handler/admin/testcase"
	userproblemhandler "go-judge-system/services/problem/internal/adapter/inbound/http/handler/user/problem"
	usertaghandler "go-judge-system/services/problem/internal/adapter/inbound/http/handler/user/tag"
	"go-judge-system/services/problem/internal/adapter/outbound/persistence/postgres"
	testcasestorage "go-judge-system/services/problem/internal/adapter/outbound/storage/minio"
	adminproblemusecase "go-judge-system/services/problem/internal/application/usecase/admin/problem"
	admintagusecase "go-judge-system/services/problem/internal/application/usecase/admin/tag"
	admintestcaseusecase "go-judge-system/services/problem/internal/application/usecase/admin/testcase"
	userproblemusecase "go-judge-system/services/problem/internal/application/usecase/user/problem"
	usertagusecase "go-judge-system/services/problem/internal/application/usecase/user/tag"
	workertestcaseusecase "go-judge-system/services/problem/internal/application/usecase/worker/testcase"

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
	postgres.NewTagRepository,
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
	adminproblemusecase.NewUpdateProblemUseCase,
	adminproblemusecase.NewGetProblemUseCase,
	adminproblemusecase.NewPublishProblemUseCase,
	adminproblemusecase.NewHiddenProblemUseCase,
	adminproblemusecase.NewDeleteProblemUseCase,

	admintagusecase.NewListTagsUseCase,
	admintagusecase.NewCreateTagUseCase,
	admintagusecase.NewUpdateTagUseCase,
	admintagusecase.NewDeleteTagUseCase,

	admintestcaseusecase.NewUploadTestCaseUseCase,
	admintestcaseusecase.NewGetTestCaseUseCase,
	admintestcaseusecase.NewDeleteTestCaseUseCase,

	userproblemusecase.NewListProblemsUseCase,
	userproblemusecase.NewListMyProblemsUseCase,
	userproblemusecase.NewGetProblemUseCase,
	usertagusecase.NewListTagsUseCase,

	workertestcaseusecase.NewGetTestCaseUseCase,
)

var InboundProviderSet = wire.NewSet(
	adminproblemhandler.NewCreateProblemHandler,
	adminproblemhandler.NewListProblemsHandler,
	adminproblemhandler.NewUpdateProblemHandler,
	adminproblemhandler.NewGetProblemHandler,
	adminproblemhandler.NewPublishProblemHandler,
	adminproblemhandler.NewHiddenProblemHandler,
	adminproblemhandler.NewDeleteProblemHandler,

	admintaghandler.NewListTagsHandler,
	admintaghandler.NewCreateTagHandler,
	admintaghandler.NewUpdateTagHandler,
	admintaghandler.NewDeleteTagHandler,

	admintestcasehandler.NewUploadTestCaseHandler,
	admintestcasehandler.NewGetTestCaseHandler,
	admintestcasehandler.NewDeleteTestCaseHandler,

	userproblemhandler.NewListProblemsHandler,
	userproblemhandler.NewListMyProblemsHandler,
	userproblemhandler.NewGetProblemHandler,
	usertaghandler.NewListTagsHandler,

	handler.NewAdminHandler,
	handler.NewUserHandler,
	http.NewRouter,

	grpctestcasehandler.NewGetTestCaseHandler,
	grpchandler.NewWorkerHandler,
	grpc.NewProblemServer,
	grpc.NewServer,
)

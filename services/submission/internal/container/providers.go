package container

import (
	"go-judge-system/pkg/database"
	"go-judge-system/pkg/logger"
	"go-judge-system/services/submission/internal/adapter/inbound/http"
	"go-judge-system/services/submission/internal/adapter/inbound/http/handler"
	subhd "go-judge-system/services/submission/internal/adapter/inbound/http/handler/submission"
	"go-judge-system/services/submission/internal/adapter/inbound/http/middleware"
	"go-judge-system/services/submission/internal/adapter/outbound/judge"
	"go-judge-system/services/submission/internal/adapter/outbound/persistence/postgres"
	"go-judge-system/services/submission/internal/adapter/outbound/problem"
	subuc "go-judge-system/services/submission/internal/application/usecase/submission"

	"github.com/google/wire"
)

var InfrastructureProviderSet = wire.NewSet(
	database.ConnectDatabase,
	logger.NewLogger,
)

var MiddlewareProviderSet = wire.NewSet(
	middleware.NewAuthMiddleware,
)

var OutboundProviderSet = wire.NewSet(
	postgres.NewSubmissionRepository,
	postgres.NewSubmissionResultRepository,
	problem.NewProblemAccessChecker,
	judge.NewNoopJudgePublisher,
)

var UseCaseProviderSet = wire.NewSet(
	subuc.NewCreateSubmissionUseCase,
	subuc.NewListSubmissionsUseCase,
	subuc.NewGetSubmissionUseCase,
)

var InboundProviderSet = wire.NewSet(
	subhd.NewCreateSubmissionHandler,
	subhd.NewListSubmissionsHandler,
	subhd.NewGetSubmissionHandler,
	handler.NewSubmissionHandler,
	http.NewRouter,
)

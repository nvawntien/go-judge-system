package container

import (
	"go-judge-system/pkg/database"
	"go-judge-system/pkg/kafka"
	"go-judge-system/pkg/logger"
	"go-judge-system/services/submission/internal/adapter/inbound/http"
	"go-judge-system/services/submission/internal/adapter/inbound/http/handler"
	subhd "go-judge-system/services/submission/internal/adapter/inbound/http/handler/submission"
	"go-judge-system/services/submission/internal/adapter/inbound/http/middleware"
	kafkain "go-judge-system/services/submission/internal/adapter/inbound/kafka"
	"go-judge-system/services/submission/internal/adapter/outbound/judge"
	"go-judge-system/services/submission/internal/adapter/outbound/persistence/postgres"
	"go-judge-system/services/submission/internal/adapter/outbound/outbox"
	"go-judge-system/services/submission/internal/adapter/outbound/problem"
	subuc "go-judge-system/services/submission/internal/application/usecase/submission"

	"github.com/google/wire"
)

var InfrastructureProviderSet = wire.NewSet(
	database.ConnectDatabase,
	logger.NewLogger,
	kafka.NewSyncProducer,
	kafka.NewConsumerGroup,
)

var MiddlewareProviderSet = wire.NewSet(
	middleware.NewAuthMiddleware,
)

var OutboundProviderSet = wire.NewSet(
	postgres.NewSubmissionRepository,
	postgres.NewSubmissionResultRepository,
	postgres.NewTransactionManager,
	postgres.NewOutboxRepository,
	problem.NewProblemAccessChecker,
	judge.NewOutboxJudgePublisher,
	outbox.NewOutboxRelay,
)

var UseCaseProviderSet = wire.NewSet(
	subuc.NewCreateSubmissionUseCase,
	subuc.NewListSubmissionsUseCase,
	subuc.NewGetSubmissionUseCase,
	subuc.NewRejudgeSubmissionUseCase,
	subuc.NewProcessJudgeResultUseCase,
)

var InboundProviderSet = wire.NewSet(
	kafkain.NewJudgeResultConsumer,
	subhd.NewCreateSubmissionHandler,
	subhd.NewListSubmissionsHandler,
	subhd.NewGetSubmissionHandler,
	subhd.NewRejudgeSubmissionHandler,
	handler.NewSubmissionHandler,
	http.NewRouter,
)

package container

import (
	"fmt"
	"strings"
	"time"

	auth "go-judge-system/pkg/auth"
	"go-judge-system/pkg/cache"
	"go-judge-system/pkg/config"
	"go-judge-system/pkg/database"
	sharedgrpc "go-judge-system/pkg/grpc"
	"go-judge-system/pkg/kafka"
	"go-judge-system/pkg/logger"
	"go-judge-system/pkg/middleware"
	judgev1 "go-judge-system/pkg/pb/judge/v1"
	problemv1 "go-judge-system/pkg/pb/problem/v1"
	"go-judge-system/services/submission/internal/adapter/inbound/http"
	"go-judge-system/services/submission/internal/adapter/inbound/http/handler"
	adminhandler "go-judge-system/services/submission/internal/adapter/inbound/http/handler/admin"
	userhandler "go-judge-system/services/submission/internal/adapter/inbound/http/handler/user"
	resultconsumer "go-judge-system/services/submission/internal/adapter/inbound/kafka"
	attemptid "go-judge-system/services/submission/internal/adapter/outbound/id"
	judgepublisher "go-judge-system/services/submission/internal/adapter/outbound/judge"
	"go-judge-system/services/submission/internal/adapter/outbound/outbox"
	"go-judge-system/services/submission/internal/adapter/outbound/persistence/postgres"
	problemreader "go-judge-system/services/submission/internal/adapter/outbound/problem"
	"go-judge-system/services/submission/internal/application/dto"
	adminusecase "go-judge-system/services/submission/internal/application/usecase/admin"
	resultusecase "go-judge-system/services/submission/internal/application/usecase/result"
	userusecase "go-judge-system/services/submission/internal/application/usecase/user"

	"github.com/google/wire"
	googlegrpc "google.golang.org/grpc"
)

func ProvideProblemGRPCConfig(cfg *config.Config) (config.ProblemGRPCConfig, error) {
	problemCfg := cfg.ProblemGRPC
	problemCfg.Address = strings.TrimSpace(problemCfg.Address)
	if problemCfg.Address == "" {
		problemCfg.Address = "problem-service:9092"
	}
	if problemCfg.Timeout == 0 {
		problemCfg.Timeout = time.Second
	}
	if problemCfg.Timeout < 0 {
		return config.ProblemGRPCConfig{}, fmt.Errorf("problem gRPC timeout must be greater than zero")
	}

	return problemCfg, nil
}

func ProvideProblemClientConn(cfg config.ProblemGRPCConfig) (*googlegrpc.ClientConn, error) {
	conn, err := sharedgrpc.NewClientConn(cfg.Address, sharedgrpc.WithInsecureTransport())
	if err != nil {
		return nil, fmt.Errorf("create Problem Service gRPC connection: %w", err)
	}
	return conn, nil
}

func ProvideProblemServiceClient(conn *googlegrpc.ClientConn) problemv1.ProblemServiceClient {
	return problemv1.NewProblemServiceClient(conn)
}

func ProvideProblemGRPCTimeout(cfg config.ProblemGRPCConfig) time.Duration {
	return cfg.Timeout
}

type JudgeClientConn struct {
	*googlegrpc.ClientConn
}

func ProvideJudgeGRPCConfig(cfg *config.Config) (config.JudgeGRPCConfig, error) {
	judgeCfg := cfg.JudgeGRPC
	judgeCfg.Address = strings.TrimSpace(judgeCfg.Address)
	if judgeCfg.Address == "" {
		judgeCfg.Address = "judge-worker:9093"
	}
	if judgeCfg.Timeout == 0 {
		judgeCfg.Timeout = 30 * time.Second
	}
	if judgeCfg.Timeout < 0 {
		return config.JudgeGRPCConfig{}, fmt.Errorf("judge gRPC timeout must be greater than zero")
	}
	return judgeCfg, nil
}

func ProvideJudgeClientConn(cfg config.JudgeGRPCConfig) (JudgeClientConn, error) {
	conn, err := sharedgrpc.NewClientConn(cfg.Address, sharedgrpc.WithInsecureTransport())
	if err != nil {
		return JudgeClientConn{}, fmt.Errorf("create Judge Worker gRPC connection: %w", err)
	}
	return JudgeClientConn{ClientConn: conn}, nil
}

func ProvideJudgeServiceClient(conn JudgeClientConn) judgev1.JudgeServiceClient {
	return judgev1.NewJudgeServiceClient(conn.ClientConn)
}

func ProvideRunCodeLimits(cfg *config.Config) dto.RunCodeLimits {
	runCfg := cfg.RunCode
	return dto.RunCodeLimits{
		MaxTestCases:           runCfg.MaxTestCases,
		MaxSourceCodeBytes:     runCfg.MaxSourceCodeBytes,
		MaxStdinBytes:          runCfg.MaxStdinBytes,
		MaxExpectedOutputBytes: runCfg.MaxExpectedOutputBytes,
		MaxCapturedOutputBytes: runCfg.MaxCapturedOutputBytes,
		RequestTimeout:         runCfg.RequestTimeout,
		DefaultTimeLimit:       runCfg.DefaultTimeLimit,
		DefaultMemoryLimitKB:   runCfg.DefaultMemoryLimitKB,
		DefaultOutputLimit:     runCfg.DefaultOutputLimitBytes,
	}
}

var InfrastructureProviderSet = wire.NewSet(
	database.ConnectDatabase,
	cache.ConnectRedis,
	logger.NewLogger,
	kafka.NewSyncProducer,
	kafka.NewConsumerGroup,
	ProvideProblemGRPCConfig,
	ProvideProblemClientConn,
	ProvideProblemServiceClient,
	ProvideProblemGRPCTimeout,
	ProvideJudgeGRPCConfig,
	ProvideJudgeClientConn,
	ProvideJudgeServiceClient,
	ProvideRunCodeLimits,
)

var MiddlewareProviderSet = wire.NewSet(
	middleware.NewAuthMiddleware,
)

var OutboundProviderSet = wire.NewSet(
	postgres.NewSubmissionRepository,
	postgres.NewSubmissionResultRepository,
	postgres.NewTransactionManager,
	postgres.NewOutboxRepository,
	attemptid.NewUUIDAttemptIDGenerator,
	judgepublisher.NewOutboxJudgePublisher,
	auth.NewRedisLogoutAllIATStore,
	outbox.NewOutboxRelay,
	resultconsumer.NewDLTPublisher,
	problemreader.NewGRPCProblemReader,
	judgepublisher.NewGRPCRunner,
)

var UseCaseProviderSet = wire.NewSet(
	adminusecase.NewListAdminSubmissionsUseCase,
	resultusecase.NewApplyJudgeResultUseCase,
	userusecase.NewCreateSubmissionUseCase,
	userusecase.NewRunCodeUseCase,
	userusecase.NewGetSubmissionUseCase,
	userusecase.NewListMySubmissionsUseCase,
)

var InboundProviderSet = wire.NewSet(
	adminhandler.NewListSubmissionsHandler,
	userhandler.NewCreateSubmissionHandler,
	userhandler.NewRunCodeHandler,
	userhandler.NewGetSubmissionHandler,
	userhandler.NewListMySubmissionsHandler,
	handler.NewAdminHandler,
	handler.NewUserHandler,
	http.NewRouter,
	resultconsumer.NewJudgeResultConsumer,
)

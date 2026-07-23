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
	problemv1 "go-judge-system/pkg/pb/problem/v1"
	"go-judge-system/services/submission/internal/adapter/inbound/http"
	"go-judge-system/services/submission/internal/adapter/inbound/http/handler"
	userhandler "go-judge-system/services/submission/internal/adapter/inbound/http/handler/user"
	judgepublisher "go-judge-system/services/submission/internal/adapter/outbound/judge"
	"go-judge-system/services/submission/internal/adapter/outbound/outbox"
	"go-judge-system/services/submission/internal/adapter/outbound/persistence/postgres"
	problemreader "go-judge-system/services/submission/internal/adapter/outbound/problem"
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

var InfrastructureProviderSet = wire.NewSet(
	database.ConnectDatabase,
	cache.ConnectRedis,
	logger.NewLogger,
	kafka.NewSyncProducer,
	ProvideProblemGRPCConfig,
	ProvideProblemClientConn,
	ProvideProblemServiceClient,
	ProvideProblemGRPCTimeout,
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
	problemreader.NewGRPCProblemReader,
)

var UseCaseProviderSet = wire.NewSet(
	userusecase.NewCreateSubmissionUseCase,
	userusecase.NewGetSubmissionUseCase,
)

var InboundProviderSet = wire.NewSet(
	userhandler.NewCreateSubmissionHandler,
	userhandler.NewGetSubmissionHandler,
	handler.NewUserHandler,
	http.NewRouter,
)

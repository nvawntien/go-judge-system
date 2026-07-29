package container

import (
	"fmt"
	"strings"
	"time"

	"go-judge-system/pkg/config"
	sharedgrpc "go-judge-system/pkg/grpc"
	"go-judge-system/pkg/kafka"
	"go-judge-system/pkg/logger"
	problemv1 "go-judge-system/pkg/pb/problem/v1"
	grpcin "go-judge-system/workers/judge/internal/adapter/inbound/grpc"
	grpchandler "go-judge-system/workers/judge/internal/adapter/inbound/grpc/handler"
	kafkain "go-judge-system/workers/judge/internal/adapter/inbound/kafka"
	"go-judge-system/workers/judge/internal/adapter/outbound/execute"
	"go-judge-system/workers/judge/internal/adapter/outbound/judge"
	"go-judge-system/workers/judge/internal/adapter/outbound/problem"
	"go-judge-system/workers/judge/internal/application/port/inbound"
	"go-judge-system/workers/judge/internal/application/port/outbound"
	judgeuc "go-judge-system/workers/judge/internal/application/usecase/judge"

	"github.com/google/wire"
	"go.uber.org/zap"
	googlegrpc "google.golang.org/grpc"
)

var InfrastructureProviderSet = wire.NewSet(
	ProvideLoggerConfig,
	ProvideKafkaConfig,
	ProvideServiceName,
	logger.NewLogger,
	kafka.NewSyncProducer,
	kafka.NewConsumerGroup,
	ProvideProblemGRPCConfig,
	ProvideProblemClientConn,
	ProvideProblemServiceClient,
	ProvideProblemGRPCTimeout,
)

var OutboundProviderSet = wire.NewSet(
	ProvideGoJudgeClient,
	wire.Bind(new(outbound.CodeExecutor), new(*execute.GoJudgeClient)),
	judge.NewKafkaResultPublisher,
	wire.Bind(new(outbound.ResultPublisher), new(*judge.KafkaResultPublisher)),
	ProvideProblemClient,
	wire.Bind(new(outbound.TestCaseFetcher), new(*problem.ProblemServiceClient)),
	ProvideSandboxServiceURL,
)

var UseCaseProviderSet = wire.NewSet(
	judgeuc.NewRunCodeUseCase,
	wire.Bind(new(inbound.RunCodeUseCase), new(*judgeuc.RunCodeUseCase)),
)

var InboundProviderSet = wire.NewSet(
	judgeuc.NewProcessJudgeJobUseCase,
	wire.Bind(new(inbound.ProcessJudgeJobUseCase), new(*judgeuc.ProcessJudgeJobUseCase)),
	kafkain.NewDLTPublisher,
	kafkain.NewJudgeJobConsumer,
	grpchandler.NewRunCodeHandler,
	grpcin.NewJudgeServer,
	grpcin.NewServer,
)

// Config extract functions for Wire
func ProvideKafkaConfig(cfg *config.Config) config.KafkaConfig {
	return cfg.Kafka
}

func ProvideLoggerConfig(cfg *config.Config) config.LoggerConfig {
	return cfg.Logger
}

func ProvideServiceName() string {
	return "judge-worker"
}

type ProblemClientConn struct {
	*googlegrpc.ClientConn
}

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

func ProvideProblemClientConn(cfg config.ProblemGRPCConfig) (ProblemClientConn, error) {
	conn, err := sharedgrpc.NewClientConn(cfg.Address, sharedgrpc.WithInsecureTransport())
	if err != nil {
		return ProblemClientConn{}, fmt.Errorf("create Problem Service gRPC connection: %w", err)
	}
	return ProblemClientConn{ClientConn: conn}, nil
}

func ProvideProblemServiceClient(conn ProblemClientConn) problemv1.ProblemServiceClient {
	return problemv1.NewProblemServiceClient(conn.ClientConn)
}

func ProvideProblemGRPCTimeout(cfg config.ProblemGRPCConfig) time.Duration {
	return cfg.Timeout
}

func ProvideProblemClient(client problemv1.ProblemServiceClient, timeout time.Duration, logger *zap.Logger) *problem.ProblemServiceClient {
	return problem.NewProblemServiceClient(client, timeout, logger)
}

type SandboxServiceURL string

func ProvideSandboxServiceURL() SandboxServiceURL {
	return "http://judge_sandbox:5050"
}

func ProvideGoJudgeClient(url SandboxServiceURL, logger *zap.Logger) *execute.GoJudgeClient {
	return execute.NewGoJudgeClient(string(url), logger)
}

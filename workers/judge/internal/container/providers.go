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
	"go-judge-system/workers/judge/internal/adapter/outbound/testcase"
	"go-judge-system/workers/judge/internal/application/port/inbound"
	"go-judge-system/workers/judge/internal/application/port/outbound"
	judgeuc "go-judge-system/workers/judge/internal/application/usecase/judge"

	judgepb "github.com/criyle/go-judge/pb"
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
)

var OutboundProviderSet = wire.NewSet(
	ProvideSandboxGRPCAddress,
	ProvideSandboxClientConn,
	ProvideSandboxExecutorClient,
	ProvideTestcaseCacheConfig,
	ProvideGoJudgeClient,
	wire.Bind(new(outbound.CodeExecutor), new(*execute.GoJudgeClient)),
	judge.NewKafkaResultPublisher,
	wire.Bind(new(outbound.ResultPublisher), new(*judge.KafkaResultPublisher)),
	ProvideProblemGRPCConfig,
	ProvideProblemClientConn,
	ProvideProblemServiceClient,
	ProvideProblemGRPCTimeout,
	problem.NewGRPCMetadataReader,
	wire.Bind(new(outbound.ProblemTestCaseMetadataReader), new(*problem.GRPCMetadataReader)),
	testcase.NewOfficialLoader,
	wire.Bind(new(outbound.OfficialTestCaseLoader), new(*testcase.OfficialLoader)),
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

func ProvideProblemGRPCConfig(cfg *config.Config) (config.ProblemGRPCConfig, error) {
	problemCfg := cfg.ProblemGRPC
	problemCfg.Address = strings.TrimSpace(problemCfg.Address)
	if problemCfg.Address == "" {
		problemCfg.Address = "problem-service:9092"
	}
	if problemCfg.Timeout == 0 {
		problemCfg.Timeout = 5 * time.Second
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

type SandboxGRPCAddress string

// SandboxClientConn gives Wire a distinct lifecycle-owned dependency while
// retaining the shared grpc.ClientConn implementation.
type SandboxClientConn struct {
	*googlegrpc.ClientConn
}

func ProvideSandboxGRPCAddress() SandboxGRPCAddress {
	return "judge_sandbox:5051"
}

func ProvideSandboxClientConn(address SandboxGRPCAddress) (*SandboxClientConn, error) {
	conn, err := sharedgrpc.NewClientConn(
		string(address),
		sharedgrpc.WithInsecureTransport(),
		sharedgrpc.WithDefaultCallOptions(googlegrpc.MaxCallRecvMsgSize(execute.SandboxGRPCMaxReceiveBytes)),
	)
	if err != nil {
		return nil, fmt.Errorf("create go-judge sandbox gRPC connection: %w", err)
	}
	return &SandboxClientConn{ClientConn: conn}, nil
}

func ProvideSandboxExecutorClient(conn *SandboxClientConn) judgepb.ExecutorClient {
	return judgepb.NewExecutorClient(conn.ClientConn)
}

func ProvideTestcaseCacheConfig(cfg *config.Config) (config.TestcaseCacheConfig, error) {
	cacheCfg := cfg.TestcaseCache
	if !cacheCfg.Enabled {
		return cacheCfg, nil
	}
	if cacheCfg.MaxBytes <= 0 {
		return config.TestcaseCacheConfig{}, fmt.Errorf("testcase cache max_bytes must be greater than zero when enabled")
	}
	if cacheCfg.MaxEntries <= 0 {
		return config.TestcaseCacheConfig{}, fmt.Errorf("testcase cache max_entries must be greater than zero when enabled")
	}
	if cacheCfg.IdleTTL < 0 {
		return config.TestcaseCacheConfig{}, fmt.Errorf("testcase cache idle_ttl must not be negative")
	}
	if cacheCfg.CleanupInterval <= 0 {
		return config.TestcaseCacheConfig{}, fmt.Errorf("testcase cache cleanup_interval must be greater than zero when enabled")
	}
	return cacheCfg, nil
}

func ProvideGoJudgeClient(client judgepb.ExecutorClient, logger *zap.Logger, cacheCfg config.TestcaseCacheConfig) *execute.GoJudgeClient {
	return execute.NewGoJudgeClient(client, logger, cacheCfg)
}

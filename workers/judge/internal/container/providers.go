package container

import (
	"go-judge-system/pkg/config"
	"go-judge-system/pkg/kafka"
	"go-judge-system/pkg/logger"
	kafkain "go-judge-system/workers/judge/internal/adapter/inbound/kafka"
	"go-judge-system/workers/judge/internal/adapter/outbound/execute"
	"go-judge-system/workers/judge/internal/adapter/outbound/judge"
	"go-judge-system/workers/judge/internal/adapter/outbound/problem"
	"go-judge-system/workers/judge/internal/application/port/inbound"
	"go-judge-system/workers/judge/internal/application/port/outbound"
	judgeuc "go-judge-system/workers/judge/internal/application/usecase/judge"

	"github.com/google/wire"
	"go.uber.org/zap"
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
	ProvideGoJudgeClient,
	wire.Bind(new(outbound.CodeExecutor), new(*execute.GoJudgeClient)),
	judge.NewKafkaResultPublisher,
	wire.Bind(new(outbound.ResultPublisher), new(*judge.KafkaResultPublisher)),
	ProvideProblemClient,
	wire.Bind(new(outbound.TestCaseFetcher), new(*problem.ProblemServiceClient)),
	ProvideProblemServiceURL,
	ProvideSandboxServiceURL,
)

var UseCaseProviderSet = wire.NewSet()

var InboundProviderSet = wire.NewSet(
	judgeuc.NewProcessJudgeJobUseCase,
	wire.Bind(new(inbound.ProcessJudgeJobUseCase), new(*judgeuc.ProcessJudgeJobUseCase)),
	kafkain.NewDLTPublisher,
	kafkain.NewJudgeJobConsumer,
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

// ProblemServiceURL is a custom type to prevent Wire injection conflicts with other strings
type ProblemServiceURL string

// ProvideProblemServiceURL provides the base URL for the problem service.
// This assumes the problem service is reachable at this internal Docker DNS.
func ProvideProblemServiceURL() ProblemServiceURL {
	return "http://judge_problem:8082" // Note: the docker-compose defined problem service correctly
}

func ProvideProblemClient(url ProblemServiceURL, logger *zap.Logger) *problem.ProblemServiceClient {
	return problem.NewProblemServiceClient(string(url), logger)
}

type SandboxServiceURL string

func ProvideSandboxServiceURL() SandboxServiceURL {
	return "http://judge_sandbox:5050"
}

func ProvideGoJudgeClient(url SandboxServiceURL, logger *zap.Logger) *execute.GoJudgeClient {
	return execute.NewGoJudgeClient(string(url), logger)
}

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
	execute.NewGoJudgeClient,
	wire.Bind(new(outbound.CodeExecutor), new(*execute.GoJudgeClient)),
	judge.NewKafkaResultPublisher,
	wire.Bind(new(outbound.ResultPublisher), new(*judge.KafkaResultPublisher)),
	problem.NewProblemServiceClient,
	wire.Bind(new(outbound.TestCaseFetcher), new(*problem.ProblemServiceClient)),
	ProvideProblemServiceURL,
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
	return "http://problem-service:8080"
}

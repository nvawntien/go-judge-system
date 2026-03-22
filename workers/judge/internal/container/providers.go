package container

import (
	"go-judge-system/pkg/config"
	"go-judge-system/pkg/kafka"
	"go-judge-system/pkg/logger"
	kafkain "go-judge-system/workers/judge/internal/adapter/inbound/kafka"
	"go-judge-system/workers/judge/internal/adapter/outbound/execute"
	"go-judge-system/workers/judge/internal/adapter/outbound/judge"
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
)

var UseCaseProviderSet = wire.NewSet()

var InboundProviderSet = wire.NewSet(
	judgeuc.NewProcessJudgeJobUseCase,
	wire.Bind(new(inbound.ProcessJudgeJobUseCase), new(*judgeuc.ProcessJudgeJobUseCase)),
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



package kafka

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go-judge-system/pkg/config"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

func NewSyncProducer(cfg config.KafkaConfig, logger *zap.Logger) (sarama.SyncProducer, error) {
	brokers := parseBrokers(cfg.Brokers)
	if len(brokers) == 0 {
		return nil, errors.New("kafka brokers are required")
	}

	saramaCfg := newBaseSaramaConfig()
	saramaCfg.Producer.RequiredAcks = sarama.WaitForAll
	saramaCfg.Producer.Retry.Max = 3
	saramaCfg.Producer.Retry.Backoff = 250 * time.Millisecond
	saramaCfg.Producer.Return.Successes = true
	saramaCfg.Producer.Idempotent = true
	saramaCfg.Net.MaxOpenRequests = 1

	producer, err := sarama.NewSyncProducer(brokers, saramaCfg)
	if err != nil {
		return nil, err
	}

	logger.Info("kafka sync producer initialized", zap.Strings("brokers", brokers))
	return producer, nil
}

func NewConsumerGroup(cfg config.KafkaConfig, logger *zap.Logger) (sarama.ConsumerGroup, error) {
	brokers := parseBrokers(cfg.Brokers)
	if len(brokers) == 0 {
		return nil, errors.New("kafka brokers are required")
	}

	group := strings.TrimSpace(cfg.ConsumerGroup)
	if group == "" {
		return nil, errors.New("kafka consumer group is required")
	}

	saramaCfg := newBaseSaramaConfig()
	saramaCfg.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	saramaCfg.Consumer.Return.Errors = true

	consumerGroup, err := sarama.NewConsumerGroup(brokers, group, saramaCfg)
	if err != nil {
		return nil, fmt.Errorf("create kafka consumer group: %w", err)
	}

	logger.Info(
		"kafka consumer group initialized",
		zap.Strings("brokers", brokers),
		zap.String("group", group),
	)

	return consumerGroup, nil
}

func newBaseSaramaConfig() *sarama.Config {
	saramaCfg := sarama.NewConfig()
	saramaCfg.Version = sarama.V3_7_0_0
	saramaCfg.Net.DialTimeout = 10 * time.Second
	saramaCfg.Net.ReadTimeout = 10 * time.Second
	saramaCfg.Net.WriteTimeout = 10 * time.Second
	return saramaCfg
}

func parseBrokers(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	brokers := make([]string, 0, len(parts))
	for _, part := range parts {
		broker := strings.TrimSpace(part)
		if broker != "" {
			brokers = append(brokers, broker)
		}
	}

	return brokers
}

package kafka

import (
	"errors"
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

	saramaCfg := sarama.NewConfig()
	saramaCfg.Version = sarama.V3_7_0_0
	saramaCfg.Producer.RequiredAcks = sarama.WaitForAll
	saramaCfg.Producer.Retry.Max = 3
	saramaCfg.Producer.Retry.Backoff = 250 * time.Millisecond
	saramaCfg.Producer.Return.Successes = true
	saramaCfg.Producer.Idempotent = true

	producer, err := sarama.NewSyncProducer(brokers, saramaCfg)
	if err != nil {
		return nil, err
	}

	logger.Info("kafka sync producer initialized", zap.Strings("brokers", brokers))
	return producer, nil
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

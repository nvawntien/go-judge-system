package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go-judge-system/pkg/config"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

// DLTPublisher publishes failed messages to the Dead Letter Topic (DLT)
// for manual inspection and potential reprocessing.
type DLTPublisher struct {
	producer sarama.SyncProducer
	topic    string
	logger   *zap.Logger
}

// DLTMessage wraps the original message with error metadata.
type DLTMessage struct {
	OriginalTopic     string          `json:"original_topic"`
	OriginalPartition int32           `json:"original_partition"`
	OriginalOffset    int64           `json:"original_offset"`
	OriginalKey       string          `json:"original_key"`
	OriginalValue     json.RawMessage `json:"original_value"`
	ErrorMessage      string          `json:"error_message"`
	RetryCount        int             `json:"retry_count"`
}

func NewDLTPublisher(
	producer sarama.SyncProducer,
	kafkaCfg config.KafkaConfig,
	logger *zap.Logger,
) *DLTPublisher {
	topic := strings.TrimSpace(kafkaCfg.DLTTopic)
	if topic == "" {
		topic = "judge.submission.dlt"
	}

	return &DLTPublisher{
		producer: producer,
		topic:    topic,
		logger:   logger,
	}
}

// Publish sends a failed message to the DLT with error context.
func (p *DLTPublisher) Publish(ctx context.Context, originalMsg *sarama.ConsumerMessage, errMsg string, retryCount int) error {
	dltPayload := DLTMessage{
		OriginalTopic:     originalMsg.Topic,
		OriginalPartition: originalMsg.Partition,
		OriginalOffset:    originalMsg.Offset,
		OriginalKey:       string(originalMsg.Key),
		OriginalValue:     originalMsg.Value,
		ErrorMessage:      errMsg,
		RetryCount:        retryCount,
	}

	value, err := json.Marshal(dltPayload)
	if err != nil {
		return fmt.Errorf("marshal DLT payload: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.ByteEncoder(originalMsg.Key),
		Value: sarama.ByteEncoder(value),
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("publish to DLT: %w", err)
	}

	p.logger.Warn(
		"message sent to dead letter topic",
		zap.String("dlt_topic", p.topic),
		zap.String("original_topic", originalMsg.Topic),
		zap.Int64("original_offset", originalMsg.Offset),
		zap.String("error", errMsg),
		zap.Int("retry_count", retryCount),
		zap.Int32("dlt_partition", partition),
		zap.Int64("dlt_offset", offset),
	)

	return nil
}

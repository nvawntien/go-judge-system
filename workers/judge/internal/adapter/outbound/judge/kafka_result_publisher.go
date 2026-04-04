package judge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go-judge-system/pkg/config"
	pkgjudge "go-judge-system/pkg/judge"
	"go-judge-system/workers/judge/internal/application/port/outbound"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type KafkaResultPublisher struct {
	producer sarama.SyncProducer
	topic    string
	logger   *zap.Logger
}

func NewKafkaResultPublisher(producer sarama.SyncProducer, kafkaCfg config.KafkaConfig, logger *zap.Logger) *KafkaResultPublisher {
	topic := strings.TrimSpace(kafkaCfg.ResultTopic)
	if topic == "" {
		topic = "judge.submission.results"
	}

	return &KafkaResultPublisher{
		producer: producer,
		topic:    topic,
		logger:   logger,
	}
}

func (p *KafkaResultPublisher) PublishResult(ctx context.Context, submissionID int64, attemptID string, result *outbound.ExecutionResult) error {
	// Map outbound.ExecutionResult → typed ResultMessage
	tcResults := make([]pkgjudge.TestCaseResultItem, 0, len(result.TestCases))
	for _, tc := range result.TestCases {
		tcResults = append(tcResults, pkgjudge.TestCaseResultItem{
			Index:          tc.Index,
			Status:         tc.Status,
			ActualOutput:   tc.ActualOutput,
			Input:          tc.Input,
			ExpectedOutput: tc.ExpectedOutput,
			ExecutionTime:  intPtr(tc.ExecutionTime),
			MemoryUsed:     intPtr(tc.MemoryUsed),
		})
	}

	payload := pkgjudge.ResultMessage{
		SubmissionID:  submissionID,
		AttemptID:     attemptID,
		Status:        result.Status,
		CompileOutput: result.CompileOutput,
		ExecutionTime: intPtr(result.ExecutionTime),
		MemoryUsed:    intPtr(result.MemoryUsed),
		Error:         result.Error,
		TestCases:     tcResults,
	}

	value, err := json.Marshal(payload)
	if err != nil {
		p.logger.Error("failed to marshal result payload", zap.Error(err))
		return fmt.Errorf("marshal judge result payload: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(fmt.Sprintf("%d", submissionID)),
		Value: sarama.ByteEncoder(value),
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		p.logger.Error(
			"failed to publish judge result",
			zap.Int64("submission_id", submissionID),
			zap.Error(err),
		)
		return fmt.Errorf("publish judge result message: %w", err)
	}

	p.logger.Info(
		"judge result published",
		zap.Int64("submission_id", submissionID),
		zap.String("attempt_id", attemptID),
		zap.String("status", result.Status),
		zap.Int32("partition", partition),
		zap.Int64("offset", offset),
	)

	return nil
}

// intPtr converts int to *int. Returns nil for zero values.
func intPtr(v int) *int {
	if v == 0 {
		return nil
	}
	return &v
}


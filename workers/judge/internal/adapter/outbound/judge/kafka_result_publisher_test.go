package judge

import (
	"context"
	"encoding/json"
	"testing"

	"go-judge-system/pkg/config"
	pkgjudge "go-judge-system/pkg/judge"
	"go-judge-system/workers/judge/internal/application/port/outbound"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type resultSyncProducer struct {
	message *sarama.ProducerMessage
}

func (p *resultSyncProducer) SendMessage(message *sarama.ProducerMessage) (int32, int64, error) {
	p.message = message
	return 0, 12, nil
}
func (*resultSyncProducer) SendMessages([]*sarama.ProducerMessage) error { return nil }
func (*resultSyncProducer) Close() error                                 { return nil }
func (*resultSyncProducer) TxnStatus() sarama.ProducerTxnStatusFlag      { return 0 }
func (*resultSyncProducer) IsTransactional() bool                        { return false }
func (*resultSyncProducer) BeginTxn() error                              { return nil }
func (*resultSyncProducer) CommitTxn() error                             { return nil }
func (*resultSyncProducer) AbortTxn() error                              { return nil }
func (*resultSyncProducer) AddOffsetsToTxn(map[string][]*sarama.PartitionOffsetMetadata, string) error {
	return nil
}
func (*resultSyncProducer) AddMessageToTxn(*sarama.ConsumerMessage, string, *string) error {
	return nil
}

func TestKafkaResultPublisherIncludesSanitizedErrorMessageAndAttemptID(t *testing.T) {
	producer := &resultSyncProducer{}
	publisher := NewKafkaResultPublisher(
		producer,
		config.KafkaConfig{ResultTopic: "judge.submission.results"},
		zap.NewNop(),
	)

	errMsg := "panic: runtime error: index out of range"
	err := publisher.PublishResult(context.Background(), 77, "attempt-77", &outbound.ExecutionResult{
		Status:       "RUNTIME_ERROR",
		ErrorMessage: &errMsg,
		TestCases: []outbound.TestCaseResult{{
			Index:  1,
			Status: "RUNTIME_ERROR",
		}},
	})
	if err != nil {
		t.Fatalf("PublishResult() error = %v", err)
	}
	if producer.message == nil {
		t.Fatal("producer message = nil")
	}

	value, err := producer.message.Value.Encode()
	if err != nil {
		t.Fatalf("encode message value: %v", err)
	}
	var payload pkgjudge.ResultMessage
	if err := json.Unmarshal(value, &payload); err != nil {
		t.Fatalf("decode result payload: %v", err)
	}

	if payload.SubmissionID != 77 || payload.AttemptID != "attempt-77" || payload.Status != "RUNTIME_ERROR" {
		t.Fatalf("payload identity/status = %+v", payload)
	}
	if payload.ErrorMessage == nil || *payload.ErrorMessage != errMsg {
		t.Fatalf("payload error_message = %v, want %q", payload.ErrorMessage, errMsg)
	}
	if len(payload.TestCases) != 1 || payload.TestCases[0].Input != nil || payload.TestCases[0].ExpectedOutput != nil {
		t.Fatalf("payload testcase leaked hidden fields: %+v", payload.TestCases)
	}
}

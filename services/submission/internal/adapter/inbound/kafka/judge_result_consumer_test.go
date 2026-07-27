package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	pkgjudge "go-judge-system/pkg/judge"
	"go-judge-system/services/submission/internal/domain"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type fakeApplyJudgeResultUseCase struct {
	err    error
	calls  int
	got    pkgjudge.ResultMessage
	called chan struct{}
}

func (u *fakeApplyJudgeResultUseCase) Execute(_ context.Context, result pkgjudge.ResultMessage) error {
	u.calls++
	u.got = result
	if u.called != nil {
		u.called <- struct{}{}
	}
	return u.err
}

type fakeConsumerGroupSession struct {
	ctx      context.Context
	marked   *sarama.ConsumerMessage
	metadata string
}

func (s *fakeConsumerGroupSession) Claims() map[string][]int32               { return nil }
func (s *fakeConsumerGroupSession) MemberID() string                         { return "member" }
func (s *fakeConsumerGroupSession) GenerationID() int32                      { return 1 }
func (s *fakeConsumerGroupSession) MarkOffset(string, int32, int64, string)  {}
func (s *fakeConsumerGroupSession) Commit()                                  {}
func (s *fakeConsumerGroupSession) ResetOffset(string, int32, int64, string) {}
func (s *fakeConsumerGroupSession) MarkMessage(msg *sarama.ConsumerMessage, metadata string) {
	s.marked = msg
	s.metadata = metadata
}
func (s *fakeConsumerGroupSession) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}

func resultMessage(t *testing.T, payload pkgjudge.ResultMessage) *sarama.ConsumerMessage {
	t.Helper()
	value, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	return &sarama.ConsumerMessage{Topic: "judge.submission.results", Partition: 0, Offset: 10, Value: value}
}

func TestJudgeResultHandlerValidMessageInvokesUseCaseAndMarksProcessed(t *testing.T) {
	uc := &fakeApplyJudgeResultUseCase{}
	session := &fakeConsumerGroupSession{}
	handler := &judgeResultHandler{useCase: uc, maxRetries: 3, logger: zap.NewNop()}

	handler.handleMessage(session, resultMessage(t, pkgjudge.ResultMessage{
		SubmissionID: 77,
		AttemptID:    "attempt-77",
		Status:       "ACCEPTED",
	}))

	if uc.calls != 1 || uc.got.SubmissionID != 77 || uc.got.AttemptID != "attempt-77" {
		t.Fatalf("usecase calls=%d payload=%+v", uc.calls, uc.got)
	}
	if session.marked == nil || session.metadata != "processed" {
		t.Fatalf("mark metadata = %q, want processed", session.metadata)
	}
}

func TestJudgeResultHandlerMalformedMessageIsMarkedDLT(t *testing.T) {
	uc := &fakeApplyJudgeResultUseCase{}
	session := &fakeConsumerGroupSession{}
	handler := &judgeResultHandler{useCase: uc, maxRetries: 3, logger: zap.NewNop()}

	handler.handleMessage(session, &sarama.ConsumerMessage{Topic: "judge.submission.results", Offset: 11, Value: []byte("{")})

	if uc.calls != 0 {
		t.Fatalf("usecase calls = %d, want 0", uc.calls)
	}
	if session.marked == nil || session.metadata != "invalid_payload_dlt" {
		t.Fatalf("mark metadata = %q, want invalid_payload_dlt", session.metadata)
	}
}

func TestJudgeResultHandlerNonRetryableErrorIsMarkedDLTWithoutRetry(t *testing.T) {
	uc := &fakeApplyJudgeResultUseCase{err: domain.ErrInvalidJudgeResult}
	session := &fakeConsumerGroupSession{}
	handler := &judgeResultHandler{useCase: uc, maxRetries: 3, logger: zap.NewNop()}

	handler.handleMessage(session, resultMessage(t, pkgjudge.ResultMessage{SubmissionID: 77, AttemptID: "attempt-77", Status: "BOGUS"}))

	if uc.calls != 1 {
		t.Fatalf("usecase calls = %d, want 1", uc.calls)
	}
	if session.marked == nil || session.metadata != "non_retryable_dlt" {
		t.Fatalf("mark metadata = %q, want non_retryable_dlt", session.metadata)
	}
}

func TestJudgeResultHandlerRetryableFailureExhaustsRetriesThenMarksDLT(t *testing.T) {
	uc := &fakeApplyJudgeResultUseCase{err: errors.New("database unavailable")}
	session := &fakeConsumerGroupSession{}
	handler := &judgeResultHandler{useCase: uc, maxRetries: 1, logger: zap.NewNop()}

	handler.handleMessage(session, resultMessage(t, pkgjudge.ResultMessage{SubmissionID: 77, AttemptID: "attempt-77", Status: "ACCEPTED"}))

	if uc.calls != 1 {
		t.Fatalf("usecase calls = %d, want 1", uc.calls)
	}
	if session.marked == nil || session.metadata != "dlt_forwarded" {
		t.Fatalf("mark metadata = %q, want dlt_forwarded", session.metadata)
	}
}

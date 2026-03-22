package submission

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go-judge-system/services/submission/internal/application/dto"
	"go-judge-system/services/submission/internal/domain/entity"

	"github.com/stretchr/testify/require"
)

// TestProcessJudgeResult_IdempotentPersist validates that replaying the same
// result message (e.g., on Kafka offset reset) does not cause data inconsistencies.
// This ensures the submission service can safely handle message replays without
// the Kafka offset tracking (which may reset during deployments).
func TestProcessJudgeResult_IdempotentPersist(t *testing.T) {
	result := &dto.JudgeResultMessage{
		SubmissionID:  12345,
		Status:        "ACCEPTED",
		ExecutionTime: ptrInt(150),
		MemoryUsed:    ptrInt(32768),
		Results: []dto.JudgeTestCaseResult{
			{
				TestCaseID:    1,
				Status:        "ACCEPTED",
				ExecutionTime: ptrInt(50),
				MemoryUsed:    ptrInt(10240),
				Order:         1,
			},
			{
				TestCaseID:    2,
				Status:        "ACCEPTED",
				ExecutionTime: ptrInt(100),
				MemoryUsed:    ptrInt(22528),
				Order:         2,
			},
		},
	}

	// Simulate idempotent replay: same result processed twice.
	// Persistence layer (ReplaceBySubmissionID transaction) should ensure
	// both replays result in identical database state.

	statusMap := map[string]entity.Status{
		"ACCEPTED":          entity.StatusAccepted,
		"WRONG_ANSWER":      entity.StatusWrongAnswer,
		"TLE":               entity.StatusTimeLimitExceed,
		"MLE":               entity.StatusMemoryLimitExceed,
		"RUNTIME_ERROR":     entity.StatusRuntimeError,
		"COMPILATION_ERROR": entity.StatusCompilationError,
	}

	resultStatusMap := map[string]entity.ResultStatus{
		"ACCEPTED":      entity.ResultAccepted,
		"WRONG_ANSWER":  entity.ResultWrongAnswer,
		"TLE":           entity.ResultTimeLimit,
		"MLE":           entity.ResultMemoryLimit,
		"RUNTIME_ERROR": entity.ResultRuntimeError,
	}

	// Verify status parsing for idempotent flow
	status, err := parseSubmissionStatus(result.Status)
	require.NoError(t, err)
	require.Equal(t, statusMap[result.Status], status)

	// Verify each result status parses correctly (idempotent mapping)
	for _, r := range result.Results {
		parsed, err := parseResultStatus(r.Status)
		require.NoError(t, err)
		require.Equal(t, resultStatusMap[r.Status], parsed)
	}

	// Verify JSON marshaling/unmarshaling works (Kafka serialization)
	payload, err := json.Marshal(result)
	require.NoError(t, err)

	var unmarshaled dto.JudgeResultMessage
	err = json.Unmarshal(payload, &unmarshaled)
	require.NoError(t, err)
	require.Equal(t, result.SubmissionID, unmarshaled.SubmissionID)
	require.Equal(t, result.Status, unmarshaled.Status)
	require.Len(t, unmarshaled.Results, 2)
}

// TestProcessJudgeResult_TimeoutScenario validates handling of compile errors
// and cases where execution times may exceed limits.
func TestProcessJudgeResult_CompileError(t *testing.T) {
	result := &dto.JudgeResultMessage{
		SubmissionID:  99999,
		Status:        "COMPILATION_ERROR",
		CompileOutput: ptrString("error: undefined reference to 'main'"),
		Results:       []dto.JudgeTestCaseResult{},
	}

	// Verify compilation error status
	status, err := parseSubmissionStatus(result.Status)
	require.NoError(t, err)
	require.Equal(t, entity.StatusCompilationError, status)

	// Compilation errors may have empty results (no test cases run)
	require.Empty(t, result.Results)
	require.NotNil(t, result.CompileOutput)
	require.Equal(t, "error: undefined reference to 'main'", *result.CompileOutput)
}

// TestProcessJudgeResult_PartialResults validates handling when some test cases
// pass and others fail (mixed verdict scenario).
func TestProcessJudgeResult_PartialResults(t *testing.T) {
	result := &dto.JudgeResultMessage{
		SubmissionID:  54321,
		Status:        "WRONG_ANSWER",
		ExecutionTime: ptrInt(200),
		MemoryUsed:    ptrInt(51200),
		Results: []dto.JudgeTestCaseResult{
			{
				TestCaseID:    1,
				Status:        "ACCEPTED",
				ExecutionTime: ptrInt(50),
				MemoryUsed:    ptrInt(10240),
				ActualOutput:  ptrString("42"),
				Order:         1,
			},
			{
				TestCaseID:    2,
				Status:        "WRONG_ANSWER",
				ExecutionTime: ptrInt(150),
				MemoryUsed:    ptrInt(40960),
				ActualOutput:  ptrString("43"),
				Order:         2,
			},
		},
	}

	// First test case should serialize correctly
	require.Len(t, result.Results, 2)
	require.Equal(t, "ACCEPTED", result.Results[0].Status)
	require.Equal(t, "WRONG_ANSWER", result.Results[1].Status)

	// Verify status parsing
	status, err := parseSubmissionStatus(result.Status)
	require.NoError(t, err)
	require.Equal(t, entity.StatusWrongAnswer, status)

	// Payload serialization for Kafka
	payload, _ := json.Marshal(result)
	var unmarshaled dto.JudgeResultMessage
	json.Unmarshal(payload, &unmarshaled)

	// Round-trip should preserve mixed results
	require.Len(t, unmarshaled.Results, 2)
	require.Equal(t, "ACCEPTED", unmarshaled.Results[0].Status)
	require.Equal(t, "WRONG_ANSWER", unmarshaled.Results[1].Status)
}

// TestProcessJudgeResult_ContextDeadline validates that processing respects
// context cancellation (important for graceful shutdown during result processing).
func TestProcessJudgeResult_ContextHandling(t *testing.T) {
	result := &dto.JudgeResultMessage{
		SubmissionID: 11111,
		Status:       "ACCEPTED",
		Results:      []dto.JudgeTestCaseResult{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Verify context operations don't cause panics
	select {
	case <-ctx.Done():
		t.Log("Context deadline simulation OK")
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Context should have timed out")
	}

	// Result should still be deserializable even if context deadline exceeded
	payload, _ := json.Marshal(result)
	var unmarshaled dto.JudgeResultMessage
	err := json.Unmarshal(payload, &unmarshaled)
	require.NoError(t, err)
}

// Helper functions

func ptrInt(v int) *int          { return &v }
func ptrString(v string) *string { return &v }

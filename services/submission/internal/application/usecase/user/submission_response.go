package user

import (
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain/entity"
)

func testcaseSummaryForStatus(
	status entity.Status,
	summary outbound.SubmissionResultSummary,
	found bool,
) (*int, *int) {
	if found {
		passed := summary.Passed
		total := summary.Total
		return &passed, &total
	}

	if isTerminalSubmissionStatus(status) {
		zero := 0
		return &zero, &zero
	}
	return nil, nil
}

func isTerminalSubmissionStatus(status entity.Status) bool {
	switch status {
	case entity.StatusAccepted,
		entity.StatusWrongAnswer,
		entity.StatusTimeLimitExceed,
		entity.StatusMemoryLimitExceed,
		entity.StatusRuntimeError,
		entity.StatusCompilationError,
		entity.StatusSystemError:
		return true
	default:
		return false
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

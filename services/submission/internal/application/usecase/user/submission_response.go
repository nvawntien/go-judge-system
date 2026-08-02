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

	if entity.IsTerminalStatus(status) {
		zero := 0
		return &zero, &zero
	}
	return nil, nil
}

// Package report renders a human-readable view of the machine-readable run
// artifacts. It intentionally contains only aggregate, non-secret data.
package report

import (
	"fmt"
	"strconv"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/model"
)

func Markdown(summary model.RunSummary) string {
	return fmt.Sprintf(`# Judge benchmark %s

Classification: **%s**

## Load window

- Intended / attempted / accepted: %d / %d / %d
- HTTP-201 accepted after load stop: %d
- Terminal completions during load: %d
- Client outstanding at load end: %d
- Burst POST-start spread: %v ms
- Attempted / accepted / completion rate: %.3f / %.3f / %.3f per second

## Drain

- Outstanding at start: %d
- Terminal completions: %d
- Remaining: %d
- Duration: %d ms

## Quality

%v
`, summary.RunID, summary.Classification,
		summary.Counts.Intended, summary.Counts.Attempted, summary.LoadWindow.Accepted, summary.LoadWindow.BoundaryAcceptedAfterLoad,
		summary.LoadWindow.Completed, summary.LoadWindow.OutstandingAtEnd, optionalInt(summary.LoadWindow.BurstSpreadMS),
		summary.Rates.LoadAttemptedPerSecond, summary.Rates.LoadAcceptedPerSecond, summary.Rates.LoadTerminalCompletionSecond,
		summary.Drain.OutstandingAtStart, summary.Drain.Completed, summary.Drain.Remaining, summary.Drain.DurationMS,
		summary.QualityFlags)
}

func optionalInt(value *int64) string {
	if value == nil {
		return "n/a"
	}
	return strconv.FormatInt(*value, 10)
}

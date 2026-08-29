// Package result writes immutable, local-only benchmark records with restrictive
// permissions. It refuses to overwrite a prior run directory.
package result

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/model"
)

type Writer struct{ Dir string }

func Create(root, runID string) (*Writer, error) {
	if root == "" || !safeRunID(runID) {
		return nil, errors.New("invalid result root or run ID")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create result root: %w", err)
	}
	if err := os.Chmod(root, 0o700); err != nil {
		return nil, fmt.Errorf("secure result root: %w", err)
	}
	dir := filepath.Join(root, runID)
	if err := os.Mkdir(dir, 0o700); err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("result directory already exists: %s", dir)
		}
		return nil, fmt.Errorf("create run directory: %w", err)
	}
	return &Writer{Dir: dir}, nil
}

func safeRunID(runID string) bool {
	if runID == "" || filepath.IsAbs(runID) || runID == "." || strings.Contains(runID, "..") || strings.ContainsAny(runID, `/\\`) || filepath.Base(runID) != runID {
		return false
	}
	for _, value := range runID {
		if value < 0x20 || value == 0x7f {
			return false
		}
	}
	return true
}

func (w *Writer) WriteRun(value model.RunMetadata) error { return w.writeJSON("run.json", value) }
func (w *Writer) WriteSummary(value model.RunSummary) error {
	return w.writeJSON("summary.json", value)
}

func (w *Writer) WriteReport(markdown string) error {
	return w.writeAtomic("report.md", []byte(markdown))
}

func (w *Writer) WriteSubmissions(values []model.SubmissionRecord) error {
	rows := [][]string{{"run_id", "phase", "sequence", "user_alias", "intended_at", "intended_offset_ms", "post_started_at", "post_start_offset_ms", "post_completed_at", "post_completed_offset_ms", "schedule_delay_ms", "submit_latency_ms", "attempted", "accepted", "http_status", "http_protocol", "api_code", "submission_id", "initial_status", "terminal_status", "terminal_observed_at", "terminal_offset_ms", "end_to_end_latency_ms", "accepted_to_terminal_ms", "observer_started_at", "observer_ended_at", "completion_source", "sse_connections", "sse_failures", "get_reconciliations", "ticket_started_at", "ticket_completed_at", "ticket_latency_ms", "ticket_attempted", "ticket_succeeded", "sse_started_at", "sse_established_at", "sse_closed_at", "sse_establishment_latency_ms", "sse_attempted", "sse_established", "sse_close_reason", "sse_terminal_during_hold", "sse_survived_full_hold", "rate_limited", "retry_after_ms", "outcome", "error_class", "error"}}
	for _, value := range values {
		rows = append(rows, []string{csvText(value.RunID), csvText(string(value.Phase)), integer(value.Sequence), csvText(value.UserAlias), timestamp(value.IntendedAt), int64Ptr(value.IntendedOffsetMS), timestamp(value.PostStartedAt), int64Ptr(value.PostStartOffsetMS), timestamp(value.PostCompletedAt), int64Ptr(value.PostCompletedOffsetMS), floatPtr(value.ScheduleDelayMS), floatPtr(value.SubmitLatencyMS), boolean(value.Attempted), boolean(value.Accepted), integerPtr(value.HTTPStatus), csvText(value.HTTPProtocol), integerPtr(value.APICode), int64Ptr(value.SubmissionID), csvText(value.InitialStatus), csvText(value.TerminalStatus), timestamp(value.TerminalObservedAt), int64Ptr(value.TerminalOffsetMS), floatPtr(value.EndToEndLatencyMS), floatPtr(value.AcceptedToTerminalMS), timestamp(value.ObserverStartedAt), timestamp(value.ObserverEndedAt), csvText(string(value.CompletionSource)), integer(value.SSEConnections), integer(value.SSEFailures), integer(value.GETReconciliations), timestamp(value.TicketStartedAt), timestamp(value.TicketCompletedAt), floatPtr(value.TicketLatencyMS), boolean(value.TicketAttempted), boolean(value.TicketSucceeded), timestamp(value.SSEStartedAt), timestamp(value.SSEEstablishedAt), timestamp(value.SSEClosedAt), floatPtr(value.SSEEstablishLatencyMS), boolean(value.SSEAttempted), boolean(value.SSEEstablished), csvText(value.SSECloseReason), boolean(value.SSETerminalDuringHold), boolean(value.SSESurvivedFullHold), boolean(value.RateLimited), int64Ptr(value.RetryAfterMS), csvText(string(value.Outcome)), csvText(value.ErrorClass), csvText(value.Error)})
	}
	return w.writeCSV("submissions.csv", rows)
}

func (w *Writer) WriteClientResources(values []model.ClientResourceSample) error {
	rows := [][]string{{"timestamp", "open_fds", "goroutines", "active_posts", "active_tickets", "active_sse_streams"}}
	for _, value := range values {
		rows = append(rows, []string{value.At.UTC().Format(time.RFC3339Nano), integer(value.OpenFDs), integer(value.Goroutines), integer(value.ActivePosts), integer(value.ActiveTickets), integer(value.ActiveSSEStreams)})
	}
	return w.writeCSV("client_resources.csv", rows)
}

func (w *Writer) WriteWindows(values []model.WindowRecord) error {
	rows := [][]string{{"run_id", "phase", "window_index", "window_start", "window_end", "window_start_offset_ms", "window_end_offset_ms", "window_duration_ms", "intended", "attempted", "accepted", "completed", "intended_cumulative", "attempted_cumulative", "accepted_cumulative", "completed_cumulative", "client_outstanding_start", "client_outstanding", "client_outstanding_peak", "target_arrival_rate_per_sec", "attempted_rate_per_sec", "accepted_rate_per_sec", "completion_rate_per_sec", "schedule_delay_p50_ms", "schedule_delay_p95_ms", "schedule_delay_p99_ms", "schedule_delay_max_ms", "submit_latency_p50_ms", "submit_latency_p95_ms", "e2e_latency_p50_ms", "e2e_latency_p95_ms", "rate_limited", "other_4xx", "server_errors", "transport_failures", "completion_timeouts", "active_observers_end", "peak_in_flight"}}
	for _, value := range values {
		rows = append(rows, []string{csvText(value.RunID), csvText(string(value.Phase)), integer(value.WindowIndex), value.WindowStart.UTC().Format(time.RFC3339Nano), value.WindowEnd.UTC().Format(time.RFC3339Nano), integer64(value.WindowStartOffsetMS), integer64(value.WindowEndOffsetMS), integer64(value.WindowDurationMS), integer(value.Intended), integer(value.Attempted), integer(value.Accepted), integer(value.Completed), integer(value.IntendedCumulative), integer(value.AttemptedCumulative), integer(value.AcceptedCumulative), integer(value.CompletedCumulative), integer(value.ClientOutstandingStart), integer(value.ClientOutstanding), integer(value.ClientOutstandingPeak), floatPtr(value.TargetRatePerSecond), float64Value(value.AttemptedRatePerSecond), float64Value(value.AcceptedRatePerSecond), float64Value(value.CompletionRatePerSecond), floatPtr(value.ScheduleDelayP50MS), floatPtr(value.ScheduleDelayP95MS), floatPtr(value.ScheduleDelayP99MS), floatPtr(value.ScheduleDelayMaxMS), floatPtr(value.SubmitLatencyP50MS), floatPtr(value.SubmitLatencyP95MS), floatPtr(value.E2ELatencyP50MS), floatPtr(value.E2ELatencyP95MS), integer(value.RateLimited), integer(value.Other4xx), integer(value.ServerErrors), integer(value.TransportFailures), integer(value.CompletionTimeouts), integer(value.ActiveObserversEnd), integer(value.PeakInFlight)})
	}
	return w.writeCSV("windows.csv", rows)
}

func (w *Writer) writeJSON(name string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return w.writeAtomic(name, append(data, '\n'))
}

func (w *Writer) writeCSV(name string, rows [][]string) error {
	var builder strings.Builder
	writer := csv.NewWriter(&builder)
	if err := writer.WriteAll(rows); err != nil {
		return err
	}
	return w.writeAtomic(name, []byte(builder.String()))
}

func (w *Writer) writeAtomic(name string, data []byte) error {
	path := filepath.Join(w.Dir, name)
	file, err := os.CreateTemp(w.Dir, "."+name+"-*")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

type Redactor struct{ secrets []string }

func NewRedactor(secrets ...string) Redactor {
	filtered := make([]string, 0, len(secrets))
	for _, secret := range secrets {
		if secret != "" {
			filtered = append(filtered, secret)
		}
	}
	return Redactor{secrets: filtered}
}

func (r Redactor) Sanitize(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	for _, secret := range r.secrets {
		value = strings.ReplaceAll(value, secret, "[REDACTED]")
	}
	value = strings.TrimSpace(value)
	if len(value) > 256 {
		value = value[:256]
	}
	return value
}

func timestamp(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
func boolean(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
func integer(value int) string     { return fmt.Sprintf("%d", value) }
func integer64(value int64) string { return fmt.Sprintf("%d", value) }
func integerPtr(value *int) string {
	if value == nil {
		return ""
	}
	return integer(*value)
}
func int64Ptr(value *int64) string {
	if value == nil {
		return ""
	}
	return integer64(*value)
}
func float64Value(value float64) string { return fmt.Sprintf("%.6f", value) }
func floatPtr(value *float64) string {
	if value == nil {
		return ""
	}
	return float64Value(*value)
}
func csvText(value string) string {
	if value != "" && strings.ContainsRune("=+-@", rune(value[0])) {
		return "'" + value
	}
	return value
}

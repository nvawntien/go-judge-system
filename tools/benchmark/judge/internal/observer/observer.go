// Package observer tracks one accepted submission. SSE is primary; authenticated
// GET is reconciliation only and is globally rate limited when configured.
package observer

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/client"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/model"
)

type Config struct {
	ConnectTimeout          time.Duration
	IdleTimeout             time.Duration
	SubmissionTimeout       time.Duration
	MaxReconnects           int
	BackoffBase             time.Duration
	BackoffMax              time.Duration
	SafetyReconcileInterval time.Duration
}

const maxSSEEventBytes = 1 << 20

type Result struct {
	TerminalStatus     string
	ObservedAt         time.Time
	Source             model.CompletionSource
	SSEConnections     int
	SSEFailures        int
	GETReconciliations int
	TimedOut           bool
	AuthFailure        bool
}

type event struct {
	SubmissionID int64  `json:"submission_id"`
	AttemptID    string `json:"attempt_id"`
	Status       string `json:"status"`
	UpdatedAt    string `json:"updated_at"`
}

type streamItem struct {
	// Activity means the SSE transport made forward progress and the observer
	// should reset its idle timer. Event is non-nil only after a complete SSE
	// event is dispatched by a blank line; heartbeats and id fields never carry
	// submission state.
	Event     *event
	EventName string
	Activity  bool
	End       bool
	Err       error
}

// ReconcileLimiter is shared by all observers in one benchmark run. It keeps
// optional safety GET work below the configured global QPS ceiling.
type ReconcileLimiter struct {
	ch   <-chan time.Time
	stop func()
}

func NewReconcileLimiter(qps float64) *ReconcileLimiter {
	if qps <= 0 {
		return &ReconcileLimiter{ch: nil, stop: func() {}}
	}
	interval := time.Duration(float64(time.Second) / qps)
	ticker := time.NewTicker(interval)
	return &ReconcileLimiter{ch: ticker.C, stop: ticker.Stop}
}

func (l *ReconcileLimiter) Close() { l.stop() }

func (l *ReconcileLimiter) Wait(ctx context.Context) error {
	if l == nil || l.ch == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-l.ch:
		return nil
	}
}

func Observe(ctx context.Context, api *client.API, submissionID int64, cfg Config, limiter *ReconcileLimiter) Result {
	deadline, cancel := context.WithTimeout(ctx, cfg.SubmissionTimeout)
	defer cancel()
	result := Result{}
	reconnects := 0
	for {
		if deadline.Err() != nil {
			result.TimedOut = true
			return result
		}
		ticketContext, ticketCancel := context.WithTimeout(deadline, cfg.ConnectTimeout)
		ticket, ticketErr := api.IssueTicket(ticketContext, submissionID)
		ticketCancel()
		if ticketErr == nil {
			streamContext, streamCancel := context.WithCancel(deadline)
			response, openErr := api.OpenEvents(streamContext, submissionID, ticket.Value)
			if openErr == nil {
				result.SSEConnections++
				terminal, status, source, streamErr := consumeStream(streamContext, streamCancel, response.Body, api, submissionID, cfg, limiter, &result)
				streamCancel()
				if terminal {
					result.TerminalStatus, result.Source, result.ObservedAt = status, source, time.Now()
					return result
				}
				if streamErr != nil {
					result.SSEFailures++
				}
			} else {
				streamCancel()
				result.SSEFailures++
			}
		} else {
			result.SSEFailures++
		}

		terminal, status, source, authFailure, reconcileErr := reconcile(deadline, api, submissionID, limiter, model.CompletionGETReconcile)
		result.GETReconciliations++
		if authFailure {
			result.AuthFailure = true
		}
		if reconcileErr == nil && terminal {
			result.TerminalStatus, result.Source, result.ObservedAt = status, source, time.Now()
			return result
		}
		if reconnects >= cfg.MaxReconnects {
			return degraded(deadline, api, submissionID, cfg, limiter, result)
		}
		if !sleep(deadline, backoff(cfg, reconnects)) {
			result.TimedOut = true
			return result
		}
		reconnects++
	}
}

func consumeStream(ctx context.Context, cancel context.CancelFunc, body io.ReadCloser, api *client.API, submissionID int64, cfg Config, limiter *ReconcileLimiter, result *Result) (bool, string, model.CompletionSource, error) {
	defer body.Close()
	items := make(chan streamItem, 8)
	go parseStream(ctx, body, items)
	idle := time.NewTimer(cfg.IdleTimeout)
	defer idle.Stop()
	var safety <-chan time.Time
	var safetyTimer *time.Timer
	if cfg.SafetyReconcileInterval > 0 {
		offset := stagger(submissionID, cfg.SafetyReconcileInterval)
		safetyTimer = time.NewTimer(offset)
		safety = safetyTimer.C
		defer safetyTimer.Stop()
	}
	boundAttempt := ""
	for {
		select {
		case <-ctx.Done():
			return false, "", "", ctx.Err()
		case <-idle.C:
			cancel()
			return false, "", "", errors.New("SSE idle timeout")
		case <-safety:
			terminal, status, source, authFailure, err := reconcile(ctx, api, submissionID, limiter, model.CompletionGETSafety)
			result.GETReconciliations++
			if authFailure {
				result.AuthFailure = true
			}
			if err == nil && terminal {
				cancel()
				return true, status, source, nil
			}
			safetyTimer.Reset(cfg.SafetyReconcileInterval)
		case item, ok := <-items:
			if !ok || item.End {
				if item.Err != nil {
					return false, "", "", item.Err
				}
				return false, "", "", errors.New("SSE stream closed before terminal event")
			}
			if item.Activity {
				if !idle.Stop() {
					select {
					case <-idle.C:
					default:
					}
				}
				idle.Reset(cfg.IdleTimeout)
			}
			if item.Event == nil {
				continue
			}
			if item.Event.SubmissionID != submissionID || item.Event.AttemptID == "" {
				continue
			}
			if boundAttempt == "" {
				boundAttempt = item.Event.AttemptID
			}
			if item.Event.AttemptID != boundAttempt {
				continue
			}
			if isTerminal(item.Event.Status) {
				source := model.CompletionSSEEvent
				if item.EventName == "submission.snapshot" {
					source = model.CompletionSSESnapshot
				}
				return true, item.Event.Status, source, nil
			}
		}
	}
}

func parseStream(ctx context.Context, body io.Reader, items chan<- streamItem) {
	defer close(items)
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024), 1<<20)
	var eventName string
	var data stringsBuilder
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" {
			if data.String() != "" {
				var decoded event
				if err := json.Unmarshal([]byte(data.String()), &decoded); err != nil {
					sendItem(ctx, items, streamItem{Err: fmt.Errorf("decode SSE event: %w", err), End: true})
					return
				}
				if !sendItem(ctx, items, streamItem{Event: &decoded, EventName: eventName, Activity: true}) {
					return
				}
			} else {
				if !sendItem(ctx, items, streamItem{Activity: true}) {
					return
				}
			}
			eventName = ""
			data.Reset()
			continue
		}
		if len(line) > 0 && line[0] == ':' {
			if !sendItem(ctx, items, streamItem{Activity: true}) {
				return
			}
			continue
		}
		if after, ok := trimPrefix(line, "event:"); ok {
			eventName = after
			continue
		}
		if _, ok := trimPrefix(line, "id:"); ok {
			// An id field is stream activity but not application event data.
			if !sendItem(ctx, items, streamItem{Activity: true}) {
				return
			}
			continue
		}
		if after, ok := trimPrefix(line, "data:"); ok {
			if !data.WriteLine(after) {
				sendItem(ctx, items, streamItem{Err: errors.New("SSE event exceeds size limit"), End: true})
				return
			}
			continue
		}
	}
	sendItem(ctx, items, streamItem{End: true, Err: scanner.Err()})
}

func sendItem(ctx context.Context, items chan<- streamItem, item streamItem) bool {
	select {
	case <-ctx.Done():
		return false
	case items <- item:
		return true
	}
}

// stringsBuilder avoids importing strings for one tiny incremental buffer.
type stringsBuilder struct{ bytes []byte }

func (b *stringsBuilder) WriteLine(value string) bool {
	additional := len(value)
	if len(b.bytes) > 0 {
		additional++
	}
	if len(b.bytes)+additional > maxSSEEventBytes {
		return false
	}
	if len(b.bytes) > 0 {
		b.bytes = append(b.bytes, '\n')
	}
	b.bytes = append(b.bytes, value...)
	return true
}
func (b *stringsBuilder) String() string { return string(b.bytes) }
func (b *stringsBuilder) Reset()         { b.bytes = b.bytes[:0] }

func trimPrefix(value, prefix string) (string, bool) {
	if len(value) < len(prefix) || value[:len(prefix)] != prefix {
		return "", false
	}
	result := value[len(prefix):]
	if len(result) > 0 && result[0] == ' ' {
		result = result[1:]
	}
	return result, true
}

func reconcile(ctx context.Context, api *client.API, submissionID int64, limiter *ReconcileLimiter, source model.CompletionSource) (bool, string, model.CompletionSource, bool, error) {
	if err := limiter.Wait(ctx); err != nil {
		return false, "", "", false, err
	}
	detail, err := api.GetSubmission(ctx, submissionID)
	if err != nil {
		return false, "", "", isAuthError(err), err
	}
	if isTerminal(detail.Status) {
		return true, detail.Status, source, false, nil
	}
	return false, detail.Status, source, false, nil
}

func degraded(ctx context.Context, api *client.API, submissionID int64, cfg Config, limiter *ReconcileLimiter, result Result) Result {
	interval := cfg.SafetyReconcileInterval
	if interval <= 0 {
		interval = 60 * time.Second
	}
	for {
		if !sleep(ctx, interval) {
			result.TimedOut = true
			return result
		}
		terminal, status, source, authFailure, err := reconcile(ctx, api, submissionID, limiter, model.CompletionGETFallback)
		result.GETReconciliations++
		if authFailure {
			result.AuthFailure = true
		}
		if err == nil && terminal {
			result.TerminalStatus, result.Source, result.ObservedAt = status, source, time.Now()
			return result
		}
	}
}

func sleep(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func backoff(cfg Config, attempt int) time.Duration {
	delay := cfg.BackoffBase
	for i := 0; i < attempt && delay < cfg.BackoffMax; i++ {
		delay *= 2
	}
	if delay > cfg.BackoffMax {
		return cfg.BackoffMax
	}
	return delay
}

func stagger(submissionID int64, interval time.Duration) time.Duration {
	if interval <= 0 {
		return 0
	}
	return time.Duration(uint64(submissionID) % uint64(interval))
}

func isTerminal(status string) bool {
	switch status {
	case "ACCEPTED", "WRONG_ANSWER", "TIME_LIMIT_EXCEEDED", "MEMORY_LIMIT_EXCEEDED", "OUTPUT_LIMIT_EXCEEDED", "RUNTIME_ERROR", "COMPILATION_ERROR", "SYSTEM_ERROR":
		return true
	default:
		return false
	}
}

func isAuthError(err error) bool {
	// Public client errors intentionally omit response bodies. Treating only an
	// explicit 401 string as auth degradation is conservative and reportable.
	return err != nil && (contains(err.Error(), "HTTP 401") || contains(err.Error(), "code 401"))
}

func contains(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

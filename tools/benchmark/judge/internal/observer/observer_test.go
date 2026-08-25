package observer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/client"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/credentials"
	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/model"
)

func observerAPI(t *testing.T, handler http.Handler) *client.API {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	base, _ := url.Parse(server.URL)
	sessions, err := credentials.NewSessions(credentials.File{SchemaVersion: 1, Users: []credentials.User{{Alias: "bench", Cookies: credentials.Cookies{AccessToken: "x"}}}}, base, http.DefaultTransport)
	if err != nil {
		t.Fatal(err)
	}
	api, err := client.New(base, sessions[0])
	if err != nil {
		t.Fatal(err)
	}
	return api
}

func cfg() Config {
	return Config{ConnectTimeout: time.Second, IdleTimeout: 100 * time.Millisecond, SubmissionTimeout: time.Second, MaxReconnects: 1, BackoffBase: time.Millisecond, BackoffMax: 2 * time.Millisecond, SafetyReconcileInterval: 0}
}

func TestObserveCompletesFromTerminalSnapshot(t *testing.T) {
	api := observerAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/submissions/7/events/ticket":
			_, _ = w.Write([]byte(`{"status":"success","code":20000,"data":{"ticket":"ticket-secret"}}`))
		case "/events/submissions/7":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprint(w, "event: submission.snapshot\ndata: {\"submission_id\":7,\"attempt_id\":\"attempt\",\"status\":\"ACCEPTED\"}\n\n")
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	got := Observe(context.Background(), api, 7, cfg(), nil)
	if got.TerminalStatus != "ACCEPTED" || got.Source != model.CompletionSSESnapshot || got.GETReconciliations != 0 {
		t.Fatalf("result=%+v", got)
	}
}

func TestObserveDisconnectsThenReconcilesTerminal(t *testing.T) {
	var gets atomic.Int32
	api := observerAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/submissions/7/events/ticket":
			_, _ = w.Write([]byte(`{"status":"success","code":20000,"data":{"ticket":"x"}}`))
		case "/events/submissions/7":
			w.Header().Set("Content-Type", "text/event-stream")
		case "/api/v1/submissions/7":
			gets.Add(1)
			_, _ = w.Write([]byte(`{"status":"success","code":20000,"data":{"id":7,"status":"ACCEPTED"}}`))
		default:
			t.Fatalf("unexpected %s", r.URL.Path)
		}
	}))
	got := Observe(context.Background(), api, 7, cfg(), nil)
	if got.TerminalStatus != "ACCEPTED" || got.Source != model.CompletionGETReconcile || gets.Load() != 1 {
		t.Fatalf("result=%+v gets=%d", got, gets.Load())
	}
}

func TestObserveReconnectsWithFreshTicket(t *testing.T) {
	var tickets atomic.Int32
	api := observerAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/submissions/7/events/ticket":
			value := tickets.Add(1)
			_, _ = fmt.Fprintf(w, `{"status":"success","code":20000,"data":{"ticket":"%d"}}`, value)
		case "/events/submissions/7":
			w.Header().Set("Content-Type", "text/event-stream")
			if r.URL.Query().Get("ticket") == "2" {
				_, _ = fmt.Fprint(w, "event: submission.completed\ndata: {\"submission_id\":7,\"attempt_id\":\"a\",\"status\":\"ACCEPTED\"}\n\n")
			}
		case "/api/v1/submissions/7":
			_, _ = w.Write([]byte(`{"status":"success","code":20000,"data":{"id":7,"status":"PENDING"}}`))
		default:
			t.Fatalf("unexpected %s", r.URL.Path)
		}
	}))
	got := Observe(context.Background(), api, 7, cfg(), nil)
	if got.TerminalStatus != "ACCEPTED" || tickets.Load() != 2 {
		t.Fatalf("result=%+v tickets=%d", got, tickets.Load())
	}
}

func TestObserveHeartbeatKeepsStreamAliveUntilTerminalEvent(t *testing.T) {
	api := observerAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/submissions/7/events/ticket":
			_, _ = w.Write([]byte(`{"status":"success","code":20000,"data":{"ticket":"x"}}`))
		case "/events/submissions/7":
			w.Header().Set("Content-Type", "text/event-stream")
			flusher := w.(http.Flusher)
			_, _ = fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
			time.Sleep(20 * time.Millisecond)
			_, _ = fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
			time.Sleep(20 * time.Millisecond)
			_, _ = fmt.Fprint(w, "event: submission.completed\ndata: {\"submission_id\":7,\"attempt_id\":\"a\",\"status\":\"ACCEPTED\"}\n\n")
		default:
			t.Fatalf("unexpected %s", r.URL.Path)
		}
	}))
	c := cfg()
	c.IdleTimeout = 50 * time.Millisecond
	got := Observe(context.Background(), api, 7, c, nil)
	if got.TerminalStatus != "ACCEPTED" || got.SSEFailures != 0 {
		t.Fatalf("result=%+v", got)
	}
}

func TestObserveReconnectExhaustionUsesBoundedFallback(t *testing.T) {
	var tickets atomic.Int32
	api := observerAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/submissions/7/events/ticket":
			tickets.Add(1)
			_, _ = w.Write([]byte(`{"status":"success","code":20000,"data":{"ticket":"x"}}`))
		case "/events/submissions/7":
			w.Header().Set("Content-Type", "text/event-stream")
		case "/api/v1/submissions/7":
			_, _ = w.Write([]byte(`{"status":"success","code":20000,"data":{"id":7,"status":"PENDING"}}`))
		default:
			t.Fatalf("unexpected %s", r.URL.Path)
		}
	}))
	c := cfg()
	c.SubmissionTimeout = 20 * time.Millisecond
	c.MaxReconnects = 1
	c.SafetyReconcileInterval = time.Millisecond
	got := Observe(context.Background(), api, 7, c, nil)
	if !got.TimedOut || tickets.Load() != 2 {
		t.Fatalf("result=%+v tickets=%d", got, tickets.Load())
	}
}

func TestParseStreamSupportsCRLFIDAndMultipleDataLines(t *testing.T) {
	items := parseItems(t, "id: 1\r\nevent: submission.completed\r\ndata: {\"submission_id\":7,\r\ndata: \"attempt_id\":\"a\",\"status\":\"ACCEPTED\"}\r\n\r\n")
	if len(items) < 2 || !items[0].Activity || items[0].Event != nil {
		t.Fatalf("id field must be activity without an event: items=%+v", items)
	}
	item := dispatchedEvent(t, items)
	if item.Event == nil || item.Event.SubmissionID != 7 || item.Event.Status != "ACCEPTED" || item.EventName != "submission.completed" {
		t.Fatalf("item=%+v", item)
	}
}

func TestParseStreamHeartbeatIsActivityNotSubmissionEvent(t *testing.T) {
	items := parseItems(t, ": heartbeat\n\n")
	activity, events := 0, 0
	for _, item := range items {
		if item.Activity {
			activity++
		}
		if item.Event != nil {
			events++
		}
	}
	if activity == 0 || events != 0 {
		t.Fatalf("heartbeat items=%+v", items)
	}
}

func TestParseStreamHeartbeatDoesNotConsumeFollowingEvent(t *testing.T) {
	items := parseItems(t, ": heartbeat\n\nevent: submission.completed\ndata: {\"submission_id\":7,\"attempt_id\":\"a\",\"status\":\"ACCEPTED\"}\n\n")
	event := dispatchedEvent(t, items)
	if event.EventName != "submission.completed" || event.Event.Status != "ACCEPTED" {
		t.Fatalf("event=%+v items=%+v", event, items)
	}
	events := 0
	for _, item := range items {
		if item.Event != nil {
			events++
		}
	}
	if events != 1 {
		t.Fatalf("heartbeat duplicated or consumed event: items=%+v", items)
	}
}

func TestParseStreamRejectsMalformedAndOversizedEvents(t *testing.T) {
	for _, raw := range []string{
		"event: submission.completed\ndata: not-json\n\n",
		"data: " + string(bytes.Repeat([]byte("x"), maxSSEEventBytes+1)) + "\n\n",
	} {
		items := parseItems(t, raw)
		last := items[len(items)-1]
		if !last.End || last.Err == nil {
			t.Fatalf("malformed stream did not return bounded failure: items=%+v", items)
		}
	}
}

func TestConsumeStreamCancellationClosesBodyAndUnblocksParser(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	body := &blockingBody{closed: make(chan struct{})}
	done := make(chan struct{})
	go func() {
		_, _, _, _ = consumeStream(ctx, func() {}, body, nil, 7, cfg(), nil, &Result{})
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("consumeStream did not return after cancellation")
	}
	select {
	case <-body.closed:
	case <-time.After(time.Second):
		t.Fatal("cancellation did not close SSE body")
	}
}

func parseItems(t *testing.T, raw string) []streamItem {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	items := make(chan streamItem, 8)
	go parseStream(ctx, bytes.NewBufferString(raw), items)
	var values []streamItem
	for item := range items {
		values = append(values, item)
	}
	return values
}

func dispatchedEvent(t *testing.T, items []streamItem) streamItem {
	t.Helper()
	for _, item := range items {
		if item.Event != nil {
			return item
		}
	}
	t.Fatalf("no dispatched event: items=%+v", items)
	return streamItem{}
}

type blockingBody struct {
	closed chan struct{}
}

func (b *blockingBody) Read(_ []byte) (int, error) {
	<-b.closed
	return 0, io.EOF
}

func (b *blockingBody) Close() error {
	select {
	case <-b.closed:
	default:
		close(b.closed)
	}
	return nil
}

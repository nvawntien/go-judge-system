// Package client implements the public AstraCode HTTP contract used by the
// benchmark. Submission POST is deliberately a single-attempt operation.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/credentials"
)

const maxResponseBytes = 1 << 20

type API struct {
	BaseURL     *url.URL
	Session     *credentials.Session
	Diagnostics *Diagnostics
}

// Diagnostics is shared by APIs in one benchmark run. It records only local
// transport behavior, never request bodies, cookies, or target identifiers.
type Diagnostics struct {
	newConnections    atomic.Uint64
	reusedConnections atomic.Uint64
	http1             atomic.Uint64
	http2             atomic.Uint64
	otherProtocol     atomic.Uint64
}

type DiagnosticsSnapshot struct {
	NewConnections         uint64
	ReusedConnections      uint64
	HTTP1Responses         uint64
	HTTP2Responses         uint64
	OtherProtocolResponses uint64
}

func (d *Diagnostics) Snapshot() DiagnosticsSnapshot {
	if d == nil {
		return DiagnosticsSnapshot{}
	}
	return DiagnosticsSnapshot{NewConnections: d.newConnections.Load(), ReusedConnections: d.reusedConnections.Load(), HTTP1Responses: d.http1.Load(), HTTP2Responses: d.http2.Load(), OtherProtocolResponses: d.otherProtocol.Load()}
}

type Envelope struct {
	Status string          `json:"status"`
	Code   int             `json:"code"`
	Msg    string          `json:"msg"`
	Data   json.RawMessage `json:"data"`
}

type Me struct {
	ID       string `json:"id"`
	Role     string `json:"role"`
	IsActive bool   `json:"is_active"`
}

type Problem struct {
	ID int64 `json:"id"`
}

type SubmissionRequest struct {
	ProblemID  int64  `json:"problem_id"`
	Language   string `json:"language"`
	SourceCode string `json:"source_code"`
}

type Submission struct {
	ID        int64  `json:"id"`
	ProblemID int64  `json:"problem_id"`
	Language  string `json:"language"`
	Status    string `json:"status"`
}

type SubmissionDetail struct {
	ID        int64  `json:"id"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}

type Ticket struct {
	Value     string `json:"ticket"`
	ExpiresAt string `json:"expires_at"`
}

type SubmitKind string

const (
	SubmitAccepted  SubmitKind = "accepted"
	SubmitRateLimit SubmitKind = "rate_limited"
	Submit4xx       SubmitKind = "client_error"
	Submit5xx       SubmitKind = "server_error"
)

type SubmitResult struct {
	Kind       SubmitKind
	HTTPStatus int
	APICode    int
	Submission *Submission
	RetryAfter *time.Duration
	Message    string
	Protocol   string
}

// TransportError is ambiguous: the server may have received the POST. Callers
// must record and quarantine it, never retry it automatically.
type TransportError struct{ Err error }

func (e *TransportError) Error() string { return "ambiguous submission transport failure" }
func (e *TransportError) Unwrap() error { return e.Err }

// StreamTransportError preserves an inspectable underlying local transport
// error while keeping its public text free of the one-time ticket URL.
type StreamTransportError struct{ Err error }

func (e *StreamTransportError) Error() string { return "SSE connection failed" }
func (e *StreamTransportError) Unwrap() error { return e.Err }

// TransportErrorClass keeps generator-capacity failures distinct from an
// ambiguous remote POST outcome. It never exposes endpoint/cookie details.
func TransportErrorClass(err error) string {
	switch {
	case errors.Is(err, syscall.EMFILE):
		return "client_emfile"
	case errors.Is(err, syscall.ENFILE):
		return "client_enfile"
	case errors.Is(err, syscall.EADDRNOTAVAIL):
		return "client_ephemeral_port_exhaustion"
	case errors.Is(err, context.DeadlineExceeded):
		return "transport_deadline"
	case errors.Is(err, context.Canceled):
		return "transport_cancelled"
	default:
		return "ambiguous_post"
	}
}

func New(base *url.URL, session *credentials.Session) (*API, error) {
	return NewWithDiagnostics(base, session, nil)
}

func NewWithDiagnostics(base *url.URL, session *credentials.Session, diagnostics *Diagnostics) (*API, error) {
	if base == nil || session == nil || session.Client == nil {
		return nil, errors.New("base URL and isolated session are required")
	}
	return &API{BaseURL: base, Session: session, Diagnostics: diagnostics}, nil
}

func (a *API) Me(ctx context.Context) (Me, error) {
	var data Me
	_, err := a.doJSON(ctx, http.MethodGet, "/api/v1/me", nil, &data)
	return data, err
}

func (a *API) PublicProblem(ctx context.Context, slug string) (Problem, error) {
	var data Problem
	_, err := a.doJSON(ctx, http.MethodGet, "/api/v1/problems/"+url.PathEscape(slug), nil, &data)
	return data, err
}

func (a *API) Refresh(ctx context.Context) error {
	_, err := a.doJSON(ctx, http.MethodPost, "/api/v1/auth/refresh-token", nil, nil)
	return err
}

// Submit makes exactly one HTTP request. It intentionally exposes ambiguous
// transport failures instead of hiding them behind a retry.
func (a *API) Submit(ctx context.Context, request SubmissionRequest) (SubmitResult, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return SubmitResult{}, fmt.Errorf("marshal submission request: %w", err)
	}
	response, err := a.request(ctx, http.MethodPost, "/api/v1/submissions", bytes.NewReader(body), "application/json")
	if err != nil {
		return SubmitResult{}, &TransportError{Err: err}
	}
	defer response.Body.Close()
	envelope, err := readEnvelope(response)
	if err != nil {
		return SubmitResult{Kind: Submit5xx, HTTPStatus: response.StatusCode}, &TransportError{Err: err}
	}
	result := SubmitResult{HTTPStatus: response.StatusCode, APICode: envelope.Code, Message: safeMessage(envelope.Msg), Protocol: response.Proto}
	switch {
	case response.StatusCode == http.StatusCreated:
		var submission Submission
		if envelope.Status != "success" || envelope.Code != 20100 || json.Unmarshal(envelope.Data, &submission) != nil || submission.ID <= 0 {
			return result, &TransportError{Err: errors.New("malformed HTTP 201 submission envelope")}
		}
		result.Kind = SubmitAccepted
		result.Submission = &submission
		return result, nil
	case response.StatusCode == http.StatusTooManyRequests:
		result.Kind = SubmitRateLimit
		if retryAfter, ok := parseRetryAfter(response.Header.Get("Retry-After")); ok {
			result.RetryAfter = &retryAfter
		}
		return result, nil
	case response.StatusCode >= 400 && response.StatusCode < 500:
		result.Kind = Submit4xx
		return result, nil
	default:
		result.Kind = Submit5xx
		return result, nil
	}
}

func (a *API) IssueTicket(ctx context.Context, submissionID int64) (Ticket, error) {
	var data Ticket
	_, err := a.doJSON(ctx, http.MethodPost, fmt.Sprintf("/api/v1/submissions/%d/events/ticket", submissionID), nil, &data)
	if err == nil && data.Value == "" {
		err = errors.New("ticket response contains no ticket")
	}
	return data, err
}

func (a *API) GetSubmission(ctx context.Context, submissionID int64) (SubmissionDetail, error) {
	var data SubmissionDetail
	_, err := a.doJSON(ctx, http.MethodGet, fmt.Sprintf("/api/v1/submissions/%d", submissionID), nil, &data)
	if err == nil && data.ID != submissionID {
		err = errors.New("submission detail ID mismatch")
	}
	return data, err
}

// OpenEvents returns a stream response. The ticket query must never be logged
// or persisted by callers.
func (a *API) OpenEvents(ctx context.Context, submissionID int64, ticket string) (*http.Response, error) {
	path := fmt.Sprintf("/events/submissions/%d", submissionID)
	u := a.resolve(path)
	query := u.Query()
	query.Set("ticket", ticket)
	u.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "text/event-stream")
	if a.Diagnostics != nil {
		request = request.WithContext(httptrace.WithClientTrace(request.Context(), &httptrace.ClientTrace{GotConn: func(info httptrace.GotConnInfo) {
			if info.Reused {
				a.Diagnostics.reusedConnections.Add(1)
			} else {
				a.Diagnostics.newConnections.Add(1)
			}
		}}))
	}
	response, err := a.Session.Client.Do(request)
	if err != nil {
		// The URL embeds a one-time ticket. Never surface a transport error that
		// might include the full request URL through net/http formatting.
		return nil, &StreamTransportError{Err: err}
	}
	a.trackResponse(response)
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(strings.ToLower(response.Header.Get("Content-Type")), "text/event-stream") {
		response.Body.Close()
		return nil, fmt.Errorf("SSE endpoint returned HTTP %d", response.StatusCode)
	}
	return response, nil
}

func (a *API) doJSON(ctx context.Context, method, path string, body io.Reader, output any) (Envelope, error) {
	response, err := a.request(ctx, method, path, body, "application/json")
	if err != nil {
		return Envelope{}, err
	}
	defer response.Body.Close()
	envelope, err := readEnvelope(response)
	if err != nil {
		return Envelope{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || envelope.Status != "success" {
		return envelope, fmt.Errorf("API request returned HTTP %d code %d", response.StatusCode, envelope.Code)
	}
	if output != nil && len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		if err := json.Unmarshal(envelope.Data, output); err != nil {
			return envelope, fmt.Errorf("decode API response: %w", err)
		}
	}
	return envelope, nil
}

func (a *API) request(ctx context.Context, method, path string, body io.Reader, contentType string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, method, a.resolve(path).String(), body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	request.Header.Set("Accept", "application/json")
	if a.Diagnostics != nil {
		request = request.WithContext(httptrace.WithClientTrace(request.Context(), &httptrace.ClientTrace{GotConn: func(info httptrace.GotConnInfo) {
			if info.Reused {
				a.Diagnostics.reusedConnections.Add(1)
			} else {
				a.Diagnostics.newConnections.Add(1)
			}
		}}))
	}
	response, err := a.Session.Client.Do(request)
	a.trackResponse(response)
	return response, err
}

func (a *API) trackResponse(response *http.Response) {
	if response == nil || a.Diagnostics == nil {
		return
	}
	switch response.ProtoMajor {
	case 1:
		a.Diagnostics.http1.Add(1)
	case 2:
		a.Diagnostics.http2.Add(1)
	default:
		a.Diagnostics.otherProtocol.Add(1)
	}
}

func (a *API) resolve(path string) *url.URL {
	u := *a.BaseURL
	u.Path = strings.TrimSuffix(a.BaseURL.Path, "/") + path
	u.RawQuery = ""
	u.Fragment = ""
	return &u
}

func readEnvelope(response *http.Response) (Envelope, error) {
	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return Envelope{}, fmt.Errorf("read HTTP %d API response", response.StatusCode)
	}
	if len(data) > maxResponseBytes {
		return Envelope{}, fmt.Errorf("HTTP %d API response exceeds size limit", response.StatusCode)
	}
	if len(data) == 0 {
		return Envelope{}, fmt.Errorf("HTTP %d API response is empty", response.StatusCode)
	}
	var envelope Envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return Envelope{}, fmt.Errorf("HTTP %d API response contains invalid JSON", response.StatusCode)
	}
	return envelope, nil
}

func parseRetryAfter(raw string) (time.Duration, bool) {
	seconds, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || seconds < 0 {
		return 0, false
	}
	return time.Duration(seconds) * time.Second, true
}

func safeMessage(message string) string {
	message = strings.ReplaceAll(message, "\n", " ")
	message = strings.TrimSpace(message)
	if len(message) > 256 {
		return message[:256]
	}
	return message
}

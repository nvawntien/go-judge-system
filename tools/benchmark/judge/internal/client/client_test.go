package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/credentials"
)

func testAPI(t *testing.T, handler http.Handler) *API {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	base, _ := url.Parse(server.URL)
	sessions, err := credentials.NewSessions(credentials.File{SchemaVersion: 1, Users: []credentials.User{{Alias: "bench-001", Cookies: credentials.Cookies{AccessToken: "secret"}}}}, base, http.DefaultTransport)
	if err != nil {
		t.Fatal(err)
	}
	api, err := New(base, sessions[0])
	if err != nil {
		t.Fatal(err)
	}
	return api
}

func TestSubmitParsesCreatedAndRateLimited(t *testing.T) {
	t.Run("created", func(t *testing.T) {
		api := testAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/api/v1/submissions" {
				t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":"success","code":20100,"data":{"id":42,"problem_id":1,"language":"GO","status":"PENDING"}}`))
		}))
		result, err := api.Submit(context.Background(), SubmissionRequest{ProblemID: 1, Language: "GO", SourceCode: "x"})
		if err != nil || result.Kind != SubmitAccepted || result.Submission.ID != 42 {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	})
	t.Run("rate limited", func(t *testing.T) {
		api := testAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"status":"error","code":42900,"msg":"slow down"}`))
		}))
		result, err := api.Submit(context.Background(), SubmissionRequest{ProblemID: 1, Language: "GO", SourceCode: "x"})
		if err != nil || result.Kind != SubmitRateLimit || result.RetryAfter == nil || *result.RetryAfter == 0 {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	})
}

func TestSubmitMalformed201IsAmbiguous(t *testing.T) {
	api := testAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"status":"success","code":20100,"data":{}}`))
	}))
	_, err := api.Submit(context.Background(), SubmissionRequest{ProblemID: 1, Language: "GO", SourceCode: "x"})
	if _, ok := err.(*TransportError); !ok {
		t.Fatalf("err=%T %v, want ambiguous TransportError", err, err)
	}
}

func TestSubmitClassifies4xx5xxAndNeverRetries(t *testing.T) {
	for _, test := range []struct {
		name string
		code int
		kind SubmitKind
	}{
		{name: "other 4xx", code: http.StatusBadRequest, kind: Submit4xx},
		{name: "5xx", code: http.StatusServiceUnavailable, kind: Submit5xx},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			api := testAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.WriteHeader(test.code)
				_, _ = w.Write([]byte(`{"status":"error","code":50000,"msg":"server"}`))
			}))
			got, err := api.Submit(context.Background(), SubmissionRequest{ProblemID: 1, Language: "GO", SourceCode: "x"})
			if err != nil || got.Kind != test.kind || calls != 1 {
				t.Fatalf("result=%+v calls=%d err=%v", got, calls, err)
			}
		})
	}
}

func TestSubmitRateLimitWithoutRetryAfterDoesNotFabricateValue(t *testing.T) {
	api := testAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"status":"error","code":42900,"msg":"slow down"}`))
	}))
	got, err := api.Submit(context.Background(), SubmissionRequest{ProblemID: 1, Language: "GO", SourceCode: "x"})
	if err != nil || got.Kind != SubmitRateLimit || got.RetryAfter != nil {
		t.Fatalf("result=%+v err=%v", got, err)
	}
}

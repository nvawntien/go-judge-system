package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/credentials"
)

const testPassword = "bootstrap-password-sentinel"

func TestIdentitiesUseFixedNamespaceAndSafeRange(t *testing.T) {
	identities, err := Identities(1, 100000)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct {
		index    int
		username string
		alias    string
	}{
		{0, "benchmark_judge_001", "bench-001"},
		{49, "benchmark_judge_050", "bench-050"},
		{99, "benchmark_judge_100", "bench-100"},
		{998, "benchmark_judge_999", "bench-999"},
		{999, "benchmark_judge_1000", "bench-1000"},
		{99999, "benchmark_judge_100000", "bench-100000"},
	} {
		got := identities[want.index]
		if got.Username != want.username || got.Alias != want.alias {
			t.Fatalf("identity %d = %#v", want.index, got)
		}
	}
	for _, test := range [][2]int{{0, 1}, {1, 0}, {100001, 1}, {100000, 2}, {1, int(^uint(0) >> 1)}} {
		if _, err := Identities(test[0], test[1]); !errors.Is(err, ErrInvalidRange) {
			t.Fatalf("Identities(%d, %d) error = %v", test[0], test[1], err)
		}
	}
}

func TestValidateTargetRequiresConfirmedHTTPSForRemote(t *testing.T) {
	for _, test := range []struct {
		value, confirm string
		allow          bool
		want           error
	}{
		{"http://127.0.0.1:8080", "", false, nil},
		{"http://[::1]:8080", "", false, nil},
		{"http://example.test", "example.test", true, ErrTargetUnsafe},
		{"https://example.test", "", true, ErrTargetUnsafe},
		{"https://example.test", "wrong.test", true, ErrTargetUnsafe},
		{"https://example.test", "example.test", true, nil},
		{"https://user@example.test", "example.test", true, ErrTargetUnsafe},
		{"https://example.test/?x=1", "example.test", true, ErrTargetUnsafe},
	} {
		base, err := url.Parse(test.value)
		if err != nil {
			t.Fatal(err)
		}
		if got := ValidateTarget(base, test.allow, test.confirm); !errors.Is(got, test.want) {
			t.Fatalf("ValidateTarget(%q) = %v, want %v", test.value, got, test.want)
		}
	}
}

func TestRunWritesSchemaCompatibleSessionsInOrder(t *testing.T) {
	server, seen := sessionServer(t, nil)
	defer server.Close()
	output := filepath.Join(t.TempDir(), "users.local.json")
	file, err := Run(context.Background(), testOptions(t, server.URL, output, 1, 50))
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Users) != 50 || len(*seen) != 50 {
		t.Fatalf("users=%d requests=%d", len(file.Users), len(*seen))
	}
	for index, user := range file.Users {
		want := index + 1
		if user.Alias != "bench-"+pad(want) || (*seen)[index] != "benchmark_judge_"+pad(want) {
			t.Fatalf("index %d user=%q request=%q", index, user.Alias, (*seen)[index])
		}
	}
	loaded, err := credentials.Load(output)
	if err != nil {
		t.Fatalf("output is not credentials-loader compatible: %v", err)
	}
	if len(loaded.Users) != 50 {
		t.Fatalf("loaded users = %d", len(loaded.Users))
	}
	info, err := os.Stat(output)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("output mode = %v, err=%v", info.Mode(), err)
	}
}

func TestRunFailsWithoutRequiredCookiesOrMatchingSession(t *testing.T) {
	for name, mutate := range map[string]func(http.ResponseWriter, string) bool{
		"missing access": func(w http.ResponseWriter, _ string) bool {
			http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: "refresh-sentinel", Path: "/"})
			return true
		},
		"missing refresh": func(w http.ResponseWriter, _ string) bool {
			http.SetCookie(w, &http.Cookie{Name: "access_token", Value: "access-sentinel", Path: "/"})
			return true
		},
		"mismatched me": func(http.ResponseWriter, string) bool { return false },
	} {
		t.Run(name, func(t *testing.T) {
			server, _ := sessionServer(t, mutate)
			defer server.Close()
			output := filepath.Join(t.TempDir(), "users.local.json")
			_, err := Run(context.Background(), testOptions(t, server.URL, output, 1, 1))
			if err == nil || strings.Contains(err.Error(), testPassword) || strings.Contains(err.Error(), "access-sentinel") || strings.Contains(err.Error(), "refresh-sentinel") {
				t.Fatalf("unsafe or missing error: %v", err)
			}
			if _, statErr := os.Lstat(output); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("output exists after failed bootstrap: %v", statErr)
			}
		})
	}
}

func TestRunStopsAfterFirstFailureAndLeavesNoOutput(t *testing.T) {
	var logins []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/login" {
			var request struct {
				Identifier string `json:"identifier"`
			}
			_ = json.NewDecoder(r.Body).Decode(&request)
			logins = append(logins, request.Identifier)
			if request.Identifier == "benchmark_judge_002" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "access_token", Value: "access-" + request.Identifier, Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: "refresh-" + request.Identifier, Path: "/"})
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/api/v1/me" {
			cookie, _ := r.Cookie("access_token")
			username := strings.TrimPrefix(cookie.Value, "access-")
			_ = json.NewEncoder(w).Encode(meResponse(username))
		}
	}))
	defer server.Close()
	output := filepath.Join(t.TempDir(), "users.local.json")
	_, err := Run(context.Background(), testOptions(t, server.URL, output, 1, 3))
	if err == nil || !strings.Contains(err.Error(), "bench-002") {
		t.Fatalf("error = %v", err)
	}
	if strings.Join(logins, ",") != "benchmark_judge_001,benchmark_judge_002" {
		t.Fatalf("login sequence = %v", logins)
	}
	if _, statErr := os.Lstat(output); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("output exists after partial failure: %v", statErr)
	}
}

func TestRunFailureNeverReplacesPriorCredentialOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/login" {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer server.Close()
	output := filepath.Join(t.TempDir(), "users.local.json")
	const prior = "previous-valid-credential-file"
	if err := os.WriteFile(output, []byte(prior), 0o600); err != nil {
		t.Fatal(err)
	}
	options := testOptions(t, server.URL, output, 1, 1)
	options.Replace = true
	if _, err := Run(context.Background(), options); err == nil {
		t.Fatal("failed bootstrap replaced an existing output")
	}
	actual, err := os.ReadFile(output)
	if err != nil || string(actual) != prior {
		t.Fatalf("existing output changed: %q err=%v", actual, err)
	}
}

func TestRunBoundsConcurrencyAndWritesDeterministicOrder(t *testing.T) {
	var active, peak int
	var mutex sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			mutex.Lock()
			active++
			if active > peak {
				peak = active
			}
			mutex.Unlock()
			defer func() { mutex.Lock(); active--; mutex.Unlock() }()
			var request struct {
				Identifier string `json:"identifier"`
			}
			_ = json.NewDecoder(r.Body).Decode(&request)
			time.Sleep(5 * time.Millisecond)
			http.SetCookie(w, &http.Cookie{Name: "access_token", Value: "access-" + request.Identifier, Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: "refresh-" + request.Identifier, Path: "/"})
		case "/api/v1/me":
			cookie, _ := r.Cookie("access_token")
			_ = json.NewEncoder(w).Encode(meResponse(strings.TrimPrefix(cookie.Value, "access-")))
		}
	}))
	defer server.Close()
	options := testOptions(t, server.URL, filepath.Join(t.TempDir(), "users.local.json"), 1, 20)
	options.Concurrency, options.LoginDelay = 4, 0
	file, err := Run(context.Background(), options)
	if err != nil || peak > 4 || len(file.Users) != 20 {
		t.Fatalf("err=%v peak=%d users=%d", err, peak, len(file.Users))
	}
	for index, user := range file.Users {
		if user.Alias != "bench-"+pad(index+1) {
			t.Fatalf("unordered output at %d: %q", index, user.Alias)
		}
	}
}

func TestRunRejectsRedirectAndNeverFollowsIt(t *testing.T) {
	var redirected bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/login" {
			http.Redirect(w, r, "/unexpected", http.StatusTemporaryRedirect)
			return
		}
		redirected = true
	}))
	defer server.Close()
	_, err := Run(context.Background(), testOptions(t, server.URL, filepath.Join(t.TempDir(), "users.local.json"), 1, 1))
	if err == nil || redirected {
		t.Fatalf("err=%v redirected=%v", err, redirected)
	}
}

func TestRunRefusesExistingOrSymlinkOutputAndCleansTemporaryFiles(t *testing.T) {
	server, _ := sessionServer(t, nil)
	defer server.Close()
	directory := t.TempDir()
	output := filepath.Join(directory, "users.local.json")
	if err := os.WriteFile(output, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), testOptions(t, server.URL, output, 1, 1)); err == nil {
		t.Fatal("existing output was overwritten")
	}
	replace := testOptions(t, server.URL, output, 1, 1)
	replace.Replace = true
	if _, err := Run(context.Background(), replace); err != nil {
		t.Fatalf("replace: %v", err)
	}
	link := filepath.Join(directory, "link.json")
	if err := os.Symlink(output, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), testOptions(t, server.URL, link, 1, 1)); err == nil {
		t.Fatal("symlink output accepted")
	}
	leftovers, err := filepath.Glob(filepath.Join(directory, ".users.local.*.tmp"))
	if err != nil || len(leftovers) != 0 {
		t.Fatalf("temporary credentials left behind: %v, %v", leftovers, err)
	}
}

func TestRunCancellationDoesNotStartLoginOrWriteOutput(t *testing.T) {
	server, seen := sessionServer(t, nil)
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	output := filepath.Join(t.TempDir(), "users.local.json")
	_, err := Run(ctx, testOptions(t, server.URL, output, 1, 1))
	if !errors.Is(err, context.Canceled) || len(*seen) != 0 {
		t.Fatalf("err=%v requests=%d", err, len(*seen))
	}
	if _, statErr := os.Lstat(output); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("output exists: %v", statErr)
	}
}

func testOptions(t *testing.T, rawURL, output string, start, count int) Options {
	t.Helper()
	base, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	return Options{BaseURL: base, Start: start, Count: count, Password: []byte(testPassword), Output: output, Concurrency: 1}
}

func sessionServer(t *testing.T, loginOverride func(http.ResponseWriter, string) bool) (*httptest.Server, *[]string) {
	t.Helper()
	var mutex sync.Mutex
	logins := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			var request struct {
				Identifier string `json:"identifier"`
				Password   string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Password != testPassword {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			mutex.Lock()
			logins = append(logins, request.Identifier)
			mutex.Unlock()
			if loginOverride != nil && loginOverride(w, request.Identifier) {
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "access_token", Value: "access-" + request.Identifier, Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: "refresh-" + request.Identifier, Path: "/"})
			w.WriteHeader(http.StatusOK)
		case "/api/v1/me":
			cookie, err := r.Cookie("access_token")
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			username := strings.TrimPrefix(cookie.Value, "access-")
			if loginOverride != nil && !loginOverride(w, username) {
				username = "wrong_user"
			}
			_ = json.NewEncoder(w).Encode(meResponse(username))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return server, &logins
}

func meResponse(username string) map[string]any {
	return map[string]any{"status": "success", "data": map[string]any{"username": username, "role": "user", "is_active": true}}
}

func pad(value int) string {
	return strings.Repeat("0", 3-len(strconv.Itoa(value))) + strconv.Itoa(value)
}

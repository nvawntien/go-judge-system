// Package bootstrap creates local cookie-session files through the public API.
package bootstrap

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/nvawntien/go-judge-system/tools/benchmark/judge/internal/credentials"
)

const (
	MinSequence        = 1
	MaxSequence        = 100000
	maxResponseBytes   = 1 << 20
	defaultConcurrency = 16
)

var (
	ErrInvalidRange = errors.New("invalid benchmark user range")
	ErrTargetUnsafe = errors.New("unsafe target")
)

type Options struct {
	BaseURL           *url.URL
	AllowRemote       bool
	ConfirmTargetHost string
	Start             int
	Count             int
	Password          []byte
	Output            string
	Replace           bool
	LoginDelay        time.Duration
	// Concurrency bounds normal Login + /me preparation work. It is not a
	// load-test control and defaults conservatively.
	Concurrency       int
	HTTPTransport     http.RoundTripper
	HTTPClientFactory func(jar http.CookieJar) *http.Client
}

type Identity struct {
	Sequence int
	Username string
	Alias    string
}

func Identities(start, count int) ([]Identity, error) {
	if start < MinSequence || count < 1 || start > MaxSequence || count-1 > MaxSequence-start {
		return nil, ErrInvalidRange
	}
	identities := make([]Identity, 0, count)
	for sequence := start; sequence < start+count; sequence++ {
		identities = append(identities, Identity{
			Sequence: sequence,
			Username: fmt.Sprintf("benchmark_judge_%03d", sequence),
			Alias:    fmt.Sprintf("bench-%03d", sequence),
		})
	}
	return identities, nil
}

func ValidateTarget(base *url.URL, allowRemote bool, confirmedHost string) error {
	if base == nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" || (base.Path != "" && base.Path != "/") {
		return ErrTargetUnsafe
	}
	if isLoopback(base.Hostname()) {
		return nil
	}
	if base.Scheme != "https" || !allowRemote || confirmedHost != base.Hostname() {
		return ErrTargetUnsafe
	}
	return nil
}

func Run(ctx context.Context, options Options) (credentials.File, error) {
	if err := ValidateTarget(options.BaseURL, options.AllowRemote, options.ConfirmTargetHost); err != nil {
		return credentials.File{}, err
	}
	identities, err := Identities(options.Start, options.Count)
	if err != nil {
		return credentials.File{}, err
	}
	if len(options.Password) == 0 {
		return credentials.File{}, errors.New("benchmark password is required")
	}
	if options.Output == "" {
		return credentials.File{}, errors.New("output path is required")
	}
	if err := ensureOutputTarget(options.Output, options.Replace); err != nil {
		return credentials.File{}, err
	}
	concurrency := options.Concurrency
	if concurrency == 0 {
		concurrency = defaultConcurrency
	}
	if concurrency < 1 || concurrency > len(identities) {
		if concurrency < 1 {
			return credentials.File{}, errors.New("bootstrap concurrency must be positive")
		}
		concurrency = len(identities)
	}
	transport := options.HTTPTransport
	if transport == nil {
		transport = credentials.NewBenchmarkTransport(max(64, concurrency*2))
	}
	if closer, ok := transport.(interface{ CloseIdleConnections() }); ok {
		defer closer.CloseIdleConnections()
	}

	users, err := bootstrapUsers(ctx, options, identities, concurrency, transport)
	if err != nil {
		return credentials.File{}, err
	}
	file := credentials.File{SchemaVersion: credentials.SchemaVersion, Users: users}
	if err := credentials.Validate(file); err != nil {
		return credentials.File{}, errors.New("generated credential file failed validation")
	}
	if err := writeAtomic(options.Output, file, options.Replace); err != nil {
		return credentials.File{}, err
	}
	return file, nil
}

func bootstrapUsers(ctx context.Context, options Options, identities []Identity, concurrency int, transport http.RoundTripper) ([]credentials.User, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	users := make([]credentials.User, len(identities))
	jobs := make(chan int)
	errCh := make(chan error, 1)
	var workers sync.WaitGroup
	for range concurrency {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case index, ok := <-jobs:
					if !ok {
						return
					}
					user, err := loginAndValidate(ctx, options, identities[index], transport)
					if err != nil {
						select {
						case errCh <- err:
							cancel()
						default:
						}
						return
					}
					users[index] = user
					if options.LoginDelay > 0 && !sleep(ctx, options.LoginDelay) {
						return
					}
				}
			}
		}()
	}
	for index := range identities {
		select {
		case <-ctx.Done():
			close(jobs)
			workers.Wait()
			select {
			case err := <-errCh:
				return nil, err
			default:
				return nil, ctx.Err()
			}
		case jobs <- index:
		}
	}
	close(jobs)
	workers.Wait()
	select {
	case err := <-errCh:
		return nil, err
	default:
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func loginAndValidate(ctx context.Context, options Options, identity Identity, transport http.RoundTripper) (credentials.User, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return credentials.User{}, errors.New("create cookie jar")
	}
	client := newClient(jar, options.HTTPClientFactory, transport)
	body, err := json.Marshal(struct {
		Identifier string `json:"identifier"`
		Password   string `json:"password"`
	}{Identifier: identity.Username, Password: string(options.Password)})
	if err != nil {
		return credentials.User{}, errors.New("encode login request")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint(options.BaseURL, "/api/v1/auth/login"), bytes.NewReader(body))
	if err != nil {
		return credentials.User{}, errors.New("construct login request")
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return credentials.User{}, fmt.Errorf("login failed for %s", identity.Alias)
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxResponseBytes))
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return credentials.User{}, fmt.Errorf("login failed for %s: HTTP %d", identity.Alias, response.StatusCode)
	}
	access, refresh := cookies(jar, options.BaseURL)
	if access == "" || refresh == "" {
		return credentials.User{}, fmt.Errorf("login failed for %s: required session cookies missing", identity.Alias)
	}

	meRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint(options.BaseURL, "/api/v1/me"), nil)
	if err != nil {
		return credentials.User{}, errors.New("construct session validation request")
	}
	meResponse, err := client.Do(meRequest)
	if err != nil {
		return credentials.User{}, fmt.Errorf("session validation failed for %s", identity.Alias)
	}
	defer meResponse.Body.Close()
	if meResponse.StatusCode != http.StatusOK {
		return credentials.User{}, fmt.Errorf("session validation failed for %s: HTTP %d", identity.Alias, meResponse.StatusCode)
	}
	var me struct {
		Status string `json:"status"`
		Data   struct {
			Username string `json:"username"`
			Role     string `json:"role"`
			IsActive bool   `json:"is_active"`
		} `json:"data"`
	}
	decoder := json.NewDecoder(io.LimitReader(meResponse.Body, maxResponseBytes))
	if err := decoder.Decode(&me); err != nil || me.Status != "success" {
		return credentials.User{}, fmt.Errorf("session validation failed for %s: invalid response", identity.Alias)
	}
	// /api/v1/me currently exposes username, role, and active state. Suspension
	// is enforced by Auth login but is not part of this public response.
	if me.Data.Username != identity.Username || me.Data.Role != "user" || !me.Data.IsActive {
		return credentials.User{}, fmt.Errorf("session validation failed for %s: identity mismatch", identity.Alias)
	}
	return credentials.User{Alias: identity.Alias, Cookies: credentials.Cookies{AccessToken: access, RefreshToken: refresh}}, nil
}

func newClient(jar http.CookieJar, factory func(http.CookieJar) *http.Client, transport http.RoundTripper) *http.Client {
	var client *http.Client
	if factory != nil {
		client = factory(jar)
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second, Transport: transport}
	}
	// A factory may provide a test transport, but it must never weaken the
	// session boundary or permit a redirect to replay credentials.
	client.Jar = jar
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return errors.New("redirect rejected") }
	if client.Timeout == 0 {
		client.Timeout = 10 * time.Second
	}
	return client
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

func endpoint(base *url.URL, path string) string {
	copy := *base
	copy.Path = path
	copy.RawPath = ""
	return copy.String()
}

func cookies(jar http.CookieJar, base *url.URL) (access, refresh string) {
	for _, cookie := range jar.Cookies(base) {
		switch cookie.Name {
		case "access_token":
			access = cookie.Value
		case "refresh_token":
			refresh = cookie.Value
		}
	}
	return access, refresh
}

func writeAtomic(path string, file credentials.File, replace bool) error {
	if err := ensureOutputTarget(path, replace); err != nil {
		return err
	}
	directory := filepath.Dir(path)
	random := make([]byte, 12)
	if _, err := rand.Read(random); err != nil {
		return errors.New("create output temporary name")
	}
	temporary := filepath.Join(directory, ".users.local."+hex.EncodeToString(random)+".tmp")
	f, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return errors.New("create output temporary file")
	}
	defer func() { _ = f.Close(); _ = os.Remove(temporary) }()
	if err := json.NewEncoder(f).Encode(file); err != nil {
		return errors.New("write credential output")
	}
	if err := f.Sync(); err != nil {
		return errors.New("sync credential output")
	}
	if err := f.Close(); err != nil {
		return errors.New("close credential output")
	}
	if replace {
		// Rename replaces a final symlink rather than following it, but reject a
		// symlink that appeared after the initial inspection for a clear safety
		// contract.
		if err := ensureOutputTarget(path, true); err != nil {
			return err
		}
		if err := os.Rename(temporary, path); err != nil {
			return errors.New("install credential output")
		}
	} else if err := os.Link(temporary, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return errors.New("output file already exists; use --replace to renew it")
		}
		return errors.New("install credential output")
	} else if err := os.Remove(temporary); err != nil {
		return errors.New("remove credential temporary file")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return errors.New("secure credential output permissions")
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return errors.New("verify credential output permissions")
	}
	return nil
}

func ensureOutputTarget(path string, replace bool) error {
	if path == "" {
		return errors.New("output path is required")
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return errors.New("inspect output path")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("output path must not be a symlink")
	}
	if !info.Mode().IsRegular() {
		return errors.New("output path must be a regular file")
	}
	if !replace {
		return errors.New("output file already exists; use --replace to renew it")
	}
	return nil
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func SafePath(value string) bool {
	if value == "" || value != strings.TrimSpace(value) {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

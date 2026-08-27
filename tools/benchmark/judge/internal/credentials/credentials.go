// Package credentials handles local, pre-issued benchmark sessions. It does
// not provision accounts, log in, or persist refreshed cookies.
package credentials

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"time"
)

const SchemaVersion = 1

// NewBenchmarkTransport returns the shared, explicitly bounded connection
// transport used by all isolated cookie-jar clients in one benchmark process.
// Cookie jars remain per-user; only TCP connection reuse is shared.
func NewBenchmarkTransport(maxConnsPerHost int) *http.Transport {
	if maxConnsPerHost < 1 {
		maxConnsPerHost = 1
	}
	maxIdle := maxConnsPerHost
	if maxIdle > 4096 {
		maxIdle = 4096
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = maxIdle
	transport.MaxIdleConnsPerHost = maxIdle
	transport.MaxConnsPerHost = maxConnsPerHost
	transport.IdleConnTimeout = 90 * time.Second
	return transport
}

var aliasPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
var canonicalAliasPattern = regexp.MustCompile(`^bench-([0-9]+)$`)

type File struct {
	SchemaVersion int    `json:"schema_version"`
	Users         []User `json:"users"`
}

type User struct {
	Alias   string  `json:"alias"`
	Cookies Cookies `json:"cookies"`
}

type Cookies struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// Session is deliberately per logical user. Its Jar must never be shared.
type Session struct {
	Alias  string
	Jar    http.CookieJar
	Client *http.Client
	// Subject remains memory-only and is set by authenticated preflight.
	Subject string
}

func Load(path string) (File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return File{}, fmt.Errorf("inspect credential file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return File{}, errors.New("credential file must not be a symlink")
	}
	if !info.Mode().IsRegular() {
		return File{}, errors.New("credential file must be a regular file")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return File{}, errors.New("credential file must be mode 0600 or stricter")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return File{}, fmt.Errorf("read credential file: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var file File
	if err := decoder.Decode(&file); err != nil {
		return File{}, fmt.Errorf("decode credential file: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return File{}, errors.New("credential file contains trailing JSON values")
	}
	if err := Validate(file); err != nil {
		return File{}, err
	}
	return file, nil
}

func Validate(file File) error {
	if file.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported credential schema_version %d", file.SchemaVersion)
	}
	if len(file.Users) == 0 {
		return errors.New("credential file contains no users")
	}
	aliases := make(map[string]struct{}, len(file.Users))
	tokens := make(map[string]struct{}, len(file.Users))
	for _, user := range file.Users {
		if !aliasPattern.MatchString(user.Alias) {
			return fmt.Errorf("invalid benchmark alias %q", user.Alias)
		}
		if user.Cookies.AccessToken == "" {
			return fmt.Errorf("user %q has no access_token", user.Alias)
		}
		if _, exists := aliases[user.Alias]; exists {
			return fmt.Errorf("duplicate benchmark alias %q", user.Alias)
		}
		if _, exists := tokens[user.Cookies.AccessToken]; exists {
			return errors.New("duplicate access_token in credential file")
		}
		aliases[user.Alias] = struct{}{}
		tokens[user.Cookies.AccessToken] = struct{}{}
	}
	return nil
}

// SelectCanonical returns the first count canonical benchmark aliases in
// numeric order. It deliberately rejects gaps and non-canonical aliases so a
// massive burst cannot accidentally select an arbitrary real-user session.
func SelectCanonical(file File, count int) (File, error) {
	if err := Validate(file); err != nil {
		return File{}, err
	}
	if count < 1 || count > 10000 {
		return File{}, errors.New("canonical benchmark user count must be within 1..10000")
	}
	bySequence := make(map[int]User, len(file.Users))
	for _, user := range file.Users {
		match := canonicalAliasPattern.FindStringSubmatch(user.Alias)
		if len(match) != 2 {
			return File{}, fmt.Errorf("benchmark session alias %q is not canonical", user.Alias)
		}
		sequence, err := strconv.Atoi(match[1])
		if err != nil || sequence < 1 || sequence > 10000 || user.Alias != fmt.Sprintf("bench-%03d", sequence) {
			return File{}, fmt.Errorf("benchmark session alias %q is not canonical", user.Alias)
		}
		if _, exists := bySequence[sequence]; exists {
			return File{}, fmt.Errorf("duplicate canonical benchmark alias %q", user.Alias)
		}
		bySequence[sequence] = user
	}
	selected := make([]User, 0, count)
	for sequence := 1; sequence <= count; sequence++ {
		user, ok := bySequence[sequence]
		if !ok {
			return File{}, fmt.Errorf("canonical benchmark session bench-%03d is missing", sequence)
		}
		selected = append(selected, user)
	}
	return File{SchemaVersion: SchemaVersion, Users: selected}, nil
}

func NewSessions(file File, base *url.URL, transport http.RoundTripper) ([]*Session, error) {
	if err := Validate(file); err != nil {
		return nil, err
	}
	if base == nil || base.Scheme == "" || base.Host == "" {
		return nil, errors.New("valid base URL is required to seed session cookies")
	}
	if transport == nil {
		transport = http.DefaultTransport
	}
	sessions := make([]*Session, 0, len(file.Users))
	for _, user := range file.Users {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, fmt.Errorf("create cookie jar: %w", err)
		}
		cookies := []*http.Cookie{{Name: "access_token", Value: user.Cookies.AccessToken, Path: "/", HttpOnly: true}}
		if user.Cookies.RefreshToken != "" {
			cookies = append(cookies, &http.Cookie{Name: "refresh_token", Value: user.Cookies.RefreshToken, Path: "/", HttpOnly: true})
		}
		jar.SetCookies(base, cookies)
		sessions = append(sessions, &Session{
			Alias: user.Alias,
			Jar:   jar,
			Client: &http.Client{
				Transport:     transport,
				Jar:           jar,
				CheckRedirect: sameOriginRedirect(base),
			},
		})
	}
	return sessions, nil
}

func sameOriginRedirect(_ *url.URL) func(*http.Request, []*http.Request) error {
	return func(_ *http.Request, _ []*http.Request) error {
		// Reject every redirect. Same-origin 307/308 redirects can replay a POST,
		// violating the benchmark's exactly-one submission request invariant.
		return errors.New("redirect rejected")
	}
}

// HasRefresh reports only local session capability; it does not reveal or log
// the token itself.
func (s *Session) HasRefresh(base *url.URL) bool {
	for _, cookie := range s.Jar.Cookies(base) {
		if cookie.Name == "refresh_token" && cookie.Value != "" {
			return true
		}
	}
	return false
}

// AccessToken returns the in-memory cookie only to the local runner so it can
// inspect the unverified JWT expiry for pre-run lifetime planning. Callers must
// not persist or log the returned value.
func (s *Session) AccessToken(base *url.URL) string {
	for _, cookie := range s.Jar.Cookies(base) {
		if cookie.Name == "access_token" {
			return cookie.Value
		}
	}
	return ""
}

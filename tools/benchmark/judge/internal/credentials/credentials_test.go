package credentials

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func canonicalFile(count int) File {
	users := make([]User, count)
	for index := range users {
		sequence := index + 1
		users[index] = User{Alias: fmt.Sprintf("bench-%03d", sequence), Cookies: Cookies{AccessToken: fmt.Sprintf("token-%d", sequence)}}
	}
	return File{SchemaVersion: SchemaVersion, Users: users}
}

func validFile() File {
	return File{SchemaVersion: SchemaVersion, Users: []User{{Alias: "bench-001", Cookies: Cookies{AccessToken: "one"}}, {Alias: "bench-002", Cookies: Cookies{AccessToken: "two"}}}}
}

func TestValidateRejectsDuplicateAliasAndToken(t *testing.T) {
	file := validFile()
	file.Users[1].Alias = file.Users[0].Alias
	if err := Validate(file); err == nil {
		t.Fatal("duplicate alias accepted")
	}
	file = validFile()
	file.Users[1].Cookies.AccessToken = file.Users[0].Cookies.AccessToken
	if err := Validate(file); err == nil {
		t.Fatal("duplicate token accepted")
	}
}

func TestLoadRejectsPermissiveFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission semantics")
	}
	path := filepath.Join(t.TempDir(), "users.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":1,"users":[{"alias":"bench-001","cookies":{"access_token":"x"}}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("permissive credential file accepted")
	}
}

func TestLoadRejectsTrailingJSONValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users.json")
	content := `{"schema_version":1,"users":[{"alias":"bench-001","cookies":{"access_token":"x"}}]} {}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("trailing JSON value accepted")
	}
}

func TestLoadRejectsSymlinkAndNonRegularFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions differ on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target.json")
	if err := os.WriteFile(target, []byte(`{"schema_version":1,"users":[{"alias":"bench-001","cookies":{"access_token":"x"}}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(link); err == nil {
		t.Fatal("symlink credential file accepted")
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("directory credential file accepted")
	}
}

func TestNewSessionsKeepsCookieJarsIsolated(t *testing.T) {
	base, _ := url.Parse("https://example.test")
	sessions, err := NewSessions(validFile(), base, http.DefaultTransport)
	if err != nil {
		t.Fatal(err)
	}
	sessions[0].Jar.SetCookies(base, []*http.Cookie{{Name: "only-first", Value: "yes", Path: "/"}})
	for _, cookie := range sessions[1].Jar.Cookies(base) {
		if cookie.Name == "only-first" {
			t.Fatal("cookie jar leaked between benchmark users")
		}
	}
}

func TestSessionRejectsPOSTRedirectBeforeReplay(t *testing.T) {
	base, _ := url.Parse("https://example.test")
	sessions, err := NewSessions(validFile(), base, http.DefaultTransport)
	if err != nil {
		t.Fatal(err)
	}
	request, _ := http.NewRequest(http.MethodPost, "https://example.test/submissions", nil)
	redirect, _ := http.NewRequest(http.MethodPost, "https://example.test/other", nil)
	if err := sessions[0].Client.CheckRedirect(redirect, []*http.Request{request}); err == nil {
		t.Fatal("redirect that could replay POST was accepted")
	}
}

func TestSelectCanonicalSupportsMassiveBurstRanges(t *testing.T) {
	file := canonicalFile(10000)
	for _, count := range []int{1000, 5000, 10000} {
		selected, err := SelectCanonical(file, count)
		if err != nil || len(selected.Users) != count || selected.Users[0].Alias != "bench-001" || selected.Users[count-1].Alias != fmt.Sprintf("bench-%03d", count) {
			t.Fatalf("count=%d users=%d err=%v", count, len(selected.Users), err)
		}
	}
	if _, err := SelectCanonical(file, 10001); err == nil {
		t.Fatal("out-of-range canonical selection accepted")
	}
	file.Users[1].Alias = "other-user"
	if _, err := SelectCanonical(file, 2); err == nil {
		t.Fatal("noncanonical session alias accepted")
	}
}

func TestNewBenchmarkTransportHasExplicitHighScaleLimits(t *testing.T) {
	transport := NewBenchmarkTransport(10000)
	if transport.MaxConnsPerHost != 10000 || transport.MaxIdleConnsPerHost != 4096 || transport.MaxIdleConns != 4096 {
		t.Fatalf("transport=%+v", transport)
	}
}

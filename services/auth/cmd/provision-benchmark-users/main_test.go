package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	benchmark "go-judge-system/services/auth/internal/application/usecase/benchmark"
)

func TestParseOptionsDryRunRejectsPasswordInputs(t *testing.T) {
	if _, err := parseOptions([]string{"--password-file", "secret"}, os.Stderr); err == nil {
		t.Fatal("expected dry-run password-file rejection")
	}
	if _, err := parseOptions([]string{"--apply", "--password-file", "secret"}, os.Stderr); err != nil {
		t.Fatalf("apply password-file: %v", err)
	}
}

func TestReadPasswordFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "password")
	if err := os.WriteFile(path, []byte("sentinel-password\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	password, err := readPasswordFile(path)
	if err != nil || string(password) != "sentinel-password" {
		t.Fatalf("password file read failed: %v", err)
	}
	clear(password)
	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatal(err)
	}
	if password, err := readPasswordFile(path); err != nil || string(password) != "sentinel-password" {
		t.Fatal("expected secure 0400 password file to be accepted")
	} else {
		clear(password)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readPasswordFile(path); err == nil {
		t.Fatal("expected insecure mode rejection")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("one\ntwo"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPasswordFile(path); err == nil {
		t.Fatal("expected multi-line rejection")
	}
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), maxPasswordBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPasswordFile(path); err == nil {
		t.Fatal("expected oversized password-file rejection")
	}
}

func TestReadPasswordFileRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")
	if err := os.WriteFile(target, []byte("sentinel-password"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readPasswordFile(link); err == nil {
		t.Fatal("expected symlink rejection")
	}
}

func TestSafeErrorAndTargetDoNotExposeSecrets(t *testing.T) {
	secret := "do-not-print-this-secret"
	target, err := sanitizedTarget("postgres", 5432, "auth_db")
	if err != nil {
		t.Fatal(err)
	}
	output := safeError(&secretError{secret: secret}) + " " + target
	if strings.Contains(output, secret) {
		t.Fatalf("secret leaked: %q", output)
	}
}

func TestSanitizedTargetUsesSafeIPv6Formatting(t *testing.T) {
	target, err := sanitizedTarget("2001:db8::1", 5432, "auth_db")
	if err != nil || target != "[2001:db8::1]:5432/auth_db" {
		t.Fatalf("target=%q err=%v", target, err)
	}
	for _, input := range []struct {
		host string
		port int
		db   string
	}{{"", 5432, "auth_db"}, {"postgres\n", 5432, "auth_db"}, {"postgres", 0, "auth_db"}, {"postgres", 5432, "auth/db"}} {
		if _, err := sanitizedTarget(input.host, input.port, input.db); err == nil {
			t.Fatalf("expected invalid target for %#v", input)
		}
	}
}

func TestConfirmationBindsExactTargetAndRange(t *testing.T) {
	identities, err := benchmark.Identities(1, 50)
	if err != nil {
		t.Fatal(err)
	}
	got := confirmationPhrase(identities, "postgres:5432/auth_db")
	const want = "CREATE benchmark_judge_001..benchmark_judge_050 ON postgres:5432/auth_db"
	if got != want {
		t.Fatalf("confirmation=%q", got)
	}
	other, _ := benchmark.Identities(2, 49)
	if confirmationPhrase(other, "postgres:5432/auth_db") == got || confirmationPhrase(identities, "postgres:5432/other") == got {
		t.Fatal("confirmation must bind both range and target")
	}
}

func TestRunVersionDoesNotLoadConfiguration(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run(t.Context(), []string{"--version"}, os.Stdin, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "version=") || stderr.Len() != 0 {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestInteractivePasswordRejectsNonTTYExplicitly(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := acquirePassword(context.Background(), file, io.Discard, ""); !errors.Is(err, errInteractivePasswordTTY) {
		t.Fatalf("error=%v", err)
	}
	if got := safeError(errInteractivePasswordTTY); got != errInteractivePasswordTTY.Error() {
		t.Fatalf("safe error=%q", got)
	}
}

type secretError struct{ secret string }

func (e *secretError) Error() string { return e.secret }

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
	"time"

	"go-judge-system/pkg/rbac"
	"go-judge-system/services/auth/internal/application/port/outbound"
	benchmark "go-judge-system/services/auth/internal/application/usecase/benchmark"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/entity"
)

func TestParseOptionsDryRunRejectsPasswordInputs(t *testing.T) {
	if _, err := parseOptions([]string{"--password-file", "secret"}, os.Stderr); err == nil {
		t.Fatal("expected dry-run password-file rejection")
	}
	if _, err := parseOptions([]string{"--apply", "--password-file", "secret"}, os.Stderr); err != nil {
		t.Fatalf("apply password-file: %v", err)
	}
}

func TestParseOptionsRotationRequiresExplicitSafeMode(t *testing.T) {
	for _, args := range [][]string{
		{"--rotate-password"},
		{"--rotate-password", "--apply"},
		{"--rotate-password", "--apply", "--password-file", "secret"},
		{"--rotate-password", "--dry-run", "--password-file", "secret"},
	} {
		if _, err := parseOptions(args, os.Stderr); err == nil {
			t.Fatalf("expected rotation rejection for %v", args)
		}
	}
	if _, err := parseOptions([]string{"--rotate-password", "--dry-run"}, os.Stderr); err != nil {
		t.Fatalf("explicit rotation dry-run: %v", err)
	}
	if _, err := parseOptions([]string{"--rotate-password", "--apply", "--password-file", "secret", "--confirm", "ROTATE PASSWORD example"}, os.Stderr); err != nil {
		t.Fatalf("rotation apply: %v", err)
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
	got := confirmationPhrase(identities, "postgres:5432/auth_db", false)
	const want = "CREATE benchmark_judge_001..benchmark_judge_050 ON postgres:5432/auth_db"
	if got != want {
		t.Fatalf("confirmation=%q", got)
	}
	other, _ := benchmark.Identities(2, 49)
	if confirmationPhrase(other, "postgres:5432/auth_db", false) == got || confirmationPhrase(identities, "postgres:5432/other", false) == got {
		t.Fatal("confirmation must bind both range and target")
	}
	rotate := confirmationPhrase(identities, "postgres:5432/auth_db", true)
	const wantRotate = "ROTATE PASSWORD benchmark_judge_001..benchmark_judge_050 ON postgres:5432/auth_db"
	if rotate != wantRotate || rotate == got {
		t.Fatalf("rotation confirmation=%q", rotate)
	}
}

func TestRotationCreateConfirmationCannotAuthorizePasswordRotation(t *testing.T) {
	identities, err := benchmark.Identities(51, 1)
	if err != nil {
		t.Fatal(err)
	}
	target := "postgres:5432/auth_db"
	createConfirmation := confirmationPhrase(identities, target, false)
	rotateConfirmation := confirmationPhrase(identities, target, true)
	repo := &commandTestRepository{user: &entity.User{ID: "id-051", Username: identities[0].Username, Email: identities[0].Email, FullName: identities[0].FullName, Role: rbac.RoleUser, IsActive: true}}
	var stdout, stderr bytes.Buffer
	err = runWithRuntime(t.Context(), []string{
		"--rotate-password", "--apply", "--start", "51", "--count", "1",
		"--password-file", filepath.Join(t.TempDir(), "must-not-be-read"),
		"--confirm", createConfirmation,
	}, os.Stdin, &stdout, &stderr, func(string) (commandRuntime, error) {
		return commandRuntime{
			target:      target,
			provisioner: benchmark.NewProvisioner(repo, nil),
			close:       func() {},
		}, nil
	})
	if err == nil || !strings.Contains(err.Error(), "confirmation") {
		t.Fatalf("CREATE confirmation authorized password rotation: %v", err)
	}
	if !strings.Contains(stderr.String(), rotateConfirmation) || strings.Contains(stderr.String(), createConfirmation) {
		t.Fatalf("unexpected confirmation output: %q", stderr.String())
	}
	if repo.rotations != 0 {
		t.Fatalf("rotation ran despite rejected confirmation: %d", repo.rotations)
	}
	if strings.Contains(err.Error(), "password file") {
		t.Fatalf("password acquisition ran before confirmation rejection: %v", err)
	}
}

func TestRotationDryRunOutputUsesDistinctConfirmationAndNeverPassword(t *testing.T) {
	identities, err := benchmark.Identities(51, 2)
	if err != nil {
		t.Fatal(err)
	}
	confirmation := confirmationPhrase(identities, "postgres:5432/auth_db", true)
	var output bytes.Buffer
	printResult(&output, false, true, "postgres:5432/auth_db", identities, confirmation, benchmark.Result{Entries: []benchmark.Entry{{Identity: identities[0], Status: benchmark.StatusWouldRotate}, {Identity: identities[1], Status: benchmark.StatusWouldRotate}}}, time.Millisecond)
	text := output.String()
	if !strings.Contains(text, "Mode: rotate-password dry-run") || !strings.Contains(text, "Would rotate: 2") || !strings.Contains(text, confirmation) || strings.Contains(text, "sentinel-password") {
		t.Fatalf("unexpected rotation output: %q", text)
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

type commandTestRepository struct {
	outbound.UserRepository
	user      *entity.User
	rotations int
}

func (r *commandTestRepository) GetUserByUsername(_ context.Context, username string) (*entity.User, error) {
	if r.user != nil && r.user.Username == username {
		return r.user, nil
	}
	return nil, domain.ErrUserNotFound
}

func (r *commandTestRepository) GetUserByEmail(_ context.Context, email string) (*entity.User, error) {
	if r.user != nil && r.user.Email == email {
		return r.user, nil
	}
	return nil, domain.ErrUserNotFound
}

func (r *commandTestRepository) RotateBenchmarkPasswords(context.Context, []outbound.BenchmarkPasswordUpdate) error {
	r.rotations++
	return nil
}

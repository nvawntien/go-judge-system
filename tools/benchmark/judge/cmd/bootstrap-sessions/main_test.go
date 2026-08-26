package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRejectsPasswordArgumentAndInvalidArguments(t *testing.T) {
	for _, args := range [][]string{
		{"--base-url", "http://127.0.0.1:8080", "--password", "secret"},
		{"--base-url", "http://127.0.0.1:8080", "unexpected"},
		{"--base-url", "http://127.0.0.1:8080", "--login-delay", "-1s"},
	} {
		if _, err := parse(args, os.Stderr); err == nil {
			t.Fatalf("parse(%q) succeeded", args)
		}
	}
}

func TestReadPasswordFileAcceptsSecureSinglePassword(t *testing.T) {
	path := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(path, []byte("secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	password, err := readPasswordFile(path)
	if err != nil || string(password) != "secret" {
		t.Fatalf("password=%q err=%v", password, err)
	}
	clear(password)
}

func TestReadPasswordFileRejectsUnsafeInputs(t *testing.T) {
	directory := t.TempDir()
	for name, test := range map[string]struct {
		content string
		mode    os.FileMode
	}{
		"group-readable": {"secret\n", 0o640},
		"multiple-lines": {"first\nsecond\n", 0o600},
		"oversized":      {strings.Repeat("x", maxPasswordBytes+1), 0o600},
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(directory, name)
			if err := os.WriteFile(path, []byte(test.content), test.mode); err != nil {
				t.Fatal(err)
			}
			if _, err := readPasswordFile(path); err == nil {
				t.Fatal("unsafe password file accepted")
			}
		})
	}
	target := filepath.Join(directory, "target")
	if err := os.WriteFile(target, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readPasswordFile(link); err == nil {
		t.Fatal("password file symlink accepted")
	}
}

func TestAcquirePasswordRejectsNonTTYWithoutPasswordFile(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	_, err = acquirePassword(context.Background(), file, os.Stderr, "")
	if !errors.Is(err, errInteractiveTTY) {
		t.Fatalf("err=%v", err)
	}
}

func TestSafeErrorNeverEchoesSecret(t *testing.T) {
	secret := "password-sentinel"
	if value := safeError(errors.New(secret)); strings.Contains(value, secret) {
		t.Fatalf("safeError leaked secret: %q", value)
	}
}

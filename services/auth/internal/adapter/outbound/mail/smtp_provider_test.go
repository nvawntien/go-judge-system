package mail

import (
	"strings"
	"testing"

	"go-judge-system/pkg/config"

	"go.uber.org/zap"
)

func TestSMTPProviderFrontendTokenURL(t *testing.T) {
	tests := []struct {
		name        string
		frontendURL string
		token       string
		want        string
	}{
		{
			name:        "local url without trailing slash",
			frontendURL: "http://localhost:3000",
			token:       "abc123",
			want:        "http://localhost:3000/verify-email#token=abc123",
		},
		{
			name:        "trailing slash does not create double slash",
			frontendURL: "http://localhost:3000/",
			token:       "abc123",
			want:        "http://localhost:3000/verify-email#token=abc123",
		},
		{
			name:        "token is url encoded",
			frontendURL: "https://judge.example.com",
			token:       "raw token+/=",
			want:        "https://judge.example.com/verify-email#token=raw+token%2B%2F%3D",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := newTestSMTPProvider(tt.frontendURL)

			got, err := provider.frontendTokenURL("/verify-email", tt.token, true)
			if err != nil {
				t.Fatalf("frontendTokenURL() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("frontendTokenURL() = %q, want %q", got, tt.want)
			}
			if strings.Contains(got, "//verify-email") {
				t.Fatalf("frontendTokenURL() created a double slash: %q", got)
			}
		})
	}
}

func TestSMTPProviderFrontendTokenURLRejectsInvalidBase(t *testing.T) {
	provider := newTestSMTPProvider("localhost:3000")

	if _, err := provider.frontendTokenURL("/verify-email", "abc123", true); err == nil {
		t.Fatal("expected invalid frontend URL error")
	}
}

func TestSMTPProviderFrontendTokenURLCanUseQueryForPasswordReset(t *testing.T) {
	provider := newTestSMTPProvider("http://localhost:3000/")

	got, err := provider.frontendTokenURL("/reset-password", "abc123", false)
	if err != nil {
		t.Fatalf("frontendTokenURL() error = %v", err)
	}

	want := "http://localhost:3000/reset-password?token=abc123"
	if got != want {
		t.Fatalf("frontendTokenURL() = %q, want %q", got, want)
	}
}

func newTestSMTPProvider(frontendURL string) *smtpProvider {
	return NewSMTPProvider(
		config.SMTPConfig{
			Host:     "127.0.0.1",
			Port:     1025,
			FromName: "Judge System",
			From:     "noreply@example.test",
		},
		config.AppConfig{FrontendURL: frontendURL},
		zap.NewNop(),
	).(*smtpProvider)
}

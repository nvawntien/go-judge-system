package mail

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"

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

func TestSMTPProviderSendsUnauthenticatedMailToLocalServer(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("local TCP listeners are unavailable in this environment: %v", err)
	}
	defer listener.Close()

	received := make(chan string, 1)
	serverErr := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer conn.Close()
		reader := bufio.NewReader(conn)
		if _, err := conn.Write([]byte("220 local test SMTP\r\n")); err != nil {
			serverErr <- err
			return
		}
		var data strings.Builder
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				serverErr <- err
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO "):
				_, err = conn.Write([]byte("250-localhost\r\n250 OK\r\n"))
			case strings.HasPrefix(line, "MAIL FROM:"), strings.HasPrefix(line, "RCPT TO:"):
				_, err = conn.Write([]byte("250 OK\r\n"))
			case strings.HasPrefix(line, "DATA"):
				_, err = conn.Write([]byte("354 send data\r\n"))
				if err == nil {
					for {
						bodyLine, readErr := reader.ReadString('\n')
						if readErr != nil {
							serverErr <- readErr
							return
						}
						if bodyLine == ".\r\n" {
							break
						}
						data.WriteString(bodyLine)
					}
					_, err = conn.Write([]byte("250 queued\r\n"))
				}
			case strings.HasPrefix(line, "QUIT"):
				_, err = conn.Write([]byte("221 bye\r\n"))
				received <- data.String()
				return
			default:
				err = nil
			}
			if err != nil {
				serverErr <- err
				return
			}
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	provider := NewSMTPProvider(config.SMTPConfig{
		Host: "127.0.0.1", Port: port, Security: "none", Timeout: time.Second,
		FromName: "AstraCode", From: "noreply@example.test",
	}, config.AppConfig{FrontendURL: "http://localhost:3000"}, zap.NewNop()).(*smtpProvider)
	if err := provider.sendMail(context.Background(), "user@example.test", "Test subject", []byte("<p>hello</p>")); err != nil {
		t.Fatalf("sendMail() error = %v", err)
	}

	select {
	case err := <-serverErr:
		t.Fatalf("local SMTP server error = %v", err)
	case message := <-received:
		if !strings.Contains(message, "Subject: Test subject") || !strings.Contains(message, "<p>hello</p>") {
			t.Fatalf("SMTP message = %q", message)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("local SMTP server did not receive a complete message")
	}
}

func TestValidateConfig(t *testing.T) {
	validSMTP := config.SMTPConfig{
		Host:     "smtp.example.test",
		Port:     587,
		Username: "smtp-user",
		Password: "smtp-password",
		Security: "starttls",
		Timeout:  10 * time.Second,
		FromName: "AstraCode",
		From:     "no-reply@example.test",
	}
	validApp := config.AppConfig{FrontendURL: "https://astracode.example/"}
	production := config.ServerConfig{Mode: "release"}

	tests := []struct {
		name string
		smtp config.SMTPConfig
		app  config.AppConfig
		mode config.ServerConfig
		want string
	}{
		{name: "authenticated STARTTLS production configuration", smtp: validSMTP, app: validApp, mode: production},
		{
			name: "development MailHog without authentication",
			smtp: config.SMTPConfig{Host: "mailhog", Port: 1025, Security: "none", Timeout: 10 * time.Second, FromName: "AstraCode", From: "noreply@localhost"},
			app:  config.AppConfig{FrontendURL: "http://localhost:3000"},
			mode: config.ServerConfig{Mode: "debug"},
		},
		{name: "release rejects insecure SMTP", smtp: func() config.SMTPConfig { c := validSMTP; c.Security = "none"; return c }(), app: validApp, mode: production, want: "authentication requires"},
		{name: "release requires authentication", smtp: func() config.SMTPConfig { c := validSMTP; c.Username, c.Password = "", ""; return c }(), app: validApp, mode: production, want: "authentication"},
		{name: "missing sender is rejected", smtp: func() config.SMTPConfig { c := validSMTP; c.From = ""; return c }(), app: validApp, mode: production, want: "smtp.from"},
		{name: "missing sender name is rejected", smtp: func() config.SMTPConfig { c := validSMTP; c.FromName = ""; return c }(), app: validApp, mode: production, want: "smtp.from_name"},
		{name: "release requires HTTPS public URL", smtp: validSMTP, app: config.AppConfig{FrontendURL: "http://localhost:3000"}, mode: production, want: "https"},
		{name: "public URL must be an origin", smtp: validSMTP, app: config.AppConfig{FrontendURL: "https://astracode.example/app"}, mode: production, want: "frontend_url"},
		{name: "unsupported SMTP security is rejected", smtp: func() config.SMTPConfig { c := validSMTP; c.Security = "opportunistic"; return c }(), app: validApp, mode: production, want: "smtp.security"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.smtp, tt.app, tt.mode)
			if tt.want == "" && err != nil {
				t.Fatalf("ValidateConfig() error = %v", err)
			}
			if tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)) {
				t.Fatalf("ValidateConfig() error = %v, want containing %q", err, tt.want)
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
			Security: "none",
			Timeout:  10 * time.Second,
			FromName: "Judge System",
			From:     "noreply@example.test",
		},
		config.AppConfig{FrontendURL: frontendURL},
		zap.NewNop(),
	).(*smtpProvider)
}

package stream

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"go-judge-system/pkg/config"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"
)

func TestHMACSubmissionStreamTicketIssueAndVerify(t *testing.T) {
	now := time.Date(2026, 7, 30, 9, 0, 0, 0, time.UTC)
	service := newHMACSubmissionStreamTicketService(config.SSEConfig{
		TicketSecret: "test-secret",
		TicketTTL:    2 * time.Minute,
	}, func() time.Time { return now })

	ticket, expiresAt, err := service.Issue("user-123", 77)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if expiresAt != now.Add(2*time.Minute) {
		t.Fatalf("expiresAt = %v, want %v", expiresAt, now.Add(2*time.Minute))
	}

	claims, err := service.Verify(ticket)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if claims.Purpose != entity.SubmissionStreamTicketPurpose ||
		claims.UserID != "user-123" ||
		claims.SubmissionID != 77 ||
		!claims.IssuedAt.Equal(now) ||
		!claims.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("claims = %+v", claims)
	}
}

func TestHMACSubmissionStreamTicketRejectsInvalidTickets(t *testing.T) {
	now := time.Date(2026, 7, 30, 9, 0, 0, 0, time.UTC)
	cfg := config.SSEConfig{TicketSecret: "test-secret", TicketTTL: time.Minute}
	service := newHMACSubmissionStreamTicketService(cfg, func() time.Time { return now }).(*hmacSubmissionStreamTicketService)
	ticket, _, err := service.Issue("user-123", 77)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	segments := strings.Split(ticket, ".")
	var payload entity.SubmissionStreamTicketClaims
	payloadBytes, err := base64.RawURLEncoding.DecodeString(segments[0])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		t.Fatalf("decode claims: %v", err)
	}
	payload.Purpose = "rest_api"
	wrongPurposePayload, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal wrong purpose: %v", err)
	}
	wrongPurpose := base64.RawURLEncoding.EncodeToString(wrongPurposePayload) + "." +
		base64.RawURLEncoding.EncodeToString(service.sign(wrongPurposePayload))

	tests := []struct {
		name string
		raw  string
	}{
		{name: "missing segment", raw: "only-one-segment"},
		{name: "malformed payload base64", raw: "%%." + segments[1]},
		{name: "malformed signature base64", raw: segments[0] + ".%%"},
		{name: "wrong purpose", raw: wrongPurpose},
		{name: "tampered payload", raw: segments[0][:len(segments[0])-1] + "A." + segments[1]},
		{name: "tampered signature", raw: segments[0] + "." + segments[1][:len(segments[1])-1] + "A"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Verify(tt.raw)
			if !errors.Is(err, domain.ErrInvalidStreamTicket) {
				t.Fatalf("Verify() error = %v, want invalid stream ticket", err)
			}
		})
	}
}

func TestHMACSubmissionStreamTicketRejectsExpiredTicket(t *testing.T) {
	now := time.Date(2026, 7, 30, 9, 0, 0, 0, time.UTC)
	current := now
	service := newHMACSubmissionStreamTicketService(config.SSEConfig{
		TicketSecret: "test-secret",
		TicketTTL:    time.Minute,
	}, func() time.Time { return current })

	ticket, _, err := service.Issue("user-123", 77)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	current = now.Add(time.Minute)

	_, err = service.Verify(ticket)
	if !errors.Is(err, domain.ErrExpiredStreamTicket) {
		t.Fatalf("Verify() error = %v, want expired stream ticket", err)
	}
}

func TestHMACSubmissionStreamTicketRejectsInvalidConfigAndInput(t *testing.T) {
	tests := []struct {
		name         string
		cfg          config.SSEConfig
		userID       string
		submissionID int64
	}{
		{name: "empty secret", cfg: config.SSEConfig{TicketTTL: time.Minute}, userID: "user", submissionID: 1},
		{name: "zero ttl", cfg: config.SSEConfig{TicketSecret: "secret"}, userID: "user", submissionID: 1},
		{name: "empty user", cfg: config.SSEConfig{TicketSecret: "secret", TicketTTL: time.Minute}, submissionID: 1},
		{name: "invalid submission", cfg: config.SSEConfig{TicketSecret: "secret", TicketTTL: time.Minute}, userID: "user"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newHMACSubmissionStreamTicketService(tt.cfg, time.Now)
			_, _, err := service.Issue(tt.userID, tt.submissionID)
			if !errors.Is(err, domain.ErrInvalidStreamTicket) {
				t.Fatalf("Issue() error = %v, want invalid stream ticket", err)
			}
		})
	}
}

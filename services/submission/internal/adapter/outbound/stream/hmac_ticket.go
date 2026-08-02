package stream

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"go-judge-system/pkg/config"
	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain"
	"go-judge-system/services/submission/internal/domain/entity"
)

var base64RawURL = base64.RawURLEncoding

type hmacSubmissionStreamTicketService struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewHMACSubmissionStreamTicketService(cfg config.SSEConfig) outbound.SubmissionStreamTicketService {
	return newHMACSubmissionStreamTicketService(cfg, time.Now)
}

func newHMACSubmissionStreamTicketService(
	cfg config.SSEConfig,
	now func() time.Time,
) outbound.SubmissionStreamTicketService {
	return &hmacSubmissionStreamTicketService{
		secret: []byte(strings.TrimSpace(cfg.TicketSecret)),
		ttl:    cfg.TicketTTL,
		now:    now,
	}
}

func (s *hmacSubmissionStreamTicketService) Issue(
	userID string,
	submissionID int64,
) (string, time.Time, error) {
	userID = strings.TrimSpace(userID)
	if len(s.secret) == 0 || s.ttl <= 0 || userID == "" || submissionID <= 0 {
		return "", time.Time{}, domain.ErrInvalidStreamTicket
	}

	issuedAt := s.now().UTC()
	expiresAt := issuedAt.Add(s.ttl)
	claims := entity.SubmissionStreamTicketClaims{
		Purpose:      entity.SubmissionStreamTicketPurpose,
		UserID:       userID,
		SubmissionID: submissionID,
		IssuedAt:     issuedAt,
		ExpiresAt:    expiresAt,
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, domain.ErrInternalServer.Wrap(err)
	}

	return base64RawURL.EncodeToString(payload) + "." + base64RawURL.EncodeToString(s.sign(payload)), expiresAt, nil
}

func (s *hmacSubmissionStreamTicketService) Verify(
	ticket string,
) (entity.SubmissionStreamTicketClaims, error) {
	if len(s.secret) == 0 {
		return entity.SubmissionStreamTicketClaims{}, domain.ErrInvalidStreamTicket
	}

	segments := strings.Split(ticket, ".")
	if len(segments) != 2 || segments[0] == "" || segments[1] == "" {
		return entity.SubmissionStreamTicketClaims{}, domain.ErrInvalidStreamTicket
	}

	payload, err := base64RawURL.DecodeString(segments[0])
	if err != nil {
		return entity.SubmissionStreamTicketClaims{}, domain.ErrInvalidStreamTicket
	}
	signature, err := base64RawURL.DecodeString(segments[1])
	if err != nil {
		return entity.SubmissionStreamTicketClaims{}, domain.ErrInvalidStreamTicket
	}

	expectedSignature := s.sign(payload)
	if !hmac.Equal(signature, expectedSignature) {
		return entity.SubmissionStreamTicketClaims{}, domain.ErrInvalidStreamTicket
	}

	var claims entity.SubmissionStreamTicketClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return entity.SubmissionStreamTicketClaims{}, domain.ErrInvalidStreamTicket
	}
	if claims.Purpose != entity.SubmissionStreamTicketPurpose ||
		strings.TrimSpace(claims.UserID) == "" ||
		claims.SubmissionID <= 0 ||
		claims.IssuedAt.IsZero() ||
		claims.ExpiresAt.IsZero() ||
		!claims.IssuedAt.Before(claims.ExpiresAt) {
		return entity.SubmissionStreamTicketClaims{}, domain.ErrInvalidStreamTicket
	}
	if !s.now().UTC().Before(claims.ExpiresAt) {
		return entity.SubmissionStreamTicketClaims{}, domain.ErrExpiredStreamTicket
	}

	return claims, nil
}

func (s *hmacSubmissionStreamTicketService) sign(payload []byte) []byte {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write(payload)
	return mac.Sum(nil)
}

package valueobject

import (
	"net/mail"
	"strings"
	"go-judge-system/services/auth/internal/domain"
)

type Email struct{ value string }

func NewEmail(address string) (Email, error) {
	parsed, err := mail.ParseAddress(strings.TrimSpace(address))
	if err != nil {
		return Email{}, domain.ErrInvalidEmail
	}
	return Email{value: strings.ToLower(parsed.Address)}, nil
}
func (e Email) String() string { return e.value }
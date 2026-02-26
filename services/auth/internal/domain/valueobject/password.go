package valueobject

import "go-judge-system/services/auth/internal/domain"

const minPasswordLength = 8

type Password struct {
	hash string
}

func NewPasswordFromHash(hash string) Password {
	return Password{hash: hash}
}

func ValidatePlainPassword(plain string) error {
	if len(plain) < minPasswordLength {
		return domain.ErrPasswordTooShort
	}
	return nil
}

func (p Password) Hash() string {
	return p.hash
}
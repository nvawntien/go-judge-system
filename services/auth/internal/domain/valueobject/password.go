package valueobject

import (
	"go-judge-system/services/auth/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

type Password struct {
	hash string
}

const minPasswordLength = 8

func NewPasswordFromPlain(plain string) (Password, error) {
	if len(plain) < minPasswordLength {
		return Password{}, domain.ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(plain),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return Password{}, err
	}

	return Password{hash: string(hash)}, nil
}

func NewPasswordFromHash(hash string) Password {
	return Password{hash: hash}
}


func (p Password) Hash() string {
	return p.hash
}

func (p Password) Match(plain string) bool {
	err := bcrypt.CompareHashAndPassword(
		[]byte(p.hash),
		[]byte(plain),
	)
	return err == nil
}

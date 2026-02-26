package security

import (
	"go-judge-system/services/auth/internal/application/port/outbound"

	"golang.org/x/crypto/bcrypt"
)

type bcryptHasher struct {
}

func NewBcryptHasher() outbound.PasswordHasher {
	return &bcryptHasher{}
}

func (h *bcryptHasher) Hash(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword(
		[]byte(plain),
		bcrypt.DefaultCost,
	)
	return string(hash), err
}

func (h *bcryptHasher) Compare(hashedPassword, plain string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(
		[]byte(hashedPassword),
		[]byte(plain),
	)
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

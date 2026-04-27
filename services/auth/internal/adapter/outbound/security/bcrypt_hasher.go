package security

import (
	"go-judge-system/services/auth/internal/application/port/outbound"

	"golang.org/x/crypto/bcrypt"
)

type bcryptHasher struct {
}

func NewBcryptHasher() outbound.PasswordEncoder {
	return &bcryptHasher{}
}

// ref > https://medium.com/@jcox250/password-hash-salt-using-golang-b041dc94cb72
func (b *bcryptHasher) HashAndSalt(pwd []byte) (string, error) {
	// Use GenerateFromPassword to hash & salt pwd
	// MinCost is just an integer constant provided by the bcrypt
	// package along with DefaultCost & MaxCost.
	// The cost can be any value you want provided it isn't lower
	// than the MinCost (4)
	hash, err := bcrypt.GenerateFromPassword(pwd, bcrypt.MinCost)
	if err != nil {
		return "", err
	}
	// GenerateFromPassword returns a byte slice so we need to
	// convert the bytes to a string and return it
	return string(hash), nil
}

func (b *bcryptHasher) ComparePasswords(hashedPwd string, plainPwd []byte) bool {
	// Since we'll be getting the hashed password from the DB it
	// will be a string so we'll need to convert it to a byte slice
	byteHash := []byte(hashedPwd)
	err := bcrypt.CompareHashAndPassword(byteHash, plainPwd)
	if err != nil {
		return false
	}

	return true
}

package outbound

type PasswordEncoder interface {
	HashAndSalt(pwd []byte) (string, error)
	ComparePasswords(hashedPwd string, plainPwd []byte) bool
}
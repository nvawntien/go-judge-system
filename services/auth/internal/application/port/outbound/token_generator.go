package outbound

type TokenGenerator interface {
	GenerateSecureToken(userID string) string
	HashToken(token string) string
}
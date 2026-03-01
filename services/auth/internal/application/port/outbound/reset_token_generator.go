package outbound

type ResetTokenGenerator interface {
	Generate(userID string) string
	Hash(token string) string
}
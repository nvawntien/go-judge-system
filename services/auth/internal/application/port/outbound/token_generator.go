package outbound

type TokenGenerator interface {
	Generate(identifier string) string
	Hash(token string) string
}

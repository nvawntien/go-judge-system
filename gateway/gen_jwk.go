package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
)

type JWK struct {
	Kty string `json:"kty"`
	K   string `json:"k"`
	Alg string `json:"alg"`
	Use string `json:"use"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

func main() {
	secret := os.Getenv("JWT_ACCESS_SECRET")
	if secret == "" {
		fmt.Println("JWT_ACCESS_SECRET is required")
		os.Exit(1)
	}

	// Base64Url encode the secret without padding
	encoded := base64.RawURLEncoding.EncodeToString([]byte(secret))

	jwks := JWKS{
		Keys: []JWK{
			{
				Kty: "oct",
				K:   encoded,
				Alg: "HS256",
				Use: "sig",
			},
		},
	}

	data, err := json.MarshalIndent(jwks, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling JWKS:", err)
		os.Exit(1)
	}

	outputPath := os.Getenv("JWK_OUTPUT_PATH")
	if outputPath == "" {
		outputPath = "symmetric.json"
	}

	err = os.WriteFile(outputPath, data, 0600)
	if err != nil {
		fmt.Println("Error writing file:", err)
		os.Exit(1)
	}
	if err := os.Chmod(outputPath, 0600); err != nil {
		fmt.Println("Error securing output file:", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s successfully.\n", outputPath)
}

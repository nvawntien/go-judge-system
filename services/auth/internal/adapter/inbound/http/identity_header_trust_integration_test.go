package http

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// This black-box test targets Envoy, never Auth directly. The supplied JWK
// must belong to an isolated non-production stack; it is read only to create
// JWTs that KrakenD validates before it derives X-User-* headers.
//
// Example:
// AUTH_IDENTITY_HEADER_TRUST_INTEGRATION_BASE_URL=http://127.0.0.1:8080 \
// AUTH_IDENTITY_HEADER_TRUST_INTEGRATION_JWK_PATH=../../../../../../gateway/symmetric.json \
// go test ./internal/adapter/inbound/http -run TestIdentityHeaderTrustIntegration -count=1
func TestIdentityHeaderTrustIntegration(t *testing.T) {
	baseURL := strings.TrimRight(os.Getenv("AUTH_IDENTITY_HEADER_TRUST_INTEGRATION_BASE_URL"), "/")
	jwkPath := os.Getenv("AUTH_IDENTITY_HEADER_TRUST_INTEGRATION_JWK_PATH")
	if baseURL == "" || jwkPath == "" {
		t.Skip("set AUTH_IDENTITY_HEADER_TRUST_INTEGRATION_BASE_URL and AUTH_IDENTITY_HEADER_TRUST_INTEGRATION_JWK_PATH for an isolated stack")
	}

	secret := identityHeaderTrustJWKSecret(t, jwkPath)
	forged := http.Header{
		"X-User-ID":   {"victim-user-id"},
		"X-Username":  {"admin"},
		"X-Role":      {"admin"},
		"X-Token-Iat": {"9999999999"},
	}

	t.Run("missing_or_invalid_jwt_cannot_use_forged_identity", func(t *testing.T) {
		identityHeaderTrustExpectStatus(t, baseURL, "/api/v1/admin/users", "", forged, http.StatusUnauthorized)
		identityHeaderTrustExpectStatus(t, baseURL, "/api/v1/admin/users", "not-a-jwt", forged, http.StatusUnauthorized)
	})

	t.Run("normal_user_cannot_escalate_to_admin", func(t *testing.T) {
		token := identityHeaderTrustToken(t, secret, "identity-trust-user", "normal-user", "user", time.Now().Unix())
		identityHeaderTrustExpectStatus(t, baseURL, "/api/v1/admin/users", token, forged, http.StatusForbidden)
	})

	t.Run("admin_claim_wins_over_forged_lower_role", func(t *testing.T) {
		token := identityHeaderTrustToken(t, secret, "identity-trust-admin", "admin-user", "admin", time.Now().Unix())
		headers := forged.Clone()
		headers.Set("X-Role", "user")
		identityHeaderTrustExpectStatus(t, baseURL, "/api/v1/admin/users", token, headers, http.StatusOK)
	})

	t.Run("trusted_subject_and_iat_control_logout_all", func(t *testing.T) {
		issuedAt := time.Now().Add(-time.Minute).Unix()
		userID := fmt.Sprintf("identity-trust-%s-%d", strings.ReplaceAll(t.Name(), "/", "-"), time.Now().UnixNano())
		token := identityHeaderTrustToken(t, secret, userID, "normal-user", "user", issuedAt)

		identityHeaderTrustExpectStatus(t, baseURL, "/api/v1/auth/logout-all", token, forged, http.StatusOK)
		identityHeaderTrustExpectStatus(t, baseURL, "/api/v1/auth/logout", token, forged, http.StatusUnauthorized)
	})
}

func identityHeaderTrustExpectStatus(t *testing.T, baseURL, path, token string, headers http.Header, want int) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if path == "/api/v1/auth/logout-all" || path == "/api/v1/auth/logout" {
		req.Method = http.MethodPost
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for name, values := range headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != want {
		t.Fatalf("%s status=%d want=%d", path, response.StatusCode, want)
	}
}

func identityHeaderTrustToken(t *testing.T, secret []byte, subject, username, role string, issuedAt int64) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      subject,
		"username": username,
		"role":     role,
		"iat":      issuedAt,
		"exp":      time.Now().Add(5 * time.Minute).Unix(),
	})
	signed, err := token.SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func identityHeaderTrustJWKSecret(t *testing.T, path string) []byte {
	t.Helper()
	contents, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	var jwkSet struct {
		Keys []struct {
			Key string `json:"k"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(contents, &jwkSet); err != nil {
		t.Fatal(err)
	}
	if len(jwkSet.Keys) != 1 || jwkSet.Keys[0].Key == "" {
		t.Fatal("expected exactly one symmetric JWK key")
	}
	secret, err := base64.RawURLEncoding.DecodeString(jwkSet.Keys[0].Key)
	if err != nil {
		t.Fatal(err)
	}
	return secret
}

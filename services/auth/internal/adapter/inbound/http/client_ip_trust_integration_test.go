package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// This black-box test must target the public Envoy listener, not KrakenD or
// Auth directly. It proves that rotating client-supplied IP-related headers
// cannot create fresh Login IP+identifier limiter buckets.
//
// Example:
// AUTH_CLIENT_IP_TRUST_INTEGRATION_BASE_URL=http://127.0.0.1:8080 \
// go test ./internal/adapter/inbound/http -run TestClientIPTrustIntegration -count=1
func TestClientIPTrustIntegration(t *testing.T) {
	baseURL := strings.TrimRight(os.Getenv("AUTH_CLIENT_IP_TRUST_INTEGRATION_BASE_URL"), "/")
	if baseURL == "" {
		t.Skip("set AUTH_CLIENT_IP_TRUST_INTEGRATION_BASE_URL to the public Envoy listener")
	}

	identifier := fmt.Sprintf("client-ip-trust-%d@example.test", time.Now().UnixNano())
	for attempt, headers := range []map[string]string{
		{"X-Client-IP": "1.1.1.1"},
		{"X-Forwarded-For": "8.8.8.8"},
		{"X-Real-IP": "9.9.9.9"},
		{"X-Client-IP": "2.2.2.2", "X-Forwarded-For": "7.7.7.7", "X-Real-IP": "6.6.6.6"},
		{"X-Client-IP": "3.3.3.3"},
		{"X-Client-IP": "4.4.4.4", "X-Forwarded-For": "5.5.5.5"},
	} {
		response := postClientIPTrustLogin(t, baseURL, identifier, headers)
		if attempt < 5 {
			if response.StatusCode != http.StatusUnauthorized {
				response.Body.Close()
				t.Fatalf("attempt %d status=%d want=%d", attempt+1, response.StatusCode, http.StatusUnauthorized)
			}
			response.Body.Close()
			continue
		}
		if response.StatusCode != http.StatusTooManyRequests {
			response.Body.Close()
			t.Fatalf("rotating spoofed headers bypassed the trusted-IP limiter: status=%d want=%d", response.StatusCode, http.StatusTooManyRequests)
		}
		if response.Header.Get("Retry-After") == "" {
			response.Body.Close()
			t.Fatal("trusted-IP rate limit response is missing Retry-After")
		}
		response.Body.Close()
	}
}

func postClientIPTrustLogin(t *testing.T, baseURL, identifier string, headers map[string]string) *http.Response {
	t.Helper()
	body, err := json.Marshal(map[string]string{"identifier": identifier, "password": "wrong-password"})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/auth/login", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

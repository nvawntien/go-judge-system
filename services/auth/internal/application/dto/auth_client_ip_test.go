package dto

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestAuthRequestsDoNotModelClientIP(t *testing.T) {
	requests := []any{
		LoginRequest{}, RegisterRequest{}, ResendVerificationRequest{},
		ForgotPasswordRequest{}, ResetPasswordRequest{}, VerifyEmailRequest{},
	}
	for _, request := range requests {
		typeOf := reflect.TypeOf(request)
		if _, exists := typeOf.FieldByName("ClientIP"); exists {
			t.Fatalf("%s must not contain transport metadata", typeOf.Name())
		}
	}

	var request LoginRequest
	if err := json.Unmarshal([]byte(`{"identifier":"user","password":"password","client_ip":"198.51.100.1"}`), &request); err != nil {
		t.Fatal(err)
	}
	if request.Identifier != "user" || request.Password != "password" {
		t.Fatalf("unexpected request: %+v", request)
	}
}

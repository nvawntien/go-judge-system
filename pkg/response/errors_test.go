package response

import (
	"net/http"
	"testing"
)

func TestGetHTTPStatusServiceUnavailable(t *testing.T) {
	if got := GetHTTPStatus(CodeServiceUnavailable); got != http.StatusServiceUnavailable {
		t.Fatalf("GetHTTPStatus() = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

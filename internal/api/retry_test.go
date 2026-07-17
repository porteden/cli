package api

import (
	"net/http"
	"testing"
)

// isIdempotent gates whether the retry loop may re-send a request on an
// ambiguous failure. Non-idempotent writes (POST/PATCH) must not be replayed,
// or a dropped connection after the origin acted would duplicate the operation
// (send the same email twice, create two events, etc.).
func TestIsIdempotent(t *testing.T) {
	idempotent := []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions}
	for _, m := range idempotent {
		if !isIdempotent(m) {
			t.Errorf("isIdempotent(%s) = false, want true", m)
		}
	}
	for _, m := range []string{http.MethodPost, http.MethodPatch} {
		if isIdempotent(m) {
			t.Errorf("isIdempotent(%s) = true, want false", m)
		}
	}
}

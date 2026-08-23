package middleware_test

import (
	"testing"

	"gocooking/internal/middleware"
)

func TestSignReturnsNilErrorOnSuccess(t *testing.T) {
	token, err := middleware.Sign("test-secret", 42, "cook")
	if err != nil {
		t.Fatalf("Sign returned a non-nil %T error after generating token %q", err, token)
	}
	if token == "" {
		t.Fatal("Sign returned an empty token for valid claims")
	}
}

package verification

import (
	"testing"
	"time"
)

func TestToken_IsExpired_True(t *testing.T) {
	token := Token{
		ExpiresAt: time.Now().Add(-time.Hour),
	}

	if !token.IsExpired() {
		t.Error("expected token to be expired, but IsExpired() returned false")
	}
}

func TestToken_IsExpired_False(t *testing.T) {
	token := Token{
		ExpiresAt: time.Now().Add(time.Hour),
	}

	if token.IsExpired() {
		t.Error("expected token to not be expired, but IsExpired() returned true")
	}
}

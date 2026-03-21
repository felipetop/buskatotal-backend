package auth

import (
	"context"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

func createTestJWT(secret string, claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, _ := token.SignedString([]byte(secret))
	return str
}

// --------------- JWTProvider tests ---------------

func TestJWTProvider_Authenticate_ValidToken(t *testing.T) {
	secret := "test-secret"
	provider := NewJWTProvider(secret)

	tokenStr := createTestJWT(secret, jwt.MapClaims{
		"userId": "abc123",
		"exp":    time.Now().Add(time.Hour).Unix(),
	})

	result, err := provider.Authenticate(context.Background(), tokenStr)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.UserID != "abc123" {
		t.Errorf("expected userID abc123, got %s", result.UserID)
	}
	if result.Role != "user" {
		t.Errorf("expected role user, got %s", result.Role)
	}
}

func TestJWTProvider_Authenticate_ExpiredToken(t *testing.T) {
	secret := "test-secret"
	provider := NewJWTProvider(secret)

	tokenStr := createTestJWT(secret, jwt.MapClaims{
		"userId": "abc123",
		"exp":    time.Now().Add(-time.Hour).Unix(),
	})

	_, err := provider.Authenticate(context.Background(), tokenStr)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestJWTProvider_Authenticate_InvalidSignature(t *testing.T) {
	provider := NewJWTProvider("correct-secret")

	tokenStr := createTestJWT("wrong-secret", jwt.MapClaims{
		"userId": "abc123",
		"exp":    time.Now().Add(time.Hour).Unix(),
	})

	_, err := provider.Authenticate(context.Background(), tokenStr)
	if err == nil {
		t.Fatal("expected error for invalid signature, got nil")
	}
}

func TestJWTProvider_Authenticate_EmptyToken(t *testing.T) {
	provider := NewJWTProvider("test-secret")

	_, err := provider.Authenticate(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
}

func TestJWTProvider_Authenticate_WithRole(t *testing.T) {
	secret := "test-secret"
	provider := NewJWTProvider(secret)

	tokenStr := createTestJWT(secret, jwt.MapClaims{
		"userId": "abc123",
		"role":   "admin",
		"exp":    time.Now().Add(time.Hour).Unix(),
	})

	result, err := provider.Authenticate(context.Background(), tokenStr)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.UserID != "abc123" {
		t.Errorf("expected userID abc123, got %s", result.UserID)
	}
	if result.Role != "admin" {
		t.Errorf("expected role admin, got %s", result.Role)
	}
}

// --------------- MockProvider tests ---------------

func TestMockProvider_Authenticate_Success(t *testing.T) {
	provider := NewMockProvider("X-User-ID")

	result, err := provider.Authenticate(context.Background(), "user123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.UserID != "user123" {
		t.Errorf("expected userID user123, got %s", result.UserID)
	}
	if result.Role != "user" {
		t.Errorf("expected role user, got %s", result.Role)
	}
}

func TestMockProvider_Authenticate_AdminRole(t *testing.T) {
	provider := NewMockProvider("X-User-ID")

	result, err := provider.Authenticate(context.Background(), "user123:admin")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.UserID != "user123" {
		t.Errorf("expected userID user123, got %s", result.UserID)
	}
	if result.Role != "admin" {
		t.Errorf("expected role admin, got %s", result.Role)
	}
}

func TestMockProvider_Authenticate_EmptyToken(t *testing.T) {
	provider := NewMockProvider("X-User-ID")

	_, err := provider.Authenticate(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
}

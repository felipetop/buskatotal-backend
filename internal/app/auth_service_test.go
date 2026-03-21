package app

import (
	"context"
	"testing"
	"time"

	"buskatotal-backend/internal/domain/email"
	"buskatotal-backend/internal/domain/user"
	"buskatotal-backend/internal/infra/memory"

	"golang.org/x/crypto/bcrypt"
)

// ---------------------------------------------------------------------------
// Mock email sender
// ---------------------------------------------------------------------------

type mockEmailSender struct {
	sent []email.Message
}

func (m *mockEmailSender) Send(_ context.Context, msg email.Message) error {
	m.sent = append(m.sent, msg)
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestAuthService() (*AuthService, *mockEmailSender) {
	userRepo := memory.NewUserRepository()
	verificationRepo := memory.NewVerificationRepository()
	mock := &mockEmailSender{}
	evs := NewEmailVerificationService(verificationRepo, userRepo, mock)
	auth := NewAuthService(userRepo, "test-secret-key-12345", 24*time.Hour, evs)
	return auth, mock
}

func newTestUserService() *UserService {
	return NewUserService(memory.NewUserRepository())
}

// ---------------------------------------------------------------------------
// validatePassword tests
// ---------------------------------------------------------------------------

func TestValidatePassword_Valid(t *testing.T) {
	if err := validatePassword("Abcdef123!@"); err != nil {
		t.Fatalf("expected no error for valid password, got: %v", err)
	}
}

func TestValidatePassword_TooShort(t *testing.T) {
	if err := validatePassword("Ab1!"); err == nil {
		t.Fatal("expected error for short password")
	}
}

func TestValidatePassword_NoUppercase(t *testing.T) {
	if err := validatePassword("abcdef123!@"); err == nil {
		t.Fatal("expected error for password without uppercase")
	}
}

func TestValidatePassword_NoLowercase(t *testing.T) {
	if err := validatePassword("ABCDEF123!@"); err == nil {
		t.Fatal("expected error for password without lowercase")
	}
}

func TestValidatePassword_NoDigit(t *testing.T) {
	if err := validatePassword("Abcdefghi!@"); err == nil {
		t.Fatal("expected error for password without digit")
	}
}

func TestValidatePassword_NoSpecial(t *testing.T) {
	if err := validatePassword("Abcdefghi12"); err == nil {
		t.Fatal("expected error for password without special character")
	}
}

// ---------------------------------------------------------------------------
// Register tests
// ---------------------------------------------------------------------------

func TestRegister_Success(t *testing.T) {
	auth, _ := newTestAuthService()
	ctx := context.Background()

	u, token, err := auth.Register(ctx, "Test User", "test@example.com", "TestPass1@abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID == "" {
		t.Fatal("expected user ID to be set")
	}
	if u.Email != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %s", u.Email)
	}
	if u.Name != "Test User" {
		t.Fatalf("expected name Test User, got %s", u.Name)
	}
	if u.Role != user.RoleUser {
		t.Fatalf("expected role %s, got %s", user.RoleUser, u.Role)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// password hash should be verifiable
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("TestPass1@abc")); err != nil {
		t.Fatal("password hash does not match")
	}
}

func TestRegister_EmptyEmail(t *testing.T) {
	auth, _ := newTestAuthService()
	ctx := context.Background()

	_, _, err := auth.Register(ctx, "User", "", "TestPass1@abc")
	if err == nil {
		t.Fatal("expected error for empty email")
	}
}

func TestRegister_WeakPassword(t *testing.T) {
	auth, _ := newTestAuthService()
	ctx := context.Background()

	_, _, err := auth.Register(ctx, "User", "user@example.com", "weak")
	if err == nil {
		t.Fatal("expected error for weak password")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	auth, _ := newTestAuthService()
	ctx := context.Background()

	_, _, err := auth.Register(ctx, "User1", "dup@example.com", "TestPass1@abc")
	if err != nil {
		t.Fatalf("first registration should succeed: %v", err)
	}

	_, _, err = auth.Register(ctx, "User2", "dup@example.com", "TestPass1@abc")
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
}

func TestRegister_AdminEmail(t *testing.T) {
	auth, _ := newTestAuthService()
	ctx := context.Background()

	u, _, err := auth.Register(ctx, "Admin", "dcparticular2014@gmail.com", "TestPass1@abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Role != user.RoleAdmin {
		t.Fatalf("expected admin role, got %s", u.Role)
	}
}

// ---------------------------------------------------------------------------
// Login tests
// ---------------------------------------------------------------------------

func TestLogin_Success(t *testing.T) {
	auth, _ := newTestAuthService()
	ctx := context.Background()

	// Register first
	_, _, err := auth.Register(ctx, "User", "login@example.com", "TestPass1@abc")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Wait briefly for the goroutine in Register to pick up context
	time.Sleep(50 * time.Millisecond)

	u, token, err := auth.Login(ctx, "login@example.com", "TestPass1@abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Email != "login@example.com" {
		t.Fatalf("expected email login@example.com, got %s", u.Email)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	auth, _ := newTestAuthService()
	ctx := context.Background()

	_, _, _ = auth.Register(ctx, "User", "wrong@example.com", "TestPass1@abc")

	_, _, err := auth.Login(ctx, "wrong@example.com", "WrongPass1@abc")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestLogin_NonexistentEmail(t *testing.T) {
	auth, _ := newTestAuthService()
	ctx := context.Background()

	_, _, err := auth.Login(ctx, "nobody@example.com", "TestPass1@abc")
	if err == nil {
		t.Fatal("expected error for nonexistent email")
	}
}

func TestLogin_EmptyCredentials(t *testing.T) {
	auth, _ := newTestAuthService()
	ctx := context.Background()

	_, _, err := auth.Login(ctx, "", "")
	if err == nil {
		t.Fatal("expected error for empty credentials")
	}
}

// ---------------------------------------------------------------------------
// ForgotPassword tests
// ---------------------------------------------------------------------------

func TestForgotPassword_ExistingEmail(t *testing.T) {
	auth, _ := newTestAuthService()
	ctx := context.Background()

	_, _, _ = auth.Register(ctx, "User", "forgot@example.com", "TestPass1@abc")
	time.Sleep(50 * time.Millisecond)

	err := auth.ForgotPassword(ctx, "forgot@example.com")
	if err != nil {
		t.Fatalf("expected nil for existing email, got: %v", err)
	}
}

func TestForgotPassword_NonexistentEmail(t *testing.T) {
	auth, _ := newTestAuthService()
	ctx := context.Background()

	err := auth.ForgotPassword(ctx, "nonexistent@example.com")
	if err != nil {
		t.Fatalf("expected nil for nonexistent email (never reveal existence), got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// UserService tests
// ---------------------------------------------------------------------------

func TestUserService_Create_Success(t *testing.T) {
	svc := newTestUserService()
	ctx := context.Background()

	u, err := svc.Create(ctx, user.User{
		Name:         "John",
		Email:        "john@example.com",
		PasswordHash: "somehash",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID == "" {
		t.Fatal("expected user ID to be set")
	}
	if u.Name != "John" {
		t.Fatalf("expected name John, got %s", u.Name)
	}
}

func TestUserService_Create_MissingFields(t *testing.T) {
	svc := newTestUserService()
	ctx := context.Background()

	// Missing name
	_, err := svc.Create(ctx, user.User{
		Email:        "a@b.com",
		PasswordHash: "hash",
	})
	if err == nil {
		t.Fatal("expected error for missing name")
	}

	// Missing email
	_, err = svc.Create(ctx, user.User{
		Name:         "Name",
		PasswordHash: "hash",
	})
	if err == nil {
		t.Fatal("expected error for missing email")
	}

	// Missing password hash
	_, err = svc.Create(ctx, user.User{
		Name:  "Name",
		Email: "a@b.com",
	})
	if err == nil {
		t.Fatal("expected error for missing password hash")
	}
}

func TestUserService_Update_MissingID(t *testing.T) {
	svc := newTestUserService()
	ctx := context.Background()

	_, err := svc.Update(ctx, user.User{
		Name:  "Name",
		Email: "a@b.com",
	})
	if err == nil {
		t.Fatal("expected error for missing ID")
	}
}

func TestUserService_List(t *testing.T) {
	svc := newTestUserService()
	ctx := context.Background()

	// Empty list
	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}

	// Create two users
	_, _ = svc.Create(ctx, user.User{Name: "A", Email: "a@b.com", PasswordHash: "h"})
	_, _ = svc.Create(ctx, user.User{Name: "B", Email: "b@b.com", PasswordHash: "h"})

	list, err = svc.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 users, got %d", len(list))
	}
}

func TestUserService_Delete(t *testing.T) {
	svc := newTestUserService()
	ctx := context.Background()

	u, _ := svc.Create(ctx, user.User{Name: "Del", Email: "del@b.com", PasswordHash: "h"})

	err := svc.Delete(ctx, u.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not be found after deletion
	_, err = svc.GetByID(ctx, u.ID)
	if err == nil {
		t.Fatal("expected error after deletion")
	}
}

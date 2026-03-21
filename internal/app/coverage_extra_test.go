package app

import (
	"context"
	"testing"
	"time"

	"buskatotal-backend/internal/domain/email"
	"buskatotal-backend/internal/domain/payment"
	"buskatotal-backend/internal/domain/user"
	"buskatotal-backend/internal/domain/verification"
	"buskatotal-backend/internal/infra/memory"
	paymentinfra "buskatotal-backend/internal/infra/payment"
)

// --- mock email sender ---

type extraMockSender struct{ sent []email.Message }

func (m *extraMockSender) Send(_ context.Context, msg email.Message) error {
	m.sent = append(m.sent, msg)
	return nil
}

// --- helpers ---

const strongPassword = "Str0ng@Pass!"

func createExtraTestUser(t *testing.T, repo *memory.UserRepository) user.User {
	t.Helper()
	u, err := repo.Create(context.Background(), user.User{
		Name:  "Test User",
		Email: "test@example.com",
		Role:  user.RoleUser,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

func createEmailVerificationService(userRepo *memory.UserRepository) (*EmailVerificationService, *memory.VerificationRepository, *extraMockSender) {
	verRepo := memory.NewVerificationRepository()
	sender := &extraMockSender{}
	svc := NewEmailVerificationService(verRepo, userRepo, sender)
	return svc, verRepo, sender
}

// getTokenFromRepo generates a token via GenerateAndSend and retrieves it from
// the verification repo by scanning for the user's token.
func getTokenFromRepo(t *testing.T, svc *EmailVerificationService, verRepo *memory.VerificationRepository, userID, userEmail string) verification.Token {
	t.Helper()
	ctx := context.Background()
	if err := svc.GenerateAndSend(ctx, userID, userEmail); err != nil {
		t.Fatalf("GenerateAndSend: %v", err)
	}
	// The sender captured the email; extract token from verification repo.
	// We scan for the token belonging to this user.
	// There is no List method, but we can use the fact that the sender captured
	// the email with the link containing the token string.
	// Alternatively, we created the token and know it is in the repo.
	// We'll parse the token from the sent email link.
	return extractTokenFromSender(t, svc, verRepo, userID, userEmail)
}

func extractTokenFromSender(t *testing.T, _ *EmailVerificationService, verRepo *memory.VerificationRepository, userID, _ string) verification.Token {
	t.Helper()
	// Delete all tokens then recreate so we know exactly which one exists.
	// Actually, GenerateAndSend already deletes old tokens and creates a new one.
	// We need to find the token. The memory repo stores by ID, but we can use
	// a brute-force approach: create a known token directly.
	// Better approach: just create a token directly in the repo for test isolation.
	t.Fatalf("should not be called")
	return verification.Token{UserID: userID}
}

// createTokenDirectly inserts a token directly into the verification repo.
func createTokenDirectly(t *testing.T, verRepo *memory.VerificationRepository, userID string, expiresAt time.Time, used bool) verification.Token {
	t.Helper()
	tok, err := verRepo.Create(context.Background(), verification.Token{
		UserID:    userID,
		Token:     "test-token-" + userID,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if used {
		if err := verRepo.MarkUsed(context.Background(), tok.ID); err != nil {
			t.Fatalf("mark used: %v", err)
		}
		tok.Used = true
	}
	return tok
}

// =============================================================================
// ResetPassword tests
// =============================================================================

func TestResetPassword_Success(t *testing.T) {
	userRepo := memory.NewUserRepository()
	evSvc, verRepo, _ := createEmailVerificationService(userRepo)
	authSvc := NewAuthService(userRepo, "secret", time.Hour, evSvc)

	u := createExtraTestUser(t, userRepo)
	tok := createTokenDirectly(t, verRepo, u.ID, time.Now().Add(time.Hour), false)

	err := authSvc.ResetPassword(context.Background(), tok.Token, strongPassword)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify the password was actually changed
	updated, _ := userRepo.GetByID(context.Background(), u.ID)
	if updated.PasswordHash == "" {
		t.Fatal("password hash should not be empty after reset")
	}
}

func TestResetPassword_NoEmailService(t *testing.T) {
	userRepo := memory.NewUserRepository()
	authSvc := NewAuthService(userRepo, "secret", time.Hour, nil)

	err := authSvc.ResetPassword(context.Background(), "some-token", strongPassword)
	if err == nil {
		t.Fatal("expected error when email service is nil")
	}
	if err.Error() != "email service not configured" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResetPassword_EmptyToken(t *testing.T) {
	userRepo := memory.NewUserRepository()
	evSvc, _, _ := createEmailVerificationService(userRepo)
	authSvc := NewAuthService(userRepo, "secret", time.Hour, evSvc)

	err := authSvc.ResetPassword(context.Background(), "", strongPassword)
	if err == nil {
		t.Fatal("expected error for empty token")
	}
	if err.Error() != "token and new password are required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResetPassword_WeakPassword(t *testing.T) {
	userRepo := memory.NewUserRepository()
	evSvc, _, _ := createEmailVerificationService(userRepo)
	authSvc := NewAuthService(userRepo, "secret", time.Hour, evSvc)

	err := authSvc.ResetPassword(context.Background(), "some-token", "weak")
	if err == nil {
		t.Fatal("expected error for weak password")
	}
}

// =============================================================================
// ResendVerification tests
// =============================================================================

func TestResendVerification_Success(t *testing.T) {
	userRepo := memory.NewUserRepository()
	evSvc, _, sender := createEmailVerificationService(userRepo)
	authSvc := NewAuthService(userRepo, "secret", time.Hour, evSvc)

	u := createExtraTestUser(t, userRepo)

	err := authSvc.ResendVerification(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(sender.sent) == 0 {
		t.Fatal("expected verification email to be sent")
	}
}

func TestResendVerification_AlreadyVerified(t *testing.T) {
	userRepo := memory.NewUserRepository()
	evSvc, _, _ := createEmailVerificationService(userRepo)
	authSvc := NewAuthService(userRepo, "secret", time.Hour, evSvc)

	u := createExtraTestUser(t, userRepo)
	// Mark as verified
	u.EmailVerified = true
	userRepo.Update(context.Background(), u)

	err := authSvc.ResendVerification(context.Background(), u.ID)
	if err == nil {
		t.Fatal("expected error for already verified user")
	}
	if err.Error() != "email already verified" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResendVerification_NoService(t *testing.T) {
	userRepo := memory.NewUserRepository()
	authSvc := NewAuthService(userRepo, "secret", time.Hour, nil)

	err := authSvc.ResendVerification(context.Background(), "some-id")
	if err == nil {
		t.Fatal("expected error when email verification is nil")
	}
	if err.Error() != "email verification not configured" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// Verify tests
// =============================================================================

func TestVerify_Success(t *testing.T) {
	userRepo := memory.NewUserRepository()
	evSvc, verRepo, _ := createEmailVerificationService(userRepo)

	u := createExtraTestUser(t, userRepo)
	tok := createTokenDirectly(t, verRepo, u.ID, time.Now().Add(time.Hour), false)

	err := evSvc.Verify(context.Background(), tok.Token)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check user is now verified
	updated, _ := userRepo.GetByID(context.Background(), u.ID)
	if !updated.EmailVerified {
		t.Fatal("user should be email verified after Verify")
	}
}

func TestVerify_TokenNotFound(t *testing.T) {
	userRepo := memory.NewUserRepository()
	evSvc, _, _ := createEmailVerificationService(userRepo)

	err := evSvc.Verify(context.Background(), "nonexistent-token")
	if err != verification.ErrTokenNotFound {
		t.Fatalf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestVerify_TokenExpired(t *testing.T) {
	userRepo := memory.NewUserRepository()
	evSvc, verRepo, _ := createEmailVerificationService(userRepo)

	u := createExtraTestUser(t, userRepo)
	tok := createTokenDirectly(t, verRepo, u.ID, time.Now().Add(-time.Hour), false)

	err := evSvc.Verify(context.Background(), tok.Token)
	if err != verification.ErrTokenExpired {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}

func TestVerify_TokenUsed(t *testing.T) {
	userRepo := memory.NewUserRepository()
	evSvc, verRepo, _ := createEmailVerificationService(userRepo)

	u := createExtraTestUser(t, userRepo)
	tok := createTokenDirectly(t, verRepo, u.ID, time.Now().Add(time.Hour), true)

	err := evSvc.Verify(context.Background(), tok.Token)
	if err != verification.ErrTokenUsed {
		t.Fatalf("expected ErrTokenUsed, got %v", err)
	}
}

// =============================================================================
// ValidateAndConsume tests
// =============================================================================

func TestValidateAndConsume_Success(t *testing.T) {
	userRepo := memory.NewUserRepository()
	evSvc, verRepo, _ := createEmailVerificationService(userRepo)

	u := createExtraTestUser(t, userRepo)
	tok := createTokenDirectly(t, verRepo, u.ID, time.Now().Add(time.Hour), false)

	userID, err := evSvc.ValidateAndConsume(context.Background(), tok.Token)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if userID != u.ID {
		t.Fatalf("expected userID %s, got %s", u.ID, userID)
	}

	// Verify token is now used (calling again should fail)
	_, err = evSvc.ValidateAndConsume(context.Background(), tok.Token)
	if err != verification.ErrTokenUsed {
		t.Fatalf("expected ErrTokenUsed on second call, got %v", err)
	}
}

func TestValidateAndConsume_NotFound(t *testing.T) {
	userRepo := memory.NewUserRepository()
	evSvc, _, _ := createEmailVerificationService(userRepo)

	_, err := evSvc.ValidateAndConsume(context.Background(), "nonexistent")
	if err != verification.ErrTokenNotFound {
		t.Fatalf("expected ErrTokenNotFound, got %v", err)
	}
}

// =============================================================================
// ListOrders tests
// =============================================================================

func TestListOrders_Success(t *testing.T) {
	userRepo := memory.NewUserRepository()
	orderRepo := memory.NewOrderRepository()
	provider := paymentinfra.NewMockProvider()
	svc := NewPaymentService(provider, orderRepo, userRepo, "http://localhost")

	u := createExtraTestUser(t, userRepo)
	ctx := context.Background()

	// Create 3 orders with different CreatedAt (memory repo sets CreatedAt = time.Now())
	// We'll create them sequentially and verify they come back sorted desc.
	for i := 0; i < 3; i++ {
		orderRepo.Create(ctx, payment.Order{
			UserID:      u.ID,
			ReferenceID: "ref-" + string(rune('a'+i)),
			AmountCents: int64((i + 1) * 100),
			Status:      payment.StatusPending,
		})
		// Small sleep to ensure different CreatedAt timestamps
		time.Sleep(2 * time.Millisecond)
	}

	orders, err := svc.ListOrders(ctx, u.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(orders) != 3 {
		t.Fatalf("expected 3 orders, got %d", len(orders))
	}

	// Should be sorted by CreatedAt descending (most recent first)
	for i := 0; i < len(orders)-1; i++ {
		if orders[i].CreatedAt.Before(orders[i+1].CreatedAt) {
			t.Fatalf("orders not sorted descending: orders[%d].CreatedAt=%v < orders[%d].CreatedAt=%v",
				i, orders[i].CreatedAt, i+1, orders[i+1].CreatedAt)
		}
	}
}

func TestListOrders_EmptyUserID(t *testing.T) {
	userRepo := memory.NewUserRepository()
	orderRepo := memory.NewOrderRepository()
	svc := NewPaymentService(nil, orderRepo, userRepo, "http://localhost")

	_, err := svc.ListOrders(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty user ID")
	}
	if err.Error() != "user id is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// Credit tests
// =============================================================================

func TestCredit_Success(t *testing.T) {
	userRepo := memory.NewUserRepository()
	orderRepo := memory.NewOrderRepository()
	provider := paymentinfra.NewMockProvider()
	svc := NewPaymentService(provider, orderRepo, userRepo, "http://localhost")

	u := createExtraTestUser(t, userRepo)
	ctx := context.Background()

	receipt, err := svc.Credit(ctx, u.ID, 500)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if receipt.Amount != 500 {
		t.Fatalf("expected receipt amount 500, got %d", receipt.Amount)
	}
	if receipt.Provider != "mock" {
		t.Fatalf("expected provider 'mock', got %s", receipt.Provider)
	}

	// Verify balance was credited
	updated, _ := userRepo.GetByID(ctx, u.ID)
	if updated.Balance != 500 {
		t.Fatalf("expected balance 500, got %d", updated.Balance)
	}
}

func TestCredit_NoProvider(t *testing.T) {
	userRepo := memory.NewUserRepository()
	orderRepo := memory.NewOrderRepository()
	svc := NewPaymentService(nil, orderRepo, userRepo, "http://localhost")

	_, err := svc.Credit(context.Background(), "some-user", 100)
	if err == nil {
		t.Fatal("expected error when provider is nil")
	}
	if err.Error() != "payment provider not configured" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCredit_EmptyUserID(t *testing.T) {
	userRepo := memory.NewUserRepository()
	orderRepo := memory.NewOrderRepository()
	provider := paymentinfra.NewMockProvider()
	svc := NewPaymentService(provider, orderRepo, userRepo, "http://localhost")

	_, err := svc.Credit(context.Background(), "", 100)
	if err == nil {
		t.Fatal("expected error for empty user ID")
	}
	if err.Error() != "user id is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCredit_ZeroAmount(t *testing.T) {
	userRepo := memory.NewUserRepository()
	orderRepo := memory.NewOrderRepository()
	provider := paymentinfra.NewMockProvider()
	svc := NewPaymentService(provider, orderRepo, userRepo, "http://localhost")

	_, err := svc.Credit(context.Background(), "some-user", 0)
	if err == nil {
		t.Fatal("expected error for zero amount")
	}
	if err.Error() != "amount must be greater than zero" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// ReconcileOrders tests
// =============================================================================

func TestReconcileOrders_ProcessesPending(t *testing.T) {
	userRepo := memory.NewUserRepository()
	orderRepo := memory.NewOrderRepository()
	provider := paymentinfra.NewMockProvider()
	svc := NewPaymentService(provider, orderRepo, userRepo, "http://localhost")

	u := createExtraTestUser(t, userRepo)
	ctx := context.Background()

	// Create a pending order
	order, _ := orderRepo.Create(ctx, payment.Order{
		UserID:      u.ID,
		ReferenceID: "reconcile-ref-1",
		AmountCents: 1000,
		Status:      payment.StatusPending,
	})

	// ReconcileOrders should process it (MockProvider always returns StatusPaid)
	svc.ReconcileOrders(ctx)

	// Verify order is now paid
	updated, err := orderRepo.GetByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if updated.Status != payment.StatusPaid {
		t.Fatalf("expected status %s, got %s", payment.StatusPaid, updated.Status)
	}

	// Verify user balance was credited
	updatedUser, _ := userRepo.GetByID(ctx, u.ID)
	if updatedUser.Balance != 1000 {
		t.Fatalf("expected balance 1000, got %d", updatedUser.Balance)
	}
}

// =============================================================================
// LGPD LogAction tests
// =============================================================================

func TestLGPDService_LogAction(t *testing.T) {
	userRepo := memory.NewUserRepository()
	inspRepo := memory.NewInspectionRepository()
	orderRepo := memory.NewOrderRepository()
	deletionRepo := memory.NewDeletionRepository()
	logRepo := memory.NewLogRepository()
	sender := &extraMockSender{}

	svc := NewLGPDService(userRepo, inspRepo, orderRepo, deletionRepo, logRepo, sender, "dpo@test.com")

	u := createExtraTestUser(t, userRepo)
	ctx := context.Background()

	details := map[string]interface{}{"key": "value"}
	svc.LogAction(ctx, u.ID, "test_action", details)

	logs, err := logRepo.GetByUserID(ctx, u.ID)
	if err != nil {
		t.Fatalf("get logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logs))
	}
	if logs[0].Action != "test_action" {
		t.Fatalf("expected action 'test_action', got %s", logs[0].Action)
	}
	if logs[0].UserID != u.ID {
		t.Fatalf("expected userID %s, got %s", u.ID, logs[0].UserID)
	}
	if logs[0].Details["key"] != "value" {
		t.Fatalf("expected details key=value, got %v", logs[0].Details)
	}
}

package app

import (
	"context"
	"testing"

	"buskatotal-backend/internal/domain/payment"
	"buskatotal-backend/internal/domain/user"
	"buskatotal-backend/internal/infra/memory"
	paymentinfra "buskatotal-backend/internal/infra/payment"
)

func setupPaymentTest(t *testing.T, balance int64) (*PaymentService, *memory.UserRepository, *memory.OrderRepository, user.User) {
	t.Helper()
	userRepo := memory.NewUserRepository()
	orderRepo := memory.NewOrderRepository()
	u, err := userRepo.Create(context.Background(), user.User{
		Name:    "Test User",
		Email:   "test@test.com",
		Balance: balance,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	provider := paymentinfra.NewMockProvider()
	svc := NewPaymentService(provider, orderRepo, userRepo, "http://localhost:8080")
	return svc, userRepo, orderRepo, u
}

// TestProcessWebhookForUser_WrongUser verifies that a user cannot sync
// another user's order (Fix 3).
func TestProcessWebhookForUser_WrongUser(t *testing.T) {
	svc, userRepo, _, u := setupPaymentTest(t, 10000)

	// Create a second user
	otherUser, err := userRepo.Create(context.Background(), user.User{
		Name:  "Other",
		Email: "other@test.com",
	})
	if err != nil {
		t.Fatalf("create other user: %v", err)
	}

	// Create order for the first user
	order, err := svc.CreateOrder(context.Background(), u.ID, 5000, payment.Buyer{
		Document: "12345678901",
		Email:    "test@test.com",
	}, "http://localhost/return")
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	// Other user tries to sync this order
	err = svc.ProcessWebhookForUser(context.Background(), order.ReferenceID, otherUser.ID)
	if err == nil {
		t.Fatal("expected error when syncing another user's order")
	}

	if err.Error() != "order does not belong to this user" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestProcessWebhookForUser_CorrectUser verifies the owner can sync.
func TestProcessWebhookForUser_CorrectUser(t *testing.T) {
	svc, _, _, u := setupPaymentTest(t, 10000)

	order, err := svc.CreateOrder(context.Background(), u.ID, 5000, payment.Buyer{
		Document: "12345678901",
		Email:    "test@test.com",
	}, "http://localhost/return")
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	err = svc.ProcessWebhookForUser(context.Background(), order.ReferenceID, u.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestProcessWebhook_IdempotentCredit verifies that processing the same webhook
// twice does not double-credit the user.
func TestProcessWebhook_IdempotentCredit(t *testing.T) {
	svc, userRepo, _, u := setupPaymentTest(t, 0)

	order, err := svc.CreateOrder(context.Background(), u.ID, 5000, payment.Buyer{
		Document: "12345678901",
		Email:    "test@test.com",
	}, "http://localhost/return")
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	// Process webhook first time — should credit 5000
	err = svc.ProcessWebhook(context.Background(), order.ReferenceID)
	if err != nil {
		t.Fatalf("first webhook: %v", err)
	}

	updated, _ := userRepo.GetByID(context.Background(), u.ID)
	if updated.Balance != 5000 {
		t.Fatalf("expected balance 5000 after first webhook, got %d", updated.Balance)
	}

	// Process webhook second time — should NOT credit again
	err = svc.ProcessWebhook(context.Background(), order.ReferenceID)
	if err != nil {
		t.Fatalf("second webhook: %v", err)
	}

	updated, _ = userRepo.GetByID(context.Background(), u.ID)
	if updated.Balance != 5000 {
		t.Fatalf("expected balance 5000 after second webhook (idempotent), got %d", updated.Balance)
	}
}

// TestCreditBalance_AtomicConcurrent verifies that concurrent credits
// add up correctly (no lost updates).
func TestCreditBalance_AtomicConcurrent(t *testing.T) {
	userRepo := memory.NewUserRepository()
	u, _ := userRepo.Create(context.Background(), user.User{
		Name:    "Test",
		Email:   "test@test.com",
		Balance: 0,
	})

	const goroutines = 100
	const amount int64 = 100

	done := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		go func() {
			userRepo.CreditBalance(context.Background(), u.ID, amount)
			done <- struct{}{}
		}()
	}
	for i := 0; i < goroutines; i++ {
		<-done
	}

	updated, _ := userRepo.GetByID(context.Background(), u.ID)
	expected := int64(goroutines) * amount
	if updated.Balance != expected {
		t.Fatalf("expected balance %d, got %d (lost updates!)", expected, updated.Balance)
	}
}

package app

import (
	"context"
	"testing"
	"time"

	"buskatotal-backend/internal/domain/email"
	"buskatotal-backend/internal/domain/payment"
	"buskatotal-backend/internal/domain/user"
	"buskatotal-backend/internal/infra/memory"
	paymentinfra "buskatotal-backend/internal/infra/payment"

	"golang.org/x/crypto/bcrypt"
)

// --- final mock types (prefixed to avoid collisions) ---

type finalMockSender struct{ sent []email.Message }

func (m *finalMockSender) Send(_ context.Context, msg email.Message) error {
	m.sent = append(m.sent, msg)
	return nil
}

// finalCreateUser creates a user with the given email and role.
func finalCreateUser(t *testing.T, repo *memory.UserRepository, emailAddr, role string) user.User {
	t.Helper()
	u, err := repo.Create(context.Background(), user.User{
		Name:  "Final Test User",
		Email: emailAddr,
		Role:  role,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u
}

// =============================================================================
// CreateOrder tests
// =============================================================================

func TestFinal_CreateOrder_Success(t *testing.T) {
	userRepo := memory.NewUserRepository()
	orderRepo := memory.NewOrderRepository()
	provider := paymentinfra.NewMockProvider()
	svc := NewPaymentService(provider, orderRepo, userRepo, "http://localhost:8080")

	u := finalCreateUser(t, userRepo, "buyer@example.com", user.RoleUser)
	ctx := context.Background()

	buyer := payment.Buyer{
		FirstName: "Test",
		LastName:  "User",
		Document:  "123.456.789-00",
		Email:     "buyer@example.com",
		Phone:     "11999999999",
	}

	order, err := svc.CreateOrder(ctx, u.ID, 1500, buyer, "http://return.url")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if order.ID == "" {
		t.Fatal("expected order ID to be set")
	}
	if order.UserID != u.ID {
		t.Fatalf("expected userID %s, got %s", u.ID, order.UserID)
	}
	if order.AmountCents != 1500 {
		t.Fatalf("expected amount 1500, got %d", order.AmountCents)
	}
	if order.Status != payment.StatusPending {
		t.Fatalf("expected status pending, got %s", order.Status)
	}
	if order.PaymentURL == "" {
		t.Fatal("expected payment URL to be set")
	}
	if order.QRCodeText == "" {
		t.Fatal("expected QR code text to be set")
	}
	if order.QRCodeBase64 == "" {
		t.Fatal("expected QR code base64 to be set")
	}
	if order.ReferenceID == "" {
		t.Fatal("expected reference ID to be set")
	}
}

func TestFinal_CreateOrder_NoProvider(t *testing.T) {
	userRepo := memory.NewUserRepository()
	orderRepo := memory.NewOrderRepository()
	svc := NewPaymentService(nil, orderRepo, userRepo, "http://localhost:8080")

	buyer := payment.Buyer{Document: "123.456.789-00", Email: "a@b.com"}
	_, err := svc.CreateOrder(context.Background(), "some-id", 100, buyer, "")
	if err == nil {
		t.Fatal("expected error when provider is nil")
	}
	if err.Error() != "payment provider not configured" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFinal_CreateOrder_EmptyUserID(t *testing.T) {
	userRepo := memory.NewUserRepository()
	orderRepo := memory.NewOrderRepository()
	provider := paymentinfra.NewMockProvider()
	svc := NewPaymentService(provider, orderRepo, userRepo, "http://localhost:8080")

	buyer := payment.Buyer{Document: "123.456.789-00", Email: "a@b.com"}
	_, err := svc.CreateOrder(context.Background(), "", 100, buyer, "")
	if err == nil {
		t.Fatal("expected error for empty user ID")
	}
	if err.Error() != "user id is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFinal_CreateOrder_ZeroAmount(t *testing.T) {
	userRepo := memory.NewUserRepository()
	orderRepo := memory.NewOrderRepository()
	provider := paymentinfra.NewMockProvider()
	svc := NewPaymentService(provider, orderRepo, userRepo, "http://localhost:8080")

	buyer := payment.Buyer{Document: "123.456.789-00", Email: "a@b.com"}
	_, err := svc.CreateOrder(context.Background(), "some-user-id", 0, buyer, "")
	if err == nil {
		t.Fatal("expected error for zero amount")
	}
	if err.Error() != "amount must be greater than zero" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFinal_CreateOrder_NegativeAmount(t *testing.T) {
	userRepo := memory.NewUserRepository()
	orderRepo := memory.NewOrderRepository()
	provider := paymentinfra.NewMockProvider()
	svc := NewPaymentService(provider, orderRepo, userRepo, "http://localhost:8080")

	buyer := payment.Buyer{Document: "123.456.789-00", Email: "a@b.com"}
	_, err := svc.CreateOrder(context.Background(), "some-user-id", -50, buyer, "")
	if err == nil {
		t.Fatal("expected error for negative amount")
	}
	if err.Error() != "amount must be greater than zero" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFinal_CreateOrder_MissingDocument(t *testing.T) {
	userRepo := memory.NewUserRepository()
	orderRepo := memory.NewOrderRepository()
	provider := paymentinfra.NewMockProvider()
	svc := NewPaymentService(provider, orderRepo, userRepo, "http://localhost:8080")

	buyer := payment.Buyer{Email: "a@b.com"}
	_, err := svc.CreateOrder(context.Background(), "some-user-id", 100, buyer, "")
	if err == nil {
		t.Fatal("expected error for missing document")
	}
	if err.Error() != "buyer document (CPF) is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFinal_CreateOrder_MissingBuyerEmail(t *testing.T) {
	userRepo := memory.NewUserRepository()
	orderRepo := memory.NewOrderRepository()
	provider := paymentinfra.NewMockProvider()
	svc := NewPaymentService(provider, orderRepo, userRepo, "http://localhost:8080")

	buyer := payment.Buyer{Document: "123.456.789-00"}
	_, err := svc.CreateOrder(context.Background(), "some-user-id", 100, buyer, "")
	if err == nil {
		t.Fatal("expected error for missing buyer email")
	}
	if err.Error() != "buyer email is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFinal_CreateOrder_UserNotFound(t *testing.T) {
	userRepo := memory.NewUserRepository()
	orderRepo := memory.NewOrderRepository()
	provider := paymentinfra.NewMockProvider()
	svc := NewPaymentService(provider, orderRepo, userRepo, "http://localhost:8080")

	buyer := payment.Buyer{Document: "123.456.789-00", Email: "a@b.com"}
	_, err := svc.CreateOrder(context.Background(), "nonexistent-user-id-1234", 100, buyer, "")
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
	if err.Error() != "user not found" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// ProcessWebhook tests
// =============================================================================

func TestFinal_ProcessWebhook_PaidCreditsBalance(t *testing.T) {
	userRepo := memory.NewUserRepository()
	orderRepo := memory.NewOrderRepository()
	provider := paymentinfra.NewMockProvider()
	svc := NewPaymentService(provider, orderRepo, userRepo, "http://localhost:8080")

	u := finalCreateUser(t, userRepo, "webhook@example.com", user.RoleUser)
	ctx := context.Background()

	// Create a pending order directly in the repo
	order, err := orderRepo.Create(ctx, payment.Order{
		UserID:      u.ID,
		ReferenceID: "webhook-ref-001",
		AmountCents: 2000,
		Status:      payment.StatusPending,
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	// Process the webhook - mock provider returns StatusPaid
	err = svc.ProcessWebhook(ctx, order.ReferenceID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify order status updated to paid
	updated, err := orderRepo.GetByReferenceID(ctx, order.ReferenceID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if updated.Status != payment.StatusPaid {
		t.Fatalf("expected status paid, got %s", updated.Status)
	}

	// Verify user balance was credited
	updatedUser, err := userRepo.GetByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if updatedUser.Balance != 2000 {
		t.Fatalf("expected balance 2000, got %d", updatedUser.Balance)
	}
}

func TestFinal_ProcessWebhook_AlreadyPaid(t *testing.T) {
	userRepo := memory.NewUserRepository()
	orderRepo := memory.NewOrderRepository()
	provider := paymentinfra.NewMockProvider()
	svc := NewPaymentService(provider, orderRepo, userRepo, "http://localhost:8080")

	u := finalCreateUser(t, userRepo, "idempotent@example.com", user.RoleUser)
	ctx := context.Background()

	// Create a pending order and process it once
	order, _ := orderRepo.Create(ctx, payment.Order{
		UserID:      u.ID,
		ReferenceID: "idempotent-ref-001",
		AmountCents: 3000,
		Status:      payment.StatusPending,
	})

	// First webhook processing - should credit balance
	err := svc.ProcessWebhook(ctx, order.ReferenceID)
	if err != nil {
		t.Fatalf("first webhook: expected no error, got %v", err)
	}

	// Verify balance after first processing
	u1, _ := userRepo.GetByID(ctx, u.ID)
	if u1.Balance != 3000 {
		t.Fatalf("expected balance 3000 after first webhook, got %d", u1.Balance)
	}

	// Second webhook processing - idempotency, should not credit again
	err = svc.ProcessWebhook(ctx, order.ReferenceID)
	if err != nil {
		t.Fatalf("second webhook: expected no error, got %v", err)
	}

	// Verify balance unchanged (not doubled)
	u2, _ := userRepo.GetByID(ctx, u.ID)
	if u2.Balance != 3000 {
		t.Fatalf("expected balance 3000 after second webhook (idempotent), got %d", u2.Balance)
	}
}

func TestFinal_ProcessWebhook_EmptyReferenceID(t *testing.T) {
	userRepo := memory.NewUserRepository()
	orderRepo := memory.NewOrderRepository()
	provider := paymentinfra.NewMockProvider()
	svc := NewPaymentService(provider, orderRepo, userRepo, "http://localhost:8080")

	err := svc.ProcessWebhook(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty reference ID")
	}
	if err.Error() != "referenceId is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFinal_ProcessWebhook_NoProvider(t *testing.T) {
	userRepo := memory.NewUserRepository()
	orderRepo := memory.NewOrderRepository()
	svc := NewPaymentService(nil, orderRepo, userRepo, "http://localhost:8080")

	err := svc.ProcessWebhook(context.Background(), "some-ref")
	if err == nil {
		t.Fatal("expected error when provider is nil")
	}
	if err.Error() != "payment provider not configured" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFinal_ProcessWebhook_OrderNotFound(t *testing.T) {
	userRepo := memory.NewUserRepository()
	orderRepo := memory.NewOrderRepository()
	provider := paymentinfra.NewMockProvider()
	svc := NewPaymentService(provider, orderRepo, userRepo, "http://localhost:8080")

	err := svc.ProcessWebhook(context.Background(), "nonexistent-ref")
	if err == nil {
		t.Fatal("expected error for non-existent order")
	}
}

// =============================================================================
// UserService.Update tests
// =============================================================================

func TestFinal_UserService_Update_Success(t *testing.T) {
	userRepo := memory.NewUserRepository()
	svc := NewUserService(userRepo)
	ctx := context.Background()

	u := finalCreateUser(t, userRepo, "update@example.com", user.RoleUser)

	updated, err := svc.Update(ctx, user.User{
		ID:    u.ID,
		Name:  "Updated Name",
		Email: "updated@example.com",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Name != "Updated Name" {
		t.Fatalf("expected name 'Updated Name', got %s", updated.Name)
	}
	if updated.Email != "updated@example.com" {
		t.Fatalf("expected email 'updated@example.com', got %s", updated.Email)
	}
}

func TestFinal_UserService_Update_EmptyID(t *testing.T) {
	userRepo := memory.NewUserRepository()
	svc := NewUserService(userRepo)

	_, err := svc.Update(context.Background(), user.User{
		Name:  "Name",
		Email: "e@e.com",
	})
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
	if err.Error() != "id is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFinal_UserService_Update_EmptyName(t *testing.T) {
	userRepo := memory.NewUserRepository()
	svc := NewUserService(userRepo)

	_, err := svc.Update(context.Background(), user.User{
		ID:    "some-id",
		Email: "e@e.com",
	})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if err.Error() != "name and email are required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// Auth - Login admin upgrade tests
// =============================================================================

func TestFinal_Login_AdminUpgrade(t *testing.T) {
	userRepo := memory.NewUserRepository()
	authSvc := NewAuthService(userRepo, "test-secret", time.Hour, nil)
	ctx := context.Background()

	// Use one of the hardcoded admin emails
	adminEmail := "dcparticular2014@gmail.com"
	password := "Str0ng@Pass!"

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	// Create user with RoleUser (simulating pre-admin-feature user)
	_, err = userRepo.Create(ctx, user.User{
		Name:         "Admin User",
		Email:        adminEmail,
		Role:         user.RoleUser, // not admin yet
		PasswordHash: string(hash),
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Login should upgrade the role to admin
	loggedIn, token, err := authSvc.Login(ctx, adminEmail, password)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token == "" {
		t.Fatal("expected token to be non-empty")
	}
	if loggedIn.Role != user.RoleAdmin {
		t.Fatalf("expected role to be upgraded to admin, got %s", loggedIn.Role)
	}

	// Verify role persisted in repo
	fromRepo, _ := userRepo.GetByEmail(ctx, adminEmail)
	if fromRepo.Role != user.RoleAdmin {
		t.Fatalf("expected persisted role to be admin, got %s", fromRepo.Role)
	}
}

// =============================================================================
// Auth - ForgotPassword with nil email service
// =============================================================================

func TestFinal_ForgotPassword_NoEmailService(t *testing.T) {
	userRepo := memory.NewUserRepository()
	authSvc := NewAuthService(userRepo, "test-secret", time.Hour, nil)

	err := authSvc.ForgotPassword(context.Background(), "any@example.com")
	if err == nil {
		t.Fatal("expected error when email service is nil")
	}
	if err.Error() != "email service not configured" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// Auth - generateToken with empty secret
// =============================================================================

func TestFinal_GenerateToken_EmptySecret(t *testing.T) {
	userRepo := memory.NewUserRepository()
	// Empty JWT secret
	authSvc := NewAuthService(userRepo, "", time.Hour, nil)
	ctx := context.Background()

	password := "Str0ng@Pass!"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	_, err := userRepo.Create(ctx, user.User{
		Name:         "Token User",
		Email:        "tokenuser@example.com",
		Role:         user.RoleUser,
		PasswordHash: string(hash),
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Login should fail at generateToken due to empty secret
	_, token, err := authSvc.Login(ctx, "tokenuser@example.com", password)
	if err == nil {
		t.Fatal("expected error for empty JWT secret")
	}
	if token != "" {
		t.Fatal("expected empty token on error")
	}
	if err.Error() != "missing jwt secret" {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// LGPD - NewLGPDService default DPO email
// =============================================================================

func TestFinal_NewLGPDService_DefaultDpoEmail(t *testing.T) {
	userRepo := memory.NewUserRepository()
	inspRepo := memory.NewInspectionRepository()
	orderRepo := memory.NewOrderRepository()
	deletionRepo := memory.NewDeletionRepository()
	logRepo := memory.NewLogRepository()
	sender := &finalMockSender{}

	// Pass empty dpoEmail — should use the default
	svc := NewLGPDService(userRepo, inspRepo, orderRepo, deletionRepo, logRepo, sender, "")

	if svc.dpoEmail != "karinbelan43@gmail.com" {
		t.Fatalf("expected default dpoEmail 'karinbelan43@gmail.com', got %s", svc.dpoEmail)
	}
}

func TestFinal_NewLGPDService_CustomDpoEmail(t *testing.T) {
	userRepo := memory.NewUserRepository()
	inspRepo := memory.NewInspectionRepository()
	orderRepo := memory.NewOrderRepository()
	deletionRepo := memory.NewDeletionRepository()
	logRepo := memory.NewLogRepository()
	sender := &finalMockSender{}

	svc := NewLGPDService(userRepo, inspRepo, orderRepo, deletionRepo, logRepo, sender, "custom@dpo.com")

	if svc.dpoEmail != "custom@dpo.com" {
		t.Fatalf("expected dpoEmail 'custom@dpo.com', got %s", svc.dpoEmail)
	}
}

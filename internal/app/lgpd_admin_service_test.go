package app

import (
	"context"
	"testing"
	"time"

	"buskatotal-backend/internal/domain/email"
	"buskatotal-backend/internal/domain/lgpd"
	"buskatotal-backend/internal/domain/user"
	"buskatotal-backend/internal/infra/memory"
)

// lgpdMockSender implements email.Sender for testing.
type lgpdMockSender struct {
	sent []email.Message
}

func (m *lgpdMockSender) Send(_ context.Context, msg email.Message) error {
	m.sent = append(m.sent, msg)
	return nil
}

// ---------------------------------------------------------------------------
// Admin Service Tests
// ---------------------------------------------------------------------------

func TestAdminService_ListUsers(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewUserRepository()

	names := []string{"Alice", "Bob", "Charlie"}
	for _, n := range names {
		repo.Create(ctx, user.User{Name: n, Email: n + "@test.com"})
	}

	svc := NewAdminService(repo)
	users, err := svc.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers error: %v", err)
	}
	if len(users) != 3 {
		t.Fatalf("expected 3 users, got %d", len(users))
	}
}

func TestAdminService_SearchUsers_ByName(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewUserRepository()

	repo.Create(ctx, user.User{Name: "Maria Silva", Email: "maria@test.com"})
	repo.Create(ctx, user.User{Name: "João Santos", Email: "joao@test.com"})
	repo.Create(ctx, user.User{Name: "Ana Maria", Email: "ana@test.com"})

	svc := NewAdminService(repo)
	results, err := svc.SearchUsers(ctx, "Maria")
	if err != nil {
		t.Fatalf("SearchUsers error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 users matching 'Maria', got %d", len(results))
	}
}

func TestAdminService_SearchUsers_ByEmail(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewUserRepository()

	repo.Create(ctx, user.User{Name: "User A", Email: "alpha@example.com"})
	repo.Create(ctx, user.User{Name: "User B", Email: "beta@example.com"})

	svc := NewAdminService(repo)
	results, err := svc.SearchUsers(ctx, "alpha@example")
	if err != nil {
		t.Fatalf("SearchUsers error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 user, got %d", len(results))
	}
	if results[0].Name != "User A" {
		t.Fatalf("expected User A, got %s", results[0].Name)
	}
}

func TestAdminService_SearchUsers_CaseInsensitive(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewUserRepository()

	repo.Create(ctx, user.User{Name: "Carlos Souza", Email: "carlos@test.com"})

	svc := NewAdminService(repo)
	results, err := svc.SearchUsers(ctx, "carlos souza")
	if err != nil {
		t.Fatalf("SearchUsers error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 user, got %d", len(results))
	}

	results2, err := svc.SearchUsers(ctx, "CARLOS")
	if err != nil {
		t.Fatalf("SearchUsers error: %v", err)
	}
	if len(results2) != 1 {
		t.Fatalf("expected 1 user for uppercase query, got %d", len(results2))
	}
}

func TestAdminService_GetUserByID_Found(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewUserRepository()

	created, _ := repo.Create(ctx, user.User{Name: "Found User", Email: "found@test.com"})

	svc := NewAdminService(repo)
	u, err := svc.GetUserByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetUserByID error: %v", err)
	}
	if u.Name != "Found User" {
		t.Fatalf("expected 'Found User', got %q", u.Name)
	}
}

func TestAdminService_GetUserByID_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewUserRepository()

	svc := NewAdminService(repo)
	_, err := svc.GetUserByID(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent user, got nil")
	}
}

// ---------------------------------------------------------------------------
// LGPD Service Tests
// ---------------------------------------------------------------------------

func newLGPDTestService() (*LGPDService, *memory.UserRepository, *lgpdMockSender) {
	userRepo := memory.NewUserRepository()
	inspRepo := memory.NewInspectionRepository()
	orderRepo := memory.NewOrderRepository()
	deletionRepo := memory.NewDeletionRepository()
	logRepo := memory.NewLogRepository()
	sender := &lgpdMockSender{}

	svc := NewLGPDService(userRepo, inspRepo, orderRepo, deletionRepo, logRepo, sender, "dpo@test.com")
	return svc, userRepo, sender
}

func createTestUser(t *testing.T, repo *memory.UserRepository) user.User {
	t.Helper()
	ctx := context.Background()
	u, err := repo.Create(ctx, user.User{
		Name:    "Test User",
		Email:   "test@lgpd.com",
		Balance: 5000,
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return u
}

func TestLGPDService_GetUserData_Success(t *testing.T) {
	svc, userRepo, _ := newLGPDTestService()
	u := createTestUser(t, userRepo)
	ctx := context.Background()

	result, err := svc.GetUserData(ctx, u.ID)
	if err != nil {
		t.Fatalf("GetUserData error: %v", err)
	}

	resp, ok := result.(UserDataResponse)
	if !ok {
		t.Fatalf("expected UserDataResponse, got %T", result)
	}
	if resp.User.Name != "Test User" {
		t.Fatalf("expected name 'Test User', got %q", resp.User.Name)
	}
	if resp.User.Email != "test@lgpd.com" {
		t.Fatalf("expected email 'test@lgpd.com', got %q", resp.User.Email)
	}
	if resp.SaldoCents != 5000 {
		t.Fatalf("expected saldo 5000, got %d", resp.SaldoCents)
	}
}

func TestLGPDService_GetUserData_NotFound(t *testing.T) {
	svc, _, _ := newLGPDTestService()
	ctx := context.Background()

	_, err := svc.GetUserData(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent user, got nil")
	}
}

func TestLGPDService_ExportUserData_Success(t *testing.T) {
	svc, userRepo, _ := newLGPDTestService()
	u := createTestUser(t, userRepo)
	ctx := context.Background()

	result, err := svc.ExportUserData(ctx, u.ID)
	if err != nil {
		t.Fatalf("ExportUserData error: %v", err)
	}

	resp, ok := result.(ExportResponse)
	if !ok {
		t.Fatalf("expected ExportResponse, got %T", result)
	}
	if resp.Usuario.Nome != "Test User" {
		t.Fatalf("expected nome 'Test User', got %q", resp.Usuario.Nome)
	}
	if resp.Usuario.Email != "test@lgpd.com" {
		t.Fatalf("expected email 'test@lgpd.com', got %q", resp.Usuario.Email)
	}
}

func TestLGPDService_RequestDeletion_Success(t *testing.T) {
	svc, userRepo, _ := newLGPDTestService()
	u := createTestUser(t, userRepo)
	ctx := context.Background()

	req, err := svc.RequestDeletion(ctx, u.ID, "Não quero mais usar")
	if err != nil {
		t.Fatalf("RequestDeletion error: %v", err)
	}
	if req.Status != lgpd.DeletionStatusPending {
		t.Fatalf("expected status 'pending', got %q", req.Status)
	}
	if req.UserID != u.ID {
		t.Fatalf("expected userID %q, got %q", u.ID, req.UserID)
	}
	if req.Reason != "Não quero mais usar" {
		t.Fatalf("expected reason 'Não quero mais usar', got %q", req.Reason)
	}
}

func TestLGPDService_RequestDeletion_DuplicatePending(t *testing.T) {
	svc, userRepo, _ := newLGPDTestService()
	u := createTestUser(t, userRepo)
	ctx := context.Background()

	_, err := svc.RequestDeletion(ctx, u.ID, "First request")
	if err != nil {
		t.Fatalf("first RequestDeletion error: %v", err)
	}

	_, err = svc.RequestDeletion(ctx, u.ID, "Second request")
	if err == nil {
		t.Fatal("expected error for duplicate pending request, got nil")
	}
}

func TestLGPDService_ProcessDeletion_Complete(t *testing.T) {
	svc, userRepo, _ := newLGPDTestService()
	u := createTestUser(t, userRepo)
	ctx := context.Background()

	req, err := svc.RequestDeletion(ctx, u.ID, "Remove my data")
	if err != nil {
		t.Fatalf("RequestDeletion error: %v", err)
	}

	// Allow goroutines from RequestDeletion to finish
	time.Sleep(50 * time.Millisecond)

	processed, err := svc.ProcessDeletion(ctx, req.ID, lgpd.DeletionStatusCompleted, "admin-123")
	if err != nil {
		t.Fatalf("ProcessDeletion error: %v", err)
	}
	if processed.Status != lgpd.DeletionStatusCompleted {
		t.Fatalf("expected status 'completed', got %q", processed.Status)
	}
	if processed.ProcessedBy != "admin-123" {
		t.Fatalf("expected processedBy 'admin-123', got %q", processed.ProcessedBy)
	}
	if processed.ProcessedAt == nil {
		t.Fatal("expected processedAt to be set")
	}

	// Verify user was anonymized
	anonymized, err := userRepo.GetByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("failed to get anonymized user: %v", err)
	}
	if anonymized.Name != "Usuário removido" {
		t.Fatalf("expected name 'Usuário removido', got %q", anonymized.Name)
	}
	if anonymized.Balance != 0 {
		t.Fatalf("expected balance 0, got %d", anonymized.Balance)
	}
}

func TestLGPDService_ProcessDeletion_Rejected(t *testing.T) {
	svc, userRepo, _ := newLGPDTestService()
	u := createTestUser(t, userRepo)
	ctx := context.Background()

	req, err := svc.RequestDeletion(ctx, u.ID, "Remove my data")
	if err != nil {
		t.Fatalf("RequestDeletion error: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	processed, err := svc.ProcessDeletion(ctx, req.ID, lgpd.DeletionStatusRejected, "admin-456")
	if err != nil {
		t.Fatalf("ProcessDeletion error: %v", err)
	}
	if processed.Status != lgpd.DeletionStatusRejected {
		t.Fatalf("expected status 'rejected', got %q", processed.Status)
	}

	// Verify user data was NOT changed
	unchanged, err := userRepo.GetByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	if unchanged.Name != "Test User" {
		t.Fatalf("expected name 'Test User' (unchanged), got %q", unchanged.Name)
	}
	if unchanged.Balance != 5000 {
		t.Fatalf("expected balance 5000 (unchanged), got %d", unchanged.Balance)
	}
}

func TestLGPDService_ListDeletionRequests(t *testing.T) {
	svc, userRepo, _ := newLGPDTestService()
	ctx := context.Background()

	// Create two users and request deletion for each
	u1, _ := userRepo.Create(ctx, user.User{Name: "User One", Email: "one@test.com"})
	u2, _ := userRepo.Create(ctx, user.User{Name: "User Two", Email: "two@test.com"})

	_, err := svc.RequestDeletion(ctx, u1.ID, "Reason 1")
	if err != nil {
		t.Fatalf("RequestDeletion u1 error: %v", err)
	}
	_, err = svc.RequestDeletion(ctx, u2.ID, "Reason 2")
	if err != nil {
		t.Fatalf("RequestDeletion u2 error: %v", err)
	}

	requests, err := svc.ListDeletionRequests(ctx)
	if err != nil {
		t.Fatalf("ListDeletionRequests error: %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("expected 2 deletion requests, got %d", len(requests))
	}
}

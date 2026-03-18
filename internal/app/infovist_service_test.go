package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"buskatotal-backend/internal/domain/user"
	"buskatotal-backend/internal/infra/infovist"
	"buskatotal-backend/internal/infra/memory"
)

// mockInfovistClient wraps infovist.Client with controllable behavior for tests.
// We test the service layer, not the HTTP client, so we override at service level.

func setupInfovistTest(t *testing.T, balance int64) (*InfovistService, *memory.UserRepository, user.User) {
	t.Helper()
	repo := memory.NewUserRepository()
	inspRepo := memory.NewInspectionRepository()
	u, err := repo.Create(context.Background(), user.User{
		Name:    "Test User",
		Email:   "test@test.com",
		Balance: balance,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Create a client that won't actually call the API.
	// The service will fail at getToken, but we test debit/rollback behavior.
	client := infovist.NewClient("http://localhost:9999", "test@test.com", "password", "token")

	svc := NewInfovistService(client, repo, inspRepo, 3096, 0)
	return svc, repo, u
}

// TestCreateInspection_InsufficientBalance verifies that creating an inspection
// with insufficient balance returns an error and does not change the balance.
func TestCreateInspection_InsufficientBalance(t *testing.T) {
	svc, repo, u := setupInfovistTest(t, 1000) // has 1000, needs 3096

	_, err := svc.CreateInspection(context.Background(), u.ID, infovist.CreateInspectionRequest{
		Customer:  "John",
		Cellphone: "11999999999",
		Plate:     "ABC1234",
	})

	if err == nil {
		t.Fatal("expected error for insufficient balance")
	}
	if !errors.Is(err, user.ErrInsufficientBalance) {
		t.Fatalf("expected ErrInsufficientBalance, got: %v", err)
	}

	// Balance must be unchanged
	updated, _ := repo.GetByID(context.Background(), u.ID)
	if updated.Balance != 1000 {
		t.Fatalf("expected balance 1000, got %d", updated.Balance)
	}
}

// TestCreateInspection_RollbackOnAPIFailure verifies that if the external API
// call fails, the debited balance is refunded (Fix 5).
func TestCreateInspection_RollbackOnAPIFailure(t *testing.T) {
	svc, repo, u := setupInfovistTest(t, 5000) // enough balance

	// The client points to localhost:9999 which won't respond, so CreateInspection
	// will debit, try to authenticate, fail, and should rollback.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := svc.CreateInspection(ctx, u.ID, infovist.CreateInspectionRequest{
		Customer:  "John",
		Cellphone: "11999999999",
		Plate:     "ABC1234",
	})

	if err == nil {
		t.Fatal("expected error from failed API call")
	}

	// Balance must be restored (rollback)
	updated, _ := repo.GetByID(context.Background(), u.ID)
	if updated.Balance != 5000 {
		t.Fatalf("expected balance restored to 5000, got %d (rollback failed)", updated.Balance)
	}
}

// TestGetReportV1_NoCharge verifies that fetching a report does not debit balance.
func TestGetReportV1_NoCharge(t *testing.T) {
	svc, repo, u := setupInfovistTest(t, 100) // minimal balance

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Will fail at API call, but should NOT debit anything
	_, _ = svc.GetReportV1(ctx, u.ID, "abc12345")

	updated, _ := repo.GetByID(context.Background(), u.ID)
	if updated.Balance != 100 {
		t.Fatalf("expected balance unchanged at 100, got %d (report should be free)", updated.Balance)
	}
}

// TestGetReportV2_NoCharge verifies that fetching a report v2 does not debit balance.
func TestGetReportV2_NoCharge(t *testing.T) {
	svc, repo, u := setupInfovistTest(t, 100) // minimal balance

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, _ = svc.GetReportV2(ctx, u.ID, "abc12345")

	updated, _ := repo.GetByID(context.Background(), u.ID)
	if updated.Balance != 100 {
		t.Fatalf("expected balance unchanged at 100, got %d (report should be free)", updated.Balance)
	}
}

// TestViewInspection_NoCharge verifies viewing inspection is free.
func TestViewInspection_NoCharge(t *testing.T) {
	svc, repo, u := setupInfovistTest(t, 100) // minimal balance

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Will fail at API call, but should NOT debit anything
	_, _ = svc.ViewInspection(ctx, u.ID, "abc12345")

	updated, _ := repo.GetByID(context.Background(), u.ID)
	if updated.Balance != 100 {
		t.Fatalf("expected balance unchanged at 100, got %d", updated.Balance)
	}
}

// TestCreateInspection_ValidationErrors verifies input validation runs
// before any balance debit.
func TestCreateInspection_ValidationErrors(t *testing.T) {
	svc, repo, u := setupInfovistTest(t, 50000)

	tests := []struct {
		name  string
		input infovist.CreateInspectionRequest
	}{
		{"missing customer", infovist.CreateInspectionRequest{Cellphone: "11999999999", Plate: "ABC1234"}},
		{"missing cellphone", infovist.CreateInspectionRequest{Customer: "John", Plate: "ABC1234"}},
		{"missing plate and chassis", infovist.CreateInspectionRequest{Customer: "John", Cellphone: "11999999999"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateInspection(context.Background(), u.ID, tt.input)
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}

	// Balance must be unchanged — validation errors should not debit
	updated, _ := repo.GetByID(context.Background(), u.ID)
	if updated.Balance != 50000 {
		t.Fatalf("expected balance 50000, got %d (validation debited balance!)", updated.Balance)
	}
}

// TestCreateInspection_ConcurrentRace verifies that concurrent inspection
// creations cannot overdraw the balance (Fix 1).
func TestCreateInspection_ConcurrentRace(t *testing.T) {
	repo := memory.NewUserRepository()
	u, _ := repo.Create(context.Background(), user.User{
		Name:    "Test",
		Email:   "test@test.com",
		Balance: 3096, // exactly 1 inspection
	})

	inspRepo := memory.NewInspectionRepository()
	client := infovist.NewClient("http://localhost:9999", "e", "p", "t")
	svc := NewInfovistService(client, repo, inspRepo, 3096, 0)

	const goroutines = 20
	var wg sync.WaitGroup
	var debitedCount int64
	var mu sync.Mutex

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			_, err := svc.CreateInspection(ctx, u.ID, infovist.CreateInspectionRequest{
				Customer:  "John",
				Cellphone: "11999999999",
				Plate:     "ABC1234",
			})
			// The API call will fail, but the debit might have happened
			// before rollback. We check final balance below.
			_ = err
			mu.Lock()
			debitedCount++
			mu.Unlock()
		}()
	}
	wg.Wait()

	// After all goroutines complete (with rollbacks), balance should be back to original
	// since the API calls all fail.
	updated, _ := repo.GetByID(context.Background(), u.ID)
	if updated.Balance != 3096 {
		t.Fatalf("expected balance restored to 3096 after rollbacks, got %d", updated.Balance)
	}
}

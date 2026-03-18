package memory

import (
	"context"
	"sync"
	"testing"

	"buskatotal-backend/internal/domain/user"
)

func seedUser(t *testing.T, repo *UserRepository, balance int64) user.User {
	t.Helper()
	u, err := repo.Create(context.Background(), user.User{
		Name:    "Test",
		Email:   "test@test.com",
		Balance: balance,
	})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u
}

func TestDebitBalance_Success(t *testing.T) {
	repo := NewUserRepository()
	u := seedUser(t, repo, 10000)

	err := repo.DebitBalance(context.Background(), u.ID, 3000)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, _ := repo.GetByID(context.Background(), u.ID)
	if updated.Balance != 7000 {
		t.Fatalf("expected balance 7000, got %d", updated.Balance)
	}
}

func TestDebitBalance_InsufficientBalance(t *testing.T) {
	repo := NewUserRepository()
	u := seedUser(t, repo, 1000)

	err := repo.DebitBalance(context.Background(), u.ID, 5000)
	if err == nil {
		t.Fatal("expected error for insufficient balance")
	}
	if err != user.ErrInsufficientBalance {
		t.Fatalf("expected ErrInsufficientBalance, got %v", err)
	}

	// Balance should be unchanged
	updated, _ := repo.GetByID(context.Background(), u.ID)
	if updated.Balance != 1000 {
		t.Fatalf("expected balance unchanged at 1000, got %d", updated.Balance)
	}
}

func TestDebitBalance_ExactBalance(t *testing.T) {
	repo := NewUserRepository()
	u := seedUser(t, repo, 5000)

	err := repo.DebitBalance(context.Background(), u.ID, 5000)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, _ := repo.GetByID(context.Background(), u.ID)
	if updated.Balance != 0 {
		t.Fatalf("expected balance 0, got %d", updated.Balance)
	}
}

func TestDebitBalance_UserNotFound(t *testing.T) {
	repo := NewUserRepository()

	err := repo.DebitBalance(context.Background(), "nonexistent", 1000)
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestCreditBalance_Success(t *testing.T) {
	repo := NewUserRepository()
	u := seedUser(t, repo, 5000)

	err := repo.CreditBalance(context.Background(), u.ID, 3000)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, _ := repo.GetByID(context.Background(), u.ID)
	if updated.Balance != 8000 {
		t.Fatalf("expected balance 8000, got %d", updated.Balance)
	}
}

func TestCreditBalance_UserNotFound(t *testing.T) {
	repo := NewUserRepository()

	err := repo.CreditBalance(context.Background(), "nonexistent", 1000)
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

// TestDebitBalance_ConcurrentRace verifies that concurrent debits cannot
// overdraw the balance (Fix 1 — race condition).
func TestDebitBalance_ConcurrentRace(t *testing.T) {
	repo := NewUserRepository()
	// User has exactly enough for 1 debit of 10000
	u := seedUser(t, repo, 10000)

	const goroutines = 50
	var wg sync.WaitGroup
	var successCount int64
	var mu sync.Mutex

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			err := repo.DebitBalance(context.Background(), u.ID, 10000)
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if successCount != 1 {
		t.Fatalf("expected exactly 1 successful debit, got %d", successCount)
	}

	updated, _ := repo.GetByID(context.Background(), u.ID)
	if updated.Balance != 0 {
		t.Fatalf("expected balance 0, got %d", updated.Balance)
	}
}

// TestDebitBalance_ConcurrentMultiple verifies that concurrent debits
// correctly track balance across many small operations.
func TestDebitBalance_ConcurrentMultiple(t *testing.T) {
	repo := NewUserRepository()
	// User has enough for exactly 100 debits of 100
	u := seedUser(t, repo, 10000)

	const goroutines = 200
	var wg sync.WaitGroup
	var successCount int64
	var mu sync.Mutex

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			err := repo.DebitBalance(context.Background(), u.ID, 100)
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if successCount != 100 {
		t.Fatalf("expected exactly 100 successful debits, got %d", successCount)
	}

	updated, _ := repo.GetByID(context.Background(), u.ID)
	if updated.Balance != 0 {
		t.Fatalf("expected balance 0, got %d", updated.Balance)
	}
}

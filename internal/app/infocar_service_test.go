package app

import (
	"context"
	"errors"
	"sync"
	"testing"

	"buskatotal-backend/internal/domain/user"
	"buskatotal-backend/internal/infra/infocar"
	"buskatotal-backend/internal/infra/memory"
)

func setupInfocarTest(t *testing.T, balance int64) (*InfocarService, *memory.UserRepository, user.User) {
	t.Helper()
	repo := memory.NewUserRepository()
	u, err := repo.Create(context.Background(), user.User{
		Name:    "Test User",
		Email:   "test@infocar.com",
		Balance: balance,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Client pointing to unreachable host — tests debit/rollback, not HTTP calls.
	client := infocar.NewClient("http://localhost:9999", "test-key", "user", "pass")
	svc := NewInfocarService(client, repo, 150)
	return svc, repo, u
}

// ── Product registry tests ─────────────────────────────────────────────────

func TestProducts_AllRegistered(t *testing.T) {
	expected := []string{
		"agregados", "base-estadual", "base-nacional", "gravame",
		"roubo-furto", "leilao", "aquisicoes", "debitos", "proprietario",
	}
	for _, key := range expected {
		if _, ok := Products[key]; !ok {
			t.Errorf("product %q not found in registry", key)
		}
	}
}

func TestProducts_PricesMatchTable(t *testing.T) {
	// Prices from buskatotal.md: sale = cost × 3
	cases := []struct {
		key       string
		saleCents int64
	}{
		{"agregados", 150},
		{"base-estadual", 1770},
		{"base-nacional", 1800},
		{"gravame", 2400},
		{"roubo-furto", 1560},
		{"leilao", 1560},
		{"aquisicoes", 900},
		{"debitos", 690},
		{"proprietario", 3900},
	}
	for _, tc := range cases {
		p := Products[tc.key]
		if p.SaleCents != tc.saleCents {
			t.Errorf("product %q: expected SaleCents=%d, got %d", tc.key, tc.saleCents, p.SaleCents)
		}
	}
}

func TestProducts_APIPathsMatchDocs(t *testing.T) {
	cases := []struct {
		key        string
		apiVersion string
		apiPath    string
	}{
		{"agregados", "v1.0", "AgregadosB"},
		{"base-estadual", "v1.0", "BaseEstadualB"},
		{"base-nacional", "v1.0", "BaseNacionalB"},
		{"gravame", "v1.0", "GravameB"},
		{"roubo-furto", "v1.0", "HistoricoRouboFurtoB"},
		{"leilao", "v1.0", "LeilaoEssencial"},
		{"aquisicoes", "v2.0", "InfoAquisicoes"},
		{"debitos", "v4.0", "InfoDebitos"},
		{"proprietario", "v1.0", "HistoricoProprietarioA"},
	}
	for _, tc := range cases {
		p := Products[tc.key]
		if p.APIVersion != tc.apiVersion {
			t.Errorf("product %q: expected APIVersion=%q, got %q", tc.key, tc.apiVersion, p.APIVersion)
		}
		if p.APIPath != tc.apiPath {
			t.Errorf("product %q: expected APIPath=%q, got %q", tc.key, tc.apiPath, p.APIPath)
		}
	}
}

func TestProducts_AllowedTypes(t *testing.T) {
	cases := []struct {
		key      string
		allowed  []string
		rejected []string
	}{
		{"agregados", []string{"placa", "chassi", "motor"}, []string{}},
		{"base-estadual", []string{"placa", "chassi"}, []string{"motor"}},
		{"base-nacional", []string{"placa", "chassi"}, []string{"motor"}},
		{"gravame", []string{"placa", "chassi"}, []string{"motor"}},
		{"roubo-furto", []string{"placa", "chassi"}, []string{"motor"}},
		{"leilao", []string{"placa", "chassi"}, []string{"motor"}},
		{"aquisicoes", []string{"placa"}, []string{"chassi", "motor"}},
		{"debitos", []string{"placa"}, []string{"chassi", "motor"}},
		{"proprietario", []string{"placa", "chassi"}, []string{"motor"}},
	}
	for _, tc := range cases {
		p := Products[tc.key]
		typeSet := map[string]bool{}
		for _, t := range p.AllowedTypes {
			typeSet[t] = true
		}
		for _, a := range tc.allowed {
			if !typeSet[a] {
				t.Errorf("product %q: expected type %q to be allowed", tc.key, a)
			}
		}
		for _, r := range tc.rejected {
			if typeSet[r] {
				t.Errorf("product %q: expected type %q to be rejected", tc.key, r)
			}
		}
	}
}

// ── QueryProduct service tests ─────────────────────────────────────────────

func TestQueryProduct_UnknownProduct(t *testing.T) {
	svc, _, u := setupInfocarTest(t, 10000)

	_, err := svc.QueryProduct(context.Background(), u.ID, "inexistente", "placa", "ABC1234")
	if err == nil {
		t.Fatal("expected error for unknown product")
	}
}

func TestQueryProduct_InvalidType(t *testing.T) {
	svc, _, u := setupInfocarTest(t, 10000)

	// debitos only accepts "placa"
	_, err := svc.QueryProduct(context.Background(), u.ID, "debitos", "chassi", "9BWK123")
	if err == nil {
		t.Fatal("expected error for invalid type")
	}

	// gravame does not accept "motor"
	_, err = svc.QueryProduct(context.Background(), u.ID, "gravame", "motor", "MOT123")
	if err == nil {
		t.Fatal("expected error for motor on gravame")
	}
}

func TestQueryProduct_InsufficientBalance(t *testing.T) {
	cases := []struct {
		product string
		balance int64
	}{
		{"agregados", 100},       // needs 150
		{"base-estadual", 1000},  // needs 1770
		{"base-nacional", 1500},  // needs 1800
		{"gravame", 2000},        // needs 2400
		{"roubo-furto", 1000},    // needs 1560
		{"leilao", 1000},         // needs 1560
		{"aquisicoes", 500},      // needs 900
		{"debitos", 500},         // needs 690
		{"proprietario", 3000},   // needs 3900
	}

	for _, tc := range cases {
		t.Run(tc.product, func(t *testing.T) {
			svc, repo, u := setupInfocarTest(t, tc.balance)

			_, err := svc.QueryProduct(context.Background(), u.ID, tc.product, "placa", "ABC1234")
			if err == nil {
				t.Fatal("expected error for insufficient balance")
			}
			if !errors.Is(err, user.ErrInsufficientBalance) {
				t.Fatalf("expected ErrInsufficientBalance, got: %v", err)
			}

			// Balance unchanged
			updated, _ := repo.GetByID(context.Background(), u.ID)
			if updated.Balance != tc.balance {
				t.Fatalf("expected balance %d, got %d", tc.balance, updated.Balance)
			}
		})
	}
}

func TestQueryProduct_RollbackOnAPIFailure(t *testing.T) {
	// Client points to unreachable host — API call will fail, balance should rollback.
	cases := []string{
		"agregados", "base-estadual", "base-nacional", "gravame",
		"roubo-furto", "leilao", "aquisicoes", "debitos", "proprietario",
	}

	for _, product := range cases {
		t.Run(product, func(t *testing.T) {
			initialBalance := int64(100000) // R$1000
			svc, repo, u := setupInfocarTest(t, initialBalance)

			_, err := svc.QueryProduct(context.Background(), u.ID, product, "placa", "ABC1234")
			if err == nil {
				t.Fatal("expected error from unreachable API")
			}

			// Balance should be restored after rollback
			updated, _ := repo.GetByID(context.Background(), u.ID)
			if updated.Balance != initialBalance {
				t.Fatalf("expected balance %d after rollback, got %d", initialBalance, updated.Balance)
			}
		})
	}
}

func TestQueryProduct_EmptyUserID(t *testing.T) {
	svc, _, _ := setupInfocarTest(t, 10000)

	_, err := svc.QueryProduct(context.Background(), "", "agregados", "placa", "ABC1234")
	if err == nil {
		t.Fatal("expected error for empty user ID")
	}
}

func TestQueryProduct_TypeNormalization(t *testing.T) {
	svc, _, u := setupInfocarTest(t, 100000)

	// Should normalize "PLACA" to "placa" and accept it
	_, err := svc.QueryProduct(context.Background(), u.ID, "agregados", "PLACA", "ABC1234")
	// Error is expected (unreachable API), but it should NOT be a "tipo" validation error
	if err != nil && err.Error() == "tipo deve ser: placa, chassi, motor" {
		t.Fatal("type normalization failed — PLACA should be accepted")
	}

	// " Chassi " with spaces should also work
	_, err = svc.QueryProduct(context.Background(), u.ID, "base-estadual", " Chassi ", "9BWK123")
	if err != nil && err.Error() == "tipo deve ser: placa, chassi" {
		t.Fatal("type normalization failed — ' Chassi ' should be accepted")
	}
}

// ── GetAgregadosB backwards compatibility ──────────────────────────────────

func TestGetAgregadosB_BackwardsCompat(t *testing.T) {
	svc, repo, u := setupInfocarTest(t, 100000)

	// GetAgregadosB should still work and debit the correct amount
	_, err := svc.GetAgregadosB(context.Background(), u.ID, "placa", "ABC1234")
	// API fails (unreachable), balance should rollback
	if err == nil {
		t.Fatal("expected error from unreachable API")
	}

	updated, _ := repo.GetByID(context.Background(), u.ID)
	if updated.Balance != 100000 {
		t.Fatalf("expected balance 100000 after rollback, got %d", updated.Balance)
	}
}

func TestGetAgregadosB_InvalidType(t *testing.T) {
	svc, _, u := setupInfocarTest(t, 100000)

	_, err := svc.GetAgregadosB(context.Background(), u.ID, "cpf", "12345678900")
	if err == nil {
		t.Fatal("expected error for invalid type 'cpf'")
	}
}

// ── Concurrent access ──────────────────────────────────────────────────────

func TestQueryProduct_ConcurrentDebit(t *testing.T) {
	// 10 goroutines try to query simultaneously with limited balance.
	// Only some should succeed in debiting (before API fails and rolls back).
	svc, repo, u := setupInfocarTest(t, 500) // only enough for 3× agregados (150 each)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc.QueryProduct(context.Background(), u.ID, "agregados", "placa", "ABC1234")
			// All will fail at API level, so balance rolls back
		}()
	}
	wg.Wait()

	// After all rollbacks, balance should be unchanged
	updated, _ := repo.GetByID(context.Background(), u.ID)
	if updated.Balance != 500 {
		t.Fatalf("expected balance 500 after concurrent rollbacks, got %d", updated.Balance)
	}
}

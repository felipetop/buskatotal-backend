package app

import (
	"context"
	"testing"

	"buskatotal-backend/internal/domain/user"
	"buskatotal-backend/internal/infra/apifull"
	"buskatotal-backend/internal/infra/memory"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

func setupApiFullTest(t *testing.T, balance int64) (*ApiFullService, *memory.UserRepository, user.User) {
	t.Helper()
	repo := memory.NewUserRepository()
	u, err := repo.Create(context.Background(), user.User{
		Name:    "Test User",
		Email:   "test@apifull.com",
		Balance: balance,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	client := apifull.NewClient("http://localhost:9999", "test-token")
	svc := NewApiFullService(client, repo)
	return svc, repo, u
}

// ── Product registry tests ───────────────────────────────────────────────────

func TestApiFullProducts_AllRegistered(t *testing.T) {
	expected := []string{
		"cpf-simples", "cpf-completo", "cpf-ultra", "busca-nome", "busca-telefone", "cnpj",
		"placa-basica", "bin-estadual", "bin-nacional", "foto-leilao", "leilao-apifull",
		"historico-roubo-furto", "indice-risco", "proprietario-placa", "recall",
		"gravame-apifull", "fipe", "csv", "crlv", "roubo-furto-apifull",
		"spc-srs", "serasa-premium", "cred-completa", "boavista-essencial",
		"scpc-bv-basica", "cadastrais-score-dividas", "cadastrais-score-dividas-cp",
		"scr-bacen", "cenprot", "quod",
		"acoes-processos", "dossie-juridico", "cndt",
	}
	for _, key := range expected {
		if _, ok := ApiFullProducts[key]; !ok {
			t.Errorf("product %q not found in registry", key)
		}
	}
}

func TestApiFullProducts_Count(t *testing.T) {
	if got := len(ApiFullProducts); got != 33 {
		t.Errorf("expected 33 products, got %d", got)
	}
}

func TestApiFullProducts_EndpointLinkConsistency(t *testing.T) {
	// Products where Link intentionally differs from Endpoint
	exceptions := map[string]bool{
		"indice-risco":    true, // API Full typo: endpoint=inde-risco, link=indice-risco
		"acoes-processos": true, // endpoint=r-acoes-e-processos-judiciais, link=ic-processos-judiciais
	}

	for key, p := range ApiFullProducts {
		if exceptions[key] {
			continue
		}
		if p.Endpoint != p.Link {
			t.Errorf("product %q: Endpoint=%q != Link=%q (add to exceptions if intentional)", key, p.Endpoint, p.Link)
		}
	}
}

func TestApiFullProducts_BuscaTelefoneLink(t *testing.T) {
	p := ApiFullProducts["busca-telefone"]
	if p.Link != "ic-telefone" {
		t.Errorf("busca-telefone Link should be 'ic-telefone', got %q", p.Link)
	}
}

func TestApiFullProducts_PricesMatchTable(t *testing.T) {
	cases := []struct {
		key       string
		saleCents int64
	}{
		{"cpf-simples", 30},
		{"cpf-completo", 180},
		{"cpf-ultra", 351},
		{"busca-nome", 450},
		{"busca-telefone", 450},
		{"cnpj", 18},
		{"placa-basica", 30},
		{"bin-estadual", 828},
		{"bin-nacional", 900},
		{"foto-leilao", 3600},
		{"leilao-apifull", 2628},
		{"historico-roubo-furto", 2808},
		{"indice-risco", 1872},
		{"proprietario-placa", 273},
		{"recall", 1080},
		{"gravame-apifull", 660},
		{"fipe", 33},
		{"csv", 660},
		{"crlv", 6084},
		{"roubo-furto-apifull", 1080},
		{"spc-srs", 2484},
		{"serasa-premium", 2088},
		{"cred-completa", 747},
		{"boavista-essencial", 969},
		{"scpc-bv-basica", 720},
		{"cadastrais-score-dividas", 828},
		{"cadastrais-score-dividas-cp", 897},
		{"scr-bacen", 2808},
		{"cenprot", 120},
		{"quod", 1434},
		{"acoes-processos", 1242},
		{"dossie-juridico", 3528},
		{"cndt", 2160},
	}
	for _, tc := range cases {
		p, ok := ApiFullProducts[tc.key]
		if !ok {
			t.Errorf("product %q not found", tc.key)
			continue
		}
		if p.SaleCents != tc.saleCents {
			t.Errorf("product %q: sale=%d, want %d", tc.key, p.SaleCents, tc.saleCents)
		}
	}
}

// ── Input validation tests ───────────────────────────────────────────────────

func TestValidateInput_CPF(t *testing.T) {
	tests := []struct {
		value string
		valid bool
	}{
		{"12345678900", true},
		{"00000000000", true},
		{"1234567890", false},   // 10 digits
		{"123456789001", false}, // 12 digits
		{"1234567890a", false},  // letter
		{"", false},
		{"123.456.789-00", false}, // formatted
	}
	for _, tc := range tests {
		err := validateInput("cpf", tc.value)
		if tc.valid && err != nil {
			t.Errorf("cpf %q should be valid, got: %v", tc.value, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("cpf %q should be invalid", tc.value)
		}
	}
}

func TestValidateInput_CNPJ(t *testing.T) {
	tests := []struct {
		value string
		valid bool
	}{
		{"12345678000190", true},
		{"00000000000000", true},
		{"1234567800019", false},  // 13 digits
		{"123456780001901", false}, // 15 digits
		{"12.345.678/0001-90", false}, // formatted
	}
	for _, tc := range tests {
		err := validateInput("cnpj", tc.value)
		if tc.valid && err != nil {
			t.Errorf("cnpj %q should be valid, got: %v", tc.value, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("cnpj %q should be invalid", tc.value)
		}
	}
}

func TestValidateInput_Document(t *testing.T) {
	tests := []struct {
		value string
		valid bool
	}{
		{"12345678900", true},      // CPF
		{"12345678000190", true},   // CNPJ
		{"1234567890", false},      // 10 digits
		{"abcdefghijk", false},     // letters
	}
	for _, tc := range tests {
		err := validateInput("document", tc.value)
		if tc.valid && err != nil {
			t.Errorf("document %q should be valid, got: %v", tc.value, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("document %q should be invalid", tc.value)
		}
	}
}

func TestValidateInput_Placa(t *testing.T) {
	tests := []struct {
		value string
		valid bool
	}{
		{"ABC1234", true},   // old format
		{"ABC1D23", true},   // Mercosul format
		{"abc1d23", true},   // lowercase
		{"AB1234", false},   // missing letter
		{"ABCD123", false},  // extra letter
		{"ABC-1234", false}, // with dash
		{"", false},
	}
	for _, tc := range tests {
		err := validateInput("placa", tc.value)
		if tc.valid && err != nil {
			t.Errorf("placa %q should be valid, got: %v", tc.value, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("placa %q should be invalid", tc.value)
		}
	}
}

func TestValidateInput_Nome(t *testing.T) {
	tests := []struct {
		value string
		valid bool
	}{
		{"JOAO SILVA|SP", true},
		{"Ana", true},
		{"AB", false}, // too short
	}
	for _, tc := range tests {
		err := validateInput("nome", tc.value)
		if tc.valid && err != nil {
			t.Errorf("nome %q should be valid, got: %v", tc.value, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("nome %q should be invalid", tc.value)
		}
	}
}

// ── Service behavior tests ───────────────────────────────────────────────────

func TestApiFull_QueryProduct_UnknownProduct(t *testing.T) {
	svc, _, u := setupApiFullTest(t, 100000)
	_, err := svc.QueryProduct(context.Background(), u.ID, "produto-invalido", "12345678900")
	if err == nil {
		t.Error("expected error for unknown product")
	}
}

func TestApiFull_QueryProduct_EmptyValue(t *testing.T) {
	svc, _, u := setupApiFullTest(t, 100000)
	_, err := svc.QueryProduct(context.Background(), u.ID, "cpf-simples", "")
	if err == nil {
		t.Error("expected error for empty value")
	}
}

func TestApiFull_QueryProduct_EmptyUserID(t *testing.T) {
	svc, _, _ := setupApiFullTest(t, 100000)
	_, err := svc.QueryProduct(context.Background(), "", "cpf-simples", "12345678900")
	if err == nil {
		t.Error("expected error for empty user ID")
	}
}

func TestApiFull_QueryProduct_InvalidCPFFormat(t *testing.T) {
	svc, repo, u := setupApiFullTest(t, 100000)
	_, err := svc.QueryProduct(context.Background(), u.ID, "cpf-simples", "123")
	if err == nil {
		t.Error("expected error for invalid CPF")
	}
	// Balance should NOT be debited for validation errors
	updated, _ := repo.GetByID(context.Background(), u.ID)
	if updated.Balance != 100000 {
		t.Errorf("balance should be unchanged (100000), got %d", updated.Balance)
	}
}

func TestApiFull_QueryProduct_InvalidPlacaFormat(t *testing.T) {
	svc, repo, u := setupApiFullTest(t, 100000)
	_, err := svc.QueryProduct(context.Background(), u.ID, "placa-basica", "INVALIDA")
	if err == nil {
		t.Error("expected error for invalid placa")
	}
	updated, _ := repo.GetByID(context.Background(), u.ID)
	if updated.Balance != 100000 {
		t.Errorf("balance should be unchanged (100000), got %d", updated.Balance)
	}
}

func TestApiFull_QueryProduct_InsufficientBalance(t *testing.T) {
	svc, _, u := setupApiFullTest(t, 10) // 10 cents, cpf-simples costs 30
	_, err := svc.QueryProduct(context.Background(), u.ID, "cpf-simples", "12345678900")
	if err == nil {
		t.Error("expected error for insufficient balance")
	}
}

func TestApiFull_QueryProduct_DebitsOnAPICall(t *testing.T) {
	svc, repo, u := setupApiFullTest(t, 100000)
	// API call will fail (unreachable host) but balance should be rolled back
	_, err := svc.QueryProduct(context.Background(), u.ID, "cpf-simples", "12345678900")
	if err == nil {
		t.Error("expected error from unreachable API")
	}
	// Balance should be restored after API failure
	updated, _ := repo.GetByID(context.Background(), u.ID)
	if updated.Balance != 100000 {
		t.Errorf("balance should be restored to 100000 after API failure, got %d", updated.Balance)
	}
}

func TestApiFull_QueryProduct_TelefoneRequiresPipe(t *testing.T) {
	svc, _, u := setupApiFullTest(t, 100000)
	_, err := svc.QueryProduct(context.Background(), u.ID, "busca-telefone", "11999999999")
	if err == nil {
		t.Error("expected error for telefone without pipe separator")
	}
}

func TestApiFull_QueryProduct_TelefoneValidFormat(t *testing.T) {
	svc, repo, u := setupApiFullTest(t, 100000)
	// Will fail on API call but should pass validation
	_, err := svc.QueryProduct(context.Background(), u.ID, "busca-telefone", "11|999999999")
	if err == nil {
		t.Error("expected error from unreachable API")
	}
	// Balance should be restored (API failure rollback)
	updated, _ := repo.GetByID(context.Background(), u.ID)
	if updated.Balance != 100000 {
		t.Errorf("balance should be restored after API failure, got %d", updated.Balance)
	}
}

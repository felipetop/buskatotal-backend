package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"buskatotal-backend/internal/domain/user"
	"buskatotal-backend/internal/infra/apifull"
)

// ApiFullProductConfig defines an API Full product.
type ApiFullProductConfig struct {
	Key       string // route key, e.g. "cpf-simples"
	Name      string // display name
	Endpoint  string // API Full endpoint path, e.g. "pf-dadosbasicos"
	Link      string // "link" field value in request body
	InputType string // "cpf", "cnpj", "document", "placa", "nome", "telefone"
	SaleCents int64  // price charged to user
}

// ApiFullProducts is the registry of all API Full products.
// Prices from buskatotal.md — sale = custo × 3 (markup 2.0).
var ApiFullProducts = map[string]ApiFullProductConfig{
	// ── Dados Cadastrais ──
	"cpf-simples": {
		Key: "cpf-simples", Name: "CPF Simples",
		Endpoint: "pf-dadosbasicos", Link: "pf-dadosbasicos",
		InputType: "cpf", SaleCents: 30,
	},
	"cpf-completo": {
		Key: "cpf-completo", Name: "CPF Completo",
		Endpoint: "ic-cpf-completo", Link: "ic-cpf-completo",
		InputType: "cpf", SaleCents: 180,
	},
	"cpf-ultra": {
		Key: "cpf-ultra", Name: "CPF Ultra Completo",
		Endpoint: "cpf-ultra", Link: "cpf-ultra",
		InputType: "cpf", SaleCents: 351,
	},
	"busca-nome": {
		Key: "busca-nome", Name: "Busca pelo nome",
		Endpoint: "ic-nome", Link: "ic-nome",
		InputType: "nome", SaleCents: 450,
	},
	"busca-telefone": {
		Key: "busca-telefone", Name: "Busca pelo telefone",
		Endpoint: "ic-telefone", Link: "ic-nome",
		InputType: "telefone", SaleCents: 450,
	},
	"cnpj": {
		Key: "cnpj", Name: "CNPJ Completo",
		Endpoint: "cnpj", Link: "cnpj",
		InputType: "cnpj", SaleCents: 18,
	},

	// ── Veicular ──
	"placa-basica": {
		Key: "placa-basica", Name: "Placa Básica",
		Endpoint: "agregados-propria", Link: "agregados-propria",
		InputType: "placa", SaleCents: 30,
	},
	"bin-estadual": {
		Key: "bin-estadual", Name: "BIN Estadual",
		Endpoint: "ic-bin-estadual", Link: "ic-bin-estadual",
		InputType: "placa", SaleCents: 828,
	},
	"bin-nacional": {
		Key: "bin-nacional", Name: "BIN Nacional",
		Endpoint: "ic-bin-nacional", Link: "ic-bin-nacional",
		InputType: "placa", SaleCents: 900,
	},
	"foto-leilao": {
		Key: "foto-leilao", Name: "Foto Leilão",
		Endpoint: "ic-foto-leilao", Link: "ic-foto-leilao",
		InputType: "placa", SaleCents: 3600,
	},
	"leilao-apifull": {
		Key: "leilao-apifull", Name: "Leilão",
		Endpoint: "leilao", Link: "leilao",
		InputType: "placa", SaleCents: 2628,
	},
	"historico-roubo-furto": {
		Key: "historico-roubo-furto", Name: "Histórico de roubo ou furto",
		Endpoint: "ic-historico-roubo-furto", Link: "ic-historico-roubo-furto",
		InputType: "placa", SaleCents: 2808,
	},
	"indice-risco": {
		Key: "indice-risco", Name: "Índice de risco veicular",
		Endpoint: "inde-risco", Link: "indice-risco",
		InputType: "placa", SaleCents: 1872,
	},
	"proprietario-placa": {
		Key: "proprietario-placa", Name: "Proprietário placa",
		Endpoint: "ic-proprietario-atual", Link: "ic-proprietario-atual",
		InputType: "placa", SaleCents: 273,
	},
	"recall": {
		Key: "recall", Name: "Recall",
		Endpoint: "ic-recall", Link: "ic-recall",
		InputType: "placa", SaleCents: 1080,
	},
	"gravame-apifull": {
		Key: "gravame-apifull", Name: "Gravame",
		Endpoint: "gravame", Link: "gravame",
		InputType: "placa", SaleCents: 660,
	},
	"fipe": {
		Key: "fipe", Name: "Fipe",
		Endpoint: "fipe", Link: "fipe",
		InputType: "placa", SaleCents: 33,
	},
	"csv": {
		Key: "csv", Name: "Certificado de Segurança Veicular",
		Endpoint: "csv-renainf-renajud-recall-bin-proprietario", Link: "csv-renainf-renajud-recall-bin-proprietario",
		InputType: "placa", SaleCents: 660,
	},
	"crlv": {
		Key: "crlv", Name: "CRLV",
		Endpoint: "crlv", Link: "crlv",
		InputType: "placa", SaleCents: 6084,
	},
	"roubo-furto-apifull": {
		Key: "roubo-furto-apifull", Name: "Histórico de roubo e furto",
		Endpoint: "roubo-furto", Link: "roubo-furto",
		InputType: "placa", SaleCents: 1080,
	},

	// ── Dívidas e Crédito ──
	"spc-srs": {
		Key: "spc-srs", Name: "SPC e SRS",
		Endpoint: "r-spc-srs", Link: "r-spc-srs",
		InputType: "cpf", SaleCents: 2484,
	},
	"serasa-premium": {
		Key: "serasa-premium", Name: "Serasa Premium",
		Endpoint: "serasa-premium", Link: "serasa-premium",
		InputType: "document", SaleCents: 2088,
	},
	"cred-completa": {
		Key: "cred-completa", Name: "Cred Completa Plus",
		Endpoint: "e-boavista", Link: "e-boavista",
		InputType: "document", SaleCents: 747,
	},
	"boavista-essencial": {
		Key: "boavista-essencial", Name: "Boa Vista Essencial Positivo",
		Endpoint: "scpc-boavista", Link: "scpc-boavista",
		InputType: "document", SaleCents: 969,
	},
	"scpc-bv-basica": {
		Key: "scpc-bv-basica", Name: "SCPC BV Básica",
		Endpoint: "r-bv-basica", Link: "r-bv-basica",
		InputType: "cpf", SaleCents: 720,
	},
	"cadastrais-score-dividas": {
		Key: "cadastrais-score-dividas", Name: "Dados cadastrais, score e dívidas",
		Endpoint: "r-cadastrais-score-dividas", Link: "r-cadastrais-score-dividas",
		InputType: "document", SaleCents: 828,
	},
	"cadastrais-score-dividas-cp": {
		Key: "cadastrais-score-dividas-cp", Name: "Dados cadastrais, score e dívidas CP",
		Endpoint: "cp-cadastrais-score-dividas", Link: "cp-cadastrais-score-dividas",
		InputType: "document", SaleCents: 897,
	},
	"scr-bacen": {
		Key: "scr-bacen", Name: "SCR e Score (BACEN)",
		Endpoint: "ic-bacen", Link: "ic-bacen",
		InputType: "document", SaleCents: 2808,
	},
	"cenprot": {
		Key: "cenprot", Name: "CENPROT V2",
		Endpoint: "e-cenprot", Link: "e-cenprot",
		InputType: "document", SaleCents: 120,
	},
	"quod": {
		Key: "quod", Name: "QUOD",
		Endpoint: "ic-quod", Link: "ic-quod",
		InputType: "document", SaleCents: 1434,
	},

	// ── Jurídico ──
	"acoes-processos": {
		Key: "acoes-processos", Name: "Ações e processos judiciais",
		Endpoint: "ic-processos-judiciais", Link: "ic-processos-judiciais",
		InputType: "cpf", SaleCents: 1242,
	},
	"dossie-juridico": {
		Key: "dossie-juridico", Name: "Dossiê Jurídico",
		Endpoint: "ic-dossie-juridico", Link: "ic-dossie-juridico",
		InputType: "cpf", SaleCents: 3528,
	},
	"cndt": {
		Key: "cndt", Name: "Certidão Nacional de Débitos Trabalhistas",
		Endpoint: "ic-cndt", Link: "ic-cndt",
		InputType: "cpf", SaleCents: 2160,
	},
}

type ApiFullService struct {
	client   *apifull.Client
	userRepo user.Repository
}

func NewApiFullService(client *apifull.Client, userRepo user.Repository) *ApiFullService {
	return &ApiFullService{
		client:   client,
		userRepo: userRepo,
	}
}

// QueryProduct queries any registered API Full product.
func (s *ApiFullService) QueryProduct(ctx context.Context, userID, productKey, value string) (*apifull.ProductResponse, error) {
	product, ok := ApiFullProducts[productKey]
	if !ok {
		return nil, fmt.Errorf("unknown product: %s", productKey)
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("valor é obrigatório")
	}
	if userID == "" {
		return nil, errors.New("user id is required")
	}

	// Debit balance
	if err := s.userRepo.DebitBalance(ctx, userID, product.SaleCents); err != nil {
		return nil, err
	}

	// Build request body based on input type
	body := map[string]interface{}{
		"link": product.Link,
	}

	switch product.InputType {
	case "cpf":
		body["cpf"] = value
	case "cnpj":
		body["cnpj"] = value
	case "document":
		body["document"] = value
	case "placa":
		body["placa"] = value
	case "nome":
		// nome requires state — extract if provided as "nome|UF"
		parts := strings.SplitN(value, "|", 2)
		body["name"] = strings.TrimSpace(parts[0])
		if len(parts) > 1 {
			body["state"] = strings.TrimSpace(parts[1])
		}
	case "telefone":
		// telefone requires ddd and number — extract from "DDD|NUMERO"
		parts := strings.SplitN(value, "|", 2)
		if len(parts) < 2 {
			s.userRepo.CreditBalance(ctx, userID, product.SaleCents)
			return nil, errors.New("telefone deve ser enviado como DDD|NUMERO (ex: 11|999999999)")
		}
		body["ddd"] = strings.TrimSpace(parts[0])
		body["telefone"] = strings.TrimSpace(parts[1])
	default:
		s.userRepo.CreditBalance(ctx, userID, product.SaleCents)
		return nil, fmt.Errorf("input type not supported: %s", product.InputType)
	}

	result, err := s.client.QueryProduct(ctx, product.Endpoint, body)
	if err != nil {
		s.userRepo.CreditBalance(ctx, userID, product.SaleCents)
		return nil, err
	}

	return result, nil
}

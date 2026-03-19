package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"buskatotal-backend/internal/domain/user"
	"buskatotal-backend/internal/infra/infocar"
)

// ProductConfig defines an Infocar product with its API route and allowed input types.
type ProductConfig struct {
	Key          string   // internal key used in routes, e.g. "base-estadual-b"
	Name         string   // display name
	APIVersion   string   // e.g. "v1.0", "v2.0", "v4.0"
	APIPath      string   // e.g. "BaseEstadualB"
	SaleCents    int64    // price charged to the user (sale price in cents)
	AllowedTypes []string // e.g. ["placa", "chassi"] or ["placa", "chassi", "motor"]
}

// Products is the registry of all Infocar products.
// Prices from buskatotal.md — sale = custo × 3 (markup 2.0).
var Products = map[string]ProductConfig{
	"agregados": {
		Key: "agregados", Name: "AGREGADOS B",
		APIVersion: "v1.0", APIPath: "AgregadosB",
		SaleCents: 150, AllowedTypes: []string{"placa", "chassi", "motor"},
	},
	"base-estadual": {
		Key: "base-estadual", Name: "BASE ESTADUAL B",
		APIVersion: "v1.0", APIPath: "BaseEstadualB",
		SaleCents: 1770, AllowedTypes: []string{"placa", "chassi"},
	},
	"base-nacional": {
		Key: "base-nacional", Name: "BASE NACIONAL B",
		APIVersion: "v1.0", APIPath: "BaseNacionalB",
		SaleCents: 1800, AllowedTypes: []string{"placa", "chassi"},
	},
	"gravame": {
		Key: "gravame", Name: "GRAVAME B",
		APIVersion: "v1.0", APIPath: "GravameB",
		SaleCents: 2400, AllowedTypes: []string{"placa", "chassi"},
	},
	"roubo-furto": {
		Key: "roubo-furto", Name: "HISTÓRICO ROUBO E FURTO B",
		APIVersion: "v1.0", APIPath: "HistoricoRouboFurtoB",
		SaleCents: 1560, AllowedTypes: []string{"placa", "chassi"},
	},
	"leilao": {
		Key: "leilao", Name: "LEILÃO ESSENCIAL",
		APIVersion: "v1.0", APIPath: "LeilaoEssencial",
		SaleCents: 1560, AllowedTypes: []string{"placa", "chassi"},
	},
	"aquisicoes": {
		Key: "aquisicoes", Name: "INFO AQUISIÇÕES",
		APIVersion: "v2.0", APIPath: "InfoAquisicoes",
		SaleCents: 900, AllowedTypes: []string{"placa"},
	},
	"debitos": {
		Key: "debitos", Name: "INFO DÉBITOS",
		APIVersion: "v4.0", APIPath: "InfoDebitos",
		SaleCents: 690, AllowedTypes: []string{"placa"},
	},
	"proprietario": {
		Key: "proprietario", Name: "HISTÓRICO PROPRIETÁRIO",
		APIVersion: "v1.0", APIPath: "HistoricoProprietarioA",
		SaleCents: 3900, AllowedTypes: []string{"placa", "chassi"},
	},
}

type InfocarService struct {
	client   *infocar.Client
	userRepo user.Repository
	mu       sync.Mutex
	token    string
	expiry   time.Time

	// Keep for backwards compat — GetAgregadosB still works.
	costPerQuery int64
}

func NewInfocarService(client *infocar.Client, userRepo user.Repository, costPerQuery int64) *InfocarService {
	return &InfocarService{
		client:       client,
		userRepo:     userRepo,
		costPerQuery: costPerQuery,
	}
}

// GetAgregadosB keeps backwards compatibility with existing handler.
func (s *InfocarService) GetAgregadosB(ctx context.Context, userID, queryType, value string) (*infocar.AgregadosBResponse, error) {
	return s.QueryProduct(ctx, userID, "agregados", queryType, value)
}

// QueryProduct queries any registered Infocar product.
func (s *InfocarService) QueryProduct(ctx context.Context, userID, productKey, queryType, value string) (*infocar.ProductResponse, error) {
	product, ok := Products[productKey]
	if !ok {
		return nil, fmt.Errorf("unknown product: %s", productKey)
	}

	normalizedType := strings.ToLower(strings.TrimSpace(queryType))
	allowed := false
	for _, t := range product.AllowedTypes {
		if t == normalizedType {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("tipo deve ser: %s", strings.Join(product.AllowedTypes, ", "))
	}

	if userID == "" {
		return nil, errors.New("user id is required")
	}

	if err := s.userRepo.DebitBalance(ctx, userID, product.SaleCents); err != nil {
		return nil, err
	}

	token, err := s.getToken(ctx)
	if err != nil {
		s.userRepo.CreditBalance(ctx, userID, product.SaleCents)
		return nil, err
	}

	result, err := s.client.QueryProduct(ctx, token, product.APIVersion, product.APIPath, normalizedType, value)
	if err != nil {
		s.userRepo.CreditBalance(ctx, userID, product.SaleCents)
		return nil, err
	}

	return result, nil
}

func (s *InfocarService) getToken(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token != "" && time.Now().Before(s.expiry) {
		return s.token, nil
	}

	token, err := s.client.GenerateToken(ctx)
	if err != nil {
		return "", err
	}

	s.token = token
	s.expiry = time.Now().Add(8 * time.Hour).Add(-5 * time.Minute)
	return s.token, nil
}

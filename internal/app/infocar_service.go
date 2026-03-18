package app

import (
    "context"
    "errors"
    "strings"
    "sync"
    "time"

    "buskatotal-backend/internal/domain/user"
    "buskatotal-backend/internal/infra/infocar"
)

type InfocarService struct {
    client *infocar.Client
    userRepo user.Repository
    costPerQuery int64
    mu     sync.Mutex
    token  string
    expiry time.Time
}

func NewInfocarService(client *infocar.Client, userRepo user.Repository, costPerQuery int64) *InfocarService {
    return &InfocarService{
        client: client,
        userRepo: userRepo,
        costPerQuery: costPerQuery,
    }
}

func (s *InfocarService) GetAgregadosB(ctx context.Context, userID, queryType, value string) (*infocar.AgregadosBResponse, error) {
    normalizedType := strings.ToLower(strings.TrimSpace(queryType))
    if normalizedType != "placa" && normalizedType != "chassi" && normalizedType != "motor" {
        return nil, errors.New("query type must be placa, chassi or motor")
    }
    if userID == "" {
        return nil, errors.New("user id is required")
    }

    if err := s.userRepo.DebitBalance(ctx, userID, s.costPerQuery); err != nil {
        return nil, err
    }

    token, err := s.getToken(ctx)
    if err != nil {
        s.userRepo.CreditBalance(ctx, userID, s.costPerQuery)
        return nil, err
    }

    result, err := s.client.QueryAgregadosB(ctx, token, normalizedType, value)
    if err != nil {
        s.userRepo.CreditBalance(ctx, userID, s.costPerQuery)
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

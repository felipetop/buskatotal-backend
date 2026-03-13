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

    if err := s.debitBalance(ctx, userID); err != nil {
        return nil, err
    }

    token, err := s.getToken(ctx)
    if err != nil {
        return nil, err
    }

    return s.client.QueryAgregadosB(ctx, token, normalizedType, value)
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

    // Token expira em 8 horas conforme manual.
    s.token = token
    s.expiry = time.Now().Add(8 * time.Hour).Add(-5 * time.Minute)
    return s.token, nil
}

func (s *InfocarService) debitBalance(ctx context.Context, userID string) error {
    if s.userRepo == nil {
        return errors.New("user repository not configured")
    }

    userEntity, err := s.userRepo.GetByID(ctx, userID)
    if err != nil {
        return err
    }
    if userEntity.Balance < s.costPerQuery {
        return errors.New("insufficient balance")
    }

    userEntity.Balance -= s.costPerQuery
    if _, err := s.userRepo.Update(ctx, userEntity); err != nil {
        return err
    }
    return nil
}
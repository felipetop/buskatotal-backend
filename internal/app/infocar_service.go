package app

import (
    "context"
    "errors"
    "strings"
    "sync"
    "time"

    "buskatotal-backend/internal/infra/infocar"
)

type InfocarService struct {
    client *infocar.Client
    mu     sync.Mutex
    token  string
    expiry time.Time
}

func NewInfocarService(client *infocar.Client) *InfocarService {
    return &InfocarService{client: client}
}

func (s *InfocarService) GetAgregadosB(ctx context.Context, queryType, value string) (*infocar.AgregadosBResponse, error) {
    normalizedType := strings.ToLower(strings.TrimSpace(queryType))
    if normalizedType != "placa" && normalizedType != "chassi" && normalizedType != "motor" {
        return nil, errors.New("query type must be placa, chassi or motor")
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
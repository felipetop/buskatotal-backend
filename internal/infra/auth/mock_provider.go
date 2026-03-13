package auth

import (
    "context"
    "errors"
    "strings"

    domain "buskatotal-backend/internal/domain/auth"
)

type MockProvider struct {
    // headerName allows local tests to pass user id directly via header.
    headerName string
}

func NewMockProvider(headerName string) *MockProvider {
    return &MockProvider{headerName: headerName}
}

func (p *MockProvider) Authenticate(ctx context.Context, token string) (domain.Result, error) {
    if strings.TrimSpace(token) == "" {
        return domain.Result{}, errors.New("missing auth token")
    }
    return domain.Result{UserID: token}, nil
}

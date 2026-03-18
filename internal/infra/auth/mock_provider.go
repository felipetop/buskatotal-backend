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
    // In mock mode, allow passing "userID:admin" to simulate admin role
    parts := strings.SplitN(token, ":", 2)
    userID := parts[0]
    role := "user"
    if len(parts) == 2 && parts[1] == "admin" {
        role = "admin"
    }
    return domain.Result{UserID: userID, Role: role}, nil
}

package auth

import "context"

type Provider interface {
    Authenticate(ctx context.Context, token string) (Result, error)
}

type Result struct {
    UserID string
    Role   string
}
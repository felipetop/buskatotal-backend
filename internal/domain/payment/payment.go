package payment

import "context"

type Provider interface {
    Credit(ctx context.Context, userID string, amount int64) (Receipt, error)
}

type Receipt struct {
    Provider string
    Reference string
    Amount int64
}
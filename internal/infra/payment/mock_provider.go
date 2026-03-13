package payment

import (
    "context"
    "fmt"
    "time"

    domain "buskatotal-backend/internal/domain/payment"
)

type MockProvider struct {}

func NewMockProvider() *MockProvider {
    return &MockProvider{}
}

func (p *MockProvider) Credit(ctx context.Context, userID string, amount int64) (domain.Receipt, error) {
    reference := fmt.Sprintf("mock-%d", time.Now().UnixNano())
    return domain.Receipt{
        Provider: "mock",
        Reference: reference,
        Amount: amount,
    }, nil
}
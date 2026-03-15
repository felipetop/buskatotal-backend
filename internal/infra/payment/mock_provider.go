package payment

import (
	"context"
	"fmt"
	"time"

	domain "buskatotal-backend/internal/domain/payment"
)

// MockProvider is used in development / tests.
type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (p *MockProvider) CreateOrder(_ context.Context, input domain.CreateOrderInput) (domain.OrderResult, error) {
	return domain.OrderResult{
		ReferenceID:  input.ReferenceID,
		PaymentURL:   fmt.Sprintf("https://mock.picpay.com/checkout/%s", input.ReferenceID),
		QRCodeText:   fmt.Sprintf("00020101021226mock%s", input.ReferenceID),
		QRCodeBase64: "data:image/png;base64,mock",
		ExpiresAt:    time.Now().Add(30 * time.Minute),
	}, nil
}

func (p *MockProvider) GetOrderStatus(_ context.Context, referenceID string) (domain.OrderStatus, error) {
	// Mock always returns paid so integration tests can complete the flow.
	_ = referenceID
	return domain.StatusPaid, nil
}

func (p *MockProvider) Credit(_ context.Context, userID string, amount int64) (domain.Receipt, error) {
	reference := fmt.Sprintf("mock-%d", time.Now().UnixNano())
	return domain.Receipt{
		Provider:  "mock",
		Reference: reference,
		Amount:    amount,
	}, nil
}

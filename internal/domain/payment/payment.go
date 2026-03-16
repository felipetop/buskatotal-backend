package payment

import (
	"context"
	"time"
)

// OrderStatus represents the lifecycle of a PicPay payment order.
type OrderStatus string

const (
	StatusPending    OrderStatus = "pending"
	StatusPaid       OrderStatus = "paid"
	StatusExpired    OrderStatus = "expired"
	StatusCancelled  OrderStatus = "cancelled"
	StatusChargeback OrderStatus = "chargeback"
)

// Order is the aggregate root for a payment transaction.
type Order struct {
	ID           string      `json:"order_id" firestore:"ID"`
	UserID       string      `json:"user_id" firestore:"UserID"`
	ReferenceID  string      `json:"reference_id" firestore:"ReferenceID"`
	AmountCents  int64       `json:"amount_cents" firestore:"AmountCents"`
	Status       OrderStatus `json:"status" firestore:"Status"`
	PaymentURL   string      `json:"payment_url" firestore:"PaymentURL"`
	QRCodeText   string      `json:"qrcode_text" firestore:"QRCodeText"`
	QRCodeBase64 string      `json:"qrcode_base64" firestore:"QRCodeBase64"`
	CreatedAt    time.Time   `json:"created_at" firestore:"CreatedAt"`
	UpdatedAt    time.Time   `json:"updated_at" firestore:"UpdatedAt"`
}

// Buyer holds the buyer data required by PicPay.
type Buyer struct {
	FirstName string
	LastName  string
	Document  string // CPF — format: "000.000.000-00"
	Email     string
	Phone     string
}

// CreateOrderInput is the input to create a new PicPay payment order.
type CreateOrderInput struct {
	ReferenceID string
	AmountCents int64
	CallbackURL string // URL where PicPay will POST the webhook
	ReturnURL   string // URL the buyer is redirected to after payment
	Buyer       Buyer
}

// OrderResult is what PicPay returns after creating a payment.
type OrderResult struct {
	ReferenceID  string
	PaymentURL   string
	QRCodeText   string
	QRCodeBase64 string
	ExpiresAt    time.Time
}

// Receipt is returned for direct/synchronous credits (mock usage).
type Receipt struct {
	Provider  string
	Reference string
	Amount    int64
}

// Provider is the interface that payment infrastructure must implement.
type Provider interface {
	// CreateOrder initiates an async payment order on PicPay.
	CreateOrder(ctx context.Context, input CreateOrderInput) (OrderResult, error)
	// GetOrderStatus queries PicPay for the current status of an order.
	GetOrderStatus(ctx context.Context, referenceID string) (OrderStatus, error)
	// Credit applies a direct synchronous credit (used by mock/tests).
	Credit(ctx context.Context, userID string, amount int64) (Receipt, error)
}

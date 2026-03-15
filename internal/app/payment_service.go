package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"buskatotal-backend/internal/domain/payment"
	"buskatotal-backend/internal/domain/user"
)

type PaymentService struct {
	provider  payment.Provider
	orderRepo payment.OrderRepository
	userRepo  user.Repository
	// baseURL is used to build the callback URL sent to PicPay.
	baseURL string
}

func NewPaymentService(
	provider payment.Provider,
	orderRepo payment.OrderRepository,
	userRepo user.Repository,
	baseURL string,
) *PaymentService {
	return &PaymentService{
		provider:  provider,
		orderRepo: orderRepo,
		userRepo:  userRepo,
		baseURL:   baseURL,
	}
}

// CreateOrder starts an async PicPay payment order for the authenticated user.
// The user can only create orders for themselves (enforced at the handler layer).
func (s *PaymentService) CreateOrder(
	ctx context.Context,
	userID string,
	amountCents int64,
	buyer payment.Buyer,
	returnURL string,
) (payment.Order, error) {
	if userID == "" {
		return payment.Order{}, errors.New("user id is required")
	}
	if amountCents <= 0 {
		return payment.Order{}, errors.New("amount must be greater than zero")
	}
	if buyer.Document == "" {
		return payment.Order{}, errors.New("buyer document (CPF) is required")
	}
	if buyer.Email == "" {
		return payment.Order{}, errors.New("buyer email is required")
	}

	// Verify the user exists before creating the order.
	if _, err := s.userRepo.GetByID(ctx, userID); err != nil {
		return payment.Order{}, errors.New("user not found")
	}

	referenceID := fmt.Sprintf("buskatotal-%s-%d", userID[:8], time.Now().UnixNano())
	callbackURL := fmt.Sprintf("%s/payments/webhook", s.baseURL)

	result, err := s.provider.CreateOrder(ctx, payment.CreateOrderInput{
		ReferenceID: referenceID,
		AmountCents: amountCents,
		CallbackURL: callbackURL,
		ReturnURL:   returnURL,
		Buyer:       buyer,
	})
	if err != nil {
		return payment.Order{}, fmt.Errorf("provider: %w", err)
	}

	order := payment.Order{
		ID:           uuid.NewString(),
		UserID:       userID,
		ReferenceID:  result.ReferenceID,
		AmountCents:  amountCents,
		Status:       payment.StatusPending,
		PaymentURL:   result.PaymentURL,
		QRCodeText:   result.QRCodeText,
		QRCodeBase64: result.QRCodeBase64,
	}

	created, err := s.orderRepo.Create(ctx, order)
	if err != nil {
		return payment.Order{}, fmt.Errorf("save order: %w", err)
	}

	return created, nil
}

// ProcessWebhook is called when PicPay POSTs to our callback URL.
// It re-queries PicPay to confirm the status (never trusts the webhook payload alone).
// If the order is paid, it credits the user's balance.
func (s *PaymentService) ProcessWebhook(ctx context.Context, referenceID string) error {
	if referenceID == "" {
		return errors.New("referenceId is required")
	}

	order, err := s.orderRepo.GetByReferenceID(ctx, referenceID)
	if err != nil {
		return fmt.Errorf("order not found: %w", err)
	}

	// Idempotency: already processed.
	if order.Status == payment.StatusPaid {
		return nil
	}

	// Re-verify with PicPay — never trust the webhook payload alone.
	status, err := s.provider.GetOrderStatus(ctx, referenceID)
	if err != nil {
		return fmt.Errorf("verify status: %w", err)
	}

	order.Status = status
	if _, err := s.orderRepo.Update(ctx, order); err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	if status == payment.StatusPaid {
		userEntity, err := s.userRepo.GetByID(ctx, order.UserID)
		if err != nil {
			return fmt.Errorf("get user: %w", err)
		}
		userEntity.Balance += order.AmountCents
		if _, err := s.userRepo.Update(ctx, userEntity); err != nil {
			return fmt.Errorf("credit balance: %w", err)
		}
	}

	return nil
}

// ListOrders returns all orders for a user.
func (s *PaymentService) ListOrders(ctx context.Context, userID string) ([]payment.Order, error) {
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	return s.orderRepo.GetByUserID(ctx, userID)
}

// Credit keeps backward compatibility for direct/mock credits.
func (s *PaymentService) Credit(ctx context.Context, userID string, amount int64) (payment.Receipt, error) {
	if userID == "" {
		return payment.Receipt{}, errors.New("user id is required")
	}
	if amount <= 0 {
		return payment.Receipt{}, errors.New("amount must be greater than zero")
	}

	receipt, err := s.provider.Credit(ctx, userID, amount)
	if err != nil {
		return payment.Receipt{}, err
	}

	userEntity, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return payment.Receipt{}, err
	}
	userEntity.Balance += amount
	if _, err := s.userRepo.Update(ctx, userEntity); err != nil {
		return payment.Receipt{}, err
	}

	return receipt, nil
}

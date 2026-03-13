package app

import (
    "context"
    "errors"

    "buskatotal-backend/internal/domain/payment"
    "buskatotal-backend/internal/domain/user"
)

type PaymentService struct {
    provider payment.Provider
    userRepo user.Repository
}

func NewPaymentService(provider payment.Provider, userRepo user.Repository) *PaymentService {
    return &PaymentService{provider: provider, userRepo: userRepo}
}

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
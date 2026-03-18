package user

import (
    "context"
    "errors"
)

type Repository interface {
    Create(ctx context.Context, user User) (User, error)
    GetByID(ctx context.Context, id string) (User, error)
    GetByEmail(ctx context.Context, email string) (User, error)
    List(ctx context.Context) ([]User, error)
    Update(ctx context.Context, user User) (User, error)
    Delete(ctx context.Context, id string) error
    // DebitBalance atomically checks and subtracts amount from the user's balance.
    // Returns ErrInsufficientBalance if balance < amount.
    DebitBalance(ctx context.Context, id string, amount int64) error
    // CreditBalance atomically adds amount to the user's balance.
    CreditBalance(ctx context.Context, id string, amount int64) error
}

var ErrInsufficientBalance = errors.New("insufficient balance")
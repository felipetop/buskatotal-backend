package verification

import (
	"context"
	"errors"
	"time"
)

var (
	ErrTokenNotFound = errors.New("verification token not found")
	ErrTokenExpired  = errors.New("verification token expired")
	ErrTokenUsed     = errors.New("verification token already used")
)

type Token struct {
	ID        string    `json:"id" firestore:"id"`
	UserID    string    `json:"userId" firestore:"userId"`
	Token     string    `json:"token" firestore:"token"`
	Used      bool      `json:"used" firestore:"used"`
	ExpiresAt time.Time `json:"expiresAt" firestore:"expiresAt"`
	CreatedAt time.Time `json:"createdAt" firestore:"createdAt"`
}

func (t Token) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

type Repository interface {
	Create(ctx context.Context, token Token) (Token, error)
	GetByToken(ctx context.Context, token string) (Token, error)
	MarkUsed(ctx context.Context, id string) error
	DeleteByUserID(ctx context.Context, userID string) error
}

package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"buskatotal-backend/internal/domain/verification"
)

type VerificationRepository struct {
	mu    sync.RWMutex
	items map[string]verification.Token
}

func NewVerificationRepository() *VerificationRepository {
	return &VerificationRepository{items: make(map[string]verification.Token)}
}

func (r *VerificationRepository) Create(ctx context.Context, token verification.Token) (verification.Token, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	token.ID = uuid.NewString()
	token.CreatedAt = time.Now()
	r.items[token.ID] = token
	return token, nil
}

func (r *VerificationRepository) GetByToken(ctx context.Context, tokenStr string) (verification.Token, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, item := range r.items {
		if item.Token == tokenStr {
			return item, nil
		}
	}
	return verification.Token{}, verification.ErrTokenNotFound
}

func (r *VerificationRepository) MarkUsed(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.items[id]
	if !ok {
		return verification.ErrTokenNotFound
	}
	item.Used = true
	r.items[id] = item
	return nil
}

func (r *VerificationRepository) DeleteByUserID(ctx context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, item := range r.items {
		if item.UserID == userID {
			delete(r.items, id)
		}
	}
	return nil
}

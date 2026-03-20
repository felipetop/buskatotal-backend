package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"buskatotal-backend/internal/domain/lgpd"
)

type DeletionRepository struct {
	mu    sync.RWMutex
	items map[string]lgpd.DeletionRequest
}

func NewDeletionRepository() *DeletionRepository {
	return &DeletionRepository{items: make(map[string]lgpd.DeletionRequest)}
}

func (r *DeletionRepository) Create(ctx context.Context, req lgpd.DeletionRequest) (lgpd.DeletionRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	req.ID = uuid.NewString()
	req.CreatedAt = time.Now()
	r.items[req.ID] = req
	return req, nil
}

func (r *DeletionRepository) GetByID(ctx context.Context, id string) (lgpd.DeletionRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, ok := r.items[id]
	if !ok {
		return lgpd.DeletionRequest{}, errors.New("deletion request not found")
	}
	return item, nil
}

func (r *DeletionRepository) GetByUserID(ctx context.Context, userID string) ([]lgpd.DeletionRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []lgpd.DeletionRequest
	for _, item := range r.items {
		if item.UserID == userID {
			result = append(result, item)
		}
	}
	return result, nil
}

func (r *DeletionRepository) List(ctx context.Context) ([]lgpd.DeletionRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]lgpd.DeletionRequest, 0, len(r.items))
	for _, item := range r.items {
		result = append(result, item)
	}
	return result, nil
}

func (r *DeletionRepository) Update(ctx context.Context, req lgpd.DeletionRequest) (lgpd.DeletionRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[req.ID]; !ok {
		return lgpd.DeletionRequest{}, errors.New("deletion request not found")
	}
	r.items[req.ID] = req
	return req, nil
}

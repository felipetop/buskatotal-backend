package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"buskatotal-backend/internal/domain/inspection"
)

type InspectionRepository struct {
	mu    sync.RWMutex
	items map[string]inspection.Inspection
}

func NewInspectionRepository() *InspectionRepository {
	return &InspectionRepository{items: make(map[string]inspection.Inspection)}
}

func (r *InspectionRepository) Create(ctx context.Context, insp inspection.Inspection) (inspection.Inspection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	insp.ID = uuid.NewString()
	insp.CreatedAt = now
	insp.UpdatedAt = now
	r.items[insp.ID] = insp
	return insp, nil
}

func (r *InspectionRepository) GetByID(ctx context.Context, id string) (inspection.Inspection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	insp, ok := r.items[id]
	if !ok {
		return inspection.Inspection{}, errors.New("inspection not found")
	}
	return insp, nil
}

func (r *InspectionRepository) GetByProtocol(ctx context.Context, protocol string) (inspection.Inspection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, insp := range r.items {
		if insp.Protocol == protocol {
			return insp, nil
		}
	}
	return inspection.Inspection{}, errors.New("inspection not found")
}

func (r *InspectionRepository) GetByUserID(ctx context.Context, userID string) ([]inspection.Inspection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []inspection.Inspection
	for _, insp := range r.items {
		if insp.UserID == userID {
			result = append(result, insp)
		}
	}
	return result, nil
}

func (r *InspectionRepository) Update(ctx context.Context, insp inspection.Inspection) (inspection.Inspection, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[insp.ID]; !ok {
		return inspection.Inspection{}, errors.New("inspection not found")
	}
	insp.UpdatedAt = time.Now()
	r.items[insp.ID] = insp
	return insp, nil
}

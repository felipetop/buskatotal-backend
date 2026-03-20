package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"buskatotal-backend/internal/domain/lgpd"
)

type LogRepository struct {
	mu    sync.RWMutex
	items []lgpd.DataProcessingLog
}

func NewLogRepository() *LogRepository {
	return &LogRepository{}
}

func (r *LogRepository) Create(ctx context.Context, log lgpd.DataProcessingLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	log.ID = uuid.NewString()
	log.CreatedAt = time.Now()
	r.items = append(r.items, log)
	return nil
}

func (r *LogRepository) GetByUserID(ctx context.Context, userID string) ([]lgpd.DataProcessingLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []lgpd.DataProcessingLog
	for _, item := range r.items {
		if item.UserID == userID {
			result = append(result, item)
		}
	}
	return result, nil
}

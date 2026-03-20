package firestore

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	"buskatotal-backend/internal/domain/lgpd"
)

type LogRepository struct {
	client *firestore.Client
}

func NewLogRepository(client *firestore.Client) *LogRepository {
	return &LogRepository{client: client}
}

func (r *LogRepository) collection() *firestore.CollectionRef {
	return r.client.Collection("data_processing_log")
}

func (r *LogRepository) Create(ctx context.Context, log lgpd.DataProcessingLog) error {
	log.CreatedAt = time.Now()
	doc := r.collection().NewDoc()
	log.ID = doc.ID
	_, err := doc.Set(ctx, log)
	return err
}

func (r *LogRepository) GetByUserID(ctx context.Context, userID string) ([]lgpd.DataProcessingLog, error) {
	query := r.collection().Where("userId", "==", userID)
	snaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	result := make([]lgpd.DataProcessingLog, 0, len(snaps))
	for _, snap := range snaps {
		var entry lgpd.DataProcessingLog
		if err := snap.DataTo(&entry); err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	return result, nil
}

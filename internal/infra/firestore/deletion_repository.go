package firestore

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"

	"buskatotal-backend/internal/domain/lgpd"
)

type DeletionRepository struct {
	client *firestore.Client
}

func NewDeletionRepository(client *firestore.Client) *DeletionRepository {
	return &DeletionRepository{client: client}
}

func (r *DeletionRepository) collection() *firestore.CollectionRef {
	return r.client.Collection("deletion_requests")
}

func (r *DeletionRepository) Create(ctx context.Context, req lgpd.DeletionRequest) (lgpd.DeletionRequest, error) {
	req.CreatedAt = time.Now()
	doc := r.collection().NewDoc()
	req.ID = doc.ID
	if _, err := doc.Set(ctx, req); err != nil {
		return lgpd.DeletionRequest{}, err
	}
	return req, nil
}

func (r *DeletionRepository) GetByID(ctx context.Context, id string) (lgpd.DeletionRequest, error) {
	snap, err := r.collection().Doc(id).Get(ctx)
	if err != nil {
		return lgpd.DeletionRequest{}, errors.New("deletion request not found")
	}
	var req lgpd.DeletionRequest
	if err := snap.DataTo(&req); err != nil {
		return lgpd.DeletionRequest{}, err
	}
	return req, nil
}

func (r *DeletionRepository) GetByUserID(ctx context.Context, userID string) ([]lgpd.DeletionRequest, error) {
	query := r.collection().Where("userId", "==", userID)
	snaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	result := make([]lgpd.DeletionRequest, 0, len(snaps))
	for _, snap := range snaps {
		var req lgpd.DeletionRequest
		if err := snap.DataTo(&req); err != nil {
			return nil, err
		}
		result = append(result, req)
	}
	return result, nil
}

func (r *DeletionRepository) List(ctx context.Context) ([]lgpd.DeletionRequest, error) {
	snaps, err := r.collection().Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	result := make([]lgpd.DeletionRequest, 0, len(snaps))
	for _, snap := range snaps {
		var req lgpd.DeletionRequest
		if err := snap.DataTo(&req); err != nil {
			return nil, err
		}
		result = append(result, req)
	}
	return result, nil
}

func (r *DeletionRepository) Update(ctx context.Context, req lgpd.DeletionRequest) (lgpd.DeletionRequest, error) {
	if _, err := r.collection().Doc(req.ID).Set(ctx, req); err != nil {
		return lgpd.DeletionRequest{}, err
	}
	return req, nil
}

package firestore

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"

	"buskatotal-backend/internal/domain/inspection"
)

type InspectionRepository struct {
	client *firestore.Client
}

func NewInspectionRepository(client *firestore.Client) *InspectionRepository {
	return &InspectionRepository{client: client}
}

func (r *InspectionRepository) Create(ctx context.Context, insp inspection.Inspection) (inspection.Inspection, error) {
	now := time.Now()
	insp.ID = uuid.NewString()
	insp.CreatedAt = now
	insp.UpdatedAt = now

	if _, err := r.client.Collection("inspections").Doc(insp.ID).Set(ctx, insp); err != nil {
		return inspection.Inspection{}, err
	}
	return insp, nil
}

func (r *InspectionRepository) GetByID(ctx context.Context, id string) (inspection.Inspection, error) {
	snap, err := r.client.Collection("inspections").Doc(id).Get(ctx)
	if err != nil {
		return inspection.Inspection{}, err
	}
	var insp inspection.Inspection
	if err := snap.DataTo(&insp); err != nil {
		return inspection.Inspection{}, err
	}
	return insp, nil
}

func (r *InspectionRepository) GetByProtocol(ctx context.Context, protocol string) (inspection.Inspection, error) {
	snaps, err := r.client.Collection("inspections").Where("protocol", "==", protocol).Limit(1).Documents(ctx).GetAll()
	if err != nil {
		return inspection.Inspection{}, err
	}
	if len(snaps) == 0 {
		return inspection.Inspection{}, errors.New("inspection not found")
	}
	var insp inspection.Inspection
	if err := snaps[0].DataTo(&insp); err != nil {
		return inspection.Inspection{}, err
	}
	return insp, nil
}

func (r *InspectionRepository) GetByUserID(ctx context.Context, userID string) ([]inspection.Inspection, error) {
	snaps, err := r.client.Collection("inspections").
		Where("userID", "==", userID).
		OrderBy("createdAt", firestore.Desc).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	result := make([]inspection.Inspection, 0, len(snaps))
	for _, snap := range snaps {
		var insp inspection.Inspection
		if err := snap.DataTo(&insp); err != nil {
			return nil, err
		}
		result = append(result, insp)
	}
	return result, nil
}

func (r *InspectionRepository) Update(ctx context.Context, insp inspection.Inspection) (inspection.Inspection, error) {
	insp.UpdatedAt = time.Now()
	if _, err := r.client.Collection("inspections").Doc(insp.ID).Set(ctx, insp); err != nil {
		return inspection.Inspection{}, err
	}
	return insp, nil
}

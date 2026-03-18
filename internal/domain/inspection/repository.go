package inspection

import "context"

type Repository interface {
	Create(ctx context.Context, insp Inspection) (Inspection, error)
	GetByID(ctx context.Context, id string) (Inspection, error)
	GetByProtocol(ctx context.Context, protocol string) (Inspection, error)
	GetByUserID(ctx context.Context, userID string) ([]Inspection, error)
	Update(ctx context.Context, insp Inspection) (Inspection, error)
}

package lgpd

import (
	"context"
	"time"
)

const (
	DeletionStatusPending    = "pending"
	DeletionStatusProcessing = "processing"
	DeletionStatusCompleted  = "completed"
	DeletionStatusRejected   = "rejected"
)

type DeletionRequest struct {
	ID          string     `json:"id" firestore:"id"`
	UserID      string     `json:"user_id" firestore:"userId"`
	UserEmail   string     `json:"user_email" firestore:"userEmail"`
	UserName    string     `json:"user_name" firestore:"userName"`
	Reason      string     `json:"reason" firestore:"reason"`
	Status      string     `json:"status" firestore:"status"`
	CreatedAt   time.Time  `json:"created_at" firestore:"createdAt"`
	ProcessedAt *time.Time `json:"processed_at,omitempty" firestore:"processedAt"`
	ProcessedBy string     `json:"processed_by,omitempty" firestore:"processedBy"`
}

type DeletionRepository interface {
	Create(ctx context.Context, req DeletionRequest) (DeletionRequest, error)
	GetByID(ctx context.Context, id string) (DeletionRequest, error)
	GetByUserID(ctx context.Context, userID string) ([]DeletionRequest, error)
	List(ctx context.Context) ([]DeletionRequest, error)
	Update(ctx context.Context, req DeletionRequest) (DeletionRequest, error)
}

type DataProcessingLog struct {
	ID        string                 `json:"id" firestore:"id"`
	UserID    string                 `json:"user_id" firestore:"userId"`
	Action    string                 `json:"action" firestore:"action"`
	Details   map[string]interface{} `json:"details,omitempty" firestore:"details"`
	IPAddress string                 `json:"ip_address,omitempty" firestore:"ipAddress"`
	CreatedAt time.Time              `json:"created_at" firestore:"createdAt"`
}

type LogRepository interface {
	Create(ctx context.Context, log DataProcessingLog) error
	GetByUserID(ctx context.Context, userID string) ([]DataProcessingLog, error)
}

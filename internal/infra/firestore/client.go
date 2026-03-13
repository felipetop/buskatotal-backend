package firestore

import (
    "context"

    "cloud.google.com/go/firestore"
    "google.golang.org/api/option"

    "buskatotal-backend/configs"
)

func NewClient(projectID string) (*firestore.Client, error) {
    cfg := configs.Load()
    ctx := context.Background()

    if cfg.FirebaseCredsPath != "" {
        return firestore.NewClient(ctx, projectID, option.WithCredentialsFile(cfg.FirebaseCredsPath))
    }

    return firestore.NewClient(ctx, projectID)
}
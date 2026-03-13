package task

import "context"

type Repository interface {
    Create(ctx context.Context, task Task) (Task, error)
    GetByID(ctx context.Context, id string) (Task, error)
    ListByUser(ctx context.Context, userID string) ([]Task, error)
    Update(ctx context.Context, task Task) (Task, error)
    Delete(ctx context.Context, id string) error
}
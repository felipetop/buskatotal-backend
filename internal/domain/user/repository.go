package user

import "context"

type Repository interface {
    Create(ctx context.Context, user User) (User, error)
    GetByID(ctx context.Context, id string) (User, error)
    List(ctx context.Context) ([]User, error)
    Update(ctx context.Context, user User) (User, error)
    Delete(ctx context.Context, id string) error
}
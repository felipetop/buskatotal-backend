package firestore

import (
    "context"
    "time"

    "cloud.google.com/go/firestore"

    "buskatotal-backend/internal/domain/user"
)

type UserRepository struct {
    client *firestore.Client
}

func NewUserRepository(client *firestore.Client) *UserRepository {
    return &UserRepository{client: client}
}

func (r *UserRepository) Create(ctx context.Context, entity user.User) (user.User, error) {
    now := time.Now()
    entity.CreatedAt = now
    entity.UpdatedAt = now

    doc := r.client.Collection("users").NewDoc()
    entity.ID = doc.ID
    if _, err := doc.Set(ctx, entity); err != nil {
        return user.User{}, err
    }
    return entity, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (user.User, error) {
    snap, err := r.client.Collection("users").Doc(id).Get(ctx)
    if err != nil {
        return user.User{}, err
    }
    var entity user.User
    if err := snap.DataTo(&entity); err != nil {
        return user.User{}, err
    }
    return entity, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (user.User, error) {
    query := r.client.Collection("users").Where("email", "==", email).Limit(1)
    snaps, err := query.Documents(ctx).GetAll()
    if err != nil {
        return user.User{}, err
    }
    if len(snaps) == 0 {
        return user.User{}, firestore.ErrNotFound
    }
    var entity user.User
    if err := snaps[0].DataTo(&entity); err != nil {
        return user.User{}, err
    }
    return entity, nil
}

func (r *UserRepository) List(ctx context.Context) ([]user.User, error) {
    iter := r.client.Collection("users").Documents(ctx)
    snaps, err := iter.GetAll()
    if err != nil {
        return nil, err
    }
    users := make([]user.User, 0, len(snaps))
    for _, snap := range snaps {
        var entity user.User
        if err := snap.DataTo(&entity); err != nil {
            return nil, err
        }
        users = append(users, entity)
    }
    return users, nil
}

func (r *UserRepository) Update(ctx context.Context, entity user.User) (user.User, error) {
    entity.UpdatedAt = time.Now()
    if _, err := r.client.Collection("users").Doc(entity.ID).Set(ctx, entity); err != nil {
        return user.User{}, err
    }
    return entity, nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
    _, err := r.client.Collection("users").Doc(id).Delete(ctx)
    return err
}
package memory

import (
    "context"
    "errors"
    "sync"
    "time"

    "github.com/google/uuid"

    "buskatotal-backend/internal/domain/user"
)

type UserRepository struct {
    mu    sync.RWMutex
    items map[string]user.User
}

func NewUserRepository() *UserRepository {
    return &UserRepository{items: make(map[string]user.User)}
}

func (r *UserRepository) Create(ctx context.Context, entity user.User) (user.User, error) {
    r.mu.Lock()
    defer r.mu.Unlock()

    now := time.Now()
    entity.ID = uuid.NewString()
    entity.CreatedAt = now
    entity.UpdatedAt = now
    r.items[entity.ID] = entity
    return entity, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (user.User, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    entity, ok := r.items[id]
    if !ok {
        return user.User{}, errors.New("user not found")
    }
    return entity, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (user.User, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    for _, entity := range r.items {
        if entity.Email == email {
            return entity, nil
        }
    }
    return user.User{}, errors.New("user not found")
}

func (r *UserRepository) List(ctx context.Context) ([]user.User, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    users := make([]user.User, 0, len(r.items))
    for _, entity := range r.items {
        users = append(users, entity)
    }
    return users, nil
}

func (r *UserRepository) Update(ctx context.Context, entity user.User) (user.User, error) {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, ok := r.items[entity.ID]; !ok {
        return user.User{}, errors.New("user not found")
    }
    entity.UpdatedAt = time.Now()
    r.items[entity.ID] = entity
    return entity, nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, ok := r.items[id]; !ok {
        return errors.New("user not found")
    }
    delete(r.items, id)
    return nil
}

func (r *UserRepository) DebitBalance(ctx context.Context, id string, amount int64) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    entity, ok := r.items[id]
    if !ok {
        return errors.New("user not found")
    }
    if entity.Balance < amount {
        return user.ErrInsufficientBalance
    }
    entity.Balance -= amount
    entity.UpdatedAt = time.Now()
    r.items[id] = entity
    return nil
}

func (r *UserRepository) CreditBalance(ctx context.Context, id string, amount int64) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    entity, ok := r.items[id]
    if !ok {
        return errors.New("user not found")
    }
    entity.Balance += amount
    entity.UpdatedAt = time.Now()
    r.items[id] = entity
    return nil
}
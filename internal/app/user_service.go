package app

import (
    "context"
    "errors"

    "buskatotal-backend/internal/domain/user"
)

type UserService struct {
    repo user.Repository
}

func NewUserService(repo user.Repository) *UserService {
    return &UserService{repo: repo}
}

func (s *UserService) Create(ctx context.Context, input user.User) (user.User, error) {
    if input.Name == "" || input.Email == "" {
        return user.User{}, errors.New("name and email are required")
    }
    if input.Balance < 0 {
        return user.User{}, errors.New("balance cannot be negative")
    }
    return s.repo.Create(ctx, input)
}

func (s *UserService) GetByID(ctx context.Context, id string) (user.User, error) {
    return s.repo.GetByID(ctx, id)
}

func (s *UserService) List(ctx context.Context) ([]user.User, error) {
    return s.repo.List(ctx)
}

func (s *UserService) Update(ctx context.Context, input user.User) (user.User, error) {
    if input.ID == "" {
        return user.User{}, errors.New("id is required")
    }
    if input.Name == "" || input.Email == "" {
        return user.User{}, errors.New("name and email are required")
    }
    if input.Balance < 0 {
        return user.User{}, errors.New("balance cannot be negative")
    }
    return s.repo.Update(ctx, input)
}

func (s *UserService) Delete(ctx context.Context, id string) error {
    return s.repo.Delete(ctx, id)
}
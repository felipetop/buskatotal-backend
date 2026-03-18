package app

import (
	"context"
	"strings"

	"buskatotal-backend/internal/domain/user"
)

type AdminService struct {
	userRepo user.Repository
}

func NewAdminService(userRepo user.Repository) *AdminService {
	return &AdminService{userRepo: userRepo}
}

func (s *AdminService) ListUsers(ctx context.Context) ([]user.User, error) {
	return s.userRepo.List(ctx)
}

func (s *AdminService) SearchUsers(ctx context.Context, query string) ([]user.User, error) {
	all, err := s.userRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	q := strings.ToLower(strings.TrimSpace(query))
	var result []user.User
	for _, u := range all {
		if strings.Contains(strings.ToLower(u.Name), q) ||
			strings.Contains(strings.ToLower(u.Email), q) ||
			strings.Contains(u.ID, q) {
			result = append(result, u)
		}
	}
	return result, nil
}

func (s *AdminService) GetUserByID(ctx context.Context, id string) (user.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

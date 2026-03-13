package app

import (
    "context"
    "errors"

    "buskatotal-backend/internal/domain/task"
)

type TaskService struct {
    repo task.Repository
}

func NewTaskService(repo task.Repository) *TaskService {
    return &TaskService{repo: repo}
}

func (s *TaskService) Create(ctx context.Context, input task.Task) (task.Task, error) {
    if input.Title == "" || input.UserID == "" {
        return task.Task{}, errors.New("title and userId are required")
    }
    return s.repo.Create(ctx, input)
}

func (s *TaskService) GetByID(ctx context.Context, id string) (task.Task, error) {
    return s.repo.GetByID(ctx, id)
}

func (s *TaskService) ListByUser(ctx context.Context, userID string) ([]task.Task, error) {
    if userID == "" {
        return nil, errors.New("userId is required")
    }
    return s.repo.ListByUser(ctx, userID)
}

func (s *TaskService) Update(ctx context.Context, input task.Task) (task.Task, error) {
    if input.ID == "" {
        return task.Task{}, errors.New("id is required")
    }
    if input.Title == "" || input.UserID == "" {
        return task.Task{}, errors.New("title and userId are required")
    }
    return s.repo.Update(ctx, input)
}

func (s *TaskService) Delete(ctx context.Context, id string) error {
    return s.repo.Delete(ctx, id)
}
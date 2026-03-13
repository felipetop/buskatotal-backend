package memory

import (
    "context"
    "errors"
    "sync"
    "time"

    "github.com/google/uuid"

    "buskatotal-backend/internal/domain/task"
)

type TaskRepository struct {
    mu    sync.RWMutex
    items map[string]task.Task
}

func NewTaskRepository() *TaskRepository {
    return &TaskRepository{items: make(map[string]task.Task)}
}

func (r *TaskRepository) Create(ctx context.Context, entity task.Task) (task.Task, error) {
    r.mu.Lock()
    defer r.mu.Unlock()

    now := time.Now()
    entity.ID = uuid.NewString()
    entity.CreatedAt = now
    entity.UpdatedAt = now
    r.items[entity.ID] = entity
    return entity, nil
}

func (r *TaskRepository) GetByID(ctx context.Context, id string) (task.Task, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    entity, ok := r.items[id]
    if !ok {
        return task.Task{}, errors.New("task not found")
    }
    return entity, nil
}

func (r *TaskRepository) ListByUser(ctx context.Context, userID string) ([]task.Task, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    tasks := make([]task.Task, 0)
    for _, entity := range r.items {
        if entity.UserID == userID {
            tasks = append(tasks, entity)
        }
    }
    return tasks, nil
}

func (r *TaskRepository) Update(ctx context.Context, entity task.Task) (task.Task, error) {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, ok := r.items[entity.ID]; !ok {
        return task.Task{}, errors.New("task not found")
    }
    entity.UpdatedAt = time.Now()
    r.items[entity.ID] = entity
    return entity, nil
}

func (r *TaskRepository) Delete(ctx context.Context, id string) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, ok := r.items[id]; !ok {
        return errors.New("task not found")
    }
    delete(r.items, id)
    return nil
}
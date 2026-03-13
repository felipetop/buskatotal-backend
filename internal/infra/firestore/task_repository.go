package firestore

import (
    "context"
    "time"

    "cloud.google.com/go/firestore"

    "buskatotal-backend/internal/domain/task"
)

type TaskRepository struct {
    client *firestore.Client
}

func NewTaskRepository(client *firestore.Client) *TaskRepository {
    return &TaskRepository{client: client}
}

func (r *TaskRepository) Create(ctx context.Context, entity task.Task) (task.Task, error) {
    now := time.Now()
    entity.CreatedAt = now
    entity.UpdatedAt = now

    doc := r.client.Collection("tasks").NewDoc()
    entity.ID = doc.ID
    if _, err := doc.Set(ctx, entity); err != nil {
        return task.Task{}, err
    }
    return entity, nil
}

func (r *TaskRepository) GetByID(ctx context.Context, id string) (task.Task, error) {
    snap, err := r.client.Collection("tasks").Doc(id).Get(ctx)
    if err != nil {
        return task.Task{}, err
    }
    var entity task.Task
    if err := snap.DataTo(&entity); err != nil {
        return task.Task{}, err
    }
    return entity, nil
}

func (r *TaskRepository) ListByUser(ctx context.Context, userID string) ([]task.Task, error) {
    iter := r.client.Collection("tasks").Where("userId", "==", userID).Documents(ctx)
    snaps, err := iter.GetAll()
    if err != nil {
        return nil, err
    }
    tasks := make([]task.Task, 0, len(snaps))
    for _, snap := range snaps {
        var entity task.Task
        if err := snap.DataTo(&entity); err != nil {
            return nil, err
        }
        tasks = append(tasks, entity)
    }
    return tasks, nil
}

func (r *TaskRepository) Update(ctx context.Context, entity task.Task) (task.Task, error) {
    entity.UpdatedAt = time.Now()
    if _, err := r.client.Collection("tasks").Doc(entity.ID).Set(ctx, entity); err != nil {
        return task.Task{}, err
    }
    return entity, nil
}

func (r *TaskRepository) Delete(ctx context.Context, id string) error {
    _, err := r.client.Collection("tasks").Doc(id).Delete(ctx)
    return err
}
package http

import (
    "context"
    "net/http"

    "github.com/gin-gonic/gin"

    "buskatotal-backend/internal/domain/task"
)

type TaskService interface {
    Create(ctx context.Context, input task.Task) (task.Task, error)
    GetByID(ctx context.Context, id string) (task.Task, error)
    ListByUser(ctx context.Context, userID string) ([]task.Task, error)
    Update(ctx context.Context, input task.Task) (task.Task, error)
    Delete(ctx context.Context, id string) error
}

type TaskHandler struct {
    service TaskService
}

type taskInput struct {
    UserID      string `json:"userId"`
    Title       string `json:"title"`
    Description string `json:"description"`
    Done        bool   `json:"done"`
}

func NewTaskHandler(service TaskService) *TaskHandler {
    return &TaskHandler{service: service}
}

func (h *TaskHandler) Create(c *gin.Context) {
    var input taskInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    created, err := h.service.Create(c.Request.Context(), task.Task{
        UserID:      input.UserID,
        Title:       input.Title,
        Description: input.Description,
        Done:        input.Done,
    })
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusCreated, created)
}

func (h *TaskHandler) GetByID(c *gin.Context) {
    id := c.Param("id")
    taskItem, err := h.service.GetByID(c.Request.Context(), id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, taskItem)
}

func (h *TaskHandler) ListByUser(c *gin.Context) {
    userID := c.Query("userId")
    tasks, err := h.service.ListByUser(c.Request.Context(), userID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, tasks)
}

func (h *TaskHandler) Update(c *gin.Context) {
    id := c.Param("id")
    var input taskInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    updated, err := h.service.Update(c.Request.Context(), task.Task{
        ID:          id,
        UserID:      input.UserID,
        Title:       input.Title,
        Description: input.Description,
        Done:        input.Done,
    })
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, updated)
}

func (h *TaskHandler) Delete(c *gin.Context) {
    id := c.Param("id")
    if err := h.service.Delete(c.Request.Context(), id); err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }
    c.Status(http.StatusNoContent)
}
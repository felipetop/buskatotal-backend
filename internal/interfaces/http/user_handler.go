package http

import (
    "net/http"

    "github.com/gin-gonic/gin"

    "buskatotal-backend/internal/app"
    "buskatotal-backend/internal/domain/user"
)

type UserHandler struct {
    service *app.UserService
}

type userInput struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func NewUserHandler(service *app.UserService) *UserHandler {
    return &UserHandler{service: service}
}

func (h *UserHandler) Create(c *gin.Context) {
    var input userInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    created, err := h.service.Create(c.Request.Context(), user.User{
        Name:  input.Name,
        Email: input.Email,
    })
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusCreated, created)
}

func (h *UserHandler) List(c *gin.Context) {
    users, err := h.service.List(c.Request.Context())
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, users)
}

func (h *UserHandler) GetByID(c *gin.Context) {
    id := c.Param("id")
    userItem, err := h.service.GetByID(c.Request.Context(), id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, userItem)
}

func (h *UserHandler) Update(c *gin.Context) {
    id := c.Param("id")
    var input userInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    updated, err := h.service.Update(c.Request.Context(), user.User{
        ID:    id,
        Name:  input.Name,
        Email: input.Email,
    })
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, updated)
}

func (h *UserHandler) Delete(c *gin.Context) {
    id := c.Param("id")
    if err := h.service.Delete(c.Request.Context(), id); err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }
    c.Status(http.StatusNoContent)
}
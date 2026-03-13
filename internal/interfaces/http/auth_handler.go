package http

import (
    "context"
    "net/http"

    "github.com/gin-gonic/gin"

    "buskatotal-backend/internal/domain/user"
)

type AuthHandler struct {
    service AuthService
}

type AuthService interface {
    Register(ctx context.Context, name, email, password string) (user.User, string, error)
    Login(ctx context.Context, email, password string) (user.User, string, error)
}

type authInput struct {
    Name     string `json:"name"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

func NewAuthHandler(service AuthService) *AuthHandler {
    return &AuthHandler{service: service}
}

func (h *AuthHandler) Register(c *gin.Context) {
    var input authInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    userItem, token, err := h.service.Register(c.Request.Context(), input.Name, input.Email, input.Password)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{
        "user":  userItem,
        "token": token,
    })
}

func (h *AuthHandler) Login(c *gin.Context) {
    var input authInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    userItem, token, err := h.service.Login(c.Request.Context(), input.Email, input.Password)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "user":  userItem,
        "token": token,
    })
}
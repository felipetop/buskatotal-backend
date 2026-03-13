package http

import (
    "net/http"

    "github.com/gin-gonic/gin"

    "buskatotal-backend/internal/app"
)

type AuthHandler struct {
    service *app.AuthService
}

type authInput struct {
    Name     string `json:"name"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

func NewAuthHandler(service *app.AuthService) *AuthHandler {
    return &AuthHandler{service: service}
}

func (h *AuthHandler) Register(c *gin.Context) {
    var input authInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    result, err := h.service.Register(c.Request.Context(), input.Name, input.Email, input.Password)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{
        "user":  result.User,
        "token": result.Token,
    })
}

func (h *AuthHandler) Login(c *gin.Context) {
    var input authInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    result, err := h.service.Login(c.Request.Context(), input.Email, input.Password)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "user":  result.User,
        "token": result.Token,
    })
}
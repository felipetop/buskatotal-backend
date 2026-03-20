package http

import (
    "context"
    "errors"
    "net/http"

    "github.com/gin-gonic/gin"

    "buskatotal-backend/internal/domain/user"
    "buskatotal-backend/internal/domain/verification"
)

type AuthHandler struct {
    service              AuthService
    emailVerifyService   EmailVerificationService
}

type AuthService interface {
    Register(ctx context.Context, name, email, password string) (user.User, string, error)
    Login(ctx context.Context, email, password string) (user.User, string, error)
    ResendVerification(ctx context.Context, userID string) error
}

type EmailVerificationService interface {
    Verify(ctx context.Context, token string) error
    GenerateAndSend(ctx context.Context, userID, userEmail string) error
}

type authInput struct {
    Name     string `json:"name"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

func NewAuthHandler(service AuthService, emailVerifyService EmailVerificationService) *AuthHandler {
    return &AuthHandler{service: service, emailVerifyService: emailVerifyService}
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

    c.Header("Cache-Control", "no-store")
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

    c.Header("Cache-Control", "no-store")
    c.JSON(http.StatusOK, gin.H{
        "user":  userItem,
        "token": token,
    })
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
    if h.emailVerifyService == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "email verification not configured"})
        return
    }

    token := c.Query("token")
    if token == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
        return
    }

    err := h.emailVerifyService.Verify(c.Request.Context(), token)
    if err != nil {
        status := http.StatusInternalServerError
        if errors.Is(err, verification.ErrTokenNotFound) {
            status = http.StatusNotFound
        } else if errors.Is(err, verification.ErrTokenExpired) {
            status = http.StatusGone
        } else if errors.Is(err, verification.ErrTokenUsed) {
            status = http.StatusConflict
        }
        c.JSON(status, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "e-mail verificado com sucesso"})
}

func (h *AuthHandler) ResendVerification(c *gin.Context) {
    userID, ok := GetAuthUserID(c)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
        return
    }

    err := h.service.ResendVerification(c.Request.Context(), userID)
    if err != nil {
        status := http.StatusInternalServerError
        switch err.Error() {
        case "email verification not configured":
            status = http.StatusServiceUnavailable
        case "user not found":
            status = http.StatusNotFound
        case "email already verified":
            status = http.StatusConflict
        }
        c.JSON(status, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "e-mail de verificação reenviado"})
}
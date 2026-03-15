package http

import (
    "context"
    "errors"
    "net/http"
    "regexp"

    "github.com/gin-gonic/gin"
    "golang.org/x/crypto/bcrypt"

    "buskatotal-backend/internal/domain/user"
)

func validatePassword(password string) error {
    if len(password) < 10 {
        return errors.New("password must be at least 10 characters long")
    }
    if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
        return errors.New("password must contain at least one uppercase letter (A–Z)")
    }
    if !regexp.MustCompile(`[a-z]`).MatchString(password) {
        return errors.New("password must contain at least one lowercase letter (a–z)")
    }
    if !regexp.MustCompile(`[0-9]`).MatchString(password) {
        return errors.New("password must contain at least one number (0–9)")
    }
    if !regexp.MustCompile(`[@!#$%]`).MatchString(password) {
        return errors.New("password must contain at least one special character (@, !, #, $, %)")
    }
    return nil
}

type UserService interface {
    Create(ctx context.Context, input user.User) (user.User, error)
    GetByID(ctx context.Context, id string) (user.User, error)
    List(ctx context.Context) ([]user.User, error)
    Update(ctx context.Context, input user.User) (user.User, error)
    Delete(ctx context.Context, id string) error
}

type UserHandler struct {
    service UserService
}

type userInput struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Password string `json:"password"`
}

func NewUserHandler(service UserService) *UserHandler {
    return &UserHandler{service: service}
}

func (h *UserHandler) Create(c *gin.Context) {
    var input userInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if input.Password == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "password is required"})
        return
    }

    if err := validatePassword(input.Password); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "could not hash password"})
        return
    }

    created, err := h.service.Create(c.Request.Context(), user.User{
        Name:         input.Name,
        Email:        input.Email,
        PasswordHash: string(hash),
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

// GetBalance returns the current balance of the authenticated user.
// The user can only query their own balance.
func (h *UserHandler) GetBalance(c *gin.Context) {
    authUserID, ok := GetAuthUserID(c)
    if !ok || authUserID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user"})
        return
    }

    id := c.Param("id")
    if id != authUserID {
        c.JSON(http.StatusForbidden, gin.H{"error": "cannot query balance of another user"})
        return
    }

    userItem, err := h.service.GetByID(c.Request.Context(), id)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "user_id":       userItem.ID,
        "balance_cents": userItem.Balance,
        "balance_brl":   float64(userItem.Balance) / 100.0,
    })
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
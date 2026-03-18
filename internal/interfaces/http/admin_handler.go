package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"buskatotal-backend/internal/domain/user"
)

type AdminService interface {
	ListUsers(ctx context.Context) ([]user.User, error)
	SearchUsers(ctx context.Context, query string) ([]user.User, error)
	GetUserByID(ctx context.Context, id string) (user.User, error)
}

type AdminHandler struct {
	service AdminService
}

func NewAdminHandler(service AdminService) *AdminHandler {
	return &AdminHandler{service: service}
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	query := c.Query("q")

	var users []user.User
	var err error

	if query != "" {
		users, err = h.service.SearchUsers(c.Request.Context(), query)
	} else {
		users, err = h.service.ListUsers(c.Request.Context())
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if users == nil {
		users = []user.User{}
	}

	// Build response with formatted balance
	type userRow struct {
		ID        string  `json:"id"`
		Name      string  `json:"name"`
		Email     string  `json:"email"`
		Role      string  `json:"role"`
		Balance   int64   `json:"balance_cents"`
		BalanceBRL float64 `json:"balance_brl"`
		CreatedAt string  `json:"created_at"`
	}

	rows := make([]userRow, 0, len(users))
	for _, u := range users {
		rows = append(rows, userRow{
			ID:         u.ID,
			Name:       u.Name,
			Email:      u.Email,
			Role:       u.Role,
			Balance:    u.Balance,
			BalanceBRL: float64(u.Balance) / 100,
			CreatedAt:  u.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(rows),
		"users": rows,
	})
}

func (h *AdminHandler) GetUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user id is required"})
		return
	}

	u, err := h.service.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           u.ID,
		"name":         u.Name,
		"email":        u.Email,
		"role":         u.Role,
		"balance_cents": u.Balance,
		"balance_brl":  float64(u.Balance) / 100,
		"created_at":   u.CreatedAt.Format("2006-01-02 15:04:05"),
		"updated_at":   u.UpdatedAt.Format("2006-01-02 15:04:05"),
	})
}

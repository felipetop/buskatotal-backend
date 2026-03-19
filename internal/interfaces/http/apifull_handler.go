package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"buskatotal-backend/internal/infra/apifull"
)

type ApiFullService interface {
	QueryProduct(ctx context.Context, userID, productKey, value string) (*apifull.ProductResponse, error)
}

type ApiFullHandler struct {
	service ApiFullService
}

func NewApiFullHandler(service ApiFullService) *ApiFullHandler {
	return &ApiFullHandler{service: service}
}

// QueryProduct handles: POST /consultas/apifull/:produto
// Body: { "valor": "12345678900" }
func (h *ApiFullHandler) QueryProduct(c *gin.Context) {
	userID, ok := GetAuthUserID(c)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user"})
		return
	}

	productKey := c.Param("produto")
	if productKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "produto é obrigatório"})
		return
	}

	var input struct {
		Valor string `json:"valor"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Valor == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "campo 'valor' é obrigatório no body"})
		return
	}

	result, err := h.service.QueryProduct(c.Request.Context(), userID, productKey, input.Valor)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

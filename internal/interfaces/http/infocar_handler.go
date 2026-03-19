package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"buskatotal-backend/internal/infra/infocar"
)

type InfocarService interface {
	GetAgregadosB(ctx context.Context, userID, queryType, value string) (*infocar.AgregadosBResponse, error)
	QueryProduct(ctx context.Context, userID, productKey, queryType, value string) (*infocar.ProductResponse, error)
}

type InfocarHandler struct {
	service InfocarService
}

func NewInfocarHandler(service InfocarService) *InfocarHandler {
	return &InfocarHandler{service: service}
}

// GetAgregadosB keeps the existing endpoint working.
func (h *InfocarHandler) GetAgregadosB(c *gin.Context) {
	userID, ok := GetAuthUserID(c)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user"})
		return
	}

	queryType := c.Param("tipo")
	value := c.Param("valor")
	if queryType == "" || value == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tipo e valor são obrigatórios"})
		return
	}

	result, err := h.service.GetAgregadosB(c.Request.Context(), userID, queryType, value)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// QueryProduct is the generic handler for all Infocar products.
// The product key comes from the URL param :produto.
func (h *InfocarHandler) QueryProduct(c *gin.Context) {
	userID, ok := GetAuthUserID(c)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user"})
		return
	}

	productKey := c.Param("produto")
	queryType := c.Param("tipo")
	value := c.Param("valor")
	if productKey == "" || queryType == "" || value == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "produto, tipo e valor são obrigatórios"})
		return
	}

	result, err := h.service.QueryProduct(c.Request.Context(), userID, productKey, queryType, value)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

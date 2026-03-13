package http

import (
    "context"
    "net/http"

    "github.com/gin-gonic/gin"

    "buskatotal-backend/internal/infra/infocar"
)

type InfocarService interface {
    GetAgregadosB(ctx context.Context, userID, queryType, value string) (*infocar.AgregadosBResponse, error)
}

type InfocarHandler struct {
    service InfocarService
}

func NewInfocarHandler(service InfocarService) *InfocarHandler {
    return &InfocarHandler{service: service}
}

func (h *InfocarHandler) GetAgregadosB(c *gin.Context) {
    userID := c.GetHeader("X-User-Id")
    if userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "missing X-User-Id header"})
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
package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"buskatotal-backend/internal/domain/inspection"
	"buskatotal-backend/internal/infra/infovist"
)

type InfovistService interface {
	CreateInspection(ctx context.Context, userID string, input infovist.CreateInspectionRequest) (*infovist.CreateInspectionResponse, error)
	ViewInspection(ctx context.Context, userID, protocol string) (*infovist.ViewInspectionResponse, error)
	GetReportV1(ctx context.Context, userID, protocol string) (*infovist.ReportResponse, error)
	GetReportV2(ctx context.Context, userID, protocol string) (*infovist.ReportV2Response, error)
	ListInspections(ctx context.Context, userID string) ([]inspection.Inspection, error)
}

type InfovistHandler struct {
	service InfovistService
}

func NewInfovistHandler(service InfovistService) *InfovistHandler {
	return &InfovistHandler{service: service}
}

func (h *InfovistHandler) CreateInspection(c *gin.Context) {
	userID, ok := GetAuthUserID(c)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user"})
		return
	}

	var input infovist.CreateInspectionRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.CreateInspection(c.Request.Context(), userID, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *InfovistHandler) ViewInspection(c *gin.Context) {
	userID, ok := GetAuthUserID(c)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user"})
		return
	}

	protocol := c.Param("protocol")
	if protocol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "protocol is required"})
		return
	}

	result, err := h.service.ViewInspection(c.Request.Context(), userID, protocol)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *InfovistHandler) GetReportV1(c *gin.Context) {
	userID, ok := GetAuthUserID(c)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user"})
		return
	}

	protocol := c.Param("protocol")
	if protocol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "protocol is required"})
		return
	}

	result, err := h.service.GetReportV1(c.Request.Context(), userID, protocol)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *InfovistHandler) GetReportV2(c *gin.Context) {
	userID, ok := GetAuthUserID(c)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user"})
		return
	}

	protocol := c.Param("protocol")
	if protocol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "protocol is required"})
		return
	}

	result, err := h.service.GetReportV2(c.Request.Context(), userID, protocol)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *InfovistHandler) ListInspections(c *gin.Context) {
	userID, ok := GetAuthUserID(c)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user"})
		return
	}

	result, err := h.service.ListInspections(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result == nil {
		result = []inspection.Inspection{}
	}

	c.JSON(http.StatusOK, result)
}

package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"buskatotal-backend/internal/domain/lgpd"
)

type LGPDServiceInterface interface {
	GetUserData(ctx context.Context, userID string) (any, error)
	ExportUserData(ctx context.Context, userID string) (any, error)
	RequestDeletion(ctx context.Context, userID, reason string) (lgpd.DeletionRequest, error)
	ListDeletionRequests(ctx context.Context) ([]lgpd.DeletionRequest, error)
	ProcessDeletion(ctx context.Context, requestID, status, adminID string) (lgpd.DeletionRequest, error)
}

type LGPDHandler struct {
	service LGPDServiceInterface
}

func NewLGPDHandler(service LGPDServiceInterface) *LGPDHandler {
	return &LGPDHandler{service: service}
}

// GET /users/:id/data
func (h *LGPDHandler) GetUserData(c *gin.Context) {
	authUserID, ok := GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	userID := c.Param("id")
	if userID != authUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot access another user's data"})
		return
	}

	data, err := h.service.GetUserData(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

// GET /users/:id/data/export
func (h *LGPDHandler) ExportUserData(c *gin.Context) {
	authUserID, ok := GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	userID := c.Param("id")
	if userID != authUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot export another user's data"})
		return
	}

	data, err := h.service.ExportUserData(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

// POST /users/:id/data/deletion-request
func (h *LGPDHandler) RequestDeletion(c *gin.Context) {
	authUserID, ok := GetAuthUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	userID := c.Param("id")
	if userID != authUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot request deletion for another user"})
		return
	}

	var input struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req, err := h.service.RequestDeletion(c.Request.Context(), userID, input.Reason)
	if err != nil {
		status := http.StatusInternalServerError
		switch err.Error() {
		case "user not found":
			status = http.StatusNotFound
		case "deletion request already pending":
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"request_id": req.ID,
		"status":     req.Status,
		"message":    "Sua solicitação de exclusão foi registrada. Seus dados serão removidos em até 15 dias úteis conforme a LGPD.",
		"created_at": req.CreatedAt,
	})
}

// GET /admin/deletion-requests
func (h *LGPDHandler) ListDeletionRequests(c *gin.Context) {
	requests, err := h.service.ListDeletionRequests(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if requests == nil {
		requests = []lgpd.DeletionRequest{}
	}

	c.JSON(http.StatusOK, gin.H{
		"total":    len(requests),
		"requests": requests,
	})
}

// PATCH /admin/deletion-requests/:id
func (h *LGPDHandler) ProcessDeletion(c *gin.Context) {
	adminID, _ := GetAuthUserID(c)

	requestID := c.Param("id")
	if requestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request id is required"})
		return
	}

	var input struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Status != lgpd.DeletionStatusCompleted && input.Status != lgpd.DeletionStatusRejected && input.Status != lgpd.DeletionStatusProcessing {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be 'completed', 'rejected', or 'processing'"})
		return
	}

	req, err := h.service.ProcessDeletion(c.Request.Context(), requestID, input.Status, adminID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, req)
}

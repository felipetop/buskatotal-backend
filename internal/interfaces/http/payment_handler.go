package http

import (
    "context"
    "net/http"

    "github.com/gin-gonic/gin"

    "buskatotal-backend/internal/domain/payment"
)

type PaymentService interface {
    Credit(ctx context.Context, userID string, amount int64) (payment.Receipt, error)
}

type PaymentHandler struct {
    service PaymentService
}

type paymentCreditInput struct {
    Amount int64 `json:"amount"`
}

func NewPaymentHandler(service PaymentService) *PaymentHandler {
    return &PaymentHandler{service: service}
}

func (h *PaymentHandler) Credit(c *gin.Context) {
    userID := c.Param("id")
    if userID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "user id is required"})
        return
    }

    var input paymentCreditInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    receipt, err := h.service.Credit(c.Request.Context(), userID, input.Amount)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, receipt)
}
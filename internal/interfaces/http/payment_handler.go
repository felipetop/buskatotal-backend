package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"buskatotal-backend/internal/domain/payment"
)

type PaymentService interface {
	Credit(ctx context.Context, userID string, amount int64) (payment.Receipt, error)
	CreateOrder(ctx context.Context, userID string, amountCents int64, buyer payment.Buyer, returnURL string) (payment.Order, error)
	ProcessWebhook(ctx context.Context, referenceID string) error
	ListOrders(ctx context.Context, userID string) ([]payment.Order, error)
}

type PaymentHandler struct {
	service PaymentService
}

func NewPaymentHandler(service PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

// ── POST /payments/users/:id/credit ──────────────────────────────────────────
// Kept for backward compatibility / mock usage.

type paymentCreditInput struct {
	Amount int64 `json:"amount"`
}

func (h *PaymentHandler) Credit(c *gin.Context) {
	authUserID, ok := GetAuthUserID(c)
	if !ok || authUserID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user"})
		return
	}

	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user id is required"})
		return
	}
	if userID != authUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot credit another user"})
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

// ── POST /payments/users/:id/orders ──────────────────────────────────────────
// Creates a PicPay payment order. Returns a checkout URL + PIX QR code.
// The user must be authenticated and can only create orders for themselves.

type createOrderInput struct {
	AmountCents int64  `json:"amount_cents"`
	ReturnURL   string `json:"return_url"`
	Buyer       struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Document  string `json:"document"`  // CPF
		Email     string `json:"email"`
		Phone     string `json:"phone"`
	} `json:"buyer"`
}

func (h *PaymentHandler) CreateOrder(c *gin.Context) {
	authUserID, ok := GetAuthUserID(c)
	if !ok || authUserID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user"})
		return
	}

	userID := c.Param("id")
	if userID != authUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot create order for another user"})
		return
	}

	var input createOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.AmountCents <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount_cents must be greater than zero"})
		return
	}

	buyer := payment.Buyer{
		FirstName: input.Buyer.FirstName,
		LastName:  input.Buyer.LastName,
		Document:  input.Buyer.Document,
		Email:     input.Buyer.Email,
		Phone:     input.Buyer.Phone,
	}

	order, err := h.service.CreateOrder(c.Request.Context(), userID, input.AmountCents, buyer, input.ReturnURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"order_id":       order.ID,
		"reference_id":   order.ReferenceID,
		"status":         order.Status,
		"amount_cents":   order.AmountCents,
		"payment_url":    order.PaymentURL,
		"qrcode_text":    order.QRCodeText,
		"qrcode_base64":  order.QRCodeBase64,
		"created_at":     order.CreatedAt,
	})
}

// ── POST /payments/webhook ────────────────────────────────────────────────────
// Receives the PicPay callback. PicPay sends:
//   { "referenceId": "...", "authorizationId": "..." }
// We re-query PicPay to confirm the status (never trust the payload alone).

type picpayWebhookInput struct {
	ReferenceID     string `json:"referenceId"`
	AuthorizationID string `json:"authorizationId"`
}

func (h *PaymentHandler) Webhook(c *gin.Context) {
	var input picpayWebhookInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.ReferenceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "referenceId is required"})
		return
	}

	if err := h.service.ProcessWebhook(c.Request.Context(), input.ReferenceID); err != nil {
		// Return 200 to avoid PicPay retries for unknown orders; log the error internally.
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ── GET /payments/users/:id/orders ───────────────────────────────────────────
// Lists all orders for the authenticated user.

func (h *PaymentHandler) ListOrders(c *gin.Context) {
	authUserID, ok := GetAuthUserID(c)
	if !ok || authUserID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authenticated user"})
		return
	}

	userID := c.Param("id")
	if userID != authUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot list orders of another user"})
		return
	}

	orders, err := h.service.ListOrders(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orders)
}

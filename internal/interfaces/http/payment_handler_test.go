package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"buskatotal-backend/internal/domain/payment"
)

// mockPaymentService implements PaymentService for testing
type mockPaymentService struct {
	webhookCalled bool
	lastReference string
}

func (m *mockPaymentService) Credit(_ context.Context, _ string, _ int64) (payment.Receipt, error) {
	return payment.Receipt{}, nil
}
func (m *mockPaymentService) CreateOrder(_ context.Context, _ string, _ int64, _ payment.Buyer, _ string) (payment.Order, error) {
	return payment.Order{}, nil
}
func (m *mockPaymentService) ProcessWebhook(_ context.Context, referenceID string) error {
	m.webhookCalled = true
	m.lastReference = referenceID
	return nil
}
func (m *mockPaymentService) ProcessWebhookForUser(_ context.Context, _, _ string) error {
	return nil
}
func (m *mockPaymentService) ListOrders(_ context.Context, _ string) ([]payment.Order, error) {
	return nil, nil
}

func TestWebhook_NoSecret_AcceptsAll(t *testing.T) {
	svc := &mockPaymentService{}
	handler := NewPaymentHandler(svc, false)
	// No webhook secret set

	router := gin.New()
	router.POST("/payments/webhook", handler.Webhook)

	body, _ := json.Marshal(picpayWebhookInput{
		Type: "PAYMENT",
		Data: struct {
			Transaction struct {
				Status string `json:"status"`
			} `json:"transaction"`
			Charge struct {
				PaymentLinkID string `json:"paymentLinkId"`
			} `json:"charge"`
		}{
			Charge: struct {
				PaymentLinkID string `json:"paymentLinkId"`
			}{PaymentLinkID: "ref-123"},
		},
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/payments/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !svc.webhookCalled {
		t.Error("webhook should have been processed")
	}
}

func TestWebhook_WithSecret_RejectsInvalid(t *testing.T) {
	svc := &mockPaymentService{}
	handler := NewPaymentHandler(svc, false)
	handler.SetWebhookSecret("my-secret-123")

	router := gin.New()
	router.POST("/payments/webhook", handler.Webhook)

	body, _ := json.Marshal(map[string]interface{}{
		"type": "PAYMENT",
		"data": map[string]interface{}{
			"charge": map[string]interface{}{"paymentLinkId": "ref-123"},
		},
	})

	// No token
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/payments/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("no token: expected 403, got %d", w.Code)
	}

	// Wrong token
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/payments/webhook?token=wrong", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Errorf("wrong token: expected 403, got %d", w2.Code)
	}
}

func TestWebhook_WithSecret_AcceptsValid(t *testing.T) {
	svc := &mockPaymentService{}
	handler := NewPaymentHandler(svc, false)
	handler.SetWebhookSecret("my-secret-123")

	router := gin.New()
	router.POST("/payments/webhook", handler.Webhook)

	body, _ := json.Marshal(map[string]interface{}{
		"type": "PAYMENT",
		"data": map[string]interface{}{
			"charge": map[string]interface{}{"paymentLinkId": "ref-456"},
		},
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/payments/webhook?token=my-secret-123", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("valid token: expected 200, got %d", w.Code)
	}
	if !svc.webhookCalled {
		t.Error("webhook should have been processed")
	}
	if svc.lastReference != "ref-456" {
		t.Errorf("expected ref-456, got %s", svc.lastReference)
	}
}

func TestWebhook_MissingPaymentLinkID(t *testing.T) {
	svc := &mockPaymentService{}
	handler := NewPaymentHandler(svc, false)

	router := gin.New()
	router.POST("/payments/webhook", handler.Webhook)

	body, _ := json.Marshal(map[string]interface{}{
		"type": "PAYMENT",
		"data": map[string]interface{}{
			"charge": map[string]interface{}{},
		},
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/payments/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

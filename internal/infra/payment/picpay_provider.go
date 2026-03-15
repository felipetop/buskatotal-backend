package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	domain "buskatotal-backend/internal/domain/payment"
)

const picpayBaseURL = "https://appws.picpay.com/ecommerce/public/payments"

// PicPayProvider implements domain.Provider using the PicPay E-commerce API.
type PicPayProvider struct {
	token  string
	client *http.Client
}

func NewPicPayProvider(token string) *PicPayProvider {
	return &PicPayProvider{
		token:  token,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// ── PicPay request / response types ──────────────────────────────────────────

type picpayBuyer struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Document  string `json:"document"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
}

type picpayCreateRequest struct {
	ReferenceID string      `json:"referenceId"`
	CallbackURL string      `json:"callbackUrl"`
	ReturnURL   string      `json:"returnUrl,omitempty"`
	Value       float64     `json:"value"` // PicPay expects BRL float (e.g. 10.50)
	ExpiresAt   string      `json:"expiresAt"`
	Buyer       picpayBuyer `json:"buyer"`
}

type picpayQRCode struct {
	Content string `json:"content"`
	Base64  string `json:"base64"`
}

type picpayCreateResponse struct {
	ReferenceID string       `json:"referenceId"`
	PaymentURL  string       `json:"paymentUrl"`
	QRCode      picpayQRCode `json:"qrcode"`
	ExpiresAt   string       `json:"expiresAt"`
}

type picpayStatusResponse struct {
	ReferenceID     string `json:"referenceId"`
	AuthorizationID string `json:"authorizationId"`
	Status          string `json:"status"`
}

type picpayErrorResponse struct {
	Message string `json:"message"`
}

// ── Provider interface implementation ────────────────────────────────────────

func (p *PicPayProvider) CreateOrder(ctx context.Context, input domain.CreateOrderInput) (domain.OrderResult, error) {
	if p.token == "" {
		return domain.OrderResult{}, errors.New("picpay token not configured")
	}

	expiresAt := time.Now().Add(30 * time.Minute).Format(time.RFC3339)

	body := picpayCreateRequest{
		ReferenceID: input.ReferenceID,
		CallbackURL: input.CallbackURL,
		ReturnURL:   input.ReturnURL,
		Value:       centsToFloat(input.AmountCents),
		ExpiresAt:   expiresAt,
		Buyer: picpayBuyer{
			FirstName: input.Buyer.FirstName,
			LastName:  input.Buyer.LastName,
			Document:  input.Buyer.Document,
			Email:     input.Buyer.Email,
			Phone:     input.Buyer.Phone,
		},
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return domain.OrderResult{}, fmt.Errorf("picpay: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, picpayBaseURL, bytes.NewReader(raw))
	if err != nil {
		return domain.OrderResult{}, fmt.Errorf("picpay: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-picpay-token", p.token)

	resp, err := p.client.Do(req)
	if err != nil {
		return domain.OrderResult{}, fmt.Errorf("picpay: http call: %w", err)
	}
	defer resp.Body.Close()

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.OrderResult{}, fmt.Errorf("picpay: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var apiErr picpayErrorResponse
		_ = json.Unmarshal(responseBytes, &apiErr)
		if apiErr.Message != "" {
			return domain.OrderResult{}, fmt.Errorf("picpay: %s", apiErr.Message)
		}
		return domain.OrderResult{}, fmt.Errorf("picpay: unexpected status %d", resp.StatusCode)
	}

	var result picpayCreateResponse
	if err := json.Unmarshal(responseBytes, &result); err != nil {
		return domain.OrderResult{}, fmt.Errorf("picpay: decode response: %w", err)
	}

	expires, _ := time.Parse(time.RFC3339, result.ExpiresAt)

	return domain.OrderResult{
		ReferenceID:  result.ReferenceID,
		PaymentURL:   result.PaymentURL,
		QRCodeText:   result.QRCode.Content,
		QRCodeBase64: result.QRCode.Base64,
		ExpiresAt:    expires,
	}, nil
}

func (p *PicPayProvider) GetOrderStatus(ctx context.Context, referenceID string) (domain.OrderStatus, error) {
	if p.token == "" {
		return "", errors.New("picpay token not configured")
	}

	url := fmt.Sprintf("%s/%s/status", picpayBaseURL, referenceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("picpay: build status request: %w", err)
	}
	req.Header.Set("x-picpay-token", p.token)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("picpay: status http call: %w", err)
	}
	defer resp.Body.Close()

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("picpay: read status response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr picpayErrorResponse
		_ = json.Unmarshal(responseBytes, &apiErr)
		if apiErr.Message != "" {
			return "", fmt.Errorf("picpay: %s", apiErr.Message)
		}
		return "", fmt.Errorf("picpay: unexpected status %d", resp.StatusCode)
	}

	var result picpayStatusResponse
	if err := json.Unmarshal(responseBytes, &result); err != nil {
		return "", fmt.Errorf("picpay: decode status response: %w", err)
	}

	return mapPicPayStatus(result.Status), nil
}

// Credit is not applicable to PicPay (async flow only).
func (p *PicPayProvider) Credit(_ context.Context, _ string, _ int64) (domain.Receipt, error) {
	return domain.Receipt{}, errors.New("picpay: direct credit not supported; use CreateOrder")
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func centsToFloat(cents int64) float64 {
	return float64(cents) / 100.0
}

func mapPicPayStatus(s string) domain.OrderStatus {
	switch s {
	case "paid", "completed":
		return domain.StatusPaid
	case "expired":
		return domain.StatusExpired
	case "cancelled", "refunded":
		return domain.StatusCancelled
	case "chargeback":
		return domain.StatusChargeback
	default:
		return domain.StatusPending
	}
}

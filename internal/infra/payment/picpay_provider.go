package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	domain "buskatotal-backend/internal/domain/payment"
)

const (
	picpayOAuthURL  = "https://api.picpay.com/oauth2/token"
	picpayLinkURL   = "https://api.picpay.com/v1/paymentlink/create"
	picpayStatusURL = "https://api.picpay.com/v1/paymentlink"
)

// PicPayProvider implements domain.Provider using the PicPay Link de Pagamento API.
type PicPayProvider struct {
	clientID     string
	clientSecret string
	client       *http.Client

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

func NewPicPayProvider(clientID, clientSecret string) *PicPayProvider {
	return &PicPayProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		client:       &http.Client{Timeout: 15 * time.Second},
	}
}

// ── OAuth2 token management ───────────────────────────────────────────────────

func (p *PicPayProvider) getAccessToken(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.accessToken != "" && time.Now().Before(p.tokenExpiry) {
		return p.accessToken, nil
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, picpayOAuthURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("picpay oauth2: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("picpay oauth2: http call: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("picpay oauth2: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("picpay oauth2: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("picpay oauth2: decode response: %w", err)
	}

	p.accessToken = tokenResp.AccessToken
	p.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-30) * time.Second)

	return p.accessToken, nil
}

// ── PicPay request / response types ──────────────────────────────────────────

type picpayLinkRequest struct {
	Name                 string   `json:"name"`
	Description          string   `json:"description"`
	Amount               int64    `json:"amount"` // cents
	PaymentMethods       []string `json:"paymentMethods"`
	OrderNumber          string   `json:"orderNumber"`
	RedirectURL          string   `json:"redirectUrl,omitempty"`
	ExpirationDate       string   `json:"expirationDate"`
	MaxInstallmentNumber int      `json:"maxInstallmentNumber"`
}

type picpayLinkResponse struct {
	Link        string `json:"link"`
	Deeplink    string `json:"deeplink"`
	BRCode      string `json:"brcode"`
	QRCode      string `json:"qrCode"`
	TxID        string `json:"txid"`
	OrderNumber string `json:"orderNumber"`
	Status      string `json:"status"`
}

type picpayStatusResponse struct {
	OrderNumber string `json:"orderNumber"`
	Status      string `json:"status"`
}

// ── Provider interface implementation ────────────────────────────────────────

func (p *PicPayProvider) CreateOrder(ctx context.Context, input domain.CreateOrderInput) (domain.OrderResult, error) {
	if p.clientID == "" || p.clientSecret == "" {
		return domain.OrderResult{}, errors.New("picpay credentials not configured")
	}

	token, err := p.getAccessToken(ctx)
	if err != nil {
		return domain.OrderResult{}, err
	}

	expiresAt := time.Now().Add(30 * time.Minute).Format(time.RFC3339)

	body := picpayLinkRequest{
		Name:                 "Depósito BuskaTotal",
		Description:          fmt.Sprintf("Depósito de R$ %.2f", centsToFloat(input.AmountCents)),
		Amount:               input.AmountCents,
		PaymentMethods:       []string{"CREDIT_CARD", "PIX"},
		OrderNumber:          input.ReferenceID,
		RedirectURL:          input.ReturnURL,
		ExpirationDate:       expiresAt,
		MaxInstallmentNumber: 1,
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return domain.OrderResult{}, fmt.Errorf("picpay: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, picpayLinkURL, bytes.NewReader(raw))
	if err != nil {
		return domain.OrderResult{}, fmt.Errorf("picpay: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

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
		return domain.OrderResult{}, fmt.Errorf("picpay: unexpected status %d: %s", resp.StatusCode, string(responseBytes))
	}

	var result picpayLinkResponse
	if err := json.Unmarshal(responseBytes, &result); err != nil {
		return domain.OrderResult{}, fmt.Errorf("picpay: decode response: %w", err)
	}

	expires, _ := time.Parse(time.RFC3339, expiresAt)

	return domain.OrderResult{
		ReferenceID:  input.ReferenceID,
		PaymentURL:   result.Link,
		QRCodeText:   result.BRCode,
		QRCodeBase64: result.QRCode,
		ExpiresAt:    expires,
	}, nil
}

func (p *PicPayProvider) GetOrderStatus(ctx context.Context, referenceID string) (domain.OrderStatus, error) {
	if p.clientID == "" || p.clientSecret == "" {
		return "", errors.New("picpay credentials not configured")
	}

	token, err := p.getAccessToken(ctx)
	if err != nil {
		return "", err
	}

	statusURL := fmt.Sprintf("%s/%s", picpayStatusURL, referenceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return "", fmt.Errorf("picpay: build status request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

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
		return "", fmt.Errorf("picpay: unexpected status %d: %s", resp.StatusCode, string(responseBytes))
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
	case "PAID", "paid", "completed", "COMPLETED":
		return domain.StatusPaid
	case "EXPIRED", "expired":
		return domain.StatusExpired
	case "CANCELLED", "cancelled", "refunded", "REFUNDED":
		return domain.StatusCancelled
	case "CHARGEBACK", "chargeback":
		return domain.StatusChargeback
	default:
		return domain.StatusPending
	}
}

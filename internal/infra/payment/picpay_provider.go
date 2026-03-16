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
	picpayCreateURL = "https://api.picpay.com/ecommerce/v2/payments"
	picpayStatusURL = "https://api.picpay.com/ecommerce/v2/payments/%s/status"
)

// PicPayProvider implements domain.Provider using the PicPay E-commerce V2 API.
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

	tokenBody := map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     p.clientID,
		"client_secret": p.clientSecret,
	}
	tokenRaw, err := json.Marshal(tokenBody)
	if err != nil {
		return "", fmt.Errorf("picpay oauth2: marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, picpayOAuthURL, strings.NewReader(string(tokenRaw)))
	if err != nil {
		return "", fmt.Errorf("picpay oauth2: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

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

type picpayBuyer struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Document  string `json:"document"`
	Email     string `json:"email"`
	Phone     string `json:"phone,omitempty"`
}

type picpayCreateRequest struct {
	ReferenceID string      `json:"referenceId"`
	CallbackURL string      `json:"callbackUrl"`
	ReturnURL   string      `json:"returnUrl,omitempty"`
	Value       float64     `json:"value"`
	ExpiresAt   string      `json:"expiresAt"`
	Buyer       picpayBuyer `json:"buyer"`
}

type picpayCreateResponse struct {
	ReferenceID string `json:"referenceId"`
	PaymentURL  string `json:"paymentUrl"`
	ExpiresAt   string `json:"expiresAt"`
	QRCode      struct {
		Content string `json:"content"`
		Base64  string `json:"base64"`
	} `json:"qrcode"`
}

type picpayStatusResponse struct {
	ReferenceID     string `json:"referenceId"`
	AuthorizationID string `json:"authorizationId"`
	Status          string `json:"status"`
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, picpayCreateURL, bytes.NewReader(raw))
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
	if p.clientID == "" || p.clientSecret == "" {
		return "", errors.New("picpay credentials not configured")
	}

	statusToken, err := p.getAccessToken(ctx)
	if err != nil {
		return "", err
	}

	statusURL := fmt.Sprintf(picpayStatusURL, referenceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return "", fmt.Errorf("picpay: build status request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+statusToken)

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

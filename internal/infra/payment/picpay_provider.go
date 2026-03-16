package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	domain "buskatotal-backend/internal/domain/payment"
)

const (
	picpayOAuthURL  = "https://api.picpay.com/oauth2/token"
	picpayCreateURL = "https://api.picpay.com/v1/paymentlink/create"
	picpayStatusURL = "https://api.picpay.com/v1/paymentlink/%s"
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

type picpayCreateRequest struct {
	Charge  picpayCharge  `json:"charge"`
	Options picpayOptions `json:"options"`
}

type picpayCharge struct {
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	OrderNumber     string        `json:"order_number"`
	RedirectURL     string        `json:"redirect_url,omitempty"`
	NotificationURL string        `json:"notification_url,omitempty"`
	Payment         picpayPayment `json:"payment"`
	Amounts         picpayAmounts `json:"amounts"`
}

type picpayPayment struct {
	Methods            []string `json:"methods"`
	BrcodeArrangements []string `json:"brcode_arrangements"`
}

type picpayAmounts struct {
	Product int64 `json:"product"`
}

type picpayOptions struct {
	AllowCreatePixKey bool   `json:"allow_create_pix_key"`
	ExpiredAt         string `json:"expired_at"`
	NotificationURL   string `json:"notification_url,omitempty"`
}

type picpayCreateResponse struct {
	BRCode         string `json:"brcode"`
	Link           string `json:"link"`
	TxID           string `json:"txid"`
	ExpirationDate string `json:"expirationDate"`
}

type picpayStatusResponse struct {
	Status string `json:"status"`
	// transactions is a list of payments made against the link
	Transactions []struct {
		Status string `json:"status"`
	} `json:"transactions"`
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

	expiresAt := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	orderNumber := input.ReferenceID
	if len(orderNumber) > 15 {
		orderNumber = orderNumber[:15]
	}

	body := picpayCreateRequest{
		Charge: picpayCharge{
			Name:            input.Buyer.FirstName + " " + input.Buyer.LastName,
			Description:     "Pedido " + input.ReferenceID,
			OrderNumber:     orderNumber,
			RedirectURL:     input.ReturnURL,
			NotificationURL: input.CallbackURL,
			Payment: picpayPayment{
				Methods:            []string{"BRCODE"},
				BrcodeArrangements: []string{"PICPAY", "PIX"},
			},
			Amounts: picpayAmounts{
				Product: input.AmountCents,
			},
		},
		Options: picpayOptions{
			AllowCreatePixKey: true,
			ExpiredAt:         expiresAt,
			NotificationURL:   input.CallbackURL,
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

	expires, _ := time.Parse("2006-01-02T15:04:05.000000Z", result.ExpirationDate)

	// paymentLinkId is the last path segment of the link URL.
	// The webhook identifies payments by this ID, so we use it as ReferenceID.
	paymentLinkID := result.Link
	if idx := strings.LastIndex(result.Link, "/"); idx >= 0 {
		paymentLinkID = result.Link[idx+1:]
	}

	return domain.OrderResult{
		ReferenceID: paymentLinkID,
		PaymentURL:  result.Link,
		QRCodeText:  result.BRCode,
		ExpiresAt:   expires,
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

	log.Printf("picpay: status response for %s: %s", referenceID, string(responseBytes))

	// If there are transactions, use the status of the most recent one
	for _, t := range result.Transactions {
		if t.Status != "" {
			return mapPicPayStatus(t.Status), nil
		}
	}

	return mapPicPayStatus(result.Status), nil
}

// Credit is not applicable to PicPay (async flow only).
func (p *PicPayProvider) Credit(_ context.Context, _ string, _ int64) (domain.Receipt, error) {
	return domain.Receipt{}, errors.New("picpay: direct credit not supported; use CreateOrder")
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func mapPicPayStatus(s string) domain.OrderStatus {
	switch strings.ToUpper(s) {
	case "PAYED", "PAID", "COMPLETED":
		return domain.StatusPaid
	case "EXPIRED":
		return domain.StatusExpired
	case "REFUNDED", "PARTREFUNDED", "CANCELLED":
		return domain.StatusCancelled
	case "CHARGEBACK":
		return domain.StatusChargeback
	default:
		return domain.StatusPending
	}
}

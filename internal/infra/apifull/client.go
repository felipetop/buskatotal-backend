package apifull

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
	"time"
)

// Client handles communication with the API Full REST API.
// All endpoints use POST with Bearer token auth and JSON body.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// ProductResponse is the generic response from API Full.
// All products return { "status": "...", "dados": { ... } }.
type ProductResponse struct {
	Status string                 `json:"status"`
	Dados  map[string]interface{} `json:"dados"`
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// QueryProduct calls any API Full endpoint.
// endpoint: the API path (e.g. "pf-dadosbasicos")
// body: the request payload (e.g. {"cpf": "12345678900", "link": "pf-dadosbasicos"})
func (c *Client) QueryProduct(ctx context.Context, endpoint string, body map[string]interface{}) (*ProductResponse, error) {
	if c.token == "" {
		return nil, errors.New("apifull token missing: set APIFULL_TOKEN")
	}
	if endpoint == "" {
		return nil, errors.New("endpoint is required")
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("apifull %s payload: %w", endpoint, err)
	}

	url := fmt.Sprintf("%s/%s", c.baseURL, endpoint)
	log.Printf("[apifull] POST %s", url)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("apifull %s request: %w", endpoint, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("apifull %s request: %w", endpoint, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("[apifull] RESP %s | status: %d", endpoint, resp.StatusCode)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("apifull %s failed (status %d): %s", endpoint, resp.StatusCode, string(respBody))
	}

	var result ProductResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("apifull %s decode: %w (body: %s)", endpoint, err, string(respBody))
	}

	if result.Status != "" && result.Status != "sucesso" {
		return nil, fmt.Errorf("apifull %s returned status: %s", endpoint, result.Status)
	}

	return &result, nil
}

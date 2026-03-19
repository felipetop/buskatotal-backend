package infovist

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL  string
	email    string
	password string
	apiToken string
	http     *http.Client
}

type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	APIToken string `json:"api_token"`
}

type AuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	CreatedAt   int64  `json:"created_at"`
}

type CreateInspectionRequest struct {
	Customer        string `json:"customer"`
	Cellphone       string `json:"cellphone"`
	Plate           string `json:"plate,omitempty"`
	Chassis         string `json:"chassis,omitempty"`
	Notes           string `json:"notes,omitempty"`
	CPF             string `json:"cpf,omitempty"`
	CanNotify       *bool  `json:"can_notify,omitempty"`
	AppletProfileID string `json:"applet_profile_id,omitempty"`
}

type CreateInspectionResponse struct {
	Protocol string `json:"protocol"`
}

type InspectionStatus struct {
	Value      string `json:"value"`
	StatusEnum string `json:"status_enum"`
	CreatedAt  string `json:"created_at"`
}

type ViewInspectionResponse struct {
	Protocol string             `json:"protocol"`
	Statuses []InspectionStatus `json:"statuses"`
}

func NewClient(baseURL, email, password, apiToken string) *Client {
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		email:    email,
		password: password,
		apiToken: apiToken,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Authenticate(ctx context.Context) (*AuthResponse, error) {
	if c.email == "" || c.password == "" || c.apiToken == "" {
		return nil, errors.New("infovist credentials missing: set INFOVIST_EMAIL, INFOVIST_PASSWORD and INFOVIST_API_TOKEN")
	}

	payload, err := json.Marshal(AuthRequest{
		Email:    c.email,
		Password: c.password,
		APIToken: c.apiToken,
	})
	if err != nil {
		return nil, fmt.Errorf("infovist auth payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/auth/login", bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("infovist auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("infovist auth request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("infovist auth failed (status %d): %s", resp.StatusCode, string(body))
	}

	// Infovist wraps the response in a "data" field
	var wrapper struct {
		Data AuthResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, fmt.Errorf("infovist auth decode error: %w (body: %s)", err, string(body))
	}
	if wrapper.Data.AccessToken == "" {
		return nil, fmt.Errorf("infovist auth response missing access_token (body: %s)", string(body))
	}

	return &wrapper.Data, nil
}

func (c *Client) CreateInspection(ctx context.Context, token string, input CreateInspectionRequest) (*CreateInspectionResponse, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("infovist create inspection payload: %w", err)
	}

	resp, err := c.doAuthorized(ctx, http.MethodPost, "/inspection", token, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("infovist create inspection failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result CreateInspectionResponse
	if err := json.Unmarshal(body, &result); err == nil && result.Protocol != "" {
		return &result, nil
	}

	var wrapper struct {
		Data CreateInspectionResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, fmt.Errorf("infovist create inspection decode error (body: %s)", string(body))
	}
	return &wrapper.Data, nil
}

func (c *Client) ViewInspection(ctx context.Context, token, protocol string) (*ViewInspectionResponse, error) {
	resp, err := c.doAuthorized(ctx, http.MethodGet, "/inspection/"+protocol, token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("infovist view inspection failed (status %d): %s", resp.StatusCode, string(body))
	}

	// Try direct parse first, then try with "data" wrapper
	var result ViewInspectionResponse
	if err := json.Unmarshal(body, &result); err == nil && result.Protocol != "" {
		return &result, nil
	}

	var wrapper struct {
		Data ViewInspectionResponse `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, fmt.Errorf("infovist view inspection decode error (body: %s)", string(body))
	}
	return &wrapper.Data, nil
}

// ReportResponse represents the v1 report PDF response.
type ReportResponse struct {
	Data ReportData `json:"data"`
}

type ReportData struct {
	ReportPDF          string                 `json:"report_pdf"`
	Result             *ReportResult          `json:"result,omitempty"`
	InspectionID       *string                `json:"inspection_id"`
	InspectionEventData []map[string]interface{} `json:"inspection_event_data"`
	InspectionData     interface{}            `json:"inspection_data"`
}

type ReportResult struct {
	APIs                       map[string]interface{} `json:"apis"`
	IsSearchAppletInstalled    *bool                  `json:"is_search_applet_installed,omitempty"`
	IsIdentityAppletInstalled  *bool                  `json:"is_identity_applet_installed,omitempty"`
	IsQRCodeAppletInstalled    *bool                  `json:"is_qr_code_applet_installed,omitempty"`
	IsIAFlowAppletInstalled    *bool                  `json:"is_ia_flow_applet_installed,omitempty"`
}

// ReportV2Response represents the v2 report PDF response.
type ReportV2Response struct {
	Data ReportV2Data `json:"data"`
}

type ReportV2Data struct {
	InspectionData      map[string]interface{}   `json:"inspection_data"`
	InspectionCaptures  []map[string]interface{} `json:"inspection_captures"`
	APIs                map[string]interface{}   `json:"apis"`
	ReportNotes         *string                  `json:"report_notes"`
	AssessmentScorePoint int                     `json:"assessment_score_point"`
	AssessmentStatus     string                  `json:"assessment_status"`
	AssessmentStatusEnum string                  `json:"assessment_status_enum"`
	AssessmentNotes      string                  `json:"assessment_notes"`
	ReportPDF            string                  `json:"report_pdf"`
}

func (c *Client) GetReportV1(ctx context.Context, token, protocol string) (*ReportResponse, error) {
	resp, err := c.doAuthorized(ctx, http.MethodGet, "/report/pdf/"+protocol, token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := checkResponse(resp, "get report v1"); err != nil {
		return nil, err
	}

	var result ReportResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("infovist get report v1 decode: %w", err)
	}
	return &result, nil
}

func (c *Client) GetReportV2(ctx context.Context, token, protocol string) (*ReportV2Response, error) {
	// v2 uses a different base path: /api/v2 instead of /api/v1
	v2URL := strings.Replace(c.baseURL, "/api/v1", "/api/v2", 1)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v2URL+"/report/pdf/"+protocol, nil)
	if err != nil {
		return nil, fmt.Errorf("infovist request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("infovist get report v2 request: %w", err)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp, "get report v2"); err != nil {
		return nil, err
	}

	var result ReportV2Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("infovist get report v2 decode: %w", err)
	}
	return &result, nil
}

func (c *Client) doAuthorized(ctx context.Context, method, path, token string, body []byte) (*http.Response, error) {
	if token == "" {
		return nil, errors.New("infovist: token is required")
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewBuffer(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("infovist request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	return c.http.Do(req)
}

func checkResponse(resp *http.Response, operation string) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("infovist %s failed (status %d): %s", operation, resp.StatusCode, string(body))
}

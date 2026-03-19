package infocar

import (
    "bytes"
    "context"
    "encoding/base64"
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
    "strings"
    "time"
)

type Client struct {
    baseURL  string
    idKey    string
    user     string
    password string
    http     *http.Client
}

type TokenResponse struct {
    Token string `json:"token"`
}

type AgregadosBResponse struct {
    Solicitacao map[string]interface{} `json:"solicitacao"`
    Retorno     map[string]interface{} `json:"retorno"`
    Dados       map[string]interface{} `json:"dados"`
}

// ProductResponse is the generic response for all Infocar products (same shape).
type ProductResponse = AgregadosBResponse

func NewClient(baseURL, idKey, user, password string) *Client {
    return &Client{
        baseURL:  strings.TrimRight(baseURL, "/"),
        idKey:    idKey,
        user:     user,
        password: password,
        http: &http.Client{
            Timeout: 20 * time.Second,
        },
    }
}

func (c *Client) GenerateToken(ctx context.Context) (string, error) {
    if c.user == "" || c.password == "" {
        return "", errors.New("infocar credentials missing: set INFOCAR_USER and INFOCAR_PASSWORD")
    }

    base64Key := encodeBasicAuth(c.user, c.password)
    payload, err := json.Marshal(map[string]string{"chave": base64Key})
    if err != nil {
        return "", fmt.Errorf("token payload: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/Token/GerarToken", bytes.NewBuffer(payload))
    if err != nil {
        return "", fmt.Errorf("token request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.http.Do(req)
    if err != nil {
        return "", fmt.Errorf("token request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return "", fmt.Errorf("token request failed: status %d", resp.StatusCode)
    }

    var tokenResponse TokenResponse
    if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
        return "", fmt.Errorf("token decode: %w", err)
    }
    if tokenResponse.Token == "" {
        return "", errors.New("token response missing token")
    }

    return tokenResponse.Token, nil
}

func (c *Client) QueryAgregadosB(ctx context.Context, token, queryType, value string) (*AgregadosBResponse, error) {
    if c.idKey == "" {
        return nil, errors.New("infocar id key missing: set INFOCAR_ID_KEY")
    }
    if token == "" {
        return nil, errors.New("token is required")
    }
    if queryType == "" || value == "" {
        return nil, errors.New("query type and value are required")
    }

    url := fmt.Sprintf("%s/v1.0/AgregadosB/%s/%s", c.baseURL, queryType, value)
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return nil, fmt.Errorf("agregados request: %w", err)
    }
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("infocar-id-Key", c.idKey)

    resp, err := c.http.Do(req)
    if err != nil {
        return nil, fmt.Errorf("agregados request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return nil, fmt.Errorf("agregados request failed: status %d", resp.StatusCode)
    }

    var result AgregadosBResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("agregados decode: %w", err)
    }

    return &result, nil
}

// QueryProduct queries any Infocar product by API version, product path, query type and value.
// Example: QueryProduct(ctx, token, "v1.0", "BaseEstadualB", "placa", "ABC1234")
func (c *Client) QueryProduct(ctx context.Context, token, apiVersion, productPath, queryType, value string) (*ProductResponse, error) {
    if c.idKey == "" {
        return nil, errors.New("infocar id key missing: set INFOCAR_ID_KEY")
    }
    if token == "" {
        return nil, errors.New("token is required")
    }
    if queryType == "" || value == "" {
        return nil, errors.New("query type and value are required")
    }

    url := fmt.Sprintf("%s/%s/%s/%s/%s", c.baseURL, apiVersion, productPath, queryType, value)
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return nil, fmt.Errorf("infocar %s request: %w", productPath, err)
    }
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("infocar-id-Key", c.idKey)

    resp, err := c.http.Do(req)
    if err != nil {
        return nil, fmt.Errorf("infocar %s request: %w", productPath, err)
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return nil, fmt.Errorf("infocar %s failed: status %d", productPath, resp.StatusCode)
    }

    var result ProductResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("infocar %s decode: %w", productPath, err)
    }

    return &result, nil
}

func encodeBasicAuth(user, password string) string {
    payload := []byte(fmt.Sprintf("%s:%s", user, password))
    return encodeBase64(payload)
}

// encodeBase64 is separated so we can keep usage visible and comment where secrets are used.
func encodeBase64(payload []byte) string {
    // NOTE: The credentials come from INFOCAR_USER and INFOCAR_PASSWORD environment variables.
    // Never store real values in the repository.
    return strings.TrimSpace(base64.StdEncoding.EncodeToString(payload))
}
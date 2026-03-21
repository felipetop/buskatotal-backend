package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"buskatotal-backend/internal/domain/user"
	"buskatotal-backend/internal/infra/apifull"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func setAuthUser(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("authUserID", userID)
		c.Set("authUserRole", "user")
		c.Next()
	}
}

// ---------------------------------------------------------------------------
// Mock: ApiFullService
// ---------------------------------------------------------------------------

type stubApiFullService struct {
	resp *apifull.ProductResponse
	err  error
}

func (s *stubApiFullService) QueryProduct(_ context.Context, _, _, _ string) (*apifull.ProductResponse, error) {
	return s.resp, s.err
}

// ---------------------------------------------------------------------------
// Mock: AuthService
// ---------------------------------------------------------------------------

type stubAuthService struct {
	registerUser  user.User
	registerToken string
	registerErr   error

	loginUser  user.User
	loginToken string
	loginErr   error

	resendErr        error
	forgotErr        error
	resetPasswordErr error
}

func (s *stubAuthService) Register(_ context.Context, _, _, _ string) (user.User, string, error) {
	return s.registerUser, s.registerToken, s.registerErr
}

func (s *stubAuthService) Login(_ context.Context, _, _ string) (user.User, string, error) {
	return s.loginUser, s.loginToken, s.loginErr
}

func (s *stubAuthService) ResendVerification(_ context.Context, _ string) error {
	return s.resendErr
}

func (s *stubAuthService) ForgotPassword(_ context.Context, _ string) error {
	return s.forgotErr
}

func (s *stubAuthService) ResetPassword(_ context.Context, _, _ string) error {
	return s.resetPasswordErr
}

// ---------------------------------------------------------------------------
// Mock: EmailVerificationService (nil-safe stub)
// ---------------------------------------------------------------------------

type stubEmailVerifyService struct{}

func (s *stubEmailVerifyService) Verify(_ context.Context, _ string) error            { return nil }
func (s *stubEmailVerifyService) GenerateAndSend(_ context.Context, _, _ string) error { return nil }

// ---------------------------------------------------------------------------
// Mock: UserService
// ---------------------------------------------------------------------------

type stubUserService struct {
	getByIDUser user.User
	getByIDErr  error

	createUser user.User
	createErr  error

	listUsers []user.User
	listErr   error

	updateUser user.User
	updateErr  error

	deleteErr error
}

func (s *stubUserService) Create(_ context.Context, _ user.User) (user.User, error) {
	return s.createUser, s.createErr
}

func (s *stubUserService) GetByID(_ context.Context, _ string) (user.User, error) {
	return s.getByIDUser, s.getByIDErr
}

func (s *stubUserService) List(_ context.Context) ([]user.User, error) {
	return s.listUsers, s.listErr
}

func (s *stubUserService) Update(_ context.Context, _ user.User) (user.User, error) {
	return s.updateUser, s.updateErr
}

func (s *stubUserService) Delete(_ context.Context, _ string) error {
	return s.deleteErr
}

// ===========================================================================
// ApiFullHandler tests
// ===========================================================================

func TestApiFullHandler_QueryProduct_Success(t *testing.T) {
	svc := &stubApiFullService{
		resp: &apifull.ProductResponse{Status: "ok", Dados: map[string]interface{}{"key": "value"}},
	}
	handler := NewApiFullHandler(svc)

	r := gin.New()
	r.POST("/consultas/apifull/:produto", setAuthUser("user-1"), handler.QueryProduct)

	body, _ := json.Marshal(map[string]string{"valor": "12345678900"})
	req := httptest.NewRequest(http.MethodPost, "/consultas/apifull/cpf", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %v", resp["status"])
	}
}

func TestApiFullHandler_QueryProduct_NoAuth(t *testing.T) {
	svc := &stubApiFullService{}
	handler := NewApiFullHandler(svc)

	r := gin.New()
	// No auth middleware
	r.POST("/consultas/apifull/:produto", handler.QueryProduct)

	body, _ := json.Marshal(map[string]string{"valor": "12345678900"})
	req := httptest.NewRequest(http.MethodPost, "/consultas/apifull/cpf", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestApiFullHandler_QueryProduct_MissingValor(t *testing.T) {
	svc := &stubApiFullService{}
	handler := NewApiFullHandler(svc)

	r := gin.New()
	r.POST("/consultas/apifull/:produto", setAuthUser("user-1"), handler.QueryProduct)

	// Empty body
	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/consultas/apifull/cpf", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestApiFullHandler_QueryProduct_ServiceError(t *testing.T) {
	svc := &stubApiFullService{
		err: errors.New("saldo insuficiente"),
	}
	handler := NewApiFullHandler(svc)

	r := gin.New()
	r.POST("/consultas/apifull/:produto", setAuthUser("user-1"), handler.QueryProduct)

	body, _ := json.Marshal(map[string]string{"valor": "12345678900"})
	req := httptest.NewRequest(http.MethodPost, "/consultas/apifull/cpf", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "saldo insuficiente" {
		t.Errorf("expected error message 'saldo insuficiente', got %v", resp["error"])
	}
}

// ===========================================================================
// AuthHandler tests
// ===========================================================================

func TestAuthHandler_Register_Success(t *testing.T) {
	accepted := true
	svc := &stubAuthService{
		registerUser:  user.User{ID: "u1", Name: "Test", Email: "test@example.com"},
		registerToken: "jwt-token-123",
	}
	handler := NewAuthHandler(svc, &stubEmailVerifyService{})

	r := gin.New()
	r.POST("/auth/register", handler.Register)

	body, _ := json.Marshal(map[string]interface{}{
		"name":           "Test",
		"email":          "test@example.com",
		"password":       "Str0ng!Pass",
		"accepted_terms": accepted,
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["token"] != "jwt-token-123" {
		t.Errorf("expected token jwt-token-123, got %v", resp["token"])
	}

	// Check Cache-Control header
	if cc := w.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("expected Cache-Control no-store, got %q", cc)
	}
}

func TestAuthHandler_Register_InvalidInput(t *testing.T) {
	svc := &stubAuthService{}
	handler := NewAuthHandler(svc, &stubEmailVerifyService{})

	r := gin.New()
	r.POST("/auth/register", handler.Register)

	// Send invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_Register_NoTerms(t *testing.T) {
	svc := &stubAuthService{}
	handler := NewAuthHandler(svc, &stubEmailVerifyService{})

	r := gin.New()
	r.POST("/auth/register", handler.Register)

	// accepted_terms is false
	body, _ := json.Marshal(map[string]interface{}{
		"name":           "Test",
		"email":          "test@example.com",
		"password":       "Str0ng!Pass",
		"accepted_terms": false,
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	errMsg, _ := resp["error"].(string)
	if errMsg != "you must accept the terms of use and privacy policy" {
		t.Errorf("unexpected error: %s", errMsg)
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	svc := &stubAuthService{
		loginUser:  user.User{ID: "u1", Email: "test@example.com"},
		loginToken: "login-token",
	}
	handler := NewAuthHandler(svc, &stubEmailVerifyService{})

	r := gin.New()
	r.POST("/auth/login", handler.Login)

	body, _ := json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "Str0ng!Pass",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["token"] != "login-token" {
		t.Errorf("expected token login-token, got %v", resp["token"])
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	svc := &stubAuthService{
		loginErr: errors.New("invalid credentials"),
	}
	handler := NewAuthHandler(svc, &stubEmailVerifyService{})

	r := gin.New()
	r.POST("/auth/login", handler.Login)

	body, _ := json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "wrong",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_ForgotPassword_Success(t *testing.T) {
	svc := &stubAuthService{}
	handler := NewAuthHandler(svc, &stubEmailVerifyService{})

	r := gin.New()
	r.POST("/auth/forgot-password", handler.ForgotPassword)

	body, _ := json.Marshal(map[string]string{"email": "test@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_ForgotPassword_EmptyEmail(t *testing.T) {
	svc := &stubAuthService{}
	handler := NewAuthHandler(svc, &stubEmailVerifyService{})

	r := gin.New()
	r.POST("/auth/forgot-password", handler.ForgotPassword)

	body, _ := json.Marshal(map[string]string{"email": ""})
	req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_ResetPassword_Success(t *testing.T) {
	svc := &stubAuthService{}
	handler := NewAuthHandler(svc, &stubEmailVerifyService{})

	r := gin.New()
	r.POST("/auth/reset-password", handler.ResetPassword)

	body, _ := json.Marshal(map[string]string{
		"token":        "reset-token",
		"new_password": "NewStr0ng!Pass",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_ResetPassword_InvalidInput(t *testing.T) {
	svc := &stubAuthService{
		resetPasswordErr: errors.New("token and new_password are required"),
	}
	handler := NewAuthHandler(svc, &stubEmailVerifyService{})

	r := gin.New()
	r.POST("/auth/reset-password", handler.ResetPassword)

	// Send invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ===========================================================================
// UserHandler tests
// ===========================================================================

func TestUserHandler_GetBalance_Success(t *testing.T) {
	svc := &stubUserService{
		getByIDUser: user.User{ID: "user-1", Balance: 5000},
	}
	handler := NewUserHandler(svc)

	r := gin.New()
	r.GET("/users/:id/balance", setAuthUser("user-1"), handler.GetBalance)

	req := httptest.NewRequest(http.MethodGet, "/users/user-1/balance", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["user_id"] != "user-1" {
		t.Errorf("expected user_id user-1, got %v", resp["user_id"])
	}
	if resp["balance_cents"] != float64(5000) {
		t.Errorf("expected balance_cents 5000, got %v", resp["balance_cents"])
	}
	if resp["balance_brl"] != float64(50) {
		t.Errorf("expected balance_brl 50, got %v", resp["balance_brl"])
	}
}

func TestUserHandler_GetBalance_WrongUser(t *testing.T) {
	svc := &stubUserService{}
	handler := NewUserHandler(svc)

	r := gin.New()
	// Authenticated as user-1 but requesting user-2's balance
	r.GET("/users/:id/balance", setAuthUser("user-1"), handler.GetBalance)

	req := httptest.NewRequest(http.MethodGet, "/users/user-2/balance", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUserHandler_GetBalance_NoAuth(t *testing.T) {
	svc := &stubUserService{}
	handler := NewUserHandler(svc)

	r := gin.New()
	// No auth middleware
	r.GET("/users/:id/balance", handler.GetBalance)

	req := httptest.NewRequest(http.MethodGet, "/users/user-1/balance", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// ===========================================================================
// CatalogHandler tests
// ===========================================================================

func TestCatalogHandler_GetCatalog(t *testing.T) {
	handler := NewCatalogHandler(1.0) // 100% markup

	r := gin.New()
	r.GET("/catalog", handler.GetCatalog)

	req := httptest.NewRequest(http.MethodGet, "/catalog", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []CatalogCategoryResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if len(resp) == 0 {
		t.Fatal("expected at least one category")
	}

	// Verify first category is VEICULAR
	if resp[0].Key != "VEICULAR" {
		t.Errorf("expected first category key VEICULAR, got %s", resp[0].Key)
	}

	// Verify items have prices
	if len(resp[0].Items) == 0 {
		t.Fatal("expected items in first category")
	}

	// With 100% markup, the first item (AGREGADOS B, cost 50 cents) should be R$1,00
	firstPrice := resp[0].Items[0].Price
	if firstPrice != "R$1,00" {
		t.Errorf("expected price R$1,00 with 100%% markup on 50 cents cost, got %s", firstPrice)
	}
}

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"buskatotal-backend/internal/domain/inspection"
	"buskatotal-backend/internal/domain/lgpd"
	"buskatotal-backend/internal/domain/payment"
	"buskatotal-backend/internal/domain/user"
	"buskatotal-backend/internal/infra/infocar"
	"buskatotal-backend/internal/infra/infovist"
)

// ── Auth helper ──

func setFullAuth(userID, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("authUserID", userID)
		c.Set("authUserRole", role)
		c.Next()
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Mock: InfocarService
// ══════════════════════════════════════════════════════════════════════════════

type fullInfocarService struct{}

func (s *fullInfocarService) GetAgregadosB(_ context.Context, _, _, _ string) (*infocar.AgregadosBResponse, error) {
	return &infocar.AgregadosBResponse{}, nil
}

func (s *fullInfocarService) QueryProduct(_ context.Context, _, _, _, _ string) (*infocar.ProductResponse, error) {
	return &infocar.ProductResponse{}, nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Mock: InfovistService
// ══════════════════════════════════════════════════════════════════════════════

type fullInfovistService struct{}

func (s *fullInfovistService) CreateInspection(_ context.Context, _ string, _ infovist.CreateInspectionRequest) (*infovist.CreateInspectionResponse, error) {
	return &infovist.CreateInspectionResponse{Protocol: "PROTO123"}, nil
}

func (s *fullInfovistService) ViewInspection(_ context.Context, _, _ string) (*infovist.ViewInspectionResponse, error) {
	return &infovist.ViewInspectionResponse{Protocol: "PROTO123"}, nil
}

func (s *fullInfovistService) GetReportV1(_ context.Context, _, _ string) (*infovist.ReportResponse, error) {
	return &infovist.ReportResponse{}, nil
}

func (s *fullInfovistService) GetReportV2(_ context.Context, _, _ string) (*infovist.ReportV2Response, error) {
	return &infovist.ReportV2Response{}, nil
}

func (s *fullInfovistService) ListInspections(_ context.Context, _ string) ([]inspection.Inspection, error) {
	return []inspection.Inspection{}, nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Mock: PaymentService
// ══════════════════════════════════════════════════════════════════════════════

type fullPaymentService struct{}

func (s *fullPaymentService) Credit(_ context.Context, _ string, _ int64) (payment.Receipt, error) {
	return payment.Receipt{Provider: "mock", Reference: "ref1", Amount: 100}, nil
}

func (s *fullPaymentService) CreateOrder(_ context.Context, _ string, _ int64, _ payment.Buyer, _ string) (payment.Order, error) {
	return payment.Order{ID: "ord1", ReferenceID: "ref1", Status: payment.StatusPending, AmountCents: 1000}, nil
}

func (s *fullPaymentService) ProcessWebhook(_ context.Context, _ string) error {
	return nil
}

func (s *fullPaymentService) ProcessWebhookForUser(_ context.Context, _, _ string) error {
	return nil
}

func (s *fullPaymentService) ListOrders(_ context.Context, _ string) ([]payment.Order, error) {
	return []payment.Order{{ID: "ord1"}}, nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Mock: LGPDServiceInterface
// ══════════════════════════════════════════════════════════════════════════════

type fullLGPDService struct{}

func (s *fullLGPDService) GetUserData(_ context.Context, _ string) (any, error) {
	return map[string]string{"name": "Test"}, nil
}

func (s *fullLGPDService) ExportUserData(_ context.Context, _ string) (any, error) {
	return map[string]string{"export": "data"}, nil
}

func (s *fullLGPDService) RequestDeletion(_ context.Context, _, _ string) (lgpd.DeletionRequest, error) {
	return lgpd.DeletionRequest{ID: "del1", Status: lgpd.DeletionStatusPending}, nil
}

func (s *fullLGPDService) ListDeletionRequests(_ context.Context) ([]lgpd.DeletionRequest, error) {
	return []lgpd.DeletionRequest{{ID: "del1"}}, nil
}

func (s *fullLGPDService) ProcessDeletion(_ context.Context, _, _, _ string) (lgpd.DeletionRequest, error) {
	return lgpd.DeletionRequest{ID: "del1", Status: lgpd.DeletionStatusCompleted}, nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Mock: AuthService
// ══════════════════════════════════════════════════════════════════════════════

type fullAuthService struct{}

func (s *fullAuthService) Register(_ context.Context, _, _, _ string) (user.User, string, error) {
	return user.User{ID: "u1"}, "token123", nil
}

func (s *fullAuthService) Login(_ context.Context, _, _ string) (user.User, string, error) {
	return user.User{ID: "u1"}, "token123", nil
}

func (s *fullAuthService) ResendVerification(_ context.Context, _ string) error {
	return nil
}

func (s *fullAuthService) ForgotPassword(_ context.Context, _ string) error {
	return nil
}

func (s *fullAuthService) ResetPassword(_ context.Context, _, _ string) error {
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Mock: EmailVerificationService
// ══════════════════════════════════════════════════════════════════════════════

type fullEmailVerifyService struct{}

func (s *fullEmailVerifyService) Verify(_ context.Context, _ string) error {
	return nil
}

func (s *fullEmailVerifyService) GenerateAndSend(_ context.Context, _, _ string) error {
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Mock: UserService
// ══════════════════════════════════════════════════════════════════════════════

type fullUserService struct{}

func (s *fullUserService) Create(_ context.Context, input user.User) (user.User, error) {
	input.ID = "u-new"
	return input, nil
}

func (s *fullUserService) GetByID(_ context.Context, id string) (user.User, error) {
	return user.User{ID: id, Name: "Test", Balance: 5000}, nil
}

func (s *fullUserService) List(_ context.Context) ([]user.User, error) {
	return []user.User{{ID: "u1", Name: "A"}, {ID: "u2", Name: "B"}}, nil
}

func (s *fullUserService) Update(_ context.Context, input user.User) (user.User, error) {
	return input, nil
}

func (s *fullUserService) Delete(_ context.Context, _ string) error {
	return nil
}

// ══════════════════════════════════════════════════════════════════════════════
// Tests: InfocarHandler
// ══════════════════════════════════════════════════════════════════════════════

func TestFullInfocarHandler_GetAgregadosB_Success(t *testing.T) {
	handler := NewInfocarHandler(&fullInfocarService{})

	r := gin.New()
	r.GET("/infocar/:tipo/:valor", setFullAuth("user1", "user"), handler.GetAgregadosB)

	req := httptest.NewRequest(http.MethodGet, "/infocar/placa/ABC1234", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFullInfocarHandler_GetAgregadosB_NoAuth(t *testing.T) {
	handler := NewInfocarHandler(&fullInfocarService{})

	r := gin.New()
	r.GET("/infocar/:tipo/:valor", handler.GetAgregadosB)

	req := httptest.NewRequest(http.MethodGet, "/infocar/placa/ABC1234", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFullInfocarHandler_QueryProduct_Success(t *testing.T) {
	handler := NewInfocarHandler(&fullInfocarService{})

	r := gin.New()
	r.GET("/infocar/:produto/:tipo/:valor", setFullAuth("user1", "user"), handler.QueryProduct)

	req := httptest.NewRequest(http.MethodGet, "/infocar/agregados/placa/ABC1234", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFullInfocarHandler_QueryProduct_NoAuth(t *testing.T) {
	handler := NewInfocarHandler(&fullInfocarService{})

	r := gin.New()
	r.GET("/infocar/:produto/:tipo/:valor", handler.QueryProduct)

	req := httptest.NewRequest(http.MethodGet, "/infocar/agregados/placa/ABC1234", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Tests: InfovistHandler
// ══════════════════════════════════════════════════════════════════════════════

func TestFullInfovistHandler_ViewInspection_Success(t *testing.T) {
	handler := NewInfovistHandler(&fullInfovistService{})

	r := gin.New()
	r.GET("/infovist/:protocol", setFullAuth("user1", "user"), handler.ViewInspection)

	req := httptest.NewRequest(http.MethodGet, "/infovist/PROTO123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFullInfovistHandler_ViewInspection_NoAuth(t *testing.T) {
	handler := NewInfovistHandler(&fullInfovistService{})

	r := gin.New()
	r.GET("/infovist/:protocol", handler.ViewInspection)

	req := httptest.NewRequest(http.MethodGet, "/infovist/PROTO123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFullInfovistHandler_GetReportV1_Success(t *testing.T) {
	handler := NewInfovistHandler(&fullInfovistService{})

	r := gin.New()
	r.GET("/infovist/:protocol/report/v1", setFullAuth("user1", "user"), handler.GetReportV1)

	req := httptest.NewRequest(http.MethodGet, "/infovist/PROTO123/report/v1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFullInfovistHandler_GetReportV2_Success(t *testing.T) {
	handler := NewInfovistHandler(&fullInfovistService{})

	r := gin.New()
	r.GET("/infovist/:protocol/report/v2", setFullAuth("user1", "user"), handler.GetReportV2)

	req := httptest.NewRequest(http.MethodGet, "/infovist/PROTO123/report/v2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFullInfovistHandler_CreateInspection_Success(t *testing.T) {
	handler := NewInfovistHandler(&fullInfovistService{})

	body, _ := json.Marshal(infovist.CreateInspectionRequest{
		Customer:  "John",
		Cellphone: "11999999999",
		Plate:     "ABC1234",
	})

	r := gin.New()
	r.POST("/infovist", setFullAuth("user1", "user"), handler.CreateInspection)

	req := httptest.NewRequest(http.MethodPost, "/infovist", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Tests: PaymentHandler
// ══════════════════════════════════════════════════════════════════════════════

func TestFullPaymentHandler_Credit_Forbidden(t *testing.T) {
	handler := NewPaymentHandler(&fullPaymentService{}, false) // allowCredit = false

	r := gin.New()
	r.POST("/payments/users/:id/credit", setFullAuth("user1", "user"), handler.Credit)

	body, _ := json.Marshal(map[string]int64{"amount": 100})
	req := httptest.NewRequest(http.MethodPost, "/payments/users/user1/credit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFullPaymentHandler_CreateOrder_Success(t *testing.T) {
	handler := NewPaymentHandler(&fullPaymentService{}, false)

	body, _ := json.Marshal(map[string]interface{}{
		"amount_cents": 1000,
		"return_url":   "https://example.com/return",
		"buyer": map[string]string{
			"first_name": "John",
			"last_name":  "Doe",
			"document":   "12345678901",
			"email":      "john@example.com",
			"phone":      "11999999999",
		},
	})

	r := gin.New()
	r.POST("/payments/users/:id/orders", setFullAuth("user1", "user"), handler.CreateOrder)

	req := httptest.NewRequest(http.MethodPost, "/payments/users/user1/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFullPaymentHandler_CreateOrder_NoAuth(t *testing.T) {
	handler := NewPaymentHandler(&fullPaymentService{}, false)

	r := gin.New()
	r.POST("/payments/users/:id/orders", handler.CreateOrder)

	req := httptest.NewRequest(http.MethodPost, "/payments/users/user1/orders", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFullPaymentHandler_CreateOrder_WrongUser(t *testing.T) {
	handler := NewPaymentHandler(&fullPaymentService{}, false)

	body, _ := json.Marshal(map[string]interface{}{"amount_cents": 1000})

	r := gin.New()
	r.POST("/payments/users/:id/orders", setFullAuth("user1", "user"), handler.CreateOrder)

	req := httptest.NewRequest(http.MethodPost, "/payments/users/otheruser/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFullPaymentHandler_SyncOrder_Success(t *testing.T) {
	handler := NewPaymentHandler(&fullPaymentService{}, false)

	r := gin.New()
	r.POST("/payments/orders/:reference_id/sync", setFullAuth("user1", "user"), handler.SyncOrder)

	req := httptest.NewRequest(http.MethodPost, "/payments/orders/ref1/sync", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFullPaymentHandler_ListOrders_Success(t *testing.T) {
	handler := NewPaymentHandler(&fullPaymentService{}, false)

	r := gin.New()
	r.GET("/payments/users/:id/orders", setFullAuth("user1", "user"), handler.ListOrders)

	req := httptest.NewRequest(http.MethodGet, "/payments/users/user1/orders", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFullPaymentHandler_ListOrders_WrongUser(t *testing.T) {
	handler := NewPaymentHandler(&fullPaymentService{}, false)

	r := gin.New()
	r.GET("/payments/users/:id/orders", setFullAuth("user1", "user"), handler.ListOrders)

	req := httptest.NewRequest(http.MethodGet, "/payments/users/otheruser/orders", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Tests: LGPDHandler
// ══════════════════════════════════════════════════════════════════════════════

func TestFullLGPDHandler_ExportUserData_Success(t *testing.T) {
	handler := NewLGPDHandler(&fullLGPDService{})

	r := gin.New()
	r.GET("/users/:id/data/export", setFullAuth("user1", "user"), handler.ExportUserData)

	req := httptest.NewRequest(http.MethodGet, "/users/user1/data/export", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFullLGPDHandler_ListDeletionRequests_Success(t *testing.T) {
	handler := NewLGPDHandler(&fullLGPDService{})

	r := gin.New()
	r.GET("/admin/deletion-requests", setFullAuth("admin1", "admin"), handler.ListDeletionRequests)

	req := httptest.NewRequest(http.MethodGet, "/admin/deletion-requests", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["total"] != float64(1) {
		t.Fatalf("expected total=1, got %v", resp["total"])
	}
}

func TestFullLGPDHandler_ProcessDeletion_Success(t *testing.T) {
	handler := NewLGPDHandler(&fullLGPDService{})

	body, _ := json.Marshal(map[string]string{"status": lgpd.DeletionStatusCompleted})

	r := gin.New()
	r.PATCH("/admin/deletion-requests/:id", setFullAuth("admin1", "admin"), handler.ProcessDeletion)

	req := httptest.NewRequest(http.MethodPatch, "/admin/deletion-requests/del1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Tests: AuthHandler
// ══════════════════════════════════════════════════════════════════════════════

func TestFullAuthHandler_VerifyEmail_Success(t *testing.T) {
	handler := NewAuthHandler(&fullAuthService{}, &fullEmailVerifyService{})

	r := gin.New()
	r.GET("/auth/verify-email", handler.VerifyEmail)

	req := httptest.NewRequest(http.MethodGet, "/auth/verify-email?token=valid-token", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFullAuthHandler_VerifyEmail_MissingToken(t *testing.T) {
	handler := NewAuthHandler(&fullAuthService{}, &fullEmailVerifyService{})

	r := gin.New()
	r.GET("/auth/verify-email", handler.VerifyEmail)

	req := httptest.NewRequest(http.MethodGet, "/auth/verify-email", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFullAuthHandler_ResendVerification_Success(t *testing.T) {
	handler := NewAuthHandler(&fullAuthService{}, &fullEmailVerifyService{})

	r := gin.New()
	r.POST("/auth/resend-verification", setFullAuth("user1", "user"), handler.ResendVerification)

	req := httptest.NewRequest(http.MethodPost, "/auth/resend-verification", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFullAuthHandler_ResendVerification_NoAuth(t *testing.T) {
	handler := NewAuthHandler(&fullAuthService{}, &fullEmailVerifyService{})

	r := gin.New()
	r.POST("/auth/resend-verification", handler.ResendVerification)

	req := httptest.NewRequest(http.MethodPost, "/auth/resend-verification", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFullAuthHandler_ResetPassword_BadInput(t *testing.T) {
	handler := NewAuthHandler(&fullAuthService{}, &fullEmailVerifyService{})

	r := gin.New()
	r.POST("/auth/reset-password", handler.ResetPassword)

	// Send invalid JSON to trigger bind error
	req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ══════════════════════════════════════════════════════════════════════════════
// Tests: UserHandler
// ══════════════════════════════════════════════════════════════════════════════

func TestFullUserHandler_Create_Success(t *testing.T) {
	handler := NewUserHandler(&fullUserService{})

	body, _ := json.Marshal(map[string]string{
		"name":     "New User",
		"email":    "new@example.com",
		"password": "StrongPass1@",
	})

	r := gin.New()
	r.POST("/users", handler.Create)

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFullUserHandler_List_Success(t *testing.T) {
	handler := NewUserHandler(&fullUserService{})

	r := gin.New()
	r.GET("/users", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var users []user.User
	if err := json.Unmarshal(w.Body.Bytes(), &users); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
}

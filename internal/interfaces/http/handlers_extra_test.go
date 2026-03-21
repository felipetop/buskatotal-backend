package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	authinfra "buskatotal-backend/internal/infra/auth"
	"buskatotal-backend/internal/infra/infovist"

	"buskatotal-backend/internal/domain/inspection"
	"buskatotal-backend/internal/domain/lgpd"
	"buskatotal-backend/internal/domain/user"
)

// ── Extra mock: AdminService ──

type extraAdminService struct {
	users   []user.User
	userMap map[string]user.User
}

func (s *extraAdminService) ListUsers(_ context.Context) ([]user.User, error) {
	return s.users, nil
}

func (s *extraAdminService) SearchUsers(_ context.Context, _ string) ([]user.User, error) {
	return s.users, nil
}

func (s *extraAdminService) GetUserByID(_ context.Context, id string) (user.User, error) {
	u, ok := s.userMap[id]
	if !ok {
		return user.User{}, errors.New("user not found")
	}
	return u, nil
}

// ── Extra mock: InfovistService ──

type extraInfovistService struct {
	inspections []inspection.Inspection
}

func (s *extraInfovistService) CreateInspection(_ context.Context, _ string, _ infovist.CreateInspectionRequest) (*infovist.CreateInspectionResponse, error) {
	return &infovist.CreateInspectionResponse{Protocol: "PROTO-123"}, nil
}

func (s *extraInfovistService) ViewInspection(_ context.Context, _, _ string) (*infovist.ViewInspectionResponse, error) {
	return &infovist.ViewInspectionResponse{Protocol: "PROTO-123"}, nil
}

func (s *extraInfovistService) GetReportV1(_ context.Context, _, _ string) (*infovist.ReportResponse, error) {
	return nil, nil
}

func (s *extraInfovistService) GetReportV2(_ context.Context, _, _ string) (*infovist.ReportV2Response, error) {
	return nil, nil
}

func (s *extraInfovistService) ListInspections(_ context.Context, _ string) ([]inspection.Inspection, error) {
	return s.inspections, nil
}

// ── Extra mock: LGPDServiceInterface ──

type extraLGPDService struct{}

func (s *extraLGPDService) GetUserData(_ context.Context, userID string) (any, error) {
	return map[string]string{"id": userID, "name": "Test User"}, nil
}

func (s *extraLGPDService) ExportUserData(_ context.Context, userID string) (any, error) {
	return map[string]string{"id": userID}, nil
}

func (s *extraLGPDService) RequestDeletion(_ context.Context, userID, reason string) (lgpd.DeletionRequest, error) {
	return lgpd.DeletionRequest{
		ID:        "del-1",
		UserID:    userID,
		Reason:    reason,
		Status:    lgpd.DeletionStatusPending,
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}, nil
}

func (s *extraLGPDService) ListDeletionRequests(_ context.Context) ([]lgpd.DeletionRequest, error) {
	return nil, nil
}

func (s *extraLGPDService) ProcessDeletion(_ context.Context, _, _, _ string) (lgpd.DeletionRequest, error) {
	return lgpd.DeletionRequest{}, nil
}

// ── helper: set auth context ──

func extraSetAuth(c *gin.Context, userID, role string) {
	c.Set("authUserID", userID)
	c.Set("authUserRole", role)
}

// ────────────────────────────────────────────────────────
// AdminHandler tests
// ────────────────────────────────────────────────────────

func TestAdminHandler_ListUsers_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	svc := &extraAdminService{
		users: []user.User{
			{ID: "u1", Name: "Alice", Email: "alice@test.com", Role: "user", Balance: 5000, CreatedAt: now},
			{ID: "u2", Name: "Bob", Email: "bob@test.com", Role: "admin", Balance: 10000, CreatedAt: now},
		},
	}
	handler := NewAdminHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	extraSetAuth(c, "admin1", "admin")

	handler.ListUsers(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}

	total, ok := body["total"].(float64)
	if !ok || int(total) != 2 {
		t.Fatalf("expected total=2, got %v", body["total"])
	}
}

func TestAdminHandler_GetUser_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	svc := &extraAdminService{
		userMap: map[string]user.User{
			"u1": {ID: "u1", Name: "Alice", Email: "alice@test.com", Role: "user", Balance: 5000, CreatedAt: now, UpdatedAt: now},
		},
	}
	handler := NewAdminHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/users/u1", nil)
	c.Params = gin.Params{{Key: "id", Value: "u1"}}

	handler.GetUser(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}

	if body["name"] != "Alice" {
		t.Fatalf("expected name=Alice, got %v", body["name"])
	}
}

func TestAdminHandler_GetUser_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &extraAdminService{
		userMap: map[string]user.User{},
	}
	handler := NewAdminHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/users/missing", nil)
	c.Params = gin.Params{{Key: "id", Value: "missing"}}

	handler.GetUser(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

// ────────────────────────────────────────────────────────
// InfovistHandler tests
// ────────────────────────────────────────────────────────

func TestInfovistHandler_ListInspections_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	svc := &extraInfovistService{
		inspections: []inspection.Inspection{
			{ID: "i1", UserID: "user1", Protocol: "P1", Customer: "John", Status: "pending", CreatedAt: now, UpdatedAt: now},
		},
	}
	handler := NewInfovistHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/inspections", nil)
	extraSetAuth(c, "user1", "user")

	handler.ListInspections(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}

	if len(body) != 1 {
		t.Fatalf("expected 1 inspection, got %d", len(body))
	}

	if body[0]["protocol"] != "P1" {
		t.Fatalf("expected protocol=P1, got %v", body[0]["protocol"])
	}
}

func TestInfovistHandler_ListInspections_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &extraInfovistService{}
	handler := NewInfovistHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/inspections", nil)
	// No auth set

	handler.ListInspections(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestInfovistHandler_CreateInspection_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &extraInfovistService{}
	handler := NewInfovistHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"customer":"Test","cellphone":"11999999999","plate":"ABC1234"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/inspections", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	// No auth set

	handler.CreateInspection(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

// ────────────────────────────────────────────────────────
// LGPDHandler tests
// ────────────────────────────────────────────────────────

func TestLGPDHandler_GetUserData_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &extraLGPDService{}
	handler := NewLGPDHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/users/user1/data", nil)
	c.Params = gin.Params{{Key: "id", Value: "user1"}}
	extraSetAuth(c, "user1", "user")

	handler.GetUserData(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}

	if body["id"] != "user1" {
		t.Fatalf("expected id=user1, got %v", body["id"])
	}
}

func TestLGPDHandler_GetUserData_WrongUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &extraLGPDService{}
	handler := NewLGPDHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/users/user2/data", nil)
	c.Params = gin.Params{{Key: "id", Value: "user2"}}
	extraSetAuth(c, "user1", "user") // different user

	handler.GetUserData(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", w.Code)
	}
}

func TestLGPDHandler_RequestDeletion_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &extraLGPDService{}
	handler := NewLGPDHandler(svc)

	reqBody := `{"reason":"I want my data deleted"}`

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/users/user1/data/deletion-request", bytes.NewBufferString(reqBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "user1"}}
	extraSetAuth(c, "user1", "user")

	handler.RequestDeletion(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}

	if body["request_id"] != "del-1" {
		t.Fatalf("expected request_id=del-1, got %v", body["request_id"])
	}

	if body["status"] != lgpd.DeletionStatusPending {
		t.Fatalf("expected status=pending, got %v", body["status"])
	}
}

// ────────────────────────────────────────────────────────
// AuthMiddleware tests
// ────────────────────────────────────────────────────────

func TestAuthMiddleware_Handler_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	provider := authinfra.NewMockProvider("Authorization")
	middleware := NewAuthMiddleware(provider, "Authorization")

	router := gin.New()
	router.Use(middleware.Handler())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	// No Authorization header
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_AdminOnly_NotAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	provider := authinfra.NewMockProvider("Authorization")
	middleware := NewAuthMiddleware(provider, "Authorization")

	router := gin.New()
	router.Use(middleware.Handler())
	router.Use(AdminOnly())
	router.GET("/admin/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/resource", nil)
	// Mock provider: "user1" without ":admin" suffix => role="user"
	req.Header.Set("Authorization", "user1")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", w.Code)
	}
}

func TestAuthMiddleware_AdminOnly_IsAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	provider := authinfra.NewMockProvider("Authorization")
	middleware := NewAuthMiddleware(provider, "Authorization")

	router := gin.New()
	router.Use(middleware.Handler())
	router.Use(AdminOnly())
	router.GET("/admin/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/resource", nil)
	// Mock provider: "admin1:admin" => role="admin"
	req.Header.Set("Authorization", "admin1:admin")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

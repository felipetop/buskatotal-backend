package memory

import (
	"context"
	"testing"
	"time"

	"buskatotal-backend/internal/domain/inspection"
	"buskatotal-backend/internal/domain/lgpd"
	"buskatotal-backend/internal/domain/payment"
	"buskatotal-backend/internal/domain/verification"
)

// --------------- Order Repository ---------------

func TestOrderRepo_CreateAndGetByID(t *testing.T) {
	repo := NewOrderRepository()
	ctx := context.Background()

	order := payment.Order{
		UserID:      "user-1",
		AmountCents: 1500,
		Status:      payment.StatusPending,
		ReferenceID: "ref-001",
	}

	created, err := repo.Create(ctx, order)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty ID after Create")
	}
	if created.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}

	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if got.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", got.UserID, "user-1")
	}
	if got.AmountCents != 1500 {
		t.Errorf("AmountCents = %d, want %d", got.AmountCents, 1500)
	}
}

func TestOrderRepo_GetByReferenceID(t *testing.T) {
	repo := NewOrderRepository()
	ctx := context.Background()

	order := payment.Order{
		UserID:      "user-1",
		AmountCents: 2000,
		Status:      payment.StatusPending,
		ReferenceID: "ref-unique",
	}

	created, err := repo.Create(ctx, order)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	got, err := repo.GetByReferenceID(ctx, "ref-unique")
	if err != nil {
		t.Fatalf("GetByReferenceID returned error: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
}

func TestOrderRepo_GetByReferenceID_NotFound(t *testing.T) {
	repo := NewOrderRepository()
	ctx := context.Background()

	_, err := repo.GetByReferenceID(ctx, "nonexistent-ref")
	if err == nil {
		t.Fatal("expected error for nonexistent reference ID, got nil")
	}
}

func TestOrderRepo_GetByUserID(t *testing.T) {
	repo := NewOrderRepository()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		repo.Create(ctx, payment.Order{UserID: "user-A", Status: payment.StatusPending, ReferenceID: "ref"})
	}
	repo.Create(ctx, payment.Order{UserID: "user-B", Status: payment.StatusPending, ReferenceID: "ref-b"})

	orders, err := repo.GetByUserID(ctx, "user-A")
	if err != nil {
		t.Fatalf("GetByUserID returned error: %v", err)
	}
	if len(orders) != 3 {
		t.Errorf("len(orders) = %d, want 3", len(orders))
	}
}

func TestOrderRepo_GetPendingOrders(t *testing.T) {
	repo := NewOrderRepository()
	ctx := context.Background()

	repo.Create(ctx, payment.Order{UserID: "u1", Status: payment.StatusPending, ReferenceID: "r1"})
	repo.Create(ctx, payment.Order{UserID: "u2", Status: payment.StatusPending, ReferenceID: "r2"})

	paid, _ := repo.Create(ctx, payment.Order{UserID: "u3", Status: payment.StatusPending, ReferenceID: "r3"})
	paid.Status = payment.StatusPaid
	repo.Update(ctx, paid)

	pending, err := repo.GetPendingOrders(ctx)
	if err != nil {
		t.Fatalf("GetPendingOrders returned error: %v", err)
	}
	if len(pending) != 2 {
		t.Errorf("len(pending) = %d, want 2", len(pending))
	}
	for _, o := range pending {
		if o.Status != payment.StatusPending {
			t.Errorf("expected StatusPending, got %q", o.Status)
		}
	}
}

func TestOrderRepo_Update(t *testing.T) {
	repo := NewOrderRepository()
	ctx := context.Background()

	created, _ := repo.Create(ctx, payment.Order{
		UserID:      "user-1",
		AmountCents: 1000,
		Status:      payment.StatusPending,
		ReferenceID: "ref-upd",
	})

	created.Status = payment.StatusPaid
	updated, err := repo.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Status != payment.StatusPaid {
		t.Errorf("Status = %q, want %q", updated.Status, payment.StatusPaid)
	}
	if !updated.UpdatedAt.After(created.CreatedAt) || updated.UpdatedAt.Equal(created.CreatedAt) {
		// UpdatedAt should be at or after CreatedAt
	}

	got, _ := repo.GetByID(ctx, created.ID)
	if got.Status != payment.StatusPaid {
		t.Errorf("persisted Status = %q, want %q", got.Status, payment.StatusPaid)
	}
}

// --------------- Inspection Repository ---------------

func TestInspectionRepo_CreateAndGetByID(t *testing.T) {
	repo := NewInspectionRepository()
	ctx := context.Background()

	insp := inspection.Inspection{
		UserID:   "user-1",
		Protocol: "PROT-001",
		Customer: "John Doe",
		Plate:    "ABC1234",
		Status:   "pending",
	}

	created, err := repo.Create(ctx, insp)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty ID")
	}

	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if got.Protocol != "PROT-001" {
		t.Errorf("Protocol = %q, want %q", got.Protocol, "PROT-001")
	}
	if got.Customer != "John Doe" {
		t.Errorf("Customer = %q, want %q", got.Customer, "John Doe")
	}
}

func TestInspectionRepo_GetByProtocol(t *testing.T) {
	repo := NewInspectionRepository()
	ctx := context.Background()

	repo.Create(ctx, inspection.Inspection{
		UserID:   "user-1",
		Protocol: "PROT-FIND",
		Customer: "Jane",
		Plate:    "XYZ9999",
		Status:   "pending",
	})

	got, err := repo.GetByProtocol(ctx, "PROT-FIND")
	if err != nil {
		t.Fatalf("GetByProtocol returned error: %v", err)
	}
	if got.Plate != "XYZ9999" {
		t.Errorf("Plate = %q, want %q", got.Plate, "XYZ9999")
	}
}

func TestInspectionRepo_GetByProtocol_NotFound(t *testing.T) {
	repo := NewInspectionRepository()
	ctx := context.Background()

	_, err := repo.GetByProtocol(ctx, "NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for nonexistent protocol, got nil")
	}
}

func TestInspectionRepo_GetByUserID(t *testing.T) {
	repo := NewInspectionRepository()
	ctx := context.Background()

	repo.Create(ctx, inspection.Inspection{UserID: "u1", Protocol: "P1", Status: "pending"})
	repo.Create(ctx, inspection.Inspection{UserID: "u1", Protocol: "P2", Status: "pending"})
	repo.Create(ctx, inspection.Inspection{UserID: "u2", Protocol: "P3", Status: "pending"})

	result, err := repo.GetByUserID(ctx, "u1")
	if err != nil {
		t.Fatalf("GetByUserID returned error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("len = %d, want 2", len(result))
	}
}

func TestInspectionRepo_Update(t *testing.T) {
	repo := NewInspectionRepository()
	ctx := context.Background()

	created, _ := repo.Create(ctx, inspection.Inspection{
		UserID:   "user-1",
		Protocol: "PROT-UPD",
		Status:   "pending",
	})

	created.Status = "completed"
	updated, err := repo.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Status != "completed" {
		t.Errorf("Status = %q, want %q", updated.Status, "completed")
	}

	got, _ := repo.GetByID(ctx, created.ID)
	if got.Status != "completed" {
		t.Errorf("persisted Status = %q, want %q", got.Status, "completed")
	}
}

// --------------- Deletion Repository ---------------

func TestDeletionRepo_CreateAndGetByID(t *testing.T) {
	repo := NewDeletionRepository()
	ctx := context.Background()

	req := lgpd.DeletionRequest{
		UserID:    "user-1",
		UserEmail: "user@example.com",
		UserName:  "Test User",
		Reason:    "privacy concern",
		Status:    lgpd.DeletionStatusPending,
	}

	created, err := repo.Create(ctx, req)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if created.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}

	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if got.UserEmail != "user@example.com" {
		t.Errorf("UserEmail = %q, want %q", got.UserEmail, "user@example.com")
	}
	if got.Status != lgpd.DeletionStatusPending {
		t.Errorf("Status = %q, want %q", got.Status, lgpd.DeletionStatusPending)
	}
}

func TestDeletionRepo_GetByUserID(t *testing.T) {
	repo := NewDeletionRepository()
	ctx := context.Background()

	repo.Create(ctx, lgpd.DeletionRequest{UserID: "u1", Status: lgpd.DeletionStatusPending})
	repo.Create(ctx, lgpd.DeletionRequest{UserID: "u1", Status: lgpd.DeletionStatusCompleted})
	repo.Create(ctx, lgpd.DeletionRequest{UserID: "u2", Status: lgpd.DeletionStatusPending})

	result, err := repo.GetByUserID(ctx, "u1")
	if err != nil {
		t.Fatalf("GetByUserID returned error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("len = %d, want 2", len(result))
	}
}

func TestDeletionRepo_List(t *testing.T) {
	repo := NewDeletionRepository()
	ctx := context.Background()

	repo.Create(ctx, lgpd.DeletionRequest{UserID: "u1", Status: lgpd.DeletionStatusPending})
	repo.Create(ctx, lgpd.DeletionRequest{UserID: "u2", Status: lgpd.DeletionStatusProcessing})
	repo.Create(ctx, lgpd.DeletionRequest{UserID: "u3", Status: lgpd.DeletionStatusCompleted})

	all, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("len = %d, want 3", len(all))
	}
}

func TestDeletionRepo_Update(t *testing.T) {
	repo := NewDeletionRepository()
	ctx := context.Background()

	created, _ := repo.Create(ctx, lgpd.DeletionRequest{
		UserID: "user-1",
		Status: lgpd.DeletionStatusPending,
	})

	created.Status = lgpd.DeletionStatusCompleted
	updated, err := repo.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Status != lgpd.DeletionStatusCompleted {
		t.Errorf("Status = %q, want %q", updated.Status, lgpd.DeletionStatusCompleted)
	}

	got, _ := repo.GetByID(ctx, created.ID)
	if got.Status != lgpd.DeletionStatusCompleted {
		t.Errorf("persisted Status = %q, want %q", got.Status, lgpd.DeletionStatusCompleted)
	}
}

// --------------- Log Repository ---------------

func TestLogRepo_CreateAndGetByUserID(t *testing.T) {
	repo := NewLogRepository()
	ctx := context.Background()

	log1 := lgpd.DataProcessingLog{
		UserID: "user-1",
		Action: "data_export",
		Details: map[string]interface{}{
			"format": "json",
		},
		IPAddress: "192.168.1.1",
	}
	log2 := lgpd.DataProcessingLog{
		UserID: "user-1",
		Action: "consent_update",
	}
	log3 := lgpd.DataProcessingLog{
		UserID: "user-2",
		Action: "data_export",
	}

	if err := repo.Create(ctx, log1); err != nil {
		t.Fatalf("Create log1 returned error: %v", err)
	}
	if err := repo.Create(ctx, log2); err != nil {
		t.Fatalf("Create log2 returned error: %v", err)
	}
	if err := repo.Create(ctx, log3); err != nil {
		t.Fatalf("Create log3 returned error: %v", err)
	}

	logs, err := repo.GetByUserID(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetByUserID returned error: %v", err)
	}
	if len(logs) != 2 {
		t.Errorf("len = %d, want 2", len(logs))
	}
	for _, l := range logs {
		if l.UserID != "user-1" {
			t.Errorf("UserID = %q, want %q", l.UserID, "user-1")
		}
	}

	// verify empty result for unknown user
	empty, err := repo.GetByUserID(ctx, "unknown")
	if err != nil {
		t.Fatalf("GetByUserID for unknown returned error: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected empty slice, got len %d", len(empty))
	}
}

// --------------- Verification Repository ---------------

func TestVerificationRepo_CreateAndGetByToken(t *testing.T) {
	repo := NewVerificationRepository()
	ctx := context.Background()

	tok := verification.Token{
		UserID:    "user-1",
		Token:     "abc123token",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	created, err := repo.Create(ctx, tok)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty ID")
	}

	got, err := repo.GetByToken(ctx, "abc123token")
	if err != nil {
		t.Fatalf("GetByToken returned error: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
	if got.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", got.UserID, "user-1")
	}
	if got.Used {
		t.Error("expected Used to be false")
	}

	// not found case
	_, err = repo.GetByToken(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent token, got nil")
	}
}

func TestVerificationRepo_MarkUsed(t *testing.T) {
	repo := NewVerificationRepository()
	ctx := context.Background()

	created, _ := repo.Create(ctx, verification.Token{
		UserID:    "user-1",
		Token:     "mark-me",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})

	err := repo.MarkUsed(ctx, created.ID)
	if err != nil {
		t.Fatalf("MarkUsed returned error: %v", err)
	}

	got, _ := repo.GetByToken(ctx, "mark-me")
	if !got.Used {
		t.Error("expected Used to be true after MarkUsed")
	}

	// mark nonexistent
	err = repo.MarkUsed(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent ID, got nil")
	}
}

func TestVerificationRepo_DeleteByUserID(t *testing.T) {
	repo := NewVerificationRepository()
	ctx := context.Background()

	repo.Create(ctx, verification.Token{UserID: "user-del", Token: "t1", ExpiresAt: time.Now().Add(time.Hour)})
	repo.Create(ctx, verification.Token{UserID: "user-del", Token: "t2", ExpiresAt: time.Now().Add(time.Hour)})
	repo.Create(ctx, verification.Token{UserID: "user-keep", Token: "t3", ExpiresAt: time.Now().Add(time.Hour)})

	err := repo.DeleteByUserID(ctx, "user-del")
	if err != nil {
		t.Fatalf("DeleteByUserID returned error: %v", err)
	}

	_, err = repo.GetByToken(ctx, "t1")
	if err == nil {
		t.Error("expected t1 to be deleted")
	}
	_, err = repo.GetByToken(ctx, "t2")
	if err == nil {
		t.Error("expected t2 to be deleted")
	}

	got, err := repo.GetByToken(ctx, "t3")
	if err != nil {
		t.Fatalf("expected t3 to still exist, got error: %v", err)
	}
	if got.UserID != "user-keep" {
		t.Errorf("UserID = %q, want %q", got.UserID, "user-keep")
	}
}

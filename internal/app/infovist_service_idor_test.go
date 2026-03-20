package app

import (
	"context"
	"testing"

	"buskatotal-backend/internal/domain/inspection"
	"buskatotal-backend/internal/domain/user"
	"buskatotal-backend/internal/infra/infovist"
	"buskatotal-backend/internal/infra/memory"
)

func setupInfovistIDORTest(t *testing.T) (*InfovistService, *memory.UserRepository, *memory.InspectionRepository, user.User, user.User) {
	t.Helper()
	userRepo := memory.NewUserRepository()
	inspRepo := memory.NewInspectionRepository()

	userA, err := userRepo.Create(context.Background(), user.User{
		Name: "User A", Email: "a@test.com", Balance: 100000,
	})
	if err != nil {
		t.Fatalf("create user A: %v", err)
	}

	userB, err := userRepo.Create(context.Background(), user.User{
		Name: "User B", Email: "b@test.com", Balance: 100000,
	})
	if err != nil {
		t.Fatalf("create user B: %v", err)
	}

	client := infovist.NewClient("http://localhost:9999", "test@test.com", "pass", "token")
	svc := NewInfovistService(client, userRepo, inspRepo, 100, 0)

	return svc, userRepo, inspRepo, userA, userB
}

func TestViewInspection_IDOR_Blocked(t *testing.T) {
	svc, _, inspRepo, userA, userB := setupInfovistIDORTest(t)

	// Create inspection owned by User A
	inspRepo.Create(context.Background(), inspection.Inspection{
		UserID:   userA.ID,
		Protocol: "PROTO-123",
		Customer: "Test",
		Plate:    "ABC1234",
		Status:   "AWAITING_TO_SEND",
	})

	// User B tries to view User A's inspection
	_, err := svc.ViewInspection(context.Background(), userB.ID, "PROTO-123")
	if err == nil {
		t.Error("User B should NOT be able to view User A's inspection")
	}
}

func TestViewInspection_Owner_Allowed(t *testing.T) {
	svc, _, inspRepo, userA, _ := setupInfovistIDORTest(t)

	inspRepo.Create(context.Background(), inspection.Inspection{
		UserID:   userA.ID,
		Protocol: "PROTO-456",
		Customer: "Test",
		Plate:    "ABC1234",
		Status:   "AWAITING_TO_SEND",
	})

	// User A views their own inspection — will fail on API call but should pass ownership check
	_, err := svc.ViewInspection(context.Background(), userA.ID, "PROTO-456")
	if err != nil && err.Error() == "inspection not found" {
		t.Error("Owner should pass ownership check")
	}
}

func TestGetReportV1_IDOR_Blocked(t *testing.T) {
	svc, _, inspRepo, userA, userB := setupInfovistIDORTest(t)

	inspRepo.Create(context.Background(), inspection.Inspection{
		UserID:   userA.ID,
		Protocol: "PROTO-789",
		Customer: "Test",
		Plate:    "ABC1234",
		Status:   "COMPLETED",
	})

	_, err := svc.GetReportV1(context.Background(), userB.ID, "PROTO-789")
	if err == nil {
		t.Error("User B should NOT be able to get User A's report")
	}
	if err != nil && err.Error() != "inspection not found" {
		t.Errorf("expected 'inspection not found', got: %v", err)
	}
}

func TestGetReportV2_IDOR_Blocked(t *testing.T) {
	svc, _, inspRepo, userA, userB := setupInfovistIDORTest(t)

	inspRepo.Create(context.Background(), inspection.Inspection{
		UserID:   userA.ID,
		Protocol: "PROTO-101",
		Customer: "Test",
		Plate:    "ABC1234",
		Status:   "COMPLETED",
	})

	_, err := svc.GetReportV2(context.Background(), userB.ID, "PROTO-101")
	if err == nil {
		t.Error("User B should NOT be able to get User A's report v2")
	}
	if err != nil && err.Error() != "inspection not found" {
		t.Errorf("expected 'inspection not found', got: %v", err)
	}
}

func TestListInspections_OnlyOwn(t *testing.T) {
	svc, _, inspRepo, userA, userB := setupInfovistIDORTest(t)

	inspRepo.Create(context.Background(), inspection.Inspection{
		UserID: userA.ID, Protocol: "A-001", Customer: "Test A", Status: "COMPLETED",
	})
	inspRepo.Create(context.Background(), inspection.Inspection{
		UserID: userB.ID, Protocol: "B-001", Customer: "Test B", Status: "COMPLETED",
	})

	listA, err := svc.ListInspections(context.Background(), userA.ID)
	if err != nil {
		t.Fatalf("list A: %v", err)
	}
	for _, insp := range listA {
		if insp.UserID != userA.ID {
			t.Errorf("User A's list contains inspection from user %s", insp.UserID)
		}
	}

	listB, err := svc.ListInspections(context.Background(), userB.ID)
	if err != nil {
		t.Fatalf("list B: %v", err)
	}
	for _, insp := range listB {
		if insp.UserID != userB.ID {
			t.Errorf("User B's list contains inspection from user %s", insp.UserID)
		}
	}
}

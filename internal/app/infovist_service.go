package app

import (
	"context"
	"errors"
	"sync"
	"time"

	"buskatotal-backend/internal/domain/inspection"
	"buskatotal-backend/internal/domain/user"
	"buskatotal-backend/internal/infra/infovist"
)

type InfovistService struct {
	client               *infovist.Client
	userRepo             user.Repository
	inspRepo             inspection.Repository
	costCreateInspection int64
	costReport           int64
	mu                   sync.Mutex
	token                string
	expiry               time.Time
}

func NewInfovistService(client *infovist.Client, userRepo user.Repository, inspRepo inspection.Repository, costCreateInspection, costReport int64) *InfovistService {
	return &InfovistService{
		client:               client,
		userRepo:             userRepo,
		inspRepo:             inspRepo,
		costCreateInspection: costCreateInspection,
		costReport:           costReport,
	}
}

func (s *InfovistService) CreateInspection(ctx context.Context, userID string, input infovist.CreateInspectionRequest) (*infovist.CreateInspectionResponse, error) {
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	if input.Customer == "" {
		return nil, errors.New("customer is required")
	}
	if input.Cellphone == "" {
		return nil, errors.New("cellphone is required")
	}
	if input.Plate == "" && input.Chassis == "" {
		return nil, errors.New("plate or chassis is required")
	}

	if err := s.userRepo.DebitBalance(ctx, userID, s.costCreateInspection); err != nil {
		return nil, err
	}

	token, err := s.getToken(ctx)
	if err != nil {
		s.userRepo.CreditBalance(ctx, userID, s.costCreateInspection)
		return nil, err
	}

	result, err := s.client.CreateInspection(ctx, token, input)
	if err != nil {
		s.userRepo.CreditBalance(ctx, userID, s.costCreateInspection)
		return nil, err
	}

	// Save to local history
	s.inspRepo.Create(ctx, inspection.Inspection{
		UserID:    userID,
		Protocol:  result.Protocol,
		Customer:  input.Customer,
		Cellphone: input.Cellphone,
		Plate:     input.Plate,
		Chassis:   input.Chassis,
		Notes:     input.Notes,
		Status:    "AWAITING_TO_SEND",
	})

	return result, nil
}

func (s *InfovistService) ViewInspection(ctx context.Context, userID, protocol string) (*infovist.ViewInspectionResponse, error) {
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	if protocol == "" {
		return nil, errors.New("protocol is required")
	}

	// Verify ownership — reject if not found or wrong user
	insp, inspErr := s.inspRepo.GetByProtocol(ctx, protocol)
	if inspErr != nil {
		return nil, errors.New("inspection not found")
	}
	if insp.UserID != userID {
		return nil, errors.New("inspection not found")
	}

	token, err := s.getToken(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.ViewInspection(ctx, token, protocol)
	if err != nil {
		return nil, err
	}

	// Update local status if we have this inspection
	if len(resp.Statuses) > 0 {
		latest := resp.Statuses[len(resp.Statuses)-1]
		if insp, findErr := s.inspRepo.GetByProtocol(ctx, protocol); findErr == nil {
			insp.Status = latest.StatusEnum
			s.inspRepo.Update(ctx, insp)
		}
	}

	return resp, nil
}

func (s *InfovistService) GetReportV1(ctx context.Context, userID, protocol string) (*infovist.ReportResponse, error) {
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	if protocol == "" {
		return nil, errors.New("protocol is required")
	}

	// Verify ownership — reject if not found or wrong user
	insp, inspErr := s.inspRepo.GetByProtocol(ctx, protocol)
	if inspErr != nil {
		return nil, errors.New("inspection not found")
	}
	if insp.UserID != userID {
		return nil, errors.New("inspection not found")
	}

	token, err := s.getToken(ctx)
	if err != nil {
		return nil, err
	}

	return s.client.GetReportV1(ctx, token, protocol)
}

func (s *InfovistService) GetReportV2(ctx context.Context, userID, protocol string) (*infovist.ReportV2Response, error) {
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	if protocol == "" {
		return nil, errors.New("protocol is required")
	}

	// Verify ownership — reject if not found or wrong user
	insp2, inspErr2 := s.inspRepo.GetByProtocol(ctx, protocol)
	if inspErr2 != nil {
		return nil, errors.New("inspection not found")
	}
	if insp2.UserID != userID {
		return nil, errors.New("inspection not found")
	}

	token, err := s.getToken(ctx)
	if err != nil {
		return nil, err
	}

	return s.client.GetReportV2(ctx, token, protocol)
}

func (s *InfovistService) ListInspections(ctx context.Context, userID string) ([]inspection.Inspection, error) {
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	return s.inspRepo.GetByUserID(ctx, userID)
}

func (s *InfovistService) getToken(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token != "" && time.Now().Before(s.expiry) {
		return s.token, nil
	}

	authResp, err := s.client.Authenticate(ctx)
	if err != nil {
		return "", err
	}

	s.token = authResp.AccessToken
	if authResp.ExpiresIn > 0 {
		s.expiry = time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second).Add(-5 * time.Minute)
	} else {
		s.expiry = time.Now().Add(50 * time.Minute)
	}

	return s.token, nil
}

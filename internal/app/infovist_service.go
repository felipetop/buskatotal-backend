package app

import (
	"context"
	"errors"
	"sync"
	"time"

	"buskatotal-backend/internal/domain/user"
	"buskatotal-backend/internal/infra/infovist"
)

type InfovistService struct {
	client              *infovist.Client
	userRepo            user.Repository
	costCreateInspection int64 // VISTORIA DIGITAL: custo de venda em centavos
	costReport           int64 // INFOVIST: custo de venda em centavos
	mu                   sync.Mutex
	token                string
	expiry               time.Time
}

func NewInfovistService(client *infovist.Client, userRepo user.Repository, costCreateInspection, costReport int64) *InfovistService {
	return &InfovistService{
		client:              client,
		userRepo:            userRepo,
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

	if err := s.debitBalance(ctx, userID, s.costCreateInspection); err != nil {
		return nil, err
	}

	token, err := s.getToken(ctx)
	if err != nil {
		return nil, err
	}

	return s.client.CreateInspection(ctx, token, input)
}

func (s *InfovistService) ViewInspection(ctx context.Context, userID, protocol string) (*infovist.ViewInspectionResponse, error) {
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	if protocol == "" {
		return nil, errors.New("protocol is required")
	}

	token, err := s.getToken(ctx)
	if err != nil {
		return nil, err
	}

	return s.client.ViewInspection(ctx, token, protocol)
}

func (s *InfovistService) GetReportV1(ctx context.Context, userID, protocol string) (*infovist.ReportResponse, error) {
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	if protocol == "" {
		return nil, errors.New("protocol is required")
	}

	if err := s.debitBalance(ctx, userID, s.costReport); err != nil {
		return nil, err
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

	if err := s.debitBalance(ctx, userID, s.costReport); err != nil {
		return nil, err
	}

	token, err := s.getToken(ctx)
	if err != nil {
		return nil, err
	}

	return s.client.GetReportV2(ctx, token, protocol)
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
	// Token expiry based on expires_in from API, with 5 min safety margin.
	if authResp.ExpiresIn > 0 {
		s.expiry = time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second).Add(-5 * time.Minute)
	} else {
		s.expiry = time.Now().Add(50 * time.Minute)
	}

	return s.token, nil
}

func (s *InfovistService) debitBalance(ctx context.Context, userID string, cost int64) error {
	if s.userRepo == nil {
		return errors.New("user repository not configured")
	}

	userEntity, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if userEntity.Balance < cost {
		return errors.New("insufficient balance")
	}

	userEntity.Balance -= cost
	if _, err := s.userRepo.Update(ctx, userEntity); err != nil {
		return err
	}
	return nil
}

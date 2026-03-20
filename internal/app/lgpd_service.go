package app

import (
	"context"
	"errors"
	"time"

	"buskatotal-backend/internal/domain/email"
	"buskatotal-backend/internal/domain/inspection"
	"buskatotal-backend/internal/domain/lgpd"
	"buskatotal-backend/internal/domain/payment"
	"buskatotal-backend/internal/domain/user"
)

const dpoEmail = "karinbelan43@gmail.com"

type LGPDService struct {
	userRepo     user.Repository
	inspRepo     inspection.Repository
	orderRepo    payment.OrderRepository
	deletionRepo lgpd.DeletionRepository
	logRepo      lgpd.LogRepository
	emailSender  email.Sender
}

func NewLGPDService(
	userRepo user.Repository,
	inspRepo inspection.Repository,
	orderRepo payment.OrderRepository,
	deletionRepo lgpd.DeletionRepository,
	logRepo lgpd.LogRepository,
	emailSender email.Sender,
) *LGPDService {
	return &LGPDService{
		userRepo:     userRepo,
		inspRepo:     inspRepo,
		orderRepo:    orderRepo,
		deletionRepo: deletionRepo,
		logRepo:      logRepo,
		emailSender:  emailSender,
	}
}

type UserDataResponse struct {
	User struct {
		ID              string     `json:"id"`
		Name            string     `json:"name"`
		Email           string     `json:"email"`
		CreatedAt       time.Time  `json:"created_at"`
		AcceptedTermsAt *time.Time `json:"accepted_terms_at,omitempty"`
		EmailVerified   bool       `json:"email_verified"`
	} `json:"user"`
	ConsultasRealizadas int   `json:"consultas_realizadas"`
	VistoriasRealizadas int   `json:"vistorias_realizadas"`
	SaldoCents          int64 `json:"saldo_cents"`
}

func (s *LGPDService) GetUserData(ctx context.Context, userID string) (any, error) {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	inspections, _ := s.inspRepo.GetByUserID(ctx, userID)
	orders, _ := s.orderRepo.GetByUserID(ctx, userID)

	// Count paid orders as "consultas" approximation
	paidOrders := 0
	for _, o := range orders {
		if o.Status == payment.StatusPaid {
			paidOrders++
		}
	}

	var resp UserDataResponse
	resp.User.ID = u.ID
	resp.User.Name = u.Name
	resp.User.Email = u.Email
	resp.User.CreatedAt = u.CreatedAt
	resp.User.AcceptedTermsAt = u.AcceptedTermsAt
	resp.User.EmailVerified = u.EmailVerified
	resp.VistoriasRealizadas = len(inspections)
	resp.ConsultasRealizadas = paidOrders
	resp.SaldoCents = u.Balance

	s.logAction(ctx, userID, "data_access", nil)

	return resp, nil
}

type ExportResponse struct {
	Usuario struct {
		Nome         string     `json:"nome"`
		Email        string     `json:"email"`
		DataCadastro time.Time  `json:"data_cadastro"`
		AceiteTermos *time.Time `json:"aceite_termos,omitempty"`
	} `json:"usuario"`
	HistoricoVistorias  []ExportInspection `json:"historico_vistorias"`
	HistoricoPagamentos []ExportPayment    `json:"historico_pagamentos"`
}

type ExportInspection struct {
	Data      time.Time `json:"data"`
	Protocolo string    `json:"protocolo"`
	Placa     string    `json:"placa,omitempty"`
	Status    string    `json:"status"`
}

type ExportPayment struct {
	Data       time.Time `json:"data"`
	ValorCents int64     `json:"valor_cents"`
	Status     string    `json:"status"`
}

func (s *LGPDService) ExportUserData(ctx context.Context, userID string) (any, error) {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	var resp ExportResponse
	resp.Usuario.Nome = u.Name
	resp.Usuario.Email = u.Email
	resp.Usuario.DataCadastro = u.CreatedAt
	resp.Usuario.AceiteTermos = u.AcceptedTermsAt

	inspections, _ := s.inspRepo.GetByUserID(ctx, userID)
	resp.HistoricoVistorias = make([]ExportInspection, 0, len(inspections))
	for _, insp := range inspections {
		resp.HistoricoVistorias = append(resp.HistoricoVistorias, ExportInspection{
			Data:      insp.CreatedAt,
			Protocolo: insp.Protocol,
			Placa:     insp.Plate,
			Status:    insp.Status,
		})
	}

	orders, _ := s.orderRepo.GetByUserID(ctx, userID)
	resp.HistoricoPagamentos = make([]ExportPayment, 0, len(orders))
	for _, order := range orders {
		resp.HistoricoPagamentos = append(resp.HistoricoPagamentos, ExportPayment{
			Data:       order.CreatedAt,
			ValorCents: order.AmountCents,
			Status:     string(order.Status),
		})
	}

	s.logAction(ctx, userID, "data_export", nil)

	return resp, nil
}

func (s *LGPDService) RequestDeletion(ctx context.Context, userID, reason string) (lgpd.DeletionRequest, error) {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return lgpd.DeletionRequest{}, errors.New("user not found")
	}

	// Check for existing pending request
	existing, _ := s.deletionRepo.GetByUserID(ctx, userID)
	for _, req := range existing {
		if req.Status == lgpd.DeletionStatusPending || req.Status == lgpd.DeletionStatusProcessing {
			return lgpd.DeletionRequest{}, errors.New("deletion request already pending")
		}
	}

	req, err := s.deletionRepo.Create(ctx, lgpd.DeletionRequest{
		UserID:    userID,
		UserEmail: u.Email,
		UserName:  u.Name,
		Reason:    reason,
		Status:    lgpd.DeletionStatusPending,
	})
	if err != nil {
		return lgpd.DeletionRequest{}, err
	}

	// Notify DPO via email
	if s.emailSender != nil {
		go func() {
			_ = s.emailSender.Send(context.Background(), email.Message{
				From:    emailFrom,
				To:      dpoEmail,
				Subject: "Solicitação de exclusão de dados — BuskaTotal",
				HTML:    buildDeletionNotifyHTML(u.Name, u.Email, reason),
			})
		}()

		// Confirm to user
		go func() {
			_ = s.emailSender.Send(context.Background(), email.Message{
				From:    emailFrom,
				To:      u.Email,
				Subject: "Solicitação de exclusão recebida — BuskaTotal",
				HTML:    buildDeletionConfirmHTML(u.Name),
			})
		}()
	}

	s.logAction(ctx, userID, "deletion_request", map[string]interface{}{"reason": reason})

	return req, nil
}

// ListDeletionRequests returns all deletion requests (admin use).
func (s *LGPDService) ListDeletionRequests(ctx context.Context) ([]lgpd.DeletionRequest, error) {
	return s.deletionRepo.List(ctx)
}

// ProcessDeletion updates a deletion request status (admin use).
func (s *LGPDService) ProcessDeletion(ctx context.Context, requestID, status, adminID string) (lgpd.DeletionRequest, error) {
	req, err := s.deletionRepo.GetByID(ctx, requestID)
	if err != nil {
		return lgpd.DeletionRequest{}, errors.New("deletion request not found")
	}

	if req.Status != lgpd.DeletionStatusPending && req.Status != lgpd.DeletionStatusProcessing {
		return lgpd.DeletionRequest{}, errors.New("deletion request already processed")
	}

	now := time.Now()
	req.Status = status
	req.ProcessedAt = &now
	req.ProcessedBy = adminID

	if status == lgpd.DeletionStatusCompleted {
		// Anonymize user data
		u, err := s.userRepo.GetByID(ctx, req.UserID)
		if err == nil {
			u.Name = "Usuário removido"
			u.Email = "removed-" + u.ID + "@removed.buskatotal.com.br"
			u.PasswordHash = ""
			u.Balance = 0
			s.userRepo.Update(ctx, u)
		}
	}

	updated, err := s.deletionRepo.Update(ctx, req)
	if err != nil {
		return lgpd.DeletionRequest{}, err
	}

	s.logAction(ctx, req.UserID, "deletion_processed", map[string]interface{}{
		"status":   status,
		"admin_id": adminID,
	})

	return updated, nil
}

func (s *LGPDService) LogAction(ctx context.Context, userID, action string, details map[string]interface{}) {
	s.logAction(ctx, userID, action, details)
}

func (s *LGPDService) logAction(ctx context.Context, userID, action string, details map[string]interface{}) {
	_ = s.logRepo.Create(ctx, lgpd.DataProcessingLog{
		UserID:  userID,
		Action:  action,
		Details: details,
	})
}

func buildDeletionNotifyHTML(name, userEmail, reason string) string {
	return `<!DOCTYPE html>
<html lang="pt-BR">
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; padding: 20px;">
  <h2>Solicitação de Exclusão de Dados</h2>
  <p><strong>Usuário:</strong> ` + name + ` (` + userEmail + `)</p>
  <p><strong>Motivo:</strong> ` + reason + `</p>
  <p>Prazo legal: 15 dias úteis (Art. 18 LGPD).</p>
  <p>Acesse o painel admin para processar.</p>
</body>
</html>`
}

func buildDeletionConfirmHTML(name string) string {
	return `<!DOCTYPE html>
<html lang="pt-BR">
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; background-color: #f4f4f4; padding: 20px;">
  <div style="max-width: 520px; margin: 0 auto; background: #ffffff; border-radius: 8px; padding: 40px; text-align: center;">
    <h1 style="color: #1a1a2e; margin-bottom: 8px;">BuskaTotal</h1>
    <p style="color: #555; font-size: 16px;">
      Olá ` + name + `, sua solicitação de exclusão de dados foi registrada.
    </p>
    <p style="color: #555; font-size: 16px;">
      Seus dados serão removidos em até <strong>15 dias úteis</strong> conforme a LGPD.
    </p>
    <p style="color: #999; font-size: 13px; margin-top: 32px;">
      Se você não fez esta solicitação, entre em contato: karinbelan43@gmail.com
    </p>
  </div>
</body>
</html>`
}

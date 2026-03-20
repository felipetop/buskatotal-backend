package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"buskatotal-backend/internal/domain/email"
	"buskatotal-backend/internal/domain/user"
	"buskatotal-backend/internal/domain/verification"
)

const (
	tokenBytes    = 32 // 256-bit token
	tokenTTLHours = 24
	emailFrom     = "BuskaTotal <no-reply@buskatotal.com.br>"
	verifyBaseURL = "https://buskatotal.com.br/verify"
)

type EmailVerificationService struct {
	verificationRepo verification.Repository
	userRepo         user.Repository
	emailSender      email.Sender
}

func NewEmailVerificationService(
	verificationRepo verification.Repository,
	userRepo user.Repository,
	emailSender email.Sender,
) *EmailVerificationService {
	return &EmailVerificationService{
		verificationRepo: verificationRepo,
		userRepo:         userRepo,
		emailSender:      emailSender,
	}
}

// GenerateAndSend creates a secure verification token and sends the verification email.
func (s *EmailVerificationService) GenerateAndSend(ctx context.Context, userID, userEmail string) error {
	// Delete any existing tokens for this user
	if err := s.verificationRepo.DeleteByUserID(ctx, userID); err != nil {
		log.Printf("email_verification: failed to delete old tokens for user %s: %v", userID, err)
	}

	// Generate cryptographically secure token
	tokenStr, err := generateSecureToken()
	if err != nil {
		return fmt.Errorf("email_verification: generate token: %w", err)
	}

	token := verification.Token{
		UserID:    userID,
		Token:     tokenStr,
		ExpiresAt: time.Now().Add(tokenTTLHours * time.Hour),
	}

	if _, err := s.verificationRepo.Create(ctx, token); err != nil {
		return fmt.Errorf("email_verification: save token: %w", err)
	}

	verifyLink := fmt.Sprintf("%s?token=%s", verifyBaseURL, tokenStr)

	msg := email.Message{
		From:    emailFrom,
		To:      userEmail,
		Subject: "Confirme seu e-mail — BuskaTotal",
		HTML:    buildVerificationHTML(verifyLink),
	}

	if err := s.emailSender.Send(ctx, msg); err != nil {
		return fmt.Errorf("email_verification: send email: %w", err)
	}

	return nil
}

// Verify validates the token and marks the user's email as verified.
func (s *EmailVerificationService) Verify(ctx context.Context, tokenStr string) error {
	token, err := s.verificationRepo.GetByToken(ctx, tokenStr)
	if err != nil {
		return verification.ErrTokenNotFound
	}

	if token.Used {
		return verification.ErrTokenUsed
	}

	if token.IsExpired() {
		return verification.ErrTokenExpired
	}

	// Mark token as used
	if err := s.verificationRepo.MarkUsed(ctx, token.ID); err != nil {
		return fmt.Errorf("email_verification: mark used: %w", err)
	}

	// Update user's email verification status
	u, err := s.userRepo.GetByID(ctx, token.UserID)
	if err != nil {
		return fmt.Errorf("email_verification: user not found: %w", err)
	}

	u.EmailVerified = true
	if _, err := s.userRepo.Update(ctx, u); err != nil {
		return fmt.Errorf("email_verification: update user: %w", err)
	}

	return nil
}

func generateSecureToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func buildVerificationHTML(link string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="pt-BR">
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; background-color: #f4f4f4; padding: 20px;">
  <div style="max-width: 520px; margin: 0 auto; background: #ffffff; border-radius: 8px; padding: 40px; text-align: center;">
    <h1 style="color: #1a1a2e; margin-bottom: 8px;">BuskaTotal</h1>
    <p style="color: #555; font-size: 16px; margin-bottom: 24px;">
      Confirme seu endereço de e-mail para ativar sua conta.
    </p>
    <a href="%s"
       style="display: inline-block; background-color: #2563eb; color: #ffffff; text-decoration: none;
              padding: 14px 32px; border-radius: 6px; font-size: 16px; font-weight: bold;">
      Confirmar E-mail
    </a>
    <p style="color: #999; font-size: 13px; margin-top: 32px;">
      Este link expira em 24 horas.<br>
      Se você não criou uma conta, ignore este e-mail.
    </p>
  </div>
</body>
</html>`, link)
}

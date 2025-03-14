package email

import (
	"errors"
	"fmt"
	"os"

	mailjet "github.com/mailjet/mailjet-apiv3-go/v4"
)

var (
	// ErrEmailSendFailed is returned when an email fails to send
	ErrEmailSendFailed = errors.New("failed to send email")
	// ErrEmailServiceNotConfigured is returned when the email service is not properly configured
	ErrEmailServiceNotConfigured = errors.New("email service is not properly configured")
)

// EmailService defines the interface for sending emails
type EmailService interface {
	SendVerificationEmail(email, token string) error
	SendPasswordResetEmail(email, token string) error
}

// MailjetService implements EmailService using Mailjet
type MailjetService struct {
	client      *mailjet.Client
	senderName  string
	senderEmail string
	baseURL     string
}

// NewMailjetService creates a new MailjetService
func NewMailjetService() (*MailjetService, error) {
	apiKey := os.Getenv("MAILJET_API_KEY")
	apiSecret := os.Getenv("MAILJET_SECRET_KEY")
	senderEmail := os.Getenv("MAILJET_SENDER_EMAIL")
	senderName := os.Getenv("MAILJET_SENDER_NAME")
	baseURL := os.Getenv("APP_BASE_URL")

	if apiKey == "" || apiSecret == "" {
		return nil, errors.New("Mailjet API key and secret are required")
	}
	if senderEmail == "" || senderName == "" {
		return nil, errors.New("Mailjet sender email and name are required")
	}
	if baseURL == "" {
		return nil, errors.New("APP_BASE_URL is required")
	}

	client := mailjet.NewMailjetClient(apiKey, apiSecret)
	return &MailjetService{
		client:      client,
		senderName:  senderName,
		senderEmail: senderEmail,
		baseURL:     baseURL,
	}, nil
}

// SendVerificationEmail sends a verification email to the user
func (s *MailjetService) SendVerificationEmail(email, token string) error {
	// Check if the service is properly configured
	if s.client == nil || s.baseURL == "" {
		return ErrEmailServiceNotConfigured
	}

	data := &mailjet.InfoMessagesV31{
		From: &mailjet.RecipientV31{
			Email: s.senderEmail,
			Name:  s.senderName,
		},
		To: &mailjet.RecipientsV31{
			mailjet.RecipientV31{
				Email: email,
			},
		},
		Subject:  "Verify your Virtual Armory account",
		TextPart: fmt.Sprintf("Please verify your account by clicking this link: %s/verify-email?token=%s", s.baseURL, token),
		HTMLPart: fmt.Sprintf(`
			<h3>Welcome to Virtual Armory!</h3>
			<p>Please verify your account by clicking the link below:</p>
			<p><a href="%s/verify-email?token=%s">Verify Account</a></p>
			<p>If you did not create this account, please ignore this email.</p>
		`, s.baseURL, token),
	}

	messages := &mailjet.MessagesV31{Info: []mailjet.InfoMessagesV31{*data}}
	_, err := s.client.SendMailV31(messages)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEmailSendFailed, err)
	}

	return nil
}

// SendPasswordResetEmail sends a password reset email to the user
func (s *MailjetService) SendPasswordResetEmail(email, token string) error {
	// Check if the service is properly configured
	if s.client == nil || s.baseURL == "" {
		return ErrEmailServiceNotConfigured
	}

	data := &mailjet.InfoMessagesV31{
		From: &mailjet.RecipientV31{
			Email: s.senderEmail,
			Name:  s.senderName,
		},
		To: &mailjet.RecipientsV31{
			mailjet.RecipientV31{
				Email: email,
			},
		},
		Subject:  "Reset your Virtual Armory password",
		TextPart: fmt.Sprintf("Reset your password by clicking this link: %s/reset-password?token=%s", s.baseURL, token),
		HTMLPart: fmt.Sprintf(`
			<h3>Password Reset Request</h3>
			<p>You have requested to reset your password. Click the link below to proceed:</p>
			<p><a href="%s/reset-password?token=%s">Reset Password</a></p>
			<p>If you did not request this password reset, please ignore this email.</p>
		`, s.baseURL, token),
	}

	messages := &mailjet.MessagesV31{Info: []mailjet.InfoMessagesV31{*data}}
	_, err := s.client.SendMailV31(messages)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEmailSendFailed, err)
	}

	return nil
}

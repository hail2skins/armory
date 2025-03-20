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
	SendVerificationEmail(email, token, baseURL string) error
	SendEmailChangeVerification(email, token, baseURL string) error
	SendPasswordResetEmail(email, token, baseURL string) error
	SendContactEmail(name, email, subject, message string) error
}

// MailjetService implements EmailService using Mailjet
type MailjetService struct {
	client      *mailjet.Client
	senderName  string
	senderEmail string
	adminEmail  string
}

// NewMailjetService creates a new MailjetService
func NewMailjetService() (*MailjetService, error) {
	apiKey := os.Getenv("MAILJET_API_KEY")
	apiSecret := os.Getenv("MAILJET_SECRET_KEY")
	senderEmail := os.Getenv("MAILJET_SENDER_EMAIL")
	senderName := os.Getenv("MAILJET_SENDER_NAME")
	adminEmail := os.Getenv("ADMIN_EMAIL")

	if apiKey == "" || apiSecret == "" {
		return nil, errors.New("Mailjet API key and secret are required")
	}
	if senderEmail == "" || senderName == "" {
		return nil, errors.New("Mailjet sender email and name are required")
	}
	if adminEmail == "" {
		return nil, errors.New("ADMIN_EMAIL is required")
	}

	client := mailjet.NewMailjetClient(apiKey, apiSecret)

	return &MailjetService{
		client:      client,
		senderName:  senderName,
		senderEmail: senderEmail,
		adminEmail:  adminEmail,
	}, nil
}

// SendVerificationEmail sends a verification email to the user
func (s *MailjetService) SendVerificationEmail(email, token, baseURL string) error {
	// Check if the service is properly configured
	if s.client == nil {
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
		Subject:  "Verify your email for Virtual Armory",
		TextPart: fmt.Sprintf("Please verify your email by clicking this link: %s/verify-email?token=%s. This verification link will expire in 60 minutes.", baseURL, token),
		HTMLPart: fmt.Sprintf(`
			<h3>Email Verification</h3>
			<p>Thank you for registering with Virtual Armory!</p>
			<p>Please verify your email by clicking the link below:</p>
			<p><a href="%s/verify-email?token=%s">Verify Email</a></p>
			<p><strong>Note:</strong> This verification link will expire in 60 minutes.</p>
			<p>If you did not register for Virtual Armory, please ignore this email.</p>
		`, baseURL, token),
	}

	messages := &mailjet.MessagesV31{Info: []mailjet.InfoMessagesV31{*data}}
	_, err := s.client.SendMailV31(messages)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEmailSendFailed, err)
	}

	return nil
}

// SendEmailChangeVerification sends a verification email for an email address change
func (s *MailjetService) SendEmailChangeVerification(email, token, baseURL string) error {
	// Check if the service is properly configured
	if s.client == nil {
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
		Subject:  "Verify your new email address for Virtual Armory",
		TextPart: fmt.Sprintf("Please verify your new email address by clicking this link: %s/verify-email?token=%s. This verification link will expire in 60 minutes.", baseURL, token),
		HTMLPart: fmt.Sprintf(`
			<h3>Email Change Verification</h3>
			<p>You have requested to change your email address for your Virtual Armory account.</p>
			<p>Please verify your new email address by clicking the link below:</p>
			<p><a href="%s/verify-email?token=%s">Verify New Email Address</a></p>
			<p><strong>Note:</strong> This verification link will expire in 60 minutes.</p>
			<p>If you did not request this change, please ignore this email and your account will remain unchanged.</p>
		`, baseURL, token),
	}

	messages := &mailjet.MessagesV31{Info: []mailjet.InfoMessagesV31{*data}}
	_, err := s.client.SendMailV31(messages)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEmailSendFailed, err)
	}

	return nil
}

// SendPasswordResetEmail sends a password reset email to the user
func (s *MailjetService) SendPasswordResetEmail(email, token, baseURL string) error {
	// Check if the service is properly configured
	if s.client == nil {
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
		TextPart: fmt.Sprintf("Reset your password by clicking this link: %s/reset-password?token=%s. This password reset link will expire in 60 minutes.", baseURL, token),
		HTMLPart: fmt.Sprintf(`
			<h3>Password Reset Request</h3>
			<p>You have requested to reset your password. Click the link below to proceed:</p>
			<p><a href="%s/reset-password?token=%s">Reset Password</a></p>
			<p><strong>Note:</strong> This password reset link will expire in 60 minutes.</p>
			<p>If you did not request this password reset, please ignore this email.</p>
		`, baseURL, token),
	}

	messages := &mailjet.MessagesV31{Info: []mailjet.InfoMessagesV31{*data}}
	_, err := s.client.SendMailV31(messages)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEmailSendFailed, err)
	}

	return nil
}

// SendContactEmail sends a contact form submission to the admin
func (s *MailjetService) SendContactEmail(name, email, subject, message string) error {
	// Check if the service is properly configured
	if s.client == nil || s.adminEmail == "" {
		return ErrEmailServiceNotConfigured
	}

	// Create the email content
	emailSubject := fmt.Sprintf("Contact Form: %s", subject)
	textContent := fmt.Sprintf("Name: %s\nEmail: %s\nSubject: %s\nMessage: %s", name, email, subject, message)
	htmlContent := fmt.Sprintf(`
		<h3>Contact Form Submission</h3>
		<p><strong>Name:</strong> %s</p>
		<p><strong>Email:</strong> %s</p>
		<p><strong>Subject:</strong> %s</p>
		<p><strong>Message:</strong></p>
		<p>%s</p>
	`, name, email, subject, message)

	// Create the email data
	data := &mailjet.InfoMessagesV31{
		From: &mailjet.RecipientV31{
			Email: s.senderEmail,
			Name:  s.senderName,
		},
		To: &mailjet.RecipientsV31{
			mailjet.RecipientV31{
				Email: s.adminEmail,
			},
		},
		ReplyTo: &mailjet.RecipientV31{
			Email: email,
			Name:  name,
		},
		Subject:  emailSubject,
		TextPart: textContent,
		HTMLPart: htmlContent,
	}

	messages := &mailjet.MessagesV31{Info: []mailjet.InfoMessagesV31{*data}}
	_, err := s.client.SendMailV31(messages)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEmailSendFailed, err)
	}

	return nil
}

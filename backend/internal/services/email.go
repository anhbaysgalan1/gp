package services

import (
	"fmt"
	"net/smtp"
	"strings"

	"github.com/anhbaysgalan1/gp/internal/config"
)

type EmailService struct {
	config *config.Config
}

func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{
		config: cfg,
	}
}

// SendEmail sends an email using SMTP
func (es *EmailService) SendEmail(to, subject, body string) error {
	// Set up authentication information
	auth := smtp.PlainAuth(
		"",
		es.config.SMTPUsername,
		es.config.SMTPPassword,
		es.config.SMTPHost,
	)

	// Compose email
	msg := es.composeEmail(es.config.SMTPFrom, to, subject, body)

	// Connect to the server and send email
	addr := fmt.Sprintf("%s:%s", es.config.SMTPHost, es.config.SMTPPort)
	err := smtp.SendMail(addr, auth, es.config.SMTPFrom, []string{to}, []byte(msg))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// composeEmail creates a properly formatted email message
func (es *EmailService) composeEmail(from, to, subject, body string) string {
	return fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body)
}

// SendVerificationEmail sends an email verification email
func (es *EmailService) SendVerificationEmail(to, username, verificationToken string) error {
	subject := "Verify your email - Poker Platform"

	// In production, this should be your actual frontend URL
	verificationURL := fmt.Sprintf("http://localhost:3000/verify-email?token=%s", verificationToken)

	body := fmt.Sprintf(`
		<html>
		<body>
			<h2>Welcome to Poker Platform, %s!</h2>
			<p>Thank you for registering. Please verify your email address by clicking the link below:</p>
			<p><a href="%s">Verify Email Address</a></p>
			<p>If you cannot click the link, copy and paste this URL into your browser:</p>
			<p>%s</p>
			<p>This verification link will expire in 24 hours.</p>
			<p>If you did not create this account, please ignore this email.</p>
			<br>
			<p>Best regards,<br>The Poker Platform Team</p>
		</body>
		</html>
	`, strings.Title(username), verificationURL, verificationURL)

	return es.SendEmail(to, subject, body)
}

// SendPasswordResetEmail sends a password reset email
func (es *EmailService) SendPasswordResetEmail(to, username, resetToken string) error {
	subject := "Reset your password - Poker Platform"

	// In production, this should be your actual frontend URL
	resetURL := fmt.Sprintf("http://localhost:3000/reset-password?token=%s", resetToken)

	body := fmt.Sprintf(`
		<html>
		<body>
			<h2>Password Reset Request</h2>
			<p>Hello %s,</p>
			<p>We received a request to reset your password. Click the link below to reset your password:</p>
			<p><a href="%s">Reset Password</a></p>
			<p>If you cannot click the link, copy and paste this URL into your browser:</p>
			<p>%s</p>
			<p>This password reset link will expire in 1 hour.</p>
			<p>If you did not request a password reset, please ignore this email.</p>
			<br>
			<p>Best regards,<br>The Poker Platform Team</p>
		</body>
		</html>
	`, strings.Title(username), resetURL, resetURL)

	return es.SendEmail(to, subject, body)
}

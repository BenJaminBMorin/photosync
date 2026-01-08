package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"time"

	"github.com/photosync/server/internal/models"
	"github.com/photosync/server/internal/repository"
)

// SMTPService handles sending emails
type SMTPService struct {
	smtpRepo          repository.SMTPConfigRepo
	encryptionService *EncryptionService
}

// NewSMTPService creates a new SMTP service
func NewSMTPService(smtpRepo repository.SMTPConfigRepo, encryptionService *EncryptionService) *SMTPService {
	return &SMTPService{
		smtpRepo:          smtpRepo,
		encryptionService: encryptionService,
	}
}

// SendRecoveryEmail sends a password recovery email with a temporary login link
func (s *SMTPService) SendRecoveryEmail(ctx context.Context, toEmail, toName, recoveryToken, serverURL string) error {
	recoveryLink := fmt.Sprintf("%s/login?recovery=%s", serverURL, recoveryToken)

	data := RecoveryEmailData{
		Name:         toName,
		RecoveryLink: recoveryLink,
	}

	// Parse template
	tmpl, err := template.New("recovery").Parse(recoveryEmailTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse recovery email template: %w", err)
	}

	// Execute template
	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute recovery email template: %w", err)
	}

	subject := "🔐 PhotoSync Account Recovery"
	return s.sendEmail(ctx, toEmail, subject, body.String())
}

// SendTestEmail sends a test email to verify SMTP configuration
func (s *SMTPService) SendTestEmail(ctx context.Context, toEmail string) error {
	data := TestEmailData{
		Timestamp: time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
	}

	// Parse template
	tmpl, err := template.New("test").Parse(testEmailTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse test email template: %w", err)
	}

	// Execute template
	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute test email template: %w", err)
	}

	subject := "✅ PhotoSync SMTP Test"
	return s.sendEmail(ctx, toEmail, subject, body.String())
}

// SendInviteEmail sends an invitation email with a deep link to set up the app
func (s *SMTPService) SendInviteEmail(ctx context.Context, toEmail, toName, inviteToken, inviteLink string) error {
	data := InviteEmailData{
		Name:       toName,
		InviteLink: inviteLink,
		InviteCode: inviteToken,
	}

	// Parse template
	tmpl, err := template.New("invite").Parse(inviteEmailTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse invite email template: %w", err)
	}

	// Execute template
	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute invite email template: %w", err)
	}

	subject := "📸 You're invited to PhotoSync!"
	return s.sendEmail(ctx, toEmail, subject, body.String())
}

// sendEmail is the internal helper that performs the actual SMTP sending
func (s *SMTPService) sendEmail(ctx context.Context, to, subject, htmlBody string) error {
	// Get SMTP config
	config, err := s.smtpRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get SMTP config: %w", err)
	}
	if config == nil {
		return fmt.Errorf("SMTP not configured")
	}

	// Decrypt password
	password, err := s.encryptionService.Decrypt(config.Password)
	if err != nil {
		return fmt.Errorf("failed to decrypt SMTP password: %w", err)
	}

	// Build message
	from := fmt.Sprintf("%s <%s>", config.FromName, config.FromAddress)
	headers := map[string]string{
		"From":         from,
		"To":           to,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/html; charset=UTF-8",
	}

	var msg bytes.Buffer
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	// Send email
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	auth := smtp.PlainAuth("", config.Username, password, config.Host)

	if config.UseTLS {
		return s.sendWithTLS(addr, auth, config, from, to, msg.Bytes())
	}

	// Plain SMTP (not recommended for production)
	return smtp.SendMail(addr, auth, config.FromAddress, []string{to}, msg.Bytes())
}

// sendWithTLS sends email using STARTTLS
func (s *SMTPService) sendWithTLS(addr string, auth smtp.Auth, config *models.SMTPConfig, from, to string, message []byte) error {
	// Create TLS config
	tlsConfig := &tls.Config{
		ServerName:         config.Host,
		InsecureSkipVerify: config.SkipVerify,
	}

	// Connect with TLS
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial failed: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, config.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Quit()

	// Authenticate
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	// Set sender
	if err := client.Mail(config.FromAddress); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipient
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	// Send message
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to send DATA command: %w", err)
	}

	if _, err := w.Write(message); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close message writer: %w", err)
	}

	return nil
}

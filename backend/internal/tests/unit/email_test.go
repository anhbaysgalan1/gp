package unit

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/evanofslack/go-poker/internal/config"
	"github.com/evanofslack/go-poker/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSMTPServer provides a simple mock SMTP server for testing
type MockSMTPServer struct {
	listener    net.Listener
	port        string
	receivedMsg string
	started     bool
}

func NewMockSMTPServer() *MockSMTPServer {
	return &MockSMTPServer{}
}

func (m *MockSMTPServer) Start() error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}

	m.listener = listener
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	m.port = port
	m.started = true

	go m.handleConnections()
	return nil
}

func (m *MockSMTPServer) Stop() {
	if m.listener != nil {
		m.listener.Close()
	}
	m.started = false
}

func (m *MockSMTPServer) GetPort() string {
	return m.port
}

func (m *MockSMTPServer) GetReceivedMessage() string {
	return m.receivedMsg
}

func (m *MockSMTPServer) handleConnections() {
	for {
		conn, err := m.listener.Accept()
		if err != nil {
			return // Server stopped
		}

		go m.handleConnection(conn)
	}
}

func (m *MockSMTPServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// Send greeting
	writer.WriteString("220 Mock SMTP Server Ready\r\n")
	writer.Flush()

	var messageData strings.Builder
	inDataMode := false

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		line = strings.TrimSpace(line)

		if inDataMode {
			if line == "." {
				// End of message data
				m.receivedMsg = messageData.String()
				writer.WriteString("250 Message accepted\r\n")
				writer.Flush()
				inDataMode = false
			} else {
				messageData.WriteString(line + "\n")
			}
			continue
		}

		// Handle SMTP commands
		switch {
		case strings.HasPrefix(line, "EHLO") || strings.HasPrefix(line, "HELO"):
			writer.WriteString("250 Hello\r\n")
		case strings.HasPrefix(line, "AUTH PLAIN"):
			writer.WriteString("235 Authentication successful\r\n")
		case strings.HasPrefix(line, "MAIL FROM:"):
			writer.WriteString("250 OK\r\n")
		case strings.HasPrefix(line, "RCPT TO:"):
			writer.WriteString("250 OK\r\n")
		case line == "DATA":
			writer.WriteString("354 Start mail input; end with <CRLF>.<CRLF>\r\n")
			inDataMode = true
		case line == "QUIT":
			writer.WriteString("221 Goodbye\r\n")
			writer.Flush()
			return
		default:
			writer.WriteString("250 OK\r\n")
		}
		writer.Flush()
	}
}

func TestEmailService_SendEmail(t *testing.T) {
	// Start mock SMTP server
	mockSMTP := NewMockSMTPServer()
	err := mockSMTP.Start()
	require.NoError(t, err)
	defer mockSMTP.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Create email service with mock SMTP config
	cfg := &config.Config{
		SMTPHost:     "127.0.0.1",
		SMTPPort:     mockSMTP.GetPort(),
		SMTPUsername: "test",
		SMTPPassword: "test",
		SMTPFrom:     "test@example.com",
	}

	emailService := services.NewEmailService(cfg)

	// Test sending email
	err = emailService.SendEmail("recipient@example.com", "Test Subject", "Test Body")
	assert.NoError(t, err)

	// Give some time for the message to be processed
	time.Sleep(200 * time.Millisecond)

	// Verify the email was received by mock server
	receivedMsg := mockSMTP.GetReceivedMessage()
	assert.Contains(t, receivedMsg, "From: test@example.com")
	assert.Contains(t, receivedMsg, "To: recipient@example.com")
	assert.Contains(t, receivedMsg, "Subject: Test Subject")
	assert.Contains(t, receivedMsg, "Test Body")
}

func TestEmailService_SendVerificationEmail(t *testing.T) {
	// Start mock SMTP server
	mockSMTP := NewMockSMTPServer()
	err := mockSMTP.Start()
	require.NoError(t, err)
	defer mockSMTP.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Create email service
	cfg := &config.Config{
		SMTPHost:     "127.0.0.1",
		SMTPPort:     mockSMTP.GetPort(),
		SMTPUsername: "test",
		SMTPPassword: "test",
		SMTPFrom:     "noreply@pokerplatform.com",
	}

	emailService := services.NewEmailService(cfg)

	// Test sending verification email
	err = emailService.SendVerificationEmail("user@example.com", "testuser", "verification-token-123")
	assert.NoError(t, err)

	// Give some time for the message to be processed
	time.Sleep(200 * time.Millisecond)

	// Verify the email content
	receivedMsg := mockSMTP.GetReceivedMessage()
	assert.Contains(t, receivedMsg, "From: noreply@pokerplatform.com")
	assert.Contains(t, receivedMsg, "To: user@example.com")
	assert.Contains(t, receivedMsg, "Subject: Verify your email - Poker Platform")
	assert.Contains(t, receivedMsg, "Welcome to Poker Platform, Testuser!")
	assert.Contains(t, receivedMsg, "verification-token-123")
	assert.Contains(t, receivedMsg, "http://localhost:3000/verify-email?token=verification-token-123")
	assert.Contains(t, receivedMsg, "Content-Type: text/html")
}

func TestEmailService_SendPasswordResetEmail(t *testing.T) {
	// Start mock SMTP server
	mockSMTP := NewMockSMTPServer()
	err := mockSMTP.Start()
	require.NoError(t, err)
	defer mockSMTP.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Create email service
	cfg := &config.Config{
		SMTPHost:     "127.0.0.1",
		SMTPPort:     mockSMTP.GetPort(),
		SMTPUsername: "test",
		SMTPPassword: "test",
		SMTPFrom:     "noreply@pokerplatform.com",
	}

	emailService := services.NewEmailService(cfg)

	// Test sending password reset email
	err = emailService.SendPasswordResetEmail("user@example.com", "testuser", "reset-token-456")
	assert.NoError(t, err)

	// Give some time for the message to be processed
	time.Sleep(200 * time.Millisecond)

	// Verify the email content
	receivedMsg := mockSMTP.GetReceivedMessage()
	assert.Contains(t, receivedMsg, "From: noreply@pokerplatform.com")
	assert.Contains(t, receivedMsg, "To: user@example.com")
	assert.Contains(t, receivedMsg, "Subject: Reset your password - Poker Platform")
	assert.Contains(t, receivedMsg, "Hello Testuser")
	assert.Contains(t, receivedMsg, "reset-token-456")
	assert.Contains(t, receivedMsg, "http://localhost:3000/reset-password?token=reset-token-456")
	assert.Contains(t, receivedMsg, "Content-Type: text/html")
}

func TestEmailService_ComposeEmail(t *testing.T) {
	cfg := &config.Config{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     "587",
		SMTPUsername: "test",
		SMTPPassword: "test",
		SMTPFrom:     "sender@example.com",
	}

	emailService := services.NewEmailService(cfg)

	// Use reflection to test the private composeEmail method by calling SendEmail with a mock
	// that will fail, but we can create a simple test of the compose logic
	from := "test@example.com"
	to := "recipient@example.com"
	subject := "Test Subject"
	body := "<h1>Hello World</h1>"

	// Expected format for reference (not used in this test since we use mock SMTP)
	_ = fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body)

	// We can't directly test the private method, but we can verify the email format
	// through integration with the mock SMTP server above
	mockSMTP := NewMockSMTPServer()
	err := mockSMTP.Start()
	require.NoError(t, err)
	defer mockSMTP.Stop()

	time.Sleep(100 * time.Millisecond)

	cfg.SMTPHost = "127.0.0.1"
	cfg.SMTPPort = mockSMTP.GetPort()
	emailService = services.NewEmailService(cfg)

	err = emailService.SendEmail(to, subject, body)
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	receivedMsg := mockSMTP.GetReceivedMessage()
	assert.Contains(t, receivedMsg, "From: test@example.com")
	assert.Contains(t, receivedMsg, "To: recipient@example.com")
	assert.Contains(t, receivedMsg, "Subject: Test Subject")
	assert.Contains(t, receivedMsg, "MIME-Version: 1.0")
	assert.Contains(t, receivedMsg, "Content-Type: text/html; charset=UTF-8")
	assert.Contains(t, receivedMsg, "<h1>Hello World</h1>")
}

func TestEmailService_InvalidSMTPConfig(t *testing.T) {
	// Test with invalid SMTP configuration
	cfg := &config.Config{
		SMTPHost:     "nonexistent.smtp.server",
		SMTPPort:     "587",
		SMTPUsername: "test",
		SMTPPassword: "test",
		SMTPFrom:     "test@example.com",
	}

	emailService := services.NewEmailService(cfg)

	// This should fail to connect to the nonexistent SMTP server
	err := emailService.SendEmail("test@example.com", "Test", "Body")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send email")
}

func TestEmailService_Templates(t *testing.T) {
	tests := []struct {
		name     string
		function func(es *services.EmailService) error
		subject  string
		keywords []string
	}{
		{
			name: "Verification email template",
			function: func(es *services.EmailService) error {
				return es.SendVerificationEmail("test@example.com", "john_doe", "token123")
			},
			subject: "Verify your email - Poker Platform",
			keywords: []string{
				"Welcome to Poker Platform, John_doe!",
				"verify your email address",
				"token123",
				"verification link will expire in 24 hours",
				"<html>",
			},
		},
		{
			name: "Password reset email template",
			function: func(es *services.EmailService) error {
				return es.SendPasswordResetEmail("test@example.com", "jane_smith", "reset456")
			},
			subject: "Reset your password - Poker Platform",
			keywords: []string{
				"Hello Jane_smith",
				"reset your password",
				"reset456",
				"password reset link will expire in 1 hour",
				"<html>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start mock SMTP server
			mockSMTP := NewMockSMTPServer()
			err := mockSMTP.Start()
			require.NoError(t, err)
			defer mockSMTP.Stop()

			time.Sleep(100 * time.Millisecond)

			// Create email service
			cfg := &config.Config{
				SMTPHost:     "127.0.0.1",
				SMTPPort:     mockSMTP.GetPort(),
				SMTPUsername: "test",
				SMTPPassword: "test",
				SMTPFrom:     "noreply@example.com",
			}

			emailService := services.NewEmailService(cfg)

			// Send email using the test function
			err = tt.function(emailService)
			assert.NoError(t, err)

			time.Sleep(200 * time.Millisecond)

			// Verify email content
			receivedMsg := mockSMTP.GetReceivedMessage()
			assert.Contains(t, receivedMsg, fmt.Sprintf("Subject: %s", tt.subject))

			for _, keyword := range tt.keywords {
				assert.Contains(t, receivedMsg, keyword, "Missing keyword: %s", keyword)
			}
		})
	}
}
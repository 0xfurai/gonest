package mail

import (
	"fmt"

	"github.com/0xfurai/gonest"
)

// MailService sends emails (stub implementation).
// In production, integrate with SMTP, SendGrid, AWS SES, etc.
type MailService struct {
	logger gonest.Logger
	from   string
}

func NewMailService(logger gonest.Logger) *MailService {
	return &MailService{
		logger: logger,
		from:   "noreply@example.com",
	}
}

// SendWelcome sends a welcome email to a new user.
func (s *MailService) SendWelcome(email, firstName string) error {
	s.logger.Log("Sending welcome email to %s (%s)", email, firstName)
	return s.send(email, "Welcome to GoNest App",
		fmt.Sprintf("Hello %s,\n\nWelcome to our application!\n\nBest regards,\nThe Team", firstName))
}

// SendPasswordReset sends a password reset email.
func (s *MailService) SendPasswordReset(email, resetToken string) error {
	s.logger.Log("Sending password reset email to %s", email)
	return s.send(email, "Password Reset Request",
		fmt.Sprintf("Click the following link to reset your password:\nhttps://example.com/reset?token=%s", resetToken))
}

// SendNotification sends a generic notification email.
func (s *MailService) SendNotification(email, subject, body string) error {
	return s.send(email, subject, body)
}

func (s *MailService) send(to, subject, body string) error {
	// Stub: log instead of sending
	s.logger.Log("MAIL [%s -> %s] Subject: %s", s.from, to, subject)
	// In production:
	// return smtp.SendMail(addr, auth, s.from, []string{to}, msg)
	return nil
}

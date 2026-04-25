package mail

import (
	"eraya/config"
	"fmt"
	"net/smtp"
)

type Mailer interface {
	SendOTP(to string, otp string) error
}

type smtpMailer struct {
	config config.SMTPConfig
}

func NewMailer(cnf config.SMTPConfig) Mailer {
	// If no SMTP host is provided, return a mock mailer that prints to console
	if cnf.Host == "" {
		return &mockMailer{}
	}
	return &smtpMailer{config: cnf}
}

func (m *smtpMailer) SendOTP(to string, otp string) error {
	subject := "Subject: Eraya | Security Verification Code\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: auto; padding: 20px; border: 1px solid #eee; border-radius: 10px;">
			<h2 style="color: #2c3e50; text-align: center;">Security Verification</h2>
			<p>Hello,</p>
			<p>You have requested a sensitive change to your account. Please use the following verification code to proceed:</p>
			<div style="background: #f8f9fa; padding: 20px; text-align: center; font-size: 32px; font-weight: bold; letter-spacing: 5px; color: #4338ca; border-radius: 8px; margin: 20px 0;">
				%s
			</div>
			<p>This code will expire in 10 minutes. If you did not request this, please secure your account immediately.</p>
			<hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
			<p style="font-size: 12px; color: #777; text-align: center;">&copy; 2026 Eraya Ecommerce. All rights reserved.</p>
		</div>
	`, otp)

	msg := []byte(subject + mime + body)
	addr := fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)
	auth := smtp.PlainAuth("", m.config.User, m.config.Password, m.config.Host)

	return smtp.SendMail(addr, auth, m.config.User, []string{to}, msg)
}

type mockMailer struct{}

func (m *mockMailer) SendOTP(to string, otp string) error {
	fmt.Printf("\n--- MOCK EMAIL SENT ---\nTo: %s\nOTP: %s\n-----------------------\n\n", to, otp)
	return nil
}

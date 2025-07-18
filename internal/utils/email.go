package utils

import (
	"mwork_backend/internal/config"

	"gopkg.in/gomail.v2"
)

type EmailSender struct {
	cfg *config.Config
}

func NewEmailSender(cfg *config.Config) *EmailSender {
	return &EmailSender{cfg: cfg}
}

func (e *EmailSender) Send(to, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", e.cfg.Email.FromEmail)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(
		e.cfg.Email.SMTPHost,
		e.cfg.Email.SMTPPort,
		e.cfg.Email.SMTPUser,
		e.cfg.Email.SMTPPassword,
	)

	return d.DialAndSend(m)
}

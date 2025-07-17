package services

import (
	"fmt"
	"mwork_front_fn/internal/utils"
)

type EmailService struct {
	sender *utils.EmailSender
}

func NewEmailService(sender *utils.EmailSender) *EmailService {
	return &EmailService{sender: sender}
}

// Письмо с подтверждением регистрации
func (e *EmailService) SendVerificationEmail(toEmail, token string) error {
	link := fmt.Sprintf("https://your-frontend.com/verify-email?token=%s", token)
	subject := "Подтвердите ваш email"
	body := fmt.Sprintf(`<p>Привет! Подтвердите ваш email, перейдя по ссылке:</p><a href="%s">%s</a>`, link, link)
	return e.sender.Send(toEmail, subject, body)
}

// Письмо для восстановления пароля
func (e *EmailService) SendPasswordResetEmail(toEmail, resetToken string) error {
	link := fmt.Sprintf("https://your-frontend.com/reset-password?token=%s", resetToken)
	subject := "Сброс пароля"
	body := fmt.Sprintf(`<p>Чтобы сбросить пароль, перейдите по ссылке:</p><a href="%s">%s</a>`, link, link)
	return e.sender.Send(toEmail, subject, body)
}

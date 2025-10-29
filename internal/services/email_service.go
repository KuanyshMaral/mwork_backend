package services

import (
	"context"
	"fmt"
	"mwork_backend/internal/email"
)

// EmailService предоставляет высокоуровневый интерфейс для работы с email
type EmailService struct {
	provider email.Provider
}

// NewEmailService создает новый экземпляр EmailService
func NewEmailService(provider email.Provider) *EmailService {
	return &EmailService{
		provider: provider,
	}
}

// SendSimpleEmail отправляет простое текстовое email сообщение
func (s *EmailService) SendSimpleEmail(ctx context.Context, to []string, subject, body string) error {
	emailMsg := &email.Email{
		To:      to,
		Subject: subject,
		Body:    body,
	}

	return s.provider.Send(emailMsg)
}

// SendHTMLEmail отправляет HTML email сообщение
func (s *EmailService) SendHTMLEmail(ctx context.Context, to []string, subject, htmlBody string) error {
	emailMsg := &email.Email{
		To:       to,
		Subject:  subject,
		HTMLBody: htmlBody,
	}

	return s.provider.Send(emailMsg)
}

// SendTemplatedEmail отправляет email используя шаблон
func (s *EmailService) SendTemplatedEmail(ctx context.Context, to []string, subject, templateName string, data email.TemplateData) error {
	emailMsg := &email.Email{
		To:      to,
		Subject: subject,
	}

	return s.provider.SendWithTemplate(templateName, data, emailMsg)
}

// SendEmailWithAttachments отправляет email с вложениями
func (s *EmailService) SendEmailWithAttachments(ctx context.Context, to []string, subject, body string, attachments []email.Attachment) error {
	emailMsg := &email.Email{
		To:          to,
		Subject:     subject,
		Body:        body,
		Attachments: attachments,
	}

	return s.provider.Send(emailMsg)
}

// SendBulkEmail отправляет массовые email сообщения
func (s *EmailService) SendBulkEmail(ctx context.Context, emails []BulkEmail) error {
	for i, emailItem := range emails {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			emailMsg := &email.Email{
				To:      []string{emailItem.To},
				Subject: emailItem.Subject,
				Body:    emailItem.Body,
			}

			if emailItem.TemplateName != "" {
				err := s.provider.SendWithTemplate(emailItem.TemplateName, emailItem.TemplateData, emailMsg)
				if err != nil {
					return fmt.Errorf("failed to send email to %s (index %d): %w", emailItem.To, i, err)
				}
			} else {
				err := s.provider.Send(emailMsg)
				if err != nil {
					return fmt.Errorf("failed to send email to %s (index %d): %w", emailItem.To, i, err)
				}
			}
		}
	}

	return nil
}

// SendWelcomeEmail отправляет приветственное письмо
func (s *EmailService) SendWelcomeEmail(ctx context.Context, to, userName string) error {
	data := email.TemplateData{
		"UserName": userName,
		"LoginURL": "https://yourapp.com/login",
	}

	return s.SendTemplatedEmail(ctx, []string{to}, "Добро пожаловать!", "welcome", data)
}

// SendPasswordResetEmail отправляет письмо для сброса пароля
func (s *EmailService) SendPasswordResetEmail(ctx context.Context, to, resetToken string) error {
	resetURL := fmt.Sprintf("https://yourapp.com/reset-password?token=%s", resetToken)

	data := email.TemplateData{
		"ResetURL":  resetURL,
		"ExpiresIn": "1 час",
	}

	return s.SendTemplatedEmail(ctx, []string{to}, "Сброс пароля", "password_reset", data)
}

// SendVerificationEmail отправляет письмо для верификации email
func (s *EmailService) SendVerificationEmail(ctx context.Context, to, verificationToken string) error {
	verifyURL := fmt.Sprintf("https://yourapp.com/verify-email?token=%s", verificationToken)

	data := email.TemplateData{
		"VerifyURL": verifyURL,
		"ExpiresIn": "24 часа",
	}

	return s.SendTemplatedEmail(ctx, []string{to}, "Подтверждение email", "email_verification", data)
}

// SendNotificationEmail отправляет уведомительное письмо
func (s *EmailService) SendNotificationEmail(ctx context.Context, to, title, message string) error {
	data := email.TemplateData{
		"Title":   title,
		"Message": message,
	}

	return s.SendTemplatedEmail(ctx, []string{to}, title, "notification", data)
}

// Validate проверяет конфигурацию email сервиса
func (s *EmailService) Validate() error {
	return s.provider.Validate()
}

// Close закрывает соединения email сервиса
func (s *EmailService) Close() error {
	return s.provider.Close()
}

// BulkEmail представляет email для массовой рассылки
type BulkEmail struct {
	To           string
	Subject      string
	Body         string
	TemplateName string
	TemplateData email.TemplateData
}

// EmailServiceConfig конфигурация для EmailService
type EmailServiceConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
	UseTLS       bool
	TemplatesDir string
}

// NewEmailServiceWithConfig создает EmailService с конфигурацией
func NewEmailServiceWithConfig(config EmailServiceConfig) (*EmailService, error) {
	emailConfig := &email.SMTPConfig{
		Host:      config.SMTPHost,
		Port:      config.SMTPPort,
		Username:  config.SMTPUsername,
		Password:  config.SMTPPassword,
		FromEmail: config.FromEmail,
		FromName:  config.FromName,
		UseTLS:    config.UseTLS,
	}

	templateManager := email.NewTemplateManager()

	// Загружаем шаблоны если указана директория
	if config.TemplatesDir != "" {
		if err := templateManager.LoadTemplates(config.TemplatesDir); err != nil {
			return nil, fmt.Errorf("failed to load email templates: %w", err)
		}
	}

	provider := email.NewSMTPProvider(emailConfig, templateManager)

	return NewEmailService(provider), nil
}

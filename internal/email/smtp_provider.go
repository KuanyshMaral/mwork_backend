package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// SMTPProvider реализует Provider для SMTP
type SMTPProvider struct {
	config   *SMTPConfig
	auth     smtp.Auth
	renderer TemplateRenderer
}

// NewSMTPProvider создает новый SMTP провайдер
func NewSMTPProvider(config *SMTPConfig, renderer TemplateRenderer) *SMTPProvider {
	var auth smtp.Auth
	if config.Username != "" && config.Password != "" {
		auth = smtp.PlainAuth("", config.Username, config.Password, config.Host)
	}

	return &SMTPProvider{
		config:   config,
		auth:     auth,
		renderer: renderer,
	}
}

// Send отправляет email сообщение
func (p *SMTPProvider) Send(email *Email) error {
	if err := p.Validate(); err != nil {
		return err
	}

	message, err := p.buildMessage(email)
	if err != nil {
		return fmt.Errorf("failed to build message: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", p.config.Host, p.config.Port)

	if p.config.UseTLS {
		tlsConfig := &tls.Config{
			ServerName: p.config.Host,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to dial TLS: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, p.config.Host)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		defer client.Close()

		return p.sendWithClient(client, email, message)
	}

	return smtp.SendMail(addr, p.auth, email.From, email.To, message)
}

// SendWithTemplate отправляет email используя шаблон
func (p *SMTPProvider) SendWithTemplate(templateName string, data TemplateData, email *Email) error {
	if p.renderer == nil {
		return fmt.Errorf("template renderer is not configured")
	}

	htmlBody, err := p.renderer.Render(templateName, data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	email.HTMLBody = htmlBody
	return p.Send(email)
}

// Validate проверяет конфигурацию SMTP
func (p *SMTPProvider) Validate() error {
	if p.config.Host == "" {
		return fmt.Errorf("SMTP host is required")
	}

	if p.config.Port <= 0 || p.config.Port > 65535 {
		return fmt.Errorf("invalid SMTP port: %d", p.config.Port)
	}

	return nil
}

// Close закрывает соединение (для SMTP обычно не требуется)
func (p *SMTPProvider) Close() error {
	return nil
}

// buildMessage строит MIME сообщение из структуры Email
func (p *SMTPProvider) buildMessage(email *Email) ([]byte, error) {
	builder := &strings.Builder{}

	// Заголовки
	builder.WriteString(fmt.Sprintf("From: %s\r\n", email.From))
	builder.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(email.To, ",")))

	if len(email.Cc) > 0 {
		builder.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(email.Cc, ",")))
	}

	builder.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))
	builder.WriteString("MIME-Version: 1.0\r\n")

	// Тело сообщения
	if len(email.Attachments) > 0 {
		return p.buildMultipartMessage(email, builder)
	}

	if email.HTMLBody != "" {
		builder.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n")
		builder.WriteString(email.HTMLBody)
	} else {
		builder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n")
		builder.WriteString(email.Body)
	}

	return []byte(builder.String()), nil
}

// buildMultipartMessage строит multipart MIME сообщение с вложениями
func (p *SMTPProvider) buildMultipartMessage(email *Email, builder *strings.Builder) ([]byte, error) {
	boundary := "boundary12345"

	builder.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", boundary))

	// Текстовая часть
	builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	if email.HTMLBody != "" {
		builder.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n")
		builder.WriteString(email.HTMLBody)
	} else {
		builder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n")
		builder.WriteString(email.Body)
	}
	builder.WriteString("\r\n")

	// Вложения
	for _, attachment := range email.Attachments {
		builder.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		builder.WriteString(fmt.Sprintf("Content-Type: %s\r\n", attachment.ContentType))
		builder.WriteString("Content-Transfer-Encoding: base64\r\n")
		builder.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", attachment.Name))

		// В реальной реализации здесь должна быть base64 кодировка
		builder.Write(attachment.Content)
		builder.WriteString("\r\n")
	}

	builder.WriteString(fmt.Sprintf("--%s--", boundary))

	return []byte(builder.String()), nil
}

// sendWithClient отправляет сообщение используя существующий SMTP клиент
func (p *SMTPProvider) sendWithClient(client *smtp.Client, email *Email, message []byte) error {
	if p.auth != nil {
		if err := client.Auth(p.auth); err != nil {
			return fmt.Errorf("failed to authenticate: %w", err)
		}
	}

	if err := client.Mail(email.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	recipients := append(append(email.To, email.Cc...), email.Bcc...)
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	if _, err := w.Write(message); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return nil
}

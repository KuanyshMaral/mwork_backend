package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

// SMTPSender реализация Sender для SMTP
type SMTPSender struct {
	config    Config
	templates *TemplateManager
	auth      smtp.Auth
	client    *smtp.Client
}

// NewSMTPSender создает новый SMTP отправитель
func NewSMTPSender(config Config) (Sender, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid email config: %w", err)
	}

	tm, err := NewTemplateManager(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create template manager: %w", err)
	}

	sender := &SMTPSender{
		config:    config,
		templates: tm,
	}

	// Настраиваем аутентификацию
	if config.Username != "" && config.Password != "" {
		sender.auth = smtp.PlainAuth("", config.Username, config.Password, config.SMTPHost)
	}

	return sender, nil
}

// Send отправляет email
func (s *SMTPSender) Send(email *Email) error {
	if len(email.To) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	// Подготавливаем сообщение
	message := s.buildMessage(email)

	// Отправляем через SMTP
	return s.sendSMTP(email.To, message)
}

// SendTemplate отправляет email используя шаблон
func (s *SMTPSender) SendTemplate(to []string, subject, templateName string, data interface{}) error {
	// Рендерим HTML из шаблона
	htmlBody, err := s.templates.Render(templateName, data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Создаем текстовую версию (упрощенную)
	textBody := s.htmlToText(htmlBody)

	email := &Email{
		To:       to,
		Subject:  subject,
		Body:     textBody,
		HTMLBody: htmlBody,
	}

	return s.Send(email)
}

// SendWelcome отправляет приветственное письмо
func (s *SMTPSender) SendWelcome(email, name, userRole string) error {
	data := WelcomeData{
		TemplateData: TemplateData{
			UserName:     name,
			Subject:      "Добро пожаловать в MWORK!",
			SupportEmail: s.config.FromEmail,
			CompanyName:  "MWORK",
		},
		UserRole: userRole,
	}

	return s.SendTemplate([]string{email}, "Добро пожаловать в MWORK!", "welcome", data)
}

// SendVerification отправляет письмо для верификации email
func (s *SMTPSender) SendVerification(email, token string) error {
	data := TemplateData{
		Subject:    "Подтверждение Email",
		ActionURL:  fmt.Sprintf("https://mwork.ru/verify?token=%s", token),
		ActionText: "Подтвердить Email",
	}

	return s.SendTemplate([]string{email}, "Подтверждение Email", "verification", data)
}

// SendResponseStatus отправляет уведомление об изменении статуса отклика
func (s *SMTPSender) SendResponseStatus(email, modelName, castingTitle, status string) error {
	data := ResponseStatusData{
		TemplateData: TemplateData{
			UserName: modelName,
			Subject:  "Обновление статуса отклика",
		},
		CastingTitle: castingTitle,
		Status:       status,
		ModelName:    modelName,
	}

	return s.SendTemplate([]string{email}, "Статус отклика обновлен", "response_status", data)
}

// SendCastingMatch отправляет уведомление о совпадении кастинга
func (s *SMTPSender) SendCastingMatch(email, modelName, castingTitle string, score float64) error {
	data := CastingMatchData{
		TemplateData: TemplateData{
			UserName:   modelName,
			Subject:    "Новый подходящий кастинг",
			ActionURL:  "https://mwork.ru/castings",
			ActionText: "Посмотреть кастинг",
		},
		CastingTitle: castingTitle,
		MatchScore:   score,
	}

	return s.SendTemplate([]string{email}, "Для вас найден подходящий кастинг!", "casting_match", data)
}

// SendNotification отправляет простое уведомление
func (s *SMTPSender) SendNotification(email, subject, message string) error {
	data := TemplateData{
		Subject: subject,
		Message: message,
	}

	return s.SendTemplate([]string{email}, subject, "notification", data)
}

// SendBulk отправляет несколько email
func (s *SMTPSender) SendBulk(emails []*Email) error {
	for i, email := range emails {
		if err := s.Send(email); err != nil {
			return fmt.Errorf("failed to send email %d: %w", i, err)
		}
		// Небольшая задержка между отправками
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

// buildMessage строит сообщение для SMTP
func (s *SMTPSender) buildMessage(email *Email) []byte {
	headers := []string{
		fmt.Sprintf("From: %s <%s>", s.config.FromName, s.config.FromEmail),
		fmt.Sprintf("To: %s", strings.Join(email.To, ", ")),
		fmt.Sprintf("Subject: %s", email.Subject),
		"MIME-version: 1.0;",
		"Content-Type: multipart/alternative; boundary=\"MWORK_BOUNDARY\"",
		"",
	}

	var body []string

	// Текстовая версия
	if email.Body != "" {
		body = append(body,
			"--MWORK_BOUNDARY",
			"Content-Type: text/plain; charset=UTF-8",
			"",
			email.Body,
			"",
		)
	}

	// HTML версия
	if email.HTMLBody != "" {
		body = append(body,
			"--MWORK_BOUNDARY",
			"Content-Type: text/html; charset=UTF-8",
			"",
			email.HTMLBody,
			"",
		)
	}

	// Завершающий boundary
	body = append(body, "--MWORK_BOUNDARY--")

	message := strings.Join(headers, "\r\n") + "\r\n" + strings.Join(body, "\r\n")
	return []byte(message)
}

// sendSMTP отправляет сообщение через SMTP
func (s *SMTPSender) sendSMTP(to []string, message []byte) error {
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)

	// Создаем соединение
	var client *smtp.Client
	var err error

	if s.config.UseSSL {
		// SSL соединение
		tlsConfig := &tls.Config{
			ServerName: s.config.SMTPHost,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect via SSL: %w", err)
		}
		client, err = smtp.NewClient(conn, s.config.SMTPHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
	} else {
		// Обычное соединение
		client, err = smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server: %w", err)
		}
	}
	defer client.Close()

	// STARTTLS если нужно
	if s.config.UseTLS && !s.config.UseSSL {
		if err = client.StartTLS(&tls.Config{ServerName: s.config.SMTPHost}); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	// Аутентификация
	if s.auth != nil {
		if err = client.Auth(s.auth); err != nil {
			return fmt.Errorf("failed to authenticate: %w", err)
		}
	}

	// Устанавливаем отправителя
	if err = client.Mail(s.config.FromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Устанавливаем получателей
	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Отправляем данные
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = w.Write(message)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return client.Quit()
}

// htmlToText преобразует HTML в простой текст
func (s *SMTPSender) htmlToText(html string) string {
	// Упрощенная конвертация
	text := strings.ReplaceAll(html, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<br>", "\n")
	text = strings.ReplaceAll(text, "<p>", "\n")
	text = strings.ReplaceAll(text, "</p>", "\n")

	// Удаляем HTML теги
	for {
		start := strings.Index(text, "<")
		if start == -1 {
			break
		}
		end := strings.Index(text, ">")
		if end == -1 {
			break
		}
		text = text[:start] + text[end+1:]
	}

	return strings.TrimSpace(text)
}

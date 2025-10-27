package email

import "html/template"

// Email представляет структуру email сообщения
type Email struct {
	To          []string
	Subject     string
	Body        string
	HTMLBody    string
	Attachments []Attachment
}

// Attachment представляет вложение в email
type Attachment struct {
	Name        string
	Content     []byte
	ContentType string
}

// TemplateData базовая структура для данных шаблонов
type TemplateData struct {
	UserName     string
	Subject      string
	Message      string
	ActionURL    string
	ActionText   string
	SupportEmail string
	CompanyName  string
}

// ResponseStatusData данные для шаблона статуса отклика
type ResponseStatusData struct {
	TemplateData
	CastingTitle string
	Status       string
	ModelName    string
	EmployerName string
}

// CastingMatchData данные для шаблона совпадения кастинга
type CastingMatchData struct {
	TemplateData
	CastingTitle string
	MatchScore   float64
	EmployerName string
	CastingDate  string
}

// WelcomeData данные для приветственного письма
type WelcomeData struct {
	TemplateData
	UserRole   string
	VerifyURL  string
	ProfileURL string
}

// Config конфигурация email сервиса
type Config struct {
	SMTPHost     string
	SMTPPort     int
	Username     string
	Password     string
	FromEmail    string
	FromName     string
	UseTLS       bool
	UseSSL       bool
	Timeout      int // in seconds
	TemplatePath string
}

// Sender интерфейс для отправки email
type Sender interface {
	Send(email *Email) error
	SendTemplate(to []string, subject, templateName string, data interface{}) error
	SendWelcome(email, name, userRole string) error
	SendVerification(email, token string) error
	SendResponseStatus(email, modelName, castingTitle, status string) error
	SendCastingMatch(email, modelName, castingTitle string, score float64) error
	SendNotification(email, subject, message string) error
	SendBulk(emails []*Email) error
}

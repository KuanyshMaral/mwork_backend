package email

// Attachment представляет вложение в email
type Attachment struct {
	Name        string
	Content     []byte
	ContentType string
}

// Email представляет структуру email сообщения
type Email struct {
	From        string
	To          []string
	Cc          []string
	Bcc         []string
	Subject     string
	Body        string
	HTMLBody    string
	Attachments []Attachment
}

// TemplateData представляет данные для шаблонов писем
type TemplateData map[string]interface{}

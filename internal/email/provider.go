package email

// Provider определяет интерфейс для отправки email
type Provider interface {
	// Send отправляет простое email сообщение
	Send(email *Email) error

	// SendWithTemplate отправляет email используя шаблон
	SendWithTemplate(templateName string, data TemplateData, email *Email) error

	// SendVerification отправляет email верификации
	SendVerification(email string, token string) error

	// SendTemplate отправляет email по шаблону (удобный метод)
	SendTemplate(to []string, subject string, templateName string, data TemplateData) error

	// Validate проверяет конфигурацию провайдера
	Validate() error

	// Close закрывает соединение с провайдером
	Close() error
}

// TemplateRenderer определяет интерфейс для рендеринга шаблонов
type TemplateRenderer interface {
	// Render рендерит шаблон с данными
	Render(templateName string, data TemplateData) (string, error)

	// AddTemplate добавляет шаблон в рендерер
	AddTemplate(name string, template string) error

	// LoadTemplates загружает шаблоны из директории
	LoadTemplates(dirPath string) error
}

package email

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
)

// TemplateManager управляет шаблонами email
type TemplateManager struct {
	templates map[string]*template.Template
	config    Config
}

// NewTemplateManager создает новый менеджер шаблонов
func NewTemplateManager(config Config) (*TemplateManager, error) {
	tm := &TemplateManager{
		templates: make(map[string]*template.Template),
		config:    config,
	}

	// Загружаем шаблоны
	templates := []string{
		"welcome",
		"notification",
		"response_status",
		"casting_match",
		"verification",
		"password_reset",
	}

	for _, name := range templates {
		tpl, err := tm.loadTemplate(name)
		if err != nil {
			return nil, fmt.Errorf("failed to load template %s: %w", name, err)
		}
		tm.templates[name] = tpl
	}

	return tm, nil
}

// loadTemplate загружает шаблон из файла
func (tm *TemplateManager) loadTemplate(name string) (*template.Template, error) {
	basePath := filepath.Join(tm.config.TemplatePath, "base.html")
	contentPath := filepath.Join(tm.config.TemplatePath, name+".html")

	// Сначала пробуем загрузить с базовым шаблоном
	tpl, err := template.ParseFiles(basePath, contentPath)
	if err != nil {
		// Если базового шаблона нет, загружаем только контент
		tpl, err = template.ParseFiles(contentPath)
		if err != nil {
			// Используем встроенные шаблоны как fallback
			return tm.getBuiltinTemplate(name)
		}
	}

	return tpl, nil
}

// getBuiltinTemplate возвращает встроенные шаблоны
func (tm *TemplateManager) getBuiltinTemplate(name string) (*template.Template, error) {
	var tplText string

	switch name {
	case "welcome":
		tplText = welcomeTemplate
	case "notification":
		tplText = notificationTemplate
	case "response_status":
		tplText = responseStatusTemplate
	case "casting_match":
		tplText = castingMatchTemplate
	case "verification":
		tplText = verificationTemplate
	case "password_reset":
		tplText = passwordResetTemplate
	default:
		return nil, fmt.Errorf("unknown template: %s", name)
	}

	return template.New(name).Parse(tplText)
}

// Render рендерит шаблон с данными
func (tm *TemplateManager) Render(templateName string, data interface{}) (string, error) {
	tpl, exists := tm.templates[templateName]
	if !exists {
		return "", fmt.Errorf("template not found: %s", templateName)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return buf.String(), nil
}

// Встроенные шаблоны как fallback
const (
	welcomeTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Добро пожаловать в MWORK</title>
</head>
<body>
    <h1>Добро пожаловать, {{.UserName}}!</h1>
    <p>Спасибо за регистрацию в MWORK - платформе для моделей и работодателей.</p>
    {{if .VerifyURL}}
    <p>Для завершения регистрации, пожалуйста, подтвердите ваш email:</p>
    <a href="{{.VerifyURL}}" style="background-color: #007bff; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Подтвердить Email</a>
    {{end}}
    <p>Если у вас есть вопросы, обращайтесь в поддержку: {{.SupportEmail}}</p>
</body>
</html>`

	notificationTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{{.Subject}}</title>
</head>
<body>
    <h2>{{.Subject}}</h2>
    <p>{{.Message}}</p>
    {{if .ActionURL}}
    <a href="{{.ActionURL}}" style="background-color: #28a745; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">{{.ActionText}}</a>
    {{end}}
</body>
</html>`

	responseStatusTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Обновление статуса отклика</title>
</head>
<body>
    <h2>Статус вашего отклика обновлен</h2>
    <p>Здравствуйте, {{.ModelName}}!</p>
    <p>Ваш отклик на кастинг "{{.CastingTitle}}" был изменен на: <strong>{{.Status}}</strong></p>
    {{if eq .Status "accepted"}}
    <p>Поздравляем! Работодатель принял ваш отклик. Свяжитесь с ним для уточнения деталей.</p>
    {{else if eq .Status "rejected"}}
    <p>К сожалению, в этот раз работодатель выбрал другого кандидата. Не расстраивайтесь!</p>
    {{end}}
    <p>С уважением, команда MWORK</p>
</body>
</html>`

	castingMatchTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Новый подходящий кастинг</title>
</head>
<body>
    <h2>Для вас найден подходящий кастинг!</h2>
    <p>Здравствуйте, {{.UserName}}!</p>
    <p>Мы нашли кастинг, который идеально соответствует вашему профилю:</p>
    <div style="background-color: #f8f9fa; padding: 15px; border-radius: 5px;">
        <h3>{{.CastingTitle}}</h3>
        <p><strong>Совпадение:</strong> {{.MatchScore}}%</p>
        {{if .CastingDate}}<p><strong>Дата кастинга:</strong> {{.CastingDate}}</p>{{end}}
    </div>
    <p>Не упустите возможность! Посмотрите детали кастинга и отправьте отклик.</p>
    <a href="{{.ActionURL}}" style="background-color: #007bff; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Посмотреть кастинг</a>
</body>
</html>`

	verificationTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Подтверждение Email</title>
</head>
<body>
    <h2>Подтвердите ваш Email</h2>
    <p>Для завершения регистрации на платформе MWORK, пожалуйста, подтвердите ваш email адрес.</p>
    <a href="{{.ActionURL}}" style="background-color: #007bff; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Подтвердить Email</a>
    <p>Если вы не регистрировались на MWORK, просто проигнорируйте это письмо.</p>
</body>
</html>`

	passwordResetTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Сброс пароля</title>
</head>
<body>
    <h2>Сброс пароля</h2>
    <p>Вы запросили сброс пароля для вашего аккаунта на MWORK.</p>
    <p>Для установки нового пароля перейдите по ссылке:</p>
    <a href="{{.ActionURL}}" style="background-color: #dc3545; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px;">Сбросить пароль</a>
    <p>Ссылка действительна в течение 24 часов.</p>
    <p>Если вы не запрашивали сброс пароля, проигнорируйте это письмо.</p>
</body>
</html>`
)

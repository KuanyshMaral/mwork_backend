package email

import "fmt"

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() Config {
	return Config{
		SMTPHost:     "localhost",
		SMTPPort:     587,
		Username:     "",
		Password:     "",
		FromEmail:    "noreply@mwork.ru",
		FromName:     "MWORK Platform",
		UseTLS:       true,
		UseSSL:       false,
		Timeout:      30,
		TemplatePath: "./templates/email",
	}
}

// Validate проверяет валидность конфигурации
func (c Config) Validate() error {
	if c.SMTPHost == "" {
		return fmt.Errorf("SMTP host is required")
	}
	if c.SMTPPort <= 0 || c.SMTPPort > 65535 {
		return fmt.Errorf("invalid SMTP port: %d", c.SMTPPort)
	}
	if c.FromEmail == "" {
		return fmt.Errorf("from email is required")
	}
	return nil
}

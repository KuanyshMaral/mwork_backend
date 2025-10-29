package email

import "time"

// SMTPConfig содержит конфигурацию SMTP сервера
type SMTPConfig struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromEmail string
	FromName  string
	UseTLS    bool
	Timeout   time.Duration
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *SMTPConfig {
	return &SMTPConfig{
		Host:    "localhost",
		Port:    587,
		UseTLS:  true,
		Timeout: 30 * time.Second,
	}
}

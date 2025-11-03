package app

import "mwork_backend/internal/email"

// MockEmailProvider используется для тестов и локальной разработки.
type MockEmailProvider struct{}

func (m *MockEmailProvider) Send(email *email.Email) error { return nil }
func (m *MockEmailProvider) SendWithTemplate(templateName string, data email.TemplateData, emailMsg *email.Email) error {
	return nil
}
func (m *MockEmailProvider) SendVerification(email string, token string) error { return nil }
func (m *MockEmailProvider) SendTemplate(to []string, subject string, templateName string, data email.TemplateData) error {
	return nil
}
func (m *MockEmailProvider) Validate() error { return nil }
func (m *MockEmailProvider) Close() error    { return nil }

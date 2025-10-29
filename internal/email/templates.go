package email

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// TemplateManager реализует TemplateRenderer для управления шаблонами email
type TemplateManager struct {
	templates map[string]*template.Template
	mutex     sync.RWMutex
}

// NewTemplateManager создает новый менеджер шаблонов
func NewTemplateManager() *TemplateManager {
	return &TemplateManager{
		templates: make(map[string]*template.Template),
	}
}

// Render рендерит шаблон с данными
func (tm *TemplateManager) Render(templateName string, data TemplateData) (string, error) {
	tm.mutex.RLock()
	tpl, exists := tm.templates[templateName]
	tm.mutex.RUnlock()

	if !exists {
		return "", fmt.Errorf("template not found: %s", templateName)
	}

	var buf strings.Builder
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// AddTemplate добавляет шаблон в менеджер
func (tm *TemplateManager) AddTemplate(name string, templateStr string) error {
	tpl, err := template.New(name).Parse(templateStr)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	tm.mutex.Lock()
	tm.templates[name] = tpl
	tm.mutex.Unlock()

	return nil
}

// LoadTemplates загружает шаблоны из директории
func (tm *TemplateManager) LoadTemplates(dirPath string) error {
	return filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read template file %s: %w", path, err)
		}

		templateName := strings.TrimSuffix(filepath.Base(path), ".html")
		if err := tm.AddTemplate(templateName, string(content)); err != nil {
			return fmt.Errorf("failed to add template %s: %w", templateName, err)
		}

		return nil
	})
}

// GetTemplate возвращает шаблон по имени (для тестирования)
func (tm *TemplateManager) GetTemplate(name string) *template.Template {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()
	return tm.templates[name]
}

// TemplateNames возвращает список имен загруженных шаблонов
func (tm *TemplateManager) TemplateNames() []string {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	names := make([]string, 0, len(tm.templates))
	for name := range tm.templates {
		names = append(names, name)
	}

	return names
}

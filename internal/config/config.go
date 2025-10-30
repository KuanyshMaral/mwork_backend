package config

import (
	"log"
	"os"

	"strconv"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
		Env  string `yaml:"env"` // <-- ✅ ДОБАВЛЕНО ЭТО ПОЛЕ
	} `yaml:"server"`

	Database struct {
		DSN string `yaml:"url"`
	} `yaml:"database"`

	Email struct {
		// --- ОБЯЗАТЕЛЬНЫЕ ПОЛЯ (из вашего services/services.go) ---

		SMTPHost string `yaml:"smtp_host"`
		SMTPPort int    `yaml:"smtp_port"`

		// ИСПРАВЛЕНИЕ: В сервисе поле называется SMTPUsername, а не SMTPUser
		// РЕКОМЕНДАЦИЯ: Для ясности, лучше использовать yaml:"smtp_username"
		SMTPUsername string `yaml:"smtp_user"`

		SMTPPassword string `yaml:"smtp_password"`
		FromEmail    string `yaml:"from_email"`

		// ИСПРАВЛЕНИЕ: Эти поля НУЖНЫ для NewEmailServiceWithConfig
		FromName     string `yaml:"from_name"`     // Нужно добавить в config.yaml
		UseTLS       bool   `yaml:"use_tls"`       // Нужно добавить в config.yaml
		TemplatesDir string `yaml:"templates_dir"` // Нужно добавить в config.yaml
	} `yaml:"email"`

	JWT struct {
		Secret string `yaml:"secret"`
		TTL    int    `yaml:"ttl"` // в минутах
	} `yaml:"jwt"`
}

var AppConfig *Config

func LoadConfig() {
	var cfg Config

	// 1. Пытаемся прочитать из ENV-переменных.
	// Ты сам задаешь их в auth_test.go -> TestMain
	dbURL := os.Getenv("DATABASE_URL")
	serverEnv := os.Getenv("SERVER_ENV")
	portStr := os.Getenv("SERVER_PORT")
	jwtSecret := os.Getenv("JWT_SECRET") // 👈 (См. Шаг 2)

	// Если мы не нашли DATABASE_URL, значит, мы не в тесте.
	// Пытаемся загрузиться из YAML (старый способ)
	if dbURL == "" {
		log.Println("Загрузка из config.yaml (режим НЕ-тест)")

		// 1. Загружаем из YAML
		configPath := os.Getenv("CONFIG_PATH")
		if configPath == "" {
			configPath = "config/config.yaml" // Твой путь по умолчанию
		}

		f, err := os.Open(configPath)
		if err != nil {
			log.Fatalf("Failed to open config file at %s: %v", configPath, err)
		}
		defer f.Close()

		decoder := yaml.NewDecoder(f)
		if err := decoder.Decode(&cfg); err != nil {
			log.Fatalf("Failed to parse config file at %s: %v", configPath, err)
		}

		AppConfig = &cfg
		return // 👈 Важно: выходим
	}

	// --- ЕСЛИ МЫ ЗДЕСЬ, ЗНАЧИТ dbURL БЫЛА НАЙДЕНА (мы в тесте) ---
	log.Println("✅ Загрузка конфигурации из ПЕРЕМЕННЫХ ОКРУЖЕНИЯ (режим теста)")

	// 2. Собираем конфиг из ENV
	cfg.Database.DSN = dbURL
	cfg.Server.Env = serverEnv
	cfg.Server.Port, _ = strconv.Atoi(portStr)

	// 3. Заполняем остальные важные поля (иначе они будут пустые)
	cfg.JWT.Secret = jwtSecret
	cfg.JWT.TTL = 60 // 60 минут для тестов

	// 4. Заполни поля Email, если они нужны для SetupRouter
	// (для тестов можно использовать "заглушки")
	cfg.Email.SMTPHost = "smtp.test.com"
	cfg.Email.SMTPPort = 587
	cfg.Email.FromEmail = "test@mwork.com"
	cfg.Email.TemplatesDir = "templates" // 👈 Убедись, что путь 'templates' виден из корня

	AppConfig = &cfg
}

func GetConfig() *Config {
	if AppConfig == nil {
		// Эта "защита" нужна, если кто-то вызовет GetConfig() до LoadConfig()
		LoadConfig()
	}
	return AppConfig
}

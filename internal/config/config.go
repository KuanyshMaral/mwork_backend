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
		Env  string `yaml:"env"`
	} `yaml:"server"`

	Database struct {
		DSN string `yaml:"url"`
	} `yaml:"database"`

	Email struct {
		SMTPHost     string `yaml:"smtp_host"`
		SMTPPort     int    `yaml:"smtp_port"`
		SMTPUsername string `yaml:"smtp_user"`
		SMTPPassword string `yaml:"smtp_password"`
		FromEmail    string `yaml:"from_email"`
		FromName     string `yaml:"from_name"`
		UseTLS       bool   `yaml:"use_tls"`
		TemplatesDir string `yaml:"templates_dir"`
	} `yaml:"email"`

	JWT struct {
		Secret string `yaml:"secret"`
		TTL    int    `yaml:"ttl"`
	} `yaml:"jwt"`

	Storage struct {
		Type       string `yaml:"type"`        // local, s3, cloudflare_r2
		BasePath   string `yaml:"base_path"`   // For local storage
		BaseURL    string `yaml:"base_url"`    // Public URL base
		Bucket     string `yaml:"bucket"`      // For S3/R2
		Region     string `yaml:"region"`      // For S3
		AccessKey  string `yaml:"access_key"`  // For S3/R2
		SecretKey  string `yaml:"secret_key"`  // For S3/R2
		Endpoint   string `yaml:"endpoint"`    // For R2 or custom S3
		UseSSL     bool   `yaml:"use_ssl"`     // For S3/R2
		PublicRead bool   `yaml:"public_read"` // Make files public
	} `yaml:"storage"`

	Upload struct {
		MaxSize        int64    `yaml:"max_size"`         // Max file size in bytes
		MaxUserStorage int64    `yaml:"max_user_storage"` // Max storage per user
		AllowedTypes   []string `yaml:"allowed_types"`    // Allowed MIME types
		ImageQuality   int      `yaml:"image_quality"`    // JPEG quality (1-100)
	} `yaml:"upload"`
}

var AppConfig *Config

func LoadConfig() {
	var cfg Config

	dbURL := os.Getenv("DATABASE_URL")
	serverEnv := os.Getenv("SERVER_ENV")
	portStr := os.Getenv("SERVER_PORT")
	jwtSecret := os.Getenv("JWT_SECRET")

	if dbURL == "" {
		log.Println("Загрузка из config.yaml (режим НЕ-тест)")

		configPath := os.Getenv("CONFIG_PATH")
		if configPath == "" {
			configPath = "config/config.yaml"
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
		initPortfolioFileConfig()
		return
	}

	log.Println("✅ Загрузка конфигурации из ПЕРЕМЕННЫХ ОКРУЖЕНИЯ (режим теста)")

	cfg.Database.DSN = dbURL
	cfg.Server.Env = serverEnv
	cfg.Server.Port, _ = strconv.Atoi(portStr)
	cfg.JWT.Secret = jwtSecret
	cfg.JWT.TTL = 60

	cfg.Email.SMTPHost = "smtp.test.com"
	cfg.Email.SMTPPort = 587
	cfg.Email.FromEmail = "test@mwork.com"
	cfg.Email.TemplatesDir = "templates"

	cfg.Storage.Type = "local"
	cfg.Storage.BasePath = "./uploads"
	cfg.Storage.BaseURL = "/api/v1/files"

	cfg.Upload.MaxSize = 10 * 1024 * 1024         // 10MB
	cfg.Upload.MaxUserStorage = 100 * 1024 * 1024 // 100MB
	cfg.Upload.AllowedTypes = []string{
		"image/jpeg", "image/png", "image/gif", "image/webp",
		"video/mp4", "video/quicktime",
	}
	cfg.Upload.ImageQuality = 85

	AppConfig = &cfg
	initPortfolioFileConfig()
}

func initPortfolioFileConfig() {
	PortfolioFileConfig.MaxSize = AppConfig.Upload.MaxSize
	PortfolioFileConfig.AllowedTypes = AppConfig.Upload.AllowedTypes
	PortfolioFileConfig.StoragePath = AppConfig.Storage.BasePath
	PortfolioFileConfig.MaxUserStorage = AppConfig.Upload.MaxUserStorage
	PortfolioFileConfig.AllowedUsages = map[string][]string{
		"model_profile": {"avatar", "cover_photo"},
		"portfolio":     {"portfolio_photo", "portfolio_video"},
		"casting":       {"casting_attachment", "casting_photo"},
	}
}

func GetConfig() *Config {
	if AppConfig == nil {
		LoadConfig()
	}
	return AppConfig
}

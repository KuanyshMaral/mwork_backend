package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv" // Import the godotenv package
)

type Config struct {
	Server struct {
		Host string
		Port int
		Env  string
	}

	Database struct {
		DSN string
	}

	Email struct {
		SMTPHost     string
		SMTPPort     int
		SMTPUsername string
		SMTPPassword string
		FromEmail    string
		FromName     string
		UseTLS       bool
		TemplatesDir string
	}

	JWT struct {
		Secret string
		TTL    int
	}

	Storage struct {
		Type       string
		BasePath   string
		BaseURL    string
		Bucket     string
		Region     string
		AccessKey  string
		SecretKey  string
		Endpoint   string
		UseSSL     bool
		PublicRead bool
	}

	Upload struct {
		MaxSize        int64
		MaxUserStorage int64
		AllowedTypes   []string
		ImageQuality   int
	}

	FirstAdminEmail    string `mapstructure:"FIRST_ADMIN_EMAIL"`
	FirstAdminPassword string `mapstructure:"FIRST_ADMIN_PASSWORD"`
}

// This struct was implied by your initPortfolioFileConfig function.
// I've added its definition here so the file is complete.
type PortfolioFileConfigType struct {
	MaxSize        int64
	AllowedTypes   []string
	StoragePath    string
	MaxUserStorage int64
	AllowedUsages  map[string][]string
}

var AppConfig *Config

func LoadConfig() {
	// Load .env file *before* reading any environment variables
	// This will load variables from a .env file in the working directory
	err := godotenv.Load()
	if err != nil {
		// It's okay if the .env file doesn't exist, we'll just use system env vars
		log.Println("Warning: Could not load .env file. Using system environment variables only.")
	} else {
		log.Println("Loaded configuration from .env file")
	}

	var cfg Config

	// Server Configuration
	cfg.Server.Host = getEnv("SERVER_HOST", "localhost")
	cfg.Server.Port = getEnvAsInt("SERVER_PORT", 4000)
	cfg.Server.Env = getEnv("SERVER_ENV", "development")

	// Database Configuration
	cfg.Database.DSN = getEnv("DATABASE_DSN", "")
	if cfg.Database.DSN == "" {
		// This is the error you were seeing
		log.Fatal("DATABASE_DSN environment variable is required (not found in .env or system env)")
	}

	// Email Configuration
	cfg.Email.SMTPHost = getEnv("EMAIL_SMTP_HOST", "smtp.gmail.com")
	cfg.Email.SMTPPort = getEnvAsInt("EMAIL_SMTP_PORT", 587)
	cfg.Email.SMTPUsername = getEnv("EMAIL_SMTP_USER", "")
	cfg.Email.SMTPPassword = getEnv("EMAIL_SMTP_PASSWORD", "")
	cfg.Email.FromEmail = getEnv("EMAIL_FROM_EMAIL", "noreply@mwork.com")
	cfg.Email.FromName = getEnv("EMAIL_FROM_NAME", "MWork")
	cfg.Email.UseTLS = getEnvAsBool("EMAIL_USE_TLS", true)
	cfg.Email.TemplatesDir = getEnv("EMAIL_TEMPLATES_DIR", "templates")

	// JWT Configuration
	cfg.JWT.Secret = getEnv("JWT_SECRET", "")
	if cfg.JWT.Secret == "" {
		log.Fatal("JWT_SECRET environment variable is required (not found in .env or system env)")
	}
	cfg.JWT.TTL = getEnvAsInt("JWT_TTL", 1440) // 24 hours in minutes

	// Storage Configuration
	cfg.Storage.Type = getEnv("STORAGE_TYPE", "local")
	cfg.Storage.BasePath = getEnv("STORAGE_BASE_PATH", "./uploads")
	cfg.Storage.BaseURL = getEnv("STORAGE_BASE_URL", "/api/v1/files")
	cfg.Storage.Bucket = getEnv("STORAGE_BUCKET", "")
	cfg.Storage.Region = getEnv("STORAGE_REGION", "")
	cfg.Storage.AccessKey = getEnv("STORAGE_ACCESS_KEY", "")
	cfg.Storage.SecretKey = getEnv("STORAGE_SECRET_KEY", "")
	cfg.Storage.Endpoint = getEnv("STORAGE_ENDPOINT", "")
	cfg.Storage.UseSSL = getEnvAsBool("STORAGE_USE_SSL", false)
	cfg.Storage.PublicRead = getEnvAsBool("STORAGE_PUBLIC_READ", true)

	// Upload Configuration
	cfg.Upload.MaxSize = getEnvAsInt64("UPLOAD_MAX_SIZE", 10485760)                 // 10MB
	cfg.Upload.MaxUserStorage = getEnvAsInt64("UPLOAD_MAX_USER_STORAGE", 104857600) // 100MB
	cfg.Upload.AllowedTypes = getEnvAsSlice("UPLOAD_ALLOWED_TYPES", []string{
		"image/jpeg", "image/png", "image/gif", "image/webp",
		"video/mp4", "video/quicktime",
	})
	cfg.Upload.ImageQuality = getEnvAsInt("UPLOAD_IMAGE_QUALITY", 85)

	cfg.FirstAdminEmail = getEnv("FIRST_ADMIN_EMAIL", "")
	cfg.FirstAdminPassword = getEnv("FIRST_ADMIN_PASSWORD", "")

	AppConfig = &cfg

	// Now this function will work correctly
	initPortfolioFileConfig()

	log.Printf("✅ Configuration loaded (env: %s)", cfg.Server.Env)
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

// getEnv retrieves a string from environment variables or returns a default value
func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvAsInt retrieves an integer from environment variables or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Warning: Invalid integer value for %s, using default: %d", key, defaultValue)
		return defaultValue
	}
	return value
}

// getEnvAsInt64 retrieves an int64 from environment variables or returns a default value
func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		log.Printf("Warning: Invalid int64 value for %s, using default: %d", key, defaultValue)
		return defaultValue
	}
	return value
}

// getEnvAsBool retrieves a boolean from environment variables or returns a default value
func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		log.Printf("Warning: Invalid boolean value for %s, using default: %t", key, defaultValue)
		return defaultValue
	}
	return value
}

// getEnvAsSlice retrieves a comma-separated string from environment variables,
// splits it into a slice, or returns a default value
func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	// Split by comma and trim spaces
	values := strings.Split(valueStr, ",")
	if len(values) == 0 {
		return defaultValue
	}

	result := make([]string, 0, len(values))
	for _, v := range values {
		trimmedV := strings.TrimSpace(v)
		if trimmedV != "" {
			result = append(result, trimmedV)
		}
	}

	if len(result) == 0 {
		return defaultValue
	}

	return result
}

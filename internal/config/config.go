package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"server"`

	Database struct {
		DSN string `yaml:"url"`
	} `yaml:"database"`

	Email struct {
		SMTPHost     string `yaml:"smtp_host"`
		SMTPPort     int    `yaml:"smtp_port"`
		SMTPUser     string `yaml:"smtp_user"`
		SMTPPassword string `yaml:"smtp_password"`
		FromEmail    string `yaml:"from_email"`
	} `yaml:"email"`

	JWT struct {
		Secret string `yaml:"secret"`
		TTL    int    `yaml:"ttl"` // в минутах
	} `yaml:"jwt"`
}

var AppConfig *Config

func LoadConfig() {
	f, err := os.Open("C:/Users/mrmar/GolandProjects/mwork-front-fn/config/config.yaml")
	if err != nil {
		panic("Failed to open config.yaml: " + err.Error())
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		panic("Failed to parse config.yaml: " + err.Error())
	}

	AppConfig = &cfg
}

func GetConfig() *Config {
	if AppConfig == nil {
		LoadConfig()
	}
	return AppConfig
}

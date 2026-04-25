package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type SupabaseConfig struct {
	URL    string
	Key    string
	Bucket string
}

type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
}

type Config struct {
	HttpPort       int
	DatabaseURL    string
	RedisURL       string
	JwtSecretKey   string
	BaseURL        string
	AllowedOrigins string
	Supabase       SupabaseConfig
	SMTP           SMTPConfig
}

var configurations *Config

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

func loadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	// Try HTTP_PORT first, then fallback to PORT (Render/Railway standard)
	portStr := getEnv("HTTP_PORT", getEnv("PORT", "8080"))
	httpPort, _ := strconv.Atoi(portStr)
	smtpPort, _ := strconv.Atoi(getEnv("SMTP_PORT", "587"))

	configurations = &Config{
		HttpPort:       httpPort,
		JwtSecretKey:   getEnv("JWT_SECRET_KEY", "secret"),
		DatabaseURL:    getEnv("DATABASE_URL", ""),
		RedisURL:       getEnv("REDIS_URL", ""),
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", ""),
		BaseURL:        getEnv("BASE_URL", "http://localhost:8080"),
		Supabase: SupabaseConfig{
			URL:    getEnv("SUPABASE_URL", ""),
			Key:    getEnv("SUPABASE_SERVICE_KEY", ""),
			Bucket: getEnv("SUPABASE_BUCKET", "eraya"),
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", ""),
			Port:     smtpPort,
			User:     getEnv("SMTP_USER", ""),
			Password: getEnv("SMTP_PASS", ""),
		},
	}
}

func GetConfig() *Config {
	if configurations == nil {
		loadConfig()
	}
	return configurations
}

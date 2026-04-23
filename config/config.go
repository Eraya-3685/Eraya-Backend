package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type RedisConfig struct {
	Addr     string
	Password string
}

type Config struct {
	HttpPort       int
	DatabaseURL    string
	JwtSecretKey   string
	BaseURL        string
	AllowedOrigins string
	Redis          *RedisConfig
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

	configurations = &Config{
		HttpPort:       httpPort,
		JwtSecretKey:   getEnv("JWT_SECRET_KEY", "secret"),
		DatabaseURL:    getEnv("DATABASE_URL", ""),
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", ""),
		BaseURL:        getEnv("BASE_URL", "http://localhost:8080"),
		Redis: &RedisConfig{
			Addr:     getEnv("REDIS_ADDRESS", getEnv("REDIS_URL", "localhost:6379")),
			Password: getEnv("REDIS_PASSWORD", ""),
		},
	}
}

func GetConfig() *Config {
	if configurations == nil {
		loadConfig()
	}
	return configurations
}

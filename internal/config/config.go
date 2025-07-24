package config

import (
	"os"
	"strconv"
)

type Config struct {
	Host      string
	Port      int
	ZepAPIURL string
	ZepAPIKey string
}

func Load() *Config {
	return &Config{
		Host:      getEnv("HOST", "0.0.0.0"),
		Port:      getEnvInt("PORT", 8082),
		ZepAPIURL: getEnv("ZEP_API_URL", "http://zep-zerver.railway.internal:8080"),
		ZepAPIKey: getEnv("ZEP_API_KEY", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
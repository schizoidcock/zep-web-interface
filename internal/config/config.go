package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Host         string
	Port         int
	ZepAPIURL    string
	ZepAPIKey    string
	ProxyURL     string
	TLSEnabled   bool
	CORSOrigins  []string
	TrustProxy   bool
	ProxyPath    string
}

func Load() *Config {
	cfg := &Config{
		Host:         getEnv("HOST", "0.0.0.0"),
		Port:         getEnvInt("PORT", 8080),
		ZepAPIURL:    getEnv("ZEP_API_URL", ""),
		ZepAPIKey:    getEnv("ZEP_API_KEY", ""),
		ProxyURL:     getEnv("PROXY_URL", ""),
		TLSEnabled:   getEnvBool("TLS_ENABLED", false),
		CORSOrigins:  getEnvSlice("CORS_ORIGINS", []string{"*"}),
		TrustProxy:   getEnvBool("TRUST_PROXY", true),
		ProxyPath:    getEnv("PROXY_PATH", ""),
	}
	
	// Validate required configuration
	if err := cfg.validate(); err != nil {
		panic(fmt.Sprintf("Configuration validation failed: %v", err))
	}
	
	return cfg
}

func (c *Config) validate() error {
	// Required fields
	if c.ZepAPIURL == "" {
		return fmt.Errorf("ZEP_API_URL environment variable is required")
	}
	
	if c.ZepAPIKey == "" {
		return fmt.Errorf("ZEP_API_KEY environment variable is required")
	}
	
	// Validate ZEP_API_URL format
	if _, err := url.Parse(c.ZepAPIURL); err != nil {
		return fmt.Errorf("ZEP_API_URL is not a valid URL: %v", err)
	}
	
	// Validate port range
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("PORT must be between 1 and 65535, got: %d", c.Port)
	}
	
	// Validate host (basic check)
	if c.Host == "" {
		return fmt.Errorf("HOST cannot be empty")
	}
	
	// Check for problematic IPv6 addresses that need bracketing
	if c.Host == "::" {
		return fmt.Errorf("HOST=:: is not supported without brackets, use HOST=0.0.0.0 instead")
	}
	
	// Validate proxy URL if provided
	if c.ProxyURL != "" {
		if _, err := url.Parse(c.ProxyURL); err != nil {
			return fmt.Errorf("PROXY_URL is not a valid URL: %v", err)
		}
	}
	
	// Validate CORS origins
	if len(c.CORSOrigins) == 0 {
		return fmt.Errorf("CORS_ORIGINS cannot be empty")
	}
	
	for _, origin := range c.CORSOrigins {
		if origin != "*" {
			if _, err := url.Parse(origin); err != nil {
				return fmt.Errorf("invalid origin in CORS_ORIGINS: %s", origin)
			}
		}
	}
	
	return nil
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

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}
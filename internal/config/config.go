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
	
	// Service URLs for monitoring and logs
	FalkorDBServiceURL     string
	GraphitiServiceURL     string
	FalkorDBBrowserURL     string
	HybridProxyURL         string
	ZepServerURL           string
}

func Load() *Config {
	// Build ZEP API URL from components
	zepHost := getEnv("ZEP_API_URL", "")
	zepPort := getEnv("ZEP_SERVER_PORT", "")
	
	// Construct full URL with http:// prefix if needed
	var zepAPIURL string
	if zepHost != "" {
		// Add http:// prefix if not present
		if !strings.HasPrefix(zepHost, "http://") && !strings.HasPrefix(zepHost, "https://") {
			if zepPort != "" {
				// Only add port if explicitly provided
				zepAPIURL = fmt.Sprintf("http://%s:%s", zepHost, zepPort)
			} else {
				// No port specified, use just the hostname
				zepAPIURL = fmt.Sprintf("http://%s", zepHost)
			}
		} else {
			// If protocol is already present, just use the host as-is
			zepAPIURL = zepHost
		}
	}
	
	cfg := &Config{
		Host:         getEnv("HOST", "0.0.0.0"),
		Port:         getEnvInt("PORT", 8080),
		ZepAPIURL:    zepAPIURL,
		ZepAPIKey:    getEnv("ZEP_API_KEY", ""),
		ProxyURL:     getEnv("PROXY_URL", ""),
		TLSEnabled:   getEnvBool("TLS_ENABLED", false),
		CORSOrigins:  getEnvSlice("CORS_ORIGINS", []string{"*"}),
		TrustProxy:   getEnvBool("TRUST_PROXY", true),
		ProxyPath:    getEnv("PROXY_PATH", ""),
		
		// Service URLs - required environment variables
		FalkorDBServiceURL:     getEnv("FALKORDB_SERVICE_URL", ""),
		GraphitiServiceURL:     getEnv("GRAPHITI_SERVICE_URL", ""),
		FalkorDBBrowserURL:     getEnv("FALKORDB_BROWSER_URL", ""),
		HybridProxyURL:         getEnv("HYBRID_PROXY_URL", ""),
		ZepServerURL:           getEnv("ZEP_SERVER_URL", ""),
	}
	
	// Debug logging for API key (show first/last 8 chars for security)
	if len(cfg.ZepAPIKey) >= 16 {
		fmt.Printf("🔑 Using API key: %s...%s (length: %d)\n", 
			cfg.ZepAPIKey[:8], cfg.ZepAPIKey[len(cfg.ZepAPIKey)-8:], len(cfg.ZepAPIKey))
	} else {
		fmt.Printf("🔑 API key length: %d\n", len(cfg.ZepAPIKey))
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
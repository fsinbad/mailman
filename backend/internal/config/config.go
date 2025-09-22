package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	OpenAI   OpenAIConfig
	OAuth2   OAuth2Config
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Host string
	Port string
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	Driver   string
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// OpenAIConfig holds OpenAI-related configuration
type OpenAIConfig struct {
	BaseURL     string
	APIKey      string
	Model       string
	MaxTokens   int
	Temperature float64
}

// OAuth2Config holds OAuth2-related configuration
type OAuth2Config struct {
	Gmail   OAuth2ProviderConfig
	Outlook OAuth2ProviderConfig
}

// OAuth2ProviderConfig holds OAuth2 provider configuration
type OAuth2ProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "localhost"),
			Port: getEnv("SERVER_PORT", "8080"),
		},
		Database: DatabaseConfig{
			Driver:   getEnv("DB_DRIVER", "sqlite"),
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "mailman"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "mailman.db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		OpenAI: OpenAIConfig{
			BaseURL:     getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
			APIKey:      getEnv("OPENAI_API_KEY", ""),
			Model:       getEnv("OPENAI_MODEL", "gpt-3.5-turbo"),
			MaxTokens:   getEnvAsInt("OPENAI_MAX_TOKENS", 1000),
			Temperature: getEnvAsFloat("OPENAI_TEMPERATURE", 0.7),
		},
		OAuth2: OAuth2Config{
			Gmail: OAuth2ProviderConfig{
				ClientID:     getEnv("GMAIL_CLIENT_ID", ""),
				ClientSecret: getEnv("GMAIL_CLIENT_SECRET", ""),
				RedirectURI:  getEnv("GMAIL_REDIRECT_URI", "http://localhost:8080/api/oauth2/callback/gmail"),
			},
			Outlook: OAuth2ProviderConfig{
				ClientID:     getEnv("OUTLOOK_CLIENT_ID", ""),
				ClientSecret: getEnv("OUTLOOK_CLIENT_SECRET", ""),
				RedirectURI:  getEnv("OUTLOOK_REDIRECT_URI", "http://localhost:8080/api/oauth2/callback/outlook"),
			},
		},
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

// getEnvAsBool gets an environment variable as boolean or returns a default value
func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

// getEnvAsFloat gets an environment variable as float or returns a default value
func getEnvAsFloat(key string, defaultValue float64) float64 {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return value
	}
	return defaultValue
}

// ServerAddress returns the full server address
func (c *Config) ServerAddress() string {
	return fmt.Sprintf("%s:%s", c.Server.Host, c.Server.Port)
}

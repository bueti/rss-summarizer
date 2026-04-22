package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Database      DatabaseConfig
	Server        ServerConfig
	LLM           LLMConfig
	Temporal      TemporalConfig
	DevMode       DevModeConfig
	OAuth         OAuthConfig
	EncryptionKey string
}

type DatabaseConfig struct {
	URL string
}

type ServerConfig struct {
	Port         int
	Env          string
	FrontendURL  string
	PublicAPIURL string
}

type LLMConfig struct {
	APIURL string
	APIKey string
	Model  string
}

type TemporalConfig struct {
	Host      string
	Namespace string
}

type DevModeConfig struct {
	Enabled bool
	UserID  string
}

type OAuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	SessionDuration    time.Duration
}

func Load() (*Config, error) {
	// Load .env file if it exists (ignore error in production)
	_ = godotenv.Load()

	port, _ := strconv.Atoi(getEnv("PORT", "8080"))
	devMode, _ := strconv.ParseBool(getEnv("DEV_MODE", "false"))
	sessionDuration, _ := time.ParseDuration(getEnv("SESSION_DURATION", "720h")) // 30 days default

	cfg := &Config{
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", ""),
		},
		Server: ServerConfig{
			Port:         port,
			Env:          getEnv("ENV", "development"),
			FrontendURL:  getEnv("FRONTEND_URL", "http://localhost:5173"),
			PublicAPIURL: getEnv("PUBLIC_API_URL", "http://localhost:8080"),
		},
		LLM: LLMConfig{
			APIURL: getEnv("LLM_API_URL", "https://api.openai.com/v1"),
			APIKey: getEnv("LLM_API_KEY", ""),
			Model:  getEnv("LLM_MODEL", "gpt-4o-mini"),
		},
		Temporal: TemporalConfig{
			Host:      getEnv("TEMPORAL_HOST", "localhost:7233"),
			Namespace: getEnv("TEMPORAL_NAMESPACE", "default"),
		},
		DevMode: DevModeConfig{
			Enabled: devMode,
			UserID:  getEnv("DEV_USER_ID", "00000000-0000-0000-0000-000000000001"),
		},
		OAuth: OAuthConfig{
			GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
			GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
			GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/google/callback"),
			SessionDuration:    sessionDuration,
		},
		EncryptionKey: getEnv("ENCRYPTION_KEY", ""),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.LLM.APIKey == "" && c.Server.Env != "test" {
		return fmt.Errorf("LLM_API_KEY is required")
	}
	if c.EncryptionKey == "" {
		return fmt.Errorf("ENCRYPTION_KEY is required")
	}
	if len(c.EncryptionKey) != 32 {
		return fmt.Errorf("ENCRYPTION_KEY must be exactly 32 bytes (256 bits), got %d bytes", len(c.EncryptionKey))
	}
	// Validate OAuth config when not in dev mode
	if !c.DevMode.Enabled {
		if c.OAuth.GoogleClientID == "" {
			return fmt.Errorf("GOOGLE_CLIENT_ID is required when not in dev mode")
		}
		if c.OAuth.GoogleClientSecret == "" {
			return fmt.Errorf("GOOGLE_CLIENT_SECRET is required when not in dev mode")
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

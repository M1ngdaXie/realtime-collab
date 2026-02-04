package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	// Server
	Port        string
	Environment string // "development", "staging", "production"

	// Database
	DatabaseURL string

	// Redis (optional)
	RedisURL string

	// Security
	JWTSecret    string
	SecureCookie bool // Derived from Environment

	// CORS
	AllowedOrigins []string

	// URLs
	FrontendURL string
	BackendURL  string // Used to construct OAuth callback URLs

	// OAuth - Google
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	// OAuth - GitHub
	GitHubClientID     string
	GitHubClientSecret string
	GitHubRedirectURL  string
}

// Load loads configuration from environment variables with optional CLI flag overrides.
// flagPort overrides the PORT env var if non-empty.
func Load(flagPort string) (*Config, error) {
	// Load .env files (ignore errors - files may not exist)
	_ = godotenv.Load()
	_ = godotenv.Overload(".env.local")

	cfg := &Config{}

	// Required fields
	var missing []string

	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}

	cfg.JWTSecret = os.Getenv("JWT_SECRET")
	if cfg.JWTSecret == "" {
		missing = append(missing, "JWT_SECRET")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	// Port: CLI flag > env var > default
	cfg.Port = getEnvOrDefault("PORT", "8080")
	if flagPort != "" {
		cfg.Port = flagPort
	}

	// Other optional vars with defaults
	cfg.Environment = getEnvOrDefault("ENVIRONMENT", "development")
	cfg.FrontendURL = getEnvOrDefault("FRONTEND_URL", "http://localhost:5173")
	cfg.BackendURL = getEnvOrDefault("BACKEND_URL", fmt.Sprintf("http://localhost:%s", cfg.Port))
	cfg.RedisURL = os.Getenv("REDIS_URL")

	// Derived values
	cfg.SecureCookie = cfg.Environment == "production"

	// CORS origins
	originsStr := os.Getenv("ALLOWED_ORIGINS")
	if originsStr != "" {
		cfg.AllowedOrigins = strings.Split(originsStr, ",")
		for i := range cfg.AllowedOrigins {
			cfg.AllowedOrigins[i] = strings.TrimSpace(cfg.AllowedOrigins[i])
		}
	} else {
		cfg.AllowedOrigins = []string{
			"http://localhost:5173",
			"http://localhost:3000",
			"https://realtime-collab-snowy.vercel.app",
		}
	}

	// OAuth - Google
	cfg.GoogleClientID = os.Getenv("GOOGLE_CLIENT_ID")
	cfg.GoogleClientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	cfg.GoogleRedirectURL = getEnvOrDefault("GOOGLE_REDIRECT_URL",
		fmt.Sprintf("%s/api/auth/google/callback", cfg.BackendURL))

	// OAuth - GitHub
	cfg.GitHubClientID = os.Getenv("GITHUB_CLIENT_ID")
	cfg.GitHubClientSecret = os.Getenv("GITHUB_CLIENT_SECRET")
	cfg.GitHubRedirectURL = getEnvOrDefault("GITHUB_REDIRECT_URL",
		fmt.Sprintf("%s/api/auth/github/callback", cfg.BackendURL))

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// HasGoogleOAuth returns true if Google OAuth is configured
func (c *Config) HasGoogleOAuth() bool {
	return c.GoogleClientID != "" && c.GoogleClientSecret != ""
}

// HasGitHubOAuth returns true if GitHub OAuth is configured
func (c *Config) HasGitHubOAuth() bool {
	return c.GitHubClientID != "" && c.GitHubClientSecret != ""
}

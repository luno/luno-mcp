package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/luno/luno-go"
	"github.com/luno/luno-go/decimal"
	"github.com/luno/luno-mcp/sdk"
)

const (
	// Environment variables
	EnvLunoAPIKeyID         = "LUNO_API_KEY_ID"
	EnvLunoAPIKeySecret     = "LUNO_API_SECRET"
	EnvLunoAPIDomain        = "LUNO_API_DOMAIN"
	EnvLunoAPIDebug         = "LUNO_API_DEBUG"
	EnvAllowWriteOperations = "ALLOW_WRITE_OPERATIONS"

	// Default Luno API domain
	DefaultLunoDomain = "api.luno.com"
)

// Config holds the configuration for the application
type Config struct {
	// Luno client
	LunoClient sdk.LunoClient
	// IsAuthenticated indicates if the LunoClient is authenticated with API keys.
	// If false, only public API calls can be made.
	IsAuthenticated bool

	// AllowWriteOperations controls whether write operations (create_order, cancel_order) are exposed
	AllowWriteOperations bool
}

// Mask a string to show only the first 4 characters and replace the rest with asterisks
func maskValue(s string) string {
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", len(s)-4)
}

// Load loads the configuration from environment variables
func Load(domainOverride, appName, appVersion string) (*Config, error) {
	apiKeyID := os.Getenv(strings.TrimSpace(EnvLunoAPIKeyID))
	apiKeySecret := os.Getenv(strings.TrimSpace(EnvLunoAPIKeySecret))

	slog.Debug("Loaded API credentials", "key_id", maskValue(apiKeyID), "key_id_len", len(apiKeyID), "secret_len", len(apiKeySecret))

	lunoClient := luno.NewClient()
	trimmedAppName := strings.TrimSpace(appName)
	trimmedAppVersion := strings.TrimSpace(appVersion)
	if trimmedAppName != "" && trimmedAppVersion != "" {
		lunoClient.SetUserAgentSuffix(fmt.Sprintf("%s/%s", trimmedAppName, trimmedAppVersion))
	}

	cfg := &Config{
		LunoClient: lunoClient,
	}

	domain := DefaultLunoDomain

	if envDomain := os.Getenv(strings.TrimSpace(EnvLunoAPIDomain)); envDomain != "" {
		domain = envDomain
		slog.Debug("Using domain from environment variable", "domain", domain)
	}

	if domainOverride != "" {
		domain = domainOverride
		slog.Debug("Using domain from command line flag", "domain", domain)
	}

	if domain != DefaultLunoDomain {
		cfg.LunoClient.SetBaseURL(fmt.Sprintf("https://%s", domain))
	}

	if apiKeyID != "" && apiKeySecret != "" {
		err := cfg.LunoClient.SetAuth(apiKeyID, apiKeySecret)
		if err != nil {
			return nil, fmt.Errorf("failed to set Luno API credentials: %w", err)
		}
		cfg.IsAuthenticated = true
		slog.Info("Luno client authenticated with provided API credentials")
	} else {
		cfg.IsAuthenticated = false
		slog.Info("Luno API credentials not found, operating in unauthenticated mode")
	}

	debugMode := parseBoolEnv(EnvLunoAPIDebug)
	slog.Debug("Debug mode", "enabled", debugMode)
	cfg.LunoClient.SetDebug(debugMode)

	allowWriteOps := parseBoolEnv(EnvAllowWriteOperations)
	if allowWriteOps {
		slog.Info("Write operations enabled via environment variable")
	}
	cfg.AllowWriteOperations = allowWriteOps
	return cfg, nil
}

// parseBoolEnv returns true if the environment variable is set to "true", "1", or "yes" (case-insensitive).
func parseBoolEnv(key string) bool {
	val := os.Getenv(strings.TrimSpace(key))
	return strings.ToLower(val) == "true" ||
		val == "1" ||
		strings.ToLower(val) == "yes"
}

// FormatCurrency formats a decimal amount with the currency code
func FormatCurrency(amount decimal.Decimal, currency string) string {
	return fmt.Sprintf("%s %s", amount.String(), strings.ToUpper(currency))
}

package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/luno/luno-go"
	"github.com/luno/luno-go/decimal"
	"github.com/luno/luno-mcp/sdk"
)

const (
	// Environment variables
	EnvLunoAPIKeyID     = "LUNO_API_KEY_ID"
	EnvLunoAPIKeySecret = "LUNO_API_SECRET"
	EnvLunoAPIDomain    = "LUNO_API_DOMAIN"
	EnvLunoAPIDebug     = "LUNO_API_DEBUG"

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
}

// Mask a string to show only the first 4 characters and replace the rest with asterisks
func maskValue(s string) string {
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", len(s)-4)
}

// Load loads the configuration from environment variables
func Load(domainOverride string) (*Config, error) {
	apiKeyID := os.Getenv(strings.TrimSpace(EnvLunoAPIKeyID))
	apiKeySecret := os.Getenv(strings.TrimSpace(EnvLunoAPIKeySecret))

	fmt.Printf("LUNO_API_KEY_ID value: %s (length: %d)\n", maskValue(apiKeyID), len(apiKeyID))
	fmt.Printf("LUNO_API_SECRET value: %s (length: %d)\n", maskValue(apiKeySecret), len(apiKeySecret))

	cfg := &Config{
		LunoClient: luno.NewClient(),
	}

	// Set domain - first check command line override, then env var, then default
	domain := DefaultLunoDomain

	// Check for environment variable override
	if envDomain := os.Getenv(strings.TrimSpace(EnvLunoAPIDomain)); envDomain != "" {
		domain = envDomain
		fmt.Printf("Using domain from environment variable: %s\n", domain)
	}

	// Command line override takes precedence if provided
	if domainOverride != "" {
		domain = domainOverride
		fmt.Printf("Using domain from command line: %s\n", domain)
	}

	if domain != DefaultLunoDomain {
		cfg.LunoClient.SetBaseURL(fmt.Sprintf("https://%s", domain))
	}

	// Only set authentication if both API Key ID and Secret are provided
	if apiKeyID != "" && apiKeySecret != "" {
		err := cfg.LunoClient.SetAuth(apiKeyID, apiKeySecret)
		if err != nil {
			return nil, fmt.Errorf("failed to set Luno API credentials: %w", err)
		}
		cfg.IsAuthenticated = true
		fmt.Println("Luno client authenticated with provided API credentials.")
	} else {
		cfg.IsAuthenticated = false
		fmt.Println("Luno API credentials not found. Operating in unauthenticated mode.")
	}

	// Check if debug mode is enabled via environment variable
	debugMode := false
	if debugEnv := os.Getenv(strings.TrimSpace(EnvLunoAPIDebug)); debugEnv != "" {
		// Enable debug mode if environment variable is set to "true", "1", or "yes"
		debugMode = strings.ToLower(debugEnv) == "true" ||
			debugEnv == "1" ||
			strings.ToLower(debugEnv) == "yes"

		if debugMode {
			fmt.Println("Debug mode enabled via environment variable")
		}
	}

	cfg.LunoClient.SetDebug(debugMode)
	return cfg, nil
}

// FormatCurrency formats a decimal amount with the currency code
func FormatCurrency(amount decimal.Decimal, currency string) string {
	return fmt.Sprintf("%s %s", amount.String(), strings.ToUpper(currency))
}

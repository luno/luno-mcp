package config

import (
	"os"
	"strings"
	"testing"

	"github.com/luno/luno-go/decimal"
)

func TestMaskValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"single character", "a", "*"},
		{"two characters", "ab", "**"},
		{"three characters", "abc", "***"},
		{"four characters", "abcd", "****"},
		{"five characters", "abcde", "abcd*"},
		{"long string", "verylongstring", "very**********"},
		{"api key example", "testkey123", "test******"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := maskValue(tc.input)
			if result != tc.expected {
				t.Errorf("maskValue(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestFormatCurrency(t *testing.T) {
	tests := []struct {
		name     string
		amount   string
		currency string
		expected string
	}{
		{"bitcoin amount", "0.12345678", "btc", "0.12345678 BTC"},
		{"zar amount", "1234.56", "zar", "1234.56 ZAR"},
		{"zero amount", "0", "usd", "0 USD"},
		{"lowercase currency", "100", "eth", "100 ETH"},
		{"mixed case currency", "50", "GbP", "50 GBP"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			amount, err := decimal.NewFromString(tc.amount)
			if err != nil {
				t.Fatalf("Failed to create decimal: %v", err)
			}

			result := FormatCurrency(amount, tc.currency)
			if result != tc.expected {
				t.Errorf("FormatCurrency(%s, %q) = %q, want %q", tc.amount, tc.currency, result, tc.expected)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	originalAPIKeyID := os.Getenv(EnvLunoAPIKeyID)
	originalAPISecret := os.Getenv(EnvLunoAPIKeySecret)
	originalAPIDomain := os.Getenv(EnvLunoAPIDomain)
	originalAPIDebug := os.Getenv(EnvLunoAPIDebug)
	originalAllowWriteOps := os.Getenv(EnvAllowWriteOperations)

	defer func() {
		// Restore original environment
		setEnvVar(EnvLunoAPIKeyID, originalAPIKeyID)
		setEnvVar(EnvLunoAPIKeySecret, originalAPISecret)
		setEnvVar(EnvLunoAPIDomain, originalAPIDomain)
		setEnvVar(EnvLunoAPIDebug, originalAPIDebug)
		setEnvVar(EnvAllowWriteOperations, originalAllowWriteOps)
	}()

	tests := []struct {
		name                 string
		apiKeyID             string
		apiSecret            string
		domainEnv            string
		domainOverride       string
		debugEnv             string
		allowWriteOpsEnv     string
		expectedError        string
		expectedDomain       string
		expectedAllowWriteOps bool
	}{
		{
			name:                 "valid credentials with defaults",
			apiKeyID:             "test_key_id",
			apiSecret:            "test_secret",
			expectedDomain:       DefaultLunoDomain,
			expectedAllowWriteOps: false,
		},
		{
			name:          "missing api key id",
			apiKeyID:      "",
			apiSecret:     "test_secret",
			expectedError: "luno API credentials not found",
		},
		{
			name:          "missing api secret",
			apiKeyID:      "test_key_id",
			apiSecret:     "",
			expectedError: "luno API credentials not found",
		},
		{
			name:           "custom domain from environment",
			apiKeyID:       "test_key_id",
			apiSecret:      "test_secret",
			domainEnv:      "sandbox.luno.com",
			expectedDomain: "sandbox.luno.com",
		},
		{
			name:           "domain override takes precedence",
			apiKeyID:       "test_key_id",
			apiSecret:      "test_secret",
			domainEnv:      "env.luno.com",
			domainOverride: "override.luno.com",
			expectedDomain: "override.luno.com",
		},
		{
			name:      "debug mode enabled with true",
			apiKeyID:  "test_key_id",
			apiSecret: "test_secret",
			debugEnv:  "true",
		},
		{
			name:      "debug mode enabled with 1",
			apiKeyID:  "test_key_id",
			apiSecret: "test_secret",
			debugEnv:  "1",
		},
		{
			name:      "debug mode enabled with yes",
			apiKeyID:  "test_key_id",
			apiSecret: "test_secret",
			debugEnv:  "yes",
		},
		{
			name:      "debug mode disabled with false",
			apiKeyID:  "test_key_id",
			apiSecret: "test_secret",
			debugEnv:  "false",
		},
		{
			name:                 "write operations enabled with true",
			apiKeyID:             "test_key_id",
			apiSecret:            "test_secret",
			allowWriteOpsEnv:     "true",
			expectedAllowWriteOps: true,
		},
		{
			name:                 "write operations enabled with 1",
			apiKeyID:             "test_key_id",
			apiSecret:            "test_secret",
			allowWriteOpsEnv:     "1",
			expectedAllowWriteOps: true,
		},
		{
			name:                 "write operations enabled with yes",
			apiKeyID:             "test_key_id",
			apiSecret:            "test_secret",
			allowWriteOpsEnv:     "yes",
			expectedAllowWriteOps: true,
		},
		{
			name:                 "write operations disabled with false",
			apiKeyID:             "test_key_id",
			apiSecret:            "test_secret",
			allowWriteOpsEnv:     "false",
			expectedAllowWriteOps: false,
		},
		{
			name:                 "write operations disabled with empty",
			apiKeyID:             "test_key_id",
			apiSecret:            "test_secret",
			allowWriteOpsEnv:     "",
			expectedAllowWriteOps: false,
		},
		{
			name:                 "write operations disabled with invalid value",
			apiKeyID:             "test_key_id",
			apiSecret:            "test_secret",
			allowWriteOpsEnv:     "invalid",
			expectedAllowWriteOps: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variables
			setEnvVar(EnvLunoAPIKeyID, tc.apiKeyID)
			setEnvVar(EnvLunoAPIKeySecret, tc.apiSecret)
			setEnvVar(EnvLunoAPIDomain, tc.domainEnv)
			setEnvVar(EnvLunoAPIDebug, tc.debugEnv)
			setEnvVar(EnvAllowWriteOperations, tc.allowWriteOpsEnv)

			cfg, err := Load(tc.domainOverride)

			if tc.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing %q, but got nil", tc.expectedError)
					return
				}
				if !strings.Contains(err.Error(), tc.expectedError) {
					t.Errorf("Expected error containing %q, got %q", tc.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if cfg == nil {
				t.Error("Expected config to be non-nil")
				return
			}

			if cfg.LunoClient == nil {
				t.Error("Expected LunoClient to be non-nil")
			}

			if cfg.AllowWriteOperations != tc.expectedAllowWriteOps {
				t.Errorf("Expected AllowWriteOperations to be %v, got %v", tc.expectedAllowWriteOps, cfg.AllowWriteOperations)
			}
		})
	}
}

// Helper function to set environment variable, handling empty values
func setEnvVar(key, value string) {
	if value == "" {
		os.Unsetenv(key)
	} else {
		os.Setenv(key, value)
	}
}

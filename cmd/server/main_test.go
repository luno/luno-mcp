package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/luno/luno-mcp/internal/config"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDefaultSSEAddr   = "localhost:8080"
	testCustomSSEAddr    = "127.0.0.1:9000"
	testStagingDomain    = "staging.api.luno.com"
	testCustomDomain     = "test.api.luno.com"
	testCustomSSEAddrAlt = "0.0.0.0:8888"
	testLogLevelInfo     = "info"
	testLogLevelDebug    = "debug"
	testLogLevelError    = "error"
	testTransportStdio   = "stdio"
	testTransportSSE     = "sse"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected slog.Level
	}{
		{
			name:     "debug level",
			level:    "debug",
			expected: slog.LevelDebug,
		},
		{
			name:     "info level",
			level:    "info",
			expected: slog.LevelInfo,
		},
		{
			name:     "warn level",
			level:    "warn",
			expected: slog.LevelWarn,
		},
		{
			name:     "error level",
			level:    "error",
			expected: slog.LevelError,
		},
		{
			name:     "uppercase DEBUG level",
			level:    "DEBUG",
			expected: slog.LevelDebug,
		},
		{
			name:     "invalid level defaults to info",
			level:    "invalid",
			expected: slog.LevelInfo,
		},
		{
			name:     "empty level defaults to info",
			level:    "",
			expected: slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLogLevel(tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected CliFlags
	}{
		{
			name: "default flags",
			args: []string{},
			expected: CliFlags{
				TransportType: testTransportStdio,
				SSEAddr:       testDefaultSSEAddr,
				LunoDomain:    "",
				LogLevel:      testLogLevelInfo,
			},
		},
		{
			name: "custom stdio flags",
			args: []string{"-transport=stdio", "-log-level=debug"},
			expected: CliFlags{
				TransportType: testTransportStdio,
				SSEAddr:       testDefaultSSEAddr,
				LunoDomain:    "",
				LogLevel:      testLogLevelDebug,
			},
		},
		{
			name: "sse transport with custom address",
			args: []string{"-transport=sse", "-sse-address=" + testCustomSSEAddr, "-domain=" + testStagingDomain},
			expected: CliFlags{
				TransportType: testTransportSSE,
				SSEAddr:       testCustomSSEAddr,
				LunoDomain:    testStagingDomain,
				LogLevel:      testLogLevelInfo,
			},
		},
		{
			name: "all custom flags",
			args: []string{"-transport=sse", "-sse-address=" + testCustomSSEAddrAlt, "-domain=" + testCustomDomain, "-log-level=error"},
			expected: CliFlags{
				TransportType: testTransportSSE,
				SSEAddr:       testCustomSSEAddrAlt,
				LunoDomain:    testCustomDomain,
				LogLevel:      testLogLevelError,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flag.CommandLine for each test
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			// Backup original os.Args
			originalArgs := os.Args
			defer func() { os.Args = originalArgs }()

			// Set test args
			os.Args = append([]string{"cmd"}, tt.args...)

			result := parseFlags()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadEnvFile(t *testing.T) {
	tests := []struct {
		name          string
		setupFiles    map[string]string
		workingDir    string
		expectedFound bool
	}{
		{
			name: "env file in current directory",
			setupFiles: map[string]string{
				".env": "TEST_VAR=value",
			},
			expectedFound: true,
		},
		{
			name: "env file in parent directory",
			setupFiles: map[string]string{
				".env": "TEST_VAR=value",
			},
			workingDir:    "subdir",
			expectedFound: true,
		},
		{
			name:          "no env file found",
			setupFiles:    map[string]string{},
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tempDir := t.TempDir()
			originalWd, _ := os.Getwd()
			defer func() { _ = os.Chdir(originalWd) }()

			// Setup files
			for relativeFilePath, content := range tt.setupFiles {
				var fullPath string
				if tt.workingDir != "" {
					// If we have a working directory, put the file in the parent (tempDir)
					fullPath = filepath.Join(tempDir, relativeFilePath)
				} else {
					// Put the file in the current directory we'll change to
					fullPath = filepath.Join(tempDir, relativeFilePath)
				}

				err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
				require.NoError(t, err)
				err = os.WriteFile(fullPath, []byte(content), 0o644)
				require.NoError(t, err)
			}

			// Change to test directory
			testDir := tempDir
			if tt.workingDir != "" {
				testDir = filepath.Join(tempDir, tt.workingDir)
				err := os.MkdirAll(testDir, 0o755)
				require.NoError(t, err)
			}
			err := os.Chdir(testDir)
			require.NoError(t, err)

			result := loadEnvFile()
			assert.Equal(t, tt.expectedFound, result)
		})
	}
}

func TestSetupLogger(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
	}{
		{
			name:     "debug level logger",
			logLevel: "debug",
		},
		{
			name:     "info level logger",
			logLevel: "info",
		},
		{
			name:     "error level logger",
			logLevel: "error",
		},
		{
			name:     "invalid level defaults to info",
			logLevel: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := setupLogger(tt.logLevel)
			assert.NotNil(t, logger)

			// Verify the logger was set as default
			defaultLogger := slog.Default()
			assert.NotNil(t, defaultLogger)
		})
	}
}

func TestCreateMCPServer(t *testing.T) {
	// Mock configuration - we'll need to set environment variables for this test
	t.Setenv("LUNO_API_KEY_ID", "test_key")
	t.Setenv("LUNO_API_SECRET", "test_secret")

	cfg, err := config.Load("", "test-app", "1.0.0")
	require.NoError(t, err)

	server := createMCPServer(cfg)
	assert.NotNil(t, server)
	assert.IsType(t, (*mcpserver.MCPServer)(nil), server)
}

func TestSetupSignalHandling(t *testing.T) {
	ctx, cancel := setupSignalHandling()
	defer cancel()

	assert.NotNil(t, ctx)
	assert.NotNil(t, cancel)

	// Verify context is not cancelled initially
	select {
	case <-ctx.Done():
		t.Fatal("Context should not be cancelled initially")
	default:
		// Expected behavior
	}

	// Test that cancel function works
	cancel()

	// Context should be cancelled immediately
	select {
	case <-ctx.Done():
		// Expected behavior
	default:
		t.Fatal("Context should be cancelled after calling cancel()")
	}
}

func TestCliFlags(t *testing.T) {
	tests := []struct {
		name     string
		flags    CliFlags
		expected CliFlags
	}{
		{
			name: "default values",
			flags: CliFlags{
				TransportType: testTransportStdio,
				SSEAddr:       testDefaultSSEAddr,
				LunoDomain:    "",
				LogLevel:      testLogLevelInfo,
			},
			expected: CliFlags{
				TransportType: testTransportStdio,
				SSEAddr:       testDefaultSSEAddr,
				LunoDomain:    "",
				LogLevel:      testLogLevelInfo,
			},
		},
		{
			name: "custom values",
			flags: CliFlags{
				TransportType: testTransportSSE,
				SSEAddr:       testCustomSSEAddr,
				LunoDomain:    testStagingDomain,
				LogLevel:      testLogLevelDebug,
			},
			expected: CliFlags{
				TransportType: testTransportSSE,
				SSEAddr:       testCustomSSEAddr,
				LunoDomain:    testStagingDomain,
				LogLevel:      testLogLevelDebug,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.flags)
		})
	}
}

// TestMainFunctionFlow tests the integration of the main function components
func TestMainFunctionFlow(t *testing.T) {
	// Set up environment variables for config loading
	t.Setenv("LUNO_API_KEY_ID", "test_key_id")
	t.Setenv("LUNO_API_SECRET", "test_secret")

	// Create temporary directory with .env file
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalWd) }()

	envContent := "LUNO_API_KEY_ID=test_env_key\nLUNO_API_SECRET=test_env_secret"
	envFile := filepath.Join(tempDir, ".env")
	err := os.WriteFile(envFile, []byte(envContent), 0o644)
	require.NoError(t, err)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test the flow components
	t.Run("load env file", func(t *testing.T) {
		found := loadEnvFile()
		assert.True(t, found)
	})

	t.Run("parse flags with defaults", func(t *testing.T) {
		// Reset flags for clean test
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		originalArgs := os.Args
		defer func() { os.Args = originalArgs }()
		os.Args = []string{"cmd"}

		flags := parseFlags()
		assert.Equal(t, testTransportStdio, flags.TransportType)
		assert.Equal(t, testDefaultSSEAddr, flags.SSEAddr)
		assert.Equal(t, "", flags.LunoDomain)
		assert.Equal(t, testLogLevelInfo, flags.LogLevel)
	})

	t.Run("setup logger", func(t *testing.T) {
		logger := setupLogger(testLogLevelInfo)
		assert.NotNil(t, logger)
	})

	t.Run("load config", func(t *testing.T) {
		cfg, err := config.Load("", "test-app", "1.0.0")
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
	})

	t.Run("create mcp server", func(t *testing.T) {
		cfg, err := config.Load("", "test-app", "1.0.0")
		require.NoError(t, err)

		server := createMCPServer(cfg)
		assert.NotNil(t, server)
		assert.IsType(t, (*mcpserver.MCPServer)(nil), server)
	})

	t.Run("setup signal handling", func(t *testing.T) {
		ctx, cancel := setupSignalHandling()
		defer cancel()

		assert.NotNil(t, ctx)
		assert.NotNil(t, cancel)

		// Verify context is not cancelled initially
		select {
		case <-ctx.Done():
			t.Fatal("Context should not be cancelled initially")
		default:
			// Expected behavior
		}
	})
}

func TestSetupEnhancedLogger(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
	}{
		{
			name:     "setup enhanced logger with debug level",
			logLevel: testLogLevelDebug,
		},
		{
			name:     "setup enhanced logger with info level",
			logLevel: testLogLevelInfo,
		},
		{
			name:     "setup enhanced logger with error level",
			logLevel: testLogLevelError,
		},
		{
			name:     "setup enhanced logger with invalid level defaults to info",
			logLevel: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables for config loading
			t.Setenv("LUNO_API_KEY_ID", "test_key")
			t.Setenv("LUNO_API_SECRET", "test_secret")

			// Load configuration
			cfg, err := config.Load("", "test-app", "1.0.0")
			require.NoError(t, err)

			// Create MCP server
			mcpServer := createMCPServer(cfg)
			require.NotNil(t, mcpServer)

			// Capture original logger to restore later
			originalLogger := slog.Default()
			defer slog.SetDefault(originalLogger)

			// Test setupEnhancedLogger - this function sets the default logger
			setupEnhancedLogger(mcpServer, tt.logLevel)

			// Verify the logger was set as default
			newLogger := slog.Default()
			assert.NotNil(t, newLogger)
			assert.NotEqual(t, originalLogger, newLogger, "Default logger should have changed")

			// Test that the logger can be used for logging
			slog.Info("Test log message from enhanced logger")
			slog.Debug("Debug message from enhanced logger")
			slog.Error("Error message from enhanced logger")
		})
	}
}

func TestStartServer(t *testing.T) {
	tests := []struct {
		name          string
		flags         CliFlags
		expectError   bool
		errorContains string
	}{
		{
			name: "invalid transport type",
			flags: CliFlags{
				TransportType: "invalid",
				SSEAddr:       testDefaultSSEAddr,
				LunoDomain:    "",
				LogLevel:      testLogLevelInfo,
			},
			expectError:   true,
			errorContains: "invalid transport type",
		},
		{
			name: "sse transport with invalid address",
			flags: CliFlags{
				TransportType: testTransportSSE,
				SSEAddr:       "invalid:99999",
				LunoDomain:    "",
				LogLevel:      testLogLevelInfo,
			},
			expectError:   true,
			errorContains: "invalid port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables for config loading
			t.Setenv("LUNO_API_KEY_ID", "test_key")
			t.Setenv("LUNO_API_SECRET", "test_secret")

			// Load configuration
			cfg, err := config.Load("", "test-app", "1.0.0")
			require.NoError(t, err)

			// Create MCP server
			mcpServer := createMCPServer(cfg)
			require.NotNil(t, mcpServer)

			ctx := context.Background()

			err = startServer(ctx, mcpServer, tt.flags)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

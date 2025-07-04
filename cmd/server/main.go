package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/luno/luno-mcp/internal/config"
	"github.com/luno/luno-mcp/internal/logging"
	"github.com/luno/luno-mcp/internal/server"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const (
	appName    = "luno-mcp"
	appVersion = "0.1.0"
)

// CliFlags holds command line flag values
type CliFlags struct {
	TransportType string
	SSEAddr       string
	LunoDomain    string
	LogLevel      string
}

// loadEnvFile attempts to load environment variables from various .env file locations
func loadEnvFile() bool {
	envPaths := []string{
		".env",    // Current directory
		"../.env", // Parent directory
	}

	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			log.Printf("Successfully loaded environment from %s", path)
			return true
		} else {
			log.Printf("Failed to load %s: %v", path, err)
		}
	}

	log.Println("Warning: No .env file found or unable to load it. Make sure environment variables are set.")
	// Print current directory for debugging
	if cwd, err := os.Getwd(); err == nil {
		log.Printf("Current working directory: %s", cwd)
	}
	return false
}

// parseFlags parses command line flags and returns CliFlags struct
func parseFlags() CliFlags {
	transportType := flag.String("transport", "stdio", "Transport type (stdio or sse)")
	sseAddr := flag.String("sse-address", "localhost:8080", "Address for SSE transport")
	lunoDomain := flag.String("domain", "", "Luno API domain (default: api.luno.com)")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	return CliFlags{
		TransportType: *transportType,
		SSEAddr:       *sseAddr,
		LunoDomain:    *lunoDomain,
		LogLevel:      *logLevel,
	}
}

// setupLogger creates and configures the basic console logger
func setupLogger(logLevel string) *slog.Logger {
	level := parseLogLevel(logLevel)
	consoleHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	logger := slog.New(consoleHandler)
	slog.SetDefault(logger)
	return logger
}

// setupEnhancedLogger creates an enhanced logger with MCP notification capability
func setupEnhancedLogger(mcpServer *mcpserver.MCPServer, logLevel string) {
	level := parseLogLevel(logLevel)
	consoleHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	mcpHandler := logging.NewMCPNotificationHandler(mcpServer, level)
	multiHandler := logging.NewMultiHandler(consoleHandler, mcpHandler)
	enhancedLogger := slog.New(multiHandler)
	slog.SetDefault(enhancedLogger)
}

// createMCPServer creates and configures the MCP server
func createMCPServer(cfg *config.Config) *mcpserver.MCPServer {
	return server.NewMCPServer(appName, appVersion, cfg, logging.MCPHooks())
}

// setupSignalHandling creates a context that will be cancelled on interrupt signals
func setupSignalHandling() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		slog.Info("Received shutdown signal")
		cancel()
	}()

	return ctx, cancel
}

// startServer starts the appropriate server based on transport type
func startServer(ctx context.Context, mcpServer *mcpserver.MCPServer, flags CliFlags) error {
	switch flags.TransportType {
	case "stdio":
		slog.Info("Starting Luno MCP server using stdio transport")
		return server.ServeStdio(ctx, mcpServer)
	case "sse":
		slog.Info("Starting Luno MCP server using SSE transport", slog.String("address", flags.SSEAddr))
		return server.ServeSSE(ctx, mcpServer, flags.SSEAddr)
	default:
		return fmt.Errorf("invalid transport type: %s. Must be 'stdio' or 'sse'", flags.TransportType)
	}
}

func main() {
	loadEnvFile()

	// Parse command line flags
	flags := parseFlags()

	// Set up basic logger first
	setupLogger(flags.LogLevel)

	// Load configuration
	cfg, err := config.Load(flags.LunoDomain, appName, appVersion)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create MCP server with logging hooks
	mcpServer := createMCPServer(cfg)

	// Now enhance the logger with MCP notification capability
	setupEnhancedLogger(mcpServer, flags.LogLevel)

	// Setup signal handling for graceful shutdown
	ctx, cancel := setupSignalHandling()
	defer cancel()

	// Start the server with the selected transport
	if err := startServer(ctx, mcpServer, flags); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func parseLogLevel(level string) slog.Level {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		return slog.LevelInfo
	}
	return l
}

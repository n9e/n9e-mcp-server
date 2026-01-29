package internal

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"

	"github.com/n9e/n9e-mcp-server/pkg/api"
	"github.com/n9e/n9e-mcp-server/pkg/client"
	"github.com/n9e/n9e-mcp-server/pkg/toolset"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gopkg.in/natefinch/lumberjack.v2"
)

// ServerConfig represents MCP Server configuration
type ServerConfig struct {
	Version         string
	Token           string
	BaseURL         string
	EnabledToolsets []string
	ReadOnly        bool
}

// NewMCPServer creates MCP Server
func NewMCPServer(cfg ServerConfig) (*mcp.Server, error) {
	// Create N9e Client
	n9eClient, err := client.NewClient(cfg.Token, cfg.BaseURL, fmt.Sprintf("n9e-mcp-server/%s", cfg.Version))
	if err != nil {
		return nil, fmt.Errorf("failed to create n9e client: %w", err)
	}

	// Create MCP Server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "n9e-mcp-server",
		Version: cfg.Version,
	}, &mcp.ServerOptions{
		Logger: slog.Default(),
	})

	// Add middleware: inject client into context
	server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			ctx = client.ContextWithClient(ctx, n9eClient)
			return next(ctx, method, req)
		}
	})

	// Create toolset group
	getClient := func(ctx context.Context) *client.Client {
		return client.ClientFromContext(ctx)
	}
	toolsetGroup := api.DefaultToolsetGroup(getClient, cfg.ReadOnly)

	// Determine enabled toolsets
	enabledToolsets := cfg.EnabledToolsets
	if len(enabledToolsets) == 0 {
		enabledToolsets = toolset.DefaultToolsets
	}

	// Enable toolsets
	if err := toolsetGroup.EnableToolsets(enabledToolsets); err != nil {
		return nil, fmt.Errorf("failed to enable toolsets: %w", err)
	}

	// Register all tools
	toolsetGroup.RegisterAll(server)

	return server, nil
}

// StdioServerConfig represents stdio mode configuration
type StdioServerConfig struct {
	Version         string
	Token           string
	BaseURL         string
	EnabledToolsets []string
	ReadOnly        bool
	LogFilePath     string
}

// RunStdioServer runs stdio mode server
func RunStdioServer(cfg StdioServerConfig) error {
	// Create context with signal interrupt support
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Configure logging (with file rotation support)
	var logOutput io.Writer = os.Stderr
	if cfg.LogFilePath != "" {
		logOutput = &lumberjack.Logger{
			Filename:   cfg.LogFilePath,
			MaxSize:    100, // MB, max file size
			MaxBackups: 3,   // Number of old files to keep
			MaxAge:     7,   // Days to keep
			Compress:   true,
		}
	}

	// Use LevelVar to support dynamic log level modification at runtime
	var logLevel slog.LevelVar
	parseLogLevel := func() slog.Level {
		switch os.Getenv("N9E_MCP_LOG_LEVEL") {
		case "debug", "DEBUG":
			return slog.LevelDebug
		case "warn", "WARN":
			return slog.LevelWarn
		case "error", "ERROR":
			return slog.LevelError
		default:
			return slog.LevelInfo
		}
	}
	logLevel.Set(parseLogLevel())
	logger := slog.New(slog.NewTextHandler(logOutput, &slog.HandlerOptions{Level: &logLevel}))
	slog.SetDefault(logger)

	// Listen to SIGUSR1 signal to reload environment variables and update log level (Unix only)
	setupSignalReload(func() {
		newLevel := parseLogLevel()
		logLevel.Set(newLevel)
		logger.Info("log level reloaded", "level", newLevel.String())
	})

	logger.Info("starting n9e-mcp-server",
		"version", cfg.Version,
		"base_url", cfg.BaseURL,
		"read_only", cfg.ReadOnly,
		"toolsets", cfg.EnabledToolsets,
	)

	// Create MCP Server
	server, err := NewMCPServer(ServerConfig{
		Version:         cfg.Version,
		Token:           cfg.Token,
		BaseURL:         cfg.BaseURL,
		EnabledToolsets: cfg.EnabledToolsets,
		ReadOnly:        cfg.ReadOnly,
	})
	if err != nil {
		return fmt.Errorf("failed to create MCP server: %w", err)
	}

	// Run server
	errC := make(chan error, 1)
	go func() {
		errC <- server.Run(ctx, &mcp.StdioTransport{})
	}()

	fmt.Fprintln(os.Stderr, "Nightingale MCP Server running on stdio")

	// Wait for exit
	select {
	case <-ctx.Done():
		logger.Info("shutting down server...")
	case err := <-errC:
		if err != nil {
			logger.Error("server error", "error", err)
			return fmt.Errorf("server error: %w", err)
		}
	}

	return nil
}

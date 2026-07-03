// Package main wires together the microservices system design MCP server.
//
// This entry point is small by design: parse environment, build the dependency
// graph via internal/mcpserver, start the chosen transport, and handle shutdown
// signals. No business logic, no tool registration, no HTTP handlers — those
// live in their respective packages.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/example/microservices-system-design-mcp-server/internal/mcpserver"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Production-shape structured logger to stderr.
	// Per the server-basics skill: stdio transport requires stderr-only logging
	// because stdout is the MCP protocol wire.
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Build the MCP server with all registered tools wired in.
	s := mcpserver.NewServer(logger)

	// Bound shutdown via signal handling.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	transport := os.Getenv("MCP_TRANSPORT")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	switch transport {
	case "streamablehttp", "http":
		logger.Info("starting MCP server",
			slog.String("transport", "streamable_http"),
			slog.String("port", port),
		)
		if err := server.NewStreamableHTTPServer(s).Start(":" + port); err != nil {
			logger.Error("streamable_http server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	case "sse":
		// SSE is deprecated in the current MCP spec (2025-11-25) in favor of streamable HTTP.
		// Retained here for backward compatibility with clients that haven't upgraded.
		logger.Warn("SSE transport is deprecated, prefer streamable_http")
		logger.Info("starting MCP server",
			slog.String("transport", "sse"),
			slog.String("port", port),
		)
		if err := server.NewSSEServer(s).Start(":" + port); err != nil {
			logger.Error("sse server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	default:
		logger.Info("starting MCP server", slog.String("transport", "stdio"))
		if err := server.ServeStdio(s); err != nil {
			logger.Error("stdio server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	// Wait for shutdown signal in non-blocking transports (the stdio/HTTP starts
	// above are blocking, so this is reached only if those return cleanly).
	<-ctx.Done()
	logger.Info("shutdown signal received")
}

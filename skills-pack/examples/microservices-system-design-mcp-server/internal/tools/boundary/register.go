// Package boundary registers the generate_service_boundary_canvas MCP tool.
//
// The handler is thin: parse input → validate → call service → format result.
// Business logic lives in internal/services/boundary. This package is the
// MCP-aware shell around the service.
package boundary

import (
	"context"
	"encoding/json"
	"log/slog"

	boundarysvc "github.com/example/microservices-system-design-mcp-server/internal/services/boundary"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Register wires the boundary canvas tool into the MCP server.
//
// Logger is used for structured tool-call events (start, completion, error).
// In production, these should also flow through an audit emitter; this example
// keeps audit and application logs combined for brevity. See the observability
// skill for the production separation.
func Register(s *server.MCPServer, svc *boundarysvc.Service, logger *slog.Logger) {
	tool := mcp.NewTool("generate_service_boundary_canvas",
		mcp.WithDescription("Generate a structured service boundary canvas for a proposed system. "+
			"Takes a system name and a list of proposed services with their business capabilities, "+
			"data ownership, dependencies, and team owners. Returns per-service assessments, boundary risks, "+
			"recommended changes, and an overall boundary score."),
		mcp.WithString("input_json",
			mcp.Required(),
			mcp.Description("JSON-encoded Input matching the schema in contracts/architecture-tools/implemented/generate_service_boundary_canvas.md")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger.Info("tool call started", slog.String("tool", "generate_service_boundary_canvas"))

		inputJSON, err := req.RequireString("input_json")
		if err != nil {
			logger.Warn("input_json missing", slog.String("tool", "generate_service_boundary_canvas"))
			return mcp.NewToolResultError("input_json is required"), nil
		}

		var input boundarysvc.Input
		if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
			logger.Warn("input_json failed to parse",
				slog.String("tool", "generate_service_boundary_canvas"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("input_json must be valid JSON matching the Input schema: " + err.Error()), nil
		}

		canvas, err := svc.GenerateCanvas(input)
		if err != nil {
			logger.Info("tool call rejected",
				slog.String("tool", "generate_service_boundary_canvas"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError(err.Error()), nil
		}

		b, err := json.MarshalIndent(canvas, "", "  ")
		if err != nil {
			// Marshaling our own struct should not fail. If it does, that's a bug.
			logger.Error("failed to marshal canvas",
				slog.String("tool", "generate_service_boundary_canvas"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("internal error: failed to format result"), nil
		}

		logger.Info("tool call completed",
			slog.String("tool", "generate_service_boundary_canvas"),
			slog.String("system", input.SystemName),
			slog.Int("services_analyzed", len(input.Services)),
			slog.Int("risks_identified", len(canvas.BoundaryRisks)),
			slog.Int("score", canvas.OverallScore),
		)

		return mcp.NewToolResultText(string(b)), nil
	})
}

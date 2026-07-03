// Package adr registers the generate_architecture_decision_record MCP tool.
//
// The handler is thin: parse input → call service → format result. Business
// logic lives in internal/services/adr.
package adr

import (
	"context"
	"encoding/json"
	"log/slog"

	adrsvc "github.com/example/microservices-system-design-mcp-server/internal/services/adr"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Register wires the ADR generator into the MCP server.
func Register(s *server.MCPServer, svc *adrsvc.Service, logger *slog.Logger) {
	tool := mcp.NewTool("generate_architecture_decision_record",
		mcp.WithDescription("Generate an Architecture Decision Record (ADR) in the canonical Michael Nygard format. "+
			"Takes a system name, decision title, context, decision, and optional drivers, options considered, "+
			"consequences, and references. Returns a structured ADR with status, drivers, context, decision, "+
			"options (with rejection reasons), consequences (positive/negative/neutral), references, a quality "+
			"score, warnings about weak ADR shapes, and a ready-to-commit markdown rendering. "+
			"Use this when a design choice needs to be documented; not for code review or general architecture review."),
		mcp.WithString("input_json",
			mcp.Required(),
			mcp.Description("JSON-encoded Input matching the schema in contracts/architecture-tools/implemented/generate_architecture_decision_record.md")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger.Info("tool call started", slog.String("tool", "generate_architecture_decision_record"))

		inputJSON, err := req.RequireString("input_json")
		if err != nil {
			logger.Warn("input_json missing", slog.String("tool", "generate_architecture_decision_record"))
			return mcp.NewToolResultError("input_json is required"), nil
		}

		var input adrsvc.Input
		if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
			logger.Warn("input_json failed to parse",
				slog.String("tool", "generate_architecture_decision_record"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("input_json must be valid JSON matching the Input schema: " + err.Error()), nil
		}

		out, err := svc.Generate(input)
		if err != nil {
			logger.Info("tool call rejected",
				slog.String("tool", "generate_architecture_decision_record"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError(err.Error()), nil
		}

		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			logger.Error("failed to marshal output",
				slog.String("tool", "generate_architecture_decision_record"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("internal error: failed to format result"), nil
		}

		logger.Info("tool call completed",
			slog.String("tool", "generate_architecture_decision_record"),
			slog.String("system", input.SystemName),
			slog.String("title", input.Title),
			slog.Int("warnings", len(out.Warnings)),
			slog.Int("quality_score", out.QualityScore),
		)

		return mcp.NewToolResultText(string(b)), nil
	})
}

// Package diagrams registers the generate_diagram_assets MCP tool.
package diagrams

import (
	"context"
	"encoding/json"
	"log/slog"

	diagramssvc "github.com/example/microservices-system-design-mcp-server/internal/services/diagrams"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Register wires the diagram assets generator into the MCP server.
func Register(s *server.MCPServer, svc *diagramssvc.GeneratorService, logger *slog.Logger) {
	tool := mcp.NewTool("generate_diagram_assets",
		mcp.WithDescription("Generate diagram assets for a microservices architecture. "+
			"Takes a system name, audience (business / technical / executive / engineering), diagram type "+
			"(context | deployment | sequence | event_flow | service_boundary), and structured architecture "+
			"data (services, events, external systems). Returns three assets in parallel: Mermaid source, "+
			"PlantUML source, and a draw.io-ready prompt; plus audience-tailored notes about how to refine. "+
			"Use when you need rendered architecture diagrams for documentation, ADRs, or reviews."),
		mcp.WithString("input_json",
			mcp.Required(),
			mcp.Description("JSON-encoded Input matching the schema in contracts/architecture-tools/implemented/generate_diagram_assets.md")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger.Info("tool call started", slog.String("tool", "generate_diagram_assets"))

		inputJSON, err := req.RequireString("input_json")
		if err != nil {
			logger.Warn("input_json missing", slog.String("tool", "generate_diagram_assets"))
			return mcp.NewToolResultError("input_json is required"), nil
		}

		var input diagramssvc.Input
		if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
			logger.Warn("input_json failed to parse",
				slog.String("tool", "generate_diagram_assets"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("input_json must be valid JSON matching the Input schema: " + err.Error()), nil
		}

		out, err := svc.Generate(input)
		if err != nil {
			logger.Info("tool call rejected",
				slog.String("tool", "generate_diagram_assets"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError(err.Error()), nil
		}

		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			logger.Error("failed to marshal output",
				slog.String("tool", "generate_diagram_assets"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("internal error: failed to format result"), nil
		}

		logger.Info("tool call completed",
			slog.String("tool", "generate_diagram_assets"),
			slog.String("system", input.SystemName),
			slog.String("diagram_type", out.DiagramType),
			slog.String("audience", out.Audience),
		)

		return mcp.NewToolResultText(string(b)), nil
	})
}

// Package azuremap registers the map_patterns_to_azure_services MCP tool.
//
// The handler is thin: parse input → validate → call service → format result.
// Business logic lives in internal/services/azuremap.
package azuremap

import (
	"context"
	"encoding/json"
	"log/slog"

	azuremapsvc "github.com/example/microservices-system-design-mcp-server/internal/services/azuremap"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Register wires the pattern-to-Azure mapping tool into the MCP server.
//
// Logger is used for structured tool-call events, matching the registration
// pattern established by the boundary tool.
func Register(s *server.MCPServer, svc *azuremapsvc.Mapper, logger *slog.Logger) {
	tool := mcp.NewTool("map_patterns_to_azure_services",
		mcp.WithDescription("Map architecture patterns to the Azure services that implement them, with "+
			"rationale and alternatives, using a curated catalog. Reports unmapped patterns, "+
			"deployment-target mismatches, missing cross-cutting patterns, and a mapping-coverage score."),
		mcp.WithString("input_json",
			mcp.Required(),
			mcp.Description("JSON-encoded Input matching the schema in contracts/architecture-tools/implemented/map_patterns_to_azure_services.md")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger.Info("tool call started", slog.String("tool", "map_patterns_to_azure_services"))

		inputJSON, err := req.RequireString("input_json")
		if err != nil {
			logger.Warn("input_json missing", slog.String("tool", "map_patterns_to_azure_services"))
			return mcp.NewToolResultError("input_json is required"), nil
		}

		var input azuremapsvc.Input
		if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
			logger.Warn("input_json failed to parse",
				slog.String("tool", "map_patterns_to_azure_services"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("input_json must be valid JSON matching the Input schema: " + err.Error()), nil
		}

		mapping, err := svc.Map(input)
		if err != nil {
			logger.Info("tool call rejected",
				slog.String("tool", "map_patterns_to_azure_services"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError(err.Error()), nil
		}

		b, err := json.MarshalIndent(mapping, "", "  ")
		if err != nil {
			logger.Error("failed to marshal mapping",
				slog.String("tool", "map_patterns_to_azure_services"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("internal error: failed to format result"), nil
		}

		logger.Info("tool call completed",
			slog.String("tool", "map_patterns_to_azure_services"),
			slog.String("system", input.SystemName),
			slog.Int("patterns", len(input.Patterns)),
			slog.Int("mapped", mapping.Coverage.MappedCount),
			slog.Int("score", mapping.MappingScore),
		)

		return mcp.NewToolResultText(string(b)), nil
	})
}

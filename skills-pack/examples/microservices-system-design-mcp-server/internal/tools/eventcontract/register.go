// Package eventcontract registers the generate_event_contract MCP tool.
package eventcontract

import (
	"context"
	"encoding/json"
	"log/slog"

	eventcontractsvc "github.com/example/microservices-system-design-mcp-server/internal/services/eventcontract"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Register wires the event contract generator into the MCP server.
func Register(s *server.MCPServer, svc *eventcontractsvc.GeneratorService, logger *slog.Logger) {
	tool := mcp.NewTool("generate_event_contract",
		mcp.WithDescription("Generate a CloudEvents-shaped event contract for a domain event. "+
			"Takes a system name, event name (past-tense), producer service, consumer services, payload fields "+
			"(name/type/required/sensitive), optional transport (service_bus | event_grid | event_hubs), and "+
			"ordering preference. Returns a CloudEvents schema with envelope fields (event_id, occurred_at, "+
			"correlation_id), transport binding (Azure service, topic, subscriptions, DLQ), per-consumer notes, "+
			"warnings on command-shaped names and sensitive-field handling, a markdown rendering, and a quality "+
			"score. Use this to standardise event contracts across teams before code is written."),
		mcp.WithString("input_json",
			mcp.Required(),
			mcp.Description("JSON-encoded Input matching the schema in contracts/architecture-tools/implemented/generate_event_contract.md")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger.Info("tool call started", slog.String("tool", "generate_event_contract"))

		inputJSON, err := req.RequireString("input_json")
		if err != nil {
			logger.Warn("input_json missing", slog.String("tool", "generate_event_contract"))
			return mcp.NewToolResultError("input_json is required"), nil
		}

		var input eventcontractsvc.Input
		if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
			logger.Warn("input_json failed to parse",
				slog.String("tool", "generate_event_contract"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("input_json must be valid JSON matching the Input schema: " + err.Error()), nil
		}

		out, err := svc.Generate(input)
		if err != nil {
			logger.Info("tool call rejected",
				slog.String("tool", "generate_event_contract"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError(err.Error()), nil
		}

		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			logger.Error("failed to marshal output",
				slog.String("tool", "generate_event_contract"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("internal error: failed to format result"), nil
		}

		logger.Info("tool call completed",
			slog.String("tool", "generate_event_contract"),
			slog.String("system", input.SystemName),
			slog.String("event", input.EventName),
			slog.Int("consumers", len(input.Consumers)),
			slog.Int("warnings", len(out.Warnings)),
			slog.Int("quality_score", out.QualityScore),
		)

		return mcp.NewToolResultText(string(b)), nil
	})
}

// Package obsplan registers the generate_observability_plan MCP tool.
//
// The handler is thin: parse input → validate → call service → format result.
// Business logic lives in internal/services/obsplan.
package obsplan

import (
	"context"
	"encoding/json"
	"log/slog"

	obsplansvc "github.com/example/microservices-system-design-mcp-server/internal/services/obsplan"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Register wires the observability-plan tool into the MCP server.
//
// Logger is used for structured tool-call events, matching the registration
// pattern established by the boundary tool.
func Register(s *server.MCPServer, svc *obsplansvc.Planner, logger *slog.Logger) {
	tool := mcp.NewTool("generate_observability_plan",
		mcp.WithDescription("Generate an observability plan for a proposed microservices system. "+
			"Takes services with their criticality and type and produces per-service SLIs, SLOs, "+
			"recommended dashboards and alerts, coverage gaps, and an observability-readiness score."),
		mcp.WithString("input_json",
			mcp.Required(),
			mcp.Description("JSON-encoded Input matching the schema in contracts/architecture-tools/implemented/generate_observability_plan.md")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger.Info("tool call started", slog.String("tool", "generate_observability_plan"))

		inputJSON, err := req.RequireString("input_json")
		if err != nil {
			logger.Warn("input_json missing", slog.String("tool", "generate_observability_plan"))
			return mcp.NewToolResultError("input_json is required"), nil
		}

		var input obsplansvc.Input
		if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
			logger.Warn("input_json failed to parse",
				slog.String("tool", "generate_observability_plan"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("input_json must be valid JSON matching the Input schema: " + err.Error()), nil
		}

		plan, err := svc.Generate(input)
		if err != nil {
			logger.Info("tool call rejected",
				slog.String("tool", "generate_observability_plan"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError(err.Error()), nil
		}

		b, err := json.MarshalIndent(plan, "", "  ")
		if err != nil {
			logger.Error("failed to marshal plan",
				slog.String("tool", "generate_observability_plan"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("internal error: failed to format result"), nil
		}

		logger.Info("tool call completed",
			slog.String("tool", "generate_observability_plan"),
			slog.String("system", input.SystemName),
			slog.Int("services_analyzed", len(input.Services)),
			slog.Int("coverage_gaps", len(plan.CoverageGaps)),
			slog.Int("score", plan.ObservabilityScore),
		)

		return mcp.NewToolResultText(string(b)), nil
	})
}

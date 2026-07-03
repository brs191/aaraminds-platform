// Package resilience registers the generate_resilience_plan MCP tool.
package resilience

import (
	"context"
	"encoding/json"
	"log/slog"

	resiliencesvc "github.com/example/microservices-system-design-mcp-server/internal/services/resilience"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Register wires the resilience plan generator into the MCP server.
func Register(s *server.MCPServer, svc *resiliencesvc.GeneratorService, logger *slog.Logger) {
	tool := mcp.NewTool("generate_resilience_plan",
		mcp.WithDescription("Generate a resilience plan for a microservices system. "+
			"Takes services (criticality, statefulness, replication), internal dependencies, external APIs, and "+
			"NFRs. Returns per-dependency timeout / retry / circuit-breaker configuration (with tighter values "+
			"for high-criticality boundaries and external APIs), bulkhead notes for stateful or single-replica "+
			"services, queue-based load-leveling notes for workers, fallback strategies (fail-fast vs. degrade) "+
			"per dependency, detection signals with alert thresholds, and a coverage score. "+
			"Use when designing or reviewing resilience posture across services."),
		mcp.WithString("input_json",
			mcp.Required(),
			mcp.Description("JSON-encoded Input matching the schema in contracts/architecture-tools/implemented/generate_resilience_plan.md")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger.Info("tool call started", slog.String("tool", "generate_resilience_plan"))

		inputJSON, err := req.RequireString("input_json")
		if err != nil {
			logger.Warn("input_json missing", slog.String("tool", "generate_resilience_plan"))
			return mcp.NewToolResultError("input_json is required"), nil
		}

		var input resiliencesvc.Input
		if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
			logger.Warn("input_json failed to parse",
				slog.String("tool", "generate_resilience_plan"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("input_json must be valid JSON matching the Input schema: " + err.Error()), nil
		}

		plan, err := svc.Generate(input)
		if err != nil {
			logger.Info("tool call rejected",
				slog.String("tool", "generate_resilience_plan"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError(err.Error()), nil
		}

		b, err := json.MarshalIndent(plan, "", "  ")
		if err != nil {
			logger.Error("failed to marshal plan",
				slog.String("tool", "generate_resilience_plan"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("internal error: failed to format result"), nil
		}

		logger.Info("tool call completed",
			slog.String("tool", "generate_resilience_plan"),
			slog.String("system", input.SystemName),
			slog.Int("dependency_controls", len(plan.DependencyControls)),
			slog.Int("fallbacks", len(plan.Fallbacks)),
			slog.Int("coverage_score", plan.CoverageScore),
		)

		return mcp.NewToolResultText(string(b)), nil
	})
}

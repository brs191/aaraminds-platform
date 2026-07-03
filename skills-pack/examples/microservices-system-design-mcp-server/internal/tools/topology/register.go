// Package topology registers the generate_deployment_topology MCP tool.
package topology

import (
	"context"
	"encoding/json"
	"log/slog"

	topologysvc "github.com/example/microservices-system-design-mcp-server/internal/services/topology"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Register wires the deployment topology generator into the MCP server.
func Register(s *server.MCPServer, svc *topologysvc.GeneratorService, logger *slog.Logger) {
	tool := mcp.NewTool("generate_deployment_topology",
		mcp.WithDescription("Generate an Azure deployment topology for a microservices system. "+
			"Takes services (type, criticality, externality, statefulness), data stores (kind, classification), "+
			"deployment target, target environments, and NFRs. Returns per-service placements (platform, replicas, "+
			"scale rule, CPU/memory hints), per-data-store placements (Azure service, tier, encryption, subnet, "+
			"backup policy), network boundaries (perimeter / application / sensitive-data isolation), the "+
			"environment promotion path, identified gaps, next steps, and a readiness score."),
		mcp.WithString("input_json",
			mcp.Required(),
			mcp.Description("JSON-encoded Input matching the schema in contracts/architecture-tools/implemented/generate_deployment_topology.md")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger.Info("tool call started", slog.String("tool", "generate_deployment_topology"))

		inputJSON, err := req.RequireString("input_json")
		if err != nil {
			logger.Warn("input_json missing", slog.String("tool", "generate_deployment_topology"))
			return mcp.NewToolResultError("input_json is required"), nil
		}

		var input topologysvc.Input
		if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
			logger.Warn("input_json failed to parse",
				slog.String("tool", "generate_deployment_topology"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("input_json must be valid JSON matching the Input schema: " + err.Error()), nil
		}

		out, err := svc.Generate(input)
		if err != nil {
			logger.Info("tool call rejected",
				slog.String("tool", "generate_deployment_topology"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError(err.Error()), nil
		}

		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			logger.Error("failed to marshal output",
				slog.String("tool", "generate_deployment_topology"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("internal error: failed to format result"), nil
		}

		logger.Info("tool call completed",
			slog.String("tool", "generate_deployment_topology"),
			slog.String("system", input.SystemName),
			slog.String("platform", out.Platform),
			slog.Int("services", len(out.ServicePlacements)),
			slog.Int("gaps", len(out.Gaps)),
			slog.Int("score", out.Score),
		)

		return mcp.NewToolResultText(string(b)), nil
	})
}

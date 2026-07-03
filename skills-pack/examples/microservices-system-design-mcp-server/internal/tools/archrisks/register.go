// Package archrisks registers the detect_architecture_risks MCP tool.
//
// The handler is thin: parse input → validate → call service → format result.
// Business logic lives in internal/services/archrisks. This package is the
// MCP-aware shell around the service.
package archrisks

import (
	"context"
	"encoding/json"
	"log/slog"

	archrisksvc "github.com/example/microservices-system-design-mcp-server/internal/services/archrisks"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Register wires the architecture-risk tool into the MCP server.
//
// Logger is used for structured tool-call events (start, completion, error),
// matching the registration pattern established by the boundary tool.
func Register(s *server.MCPServer, svc *archrisksvc.Service, logger *slog.Logger) {
	tool := mcp.NewTool("detect_architecture_risks",
		mcp.WithDescription("Detect architecture risks in a proposed microservices system. "+
			"Takes a structured description of services (criticality, state, dependencies, data stores, "+
			"resilience controls), data stores, deployment target, constraints, and non-functional "+
			"requirements. Returns named risks with severity, likelihood, affected components, and a "+
			"concrete mitigation, plus a risk-posture score, missing decisions, and next steps."),
		mcp.WithString("input_json",
			mcp.Required(),
			mcp.Description("JSON-encoded Input matching the schema in contracts/architecture-tools/implemented/detect_architecture_risks.md")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger.Info("tool call started", slog.String("tool", "detect_architecture_risks"))

		inputJSON, err := req.RequireString("input_json")
		if err != nil {
			logger.Warn("input_json missing", slog.String("tool", "detect_architecture_risks"))
			return mcp.NewToolResultError("input_json is required"), nil
		}

		var input archrisksvc.Input
		if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
			logger.Warn("input_json failed to parse",
				slog.String("tool", "detect_architecture_risks"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("input_json must be valid JSON matching the Input schema: " + err.Error()), nil
		}

		report, err := svc.Detect(input)
		if err != nil {
			logger.Info("tool call rejected",
				slog.String("tool", "detect_architecture_risks"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError(err.Error()), nil
		}

		b, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			logger.Error("failed to marshal report",
				slog.String("tool", "detect_architecture_risks"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("internal error: failed to format result"), nil
		}

		logger.Info("tool call completed",
			slog.String("tool", "detect_architecture_risks"),
			slog.String("system", input.SystemName),
			slog.Int("services_analyzed", len(input.Services)),
			slog.Int("risks_identified", len(report.Risks)),
			slog.Int("score", report.RiskPostureScore),
		)

		return mcp.NewToolResultText(string(b)), nil
	})
}

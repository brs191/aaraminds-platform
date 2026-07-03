// Package apicontract registers the generate_api_contract MCP tool.
//
// The handler is thin: parse input → validate → call service → format result.
// Business logic lives in internal/services/apicontract.
package apicontract

import (
	"context"
	"encoding/json"
	"log/slog"

	apicontractsvc "github.com/example/microservices-system-design-mcp-server/internal/services/apicontract"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Register wires the API-contract tool into the MCP server.
//
// Logger is used for structured tool-call events, matching the registration
// pattern established by the boundary tool.
func Register(s *server.MCPServer, svc *apicontractsvc.Generator, logger *slog.Logger) {
	tool := mcp.NewTool("generate_api_contract",
		mcp.WithDescription("Generate an OpenAPI-shaped API contract for a proposed microservices system. "+
			"Takes services with their resources and operations and produces REST endpoints with status "+
			"codes and error responses, per-service security, contract-quality findings, and a "+
			"contract-readiness score."),
		mcp.WithString("input_json",
			mcp.Required(),
			mcp.Description("JSON-encoded Input matching the schema in contracts/architecture-tools/implemented/generate_api_contract.md")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger.Info("tool call started", slog.String("tool", "generate_api_contract"))

		inputJSON, err := req.RequireString("input_json")
		if err != nil {
			logger.Warn("input_json missing", slog.String("tool", "generate_api_contract"))
			return mcp.NewToolResultError("input_json is required"), nil
		}

		var input apicontractsvc.Input
		if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
			logger.Warn("input_json failed to parse",
				slog.String("tool", "generate_api_contract"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("input_json must be valid JSON matching the Input schema: " + err.Error()), nil
		}

		contract, err := svc.Generate(input)
		if err != nil {
			logger.Info("tool call rejected",
				slog.String("tool", "generate_api_contract"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError(err.Error()), nil
		}

		b, err := json.MarshalIndent(contract, "", "  ")
		if err != nil {
			logger.Error("failed to marshal contract",
				slog.String("tool", "generate_api_contract"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("internal error: failed to format result"), nil
		}

		logger.Info("tool call completed",
			slog.String("tool", "generate_api_contract"),
			slog.String("system", input.SystemName),
			slog.Int("services_analyzed", len(input.Services)),
			slog.Int("operations", contract.OpenAPISummary.TotalOperations),
			slog.Int("score", contract.ContractScore),
		)

		return mcp.NewToolResultText(string(b)), nil
	})
}

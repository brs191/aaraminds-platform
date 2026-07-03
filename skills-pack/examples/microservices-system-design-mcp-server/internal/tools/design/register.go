// Package design registers the three design-review MCP tools:
//
//   - review_microservice_design
//   - recommend_microservice_patterns
//   - score_well_architected_readiness
//
// The review and score tools accept optional `input_json` for full-fidelity
// SystemInput while retaining their older typed arguments as a compatibility
// fallback. Claude Code introspected an older binary of these tools and cached
// their typed schemas; removing the typed surface would break cached clients
// until users manually restart the MCP server.
//
// The business logic still lives in internal/services/design. This package is
// the MCP-aware shell that:
//
//  1. Reads rich `input_json` or typed fallback args from the MCP request.
//  2. Synthesises the service's input struct (sparse is fine — the service
//     handles sparse cases by producing low-score / "Unsound" verdicts rather
//     than erroring).
//  3. Calls the service and marshals the JSON result.
package design

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"

	designsvc "github.com/example/microservices-system-design-mcp-server/internal/services/design"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Register wires the design-review tools into the MCP server.
//
// Logger is used for structured tool-call events (start, completion, error),
// matching the registration pattern established by the boundary tool.
func Register(s *server.MCPServer, svc *designsvc.Service, logger *slog.Logger) {
	registerReview(s, svc, logger)
	registerRecommend(s, svc, logger)
	registerScore(s, svc, logger)
}

func registerReview(s *server.MCPServer, svc *designsvc.Service, logger *slog.Logger) {
	tool := mcp.NewTool("review_microservice_design",
		mcp.WithDescription("Review a microservices system design end-to-end across the 9-dimension "+
			"architecture-review framework (boundaries, data, topology, contracts, resilience, Azure mapping, "+
			"observability, security, cost). Returns per-dimension verdicts with named defects, a 0-100 score, "+
			"hard/soft fails, missing artifacts, and prioritized next steps."),
		mcp.WithString("input_json",
			mcp.Description("Optional JSON-encoded SystemInput for full-fidelity review. When present, it overrides the sparse typed fields.")),
		mcp.WithString("system_name",
			mcp.Description("Name of the system")),
		mcp.WithString("business_capability",
			mcp.Description("Business capability being designed")),
		mcp.WithString("deployment_target",
			mcp.Description("AKS, Container Apps, App Service, or hybrid")),
		mcp.WithString("services",
			mcp.Description("Comma-separated proposed services")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger.Info("tool call started", slog.String("tool", "review_microservice_design"))

		input, err := reviewInputFromRequest(req)
		if err != nil {
			logger.Warn("review input invalid",
				slog.String("tool", "review_microservice_design"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError(err.Error()), nil
		}

		report, err := svc.Review(input)
		if err != nil {
			logger.Info("tool call rejected",
				slog.String("tool", "review_microservice_design"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError(err.Error()), nil
		}

		b, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			logger.Error("failed to marshal report",
				slog.String("tool", "review_microservice_design"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("internal error: failed to format result"), nil
		}

		logger.Info("tool call completed",
			slog.String("tool", "review_microservice_design"),
			slog.String("system", input.SystemName),
			slog.Int("services_analyzed", len(input.Services)),
			slog.Int("score", report.Score),
			slog.String("verdict", report.Verdict),
		)

		return mcp.NewToolResultText(string(b)), nil
	})
}

func registerRecommend(s *server.MCPServer, svc *designsvc.Service, logger *slog.Logger) {
	tool := mcp.NewTool("recommend_microservice_patterns",
		mcp.WithDescription("Recommend microservices patterns for a stated problem. Rules-based pattern matching "+
			"against a curated catalog (API Gateway, BFF, Saga, Outbox, CQRS, Strangler Fig, Sidecar, Circuit "+
			"Breaker, Bulkhead, Event Sourcing, Cache-Aside, etc.). Returns ranked recommendations with per-pattern "+
			"rationale tied to the specific input phrases that activated each rule, plus a not_recommended list for "+
			"any anti-patterns detected."),
		mcp.WithString("problem",
			mcp.Required(),
			mcp.Description("Problem statement or design challenge")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger.Info("tool call started", slog.String("tool", "recommend_microservice_patterns"))

		problem, err := req.RequireString("problem")
		if err != nil {
			logger.Warn("problem missing", slog.String("tool", "recommend_microservice_patterns"))
			return mcp.NewToolResultError("problem is required"), nil
		}

		input := designsvc.PatternRecommendInput{
			Problem: problem,
		}

		result, err := svc.RecommendPatterns(input)
		if err != nil {
			logger.Info("tool call rejected",
				slog.String("tool", "recommend_microservice_patterns"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError(err.Error()), nil
		}

		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			logger.Error("failed to marshal result",
				slog.String("tool", "recommend_microservice_patterns"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("internal error: failed to format result"), nil
		}

		logger.Info("tool call completed",
			slog.String("tool", "recommend_microservice_patterns"),
			slog.String("system", input.SystemName),
			slog.Int("recommendations", len(result.Recommendations)),
			slog.Int("anti_patterns_flagged", len(result.NotRecommended)),
		)

		return mcp.NewToolResultText(string(b)), nil
	})
}

func registerScore(s *server.MCPServer, svc *designsvc.Service, logger *slog.Logger) {
	tool := mcp.NewTool("score_well_architected_readiness",
		mcp.WithDescription("Score a microservices design against the five Azure Well-Architected pillars "+
			"(Reliability, Security, Operational Excellence, Performance Efficiency, Cost Optimization). "+
			"Pillar scores are computed deterministically from input signals; each pillar returns its score, "+
			"evidence (signals found), and risks (signals missing). Returns the per-pillar breakdown, an overall "+
			"0-100 score, and a categorical rating."),
		mcp.WithString("input_json",
			mcp.Description("Optional JSON-encoded SystemInput for full-fidelity scoring. When present, it overrides the system_name field.")),
		mcp.WithString("system_name",
			mcp.Description("System name")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		logger.Info("tool call started", slog.String("tool", "score_well_architected_readiness"))

		input, err := scoreInputFromRequest(req)
		if err != nil {
			logger.Warn("score input invalid",
				slog.String("tool", "score_well_architected_readiness"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError(err.Error()), nil
		}

		card, err := svc.ScoreWellArchitected(input)
		if err != nil {
			logger.Info("tool call rejected",
				slog.String("tool", "score_well_architected_readiness"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError(err.Error()), nil
		}

		b, err := json.MarshalIndent(card, "", "  ")
		if err != nil {
			logger.Error("failed to marshal scorecard",
				slog.String("tool", "score_well_architected_readiness"),
				slog.String("error", err.Error()),
			)
			return mcp.NewToolResultError("internal error: failed to format result"), nil
		}

		logger.Info("tool call completed",
			slog.String("tool", "score_well_architected_readiness"),
			slog.String("system", input.SystemName),
			slog.Int("overall_score", card.OverallScore),
			slog.String("rating", card.Rating),
		)

		return mcp.NewToolResultText(string(b)), nil
	})
}

func reviewInputFromRequest(req mcp.CallToolRequest) (designsvc.SystemInput, error) {
	fallback := designsvc.SystemInput{
		SystemName:         req.GetString("system_name", ""),
		BusinessCapability: req.GetString("business_capability", ""),
		DeploymentTarget:   req.GetString("deployment_target", ""),
		Services:           parseServicesCSV(req.GetString("services", "")),
	}
	return parseSystemInputJSON(req.GetString("input_json", ""), fallback)
}

func scoreInputFromRequest(req mcp.CallToolRequest) (designsvc.SystemInput, error) {
	fallback := designsvc.SystemInput{
		SystemName: req.GetString("system_name", ""),
	}
	return parseSystemInputJSON(req.GetString("input_json", ""), fallback)
}

func parseSystemInputJSON(inputJSON string, fallback designsvc.SystemInput) (designsvc.SystemInput, error) {
	if strings.TrimSpace(inputJSON) == "" {
		return fallback, nil
	}

	var input designsvc.SystemInput
	dec := json.NewDecoder(strings.NewReader(inputJSON))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&input); err != nil {
		return designsvc.SystemInput{}, err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return designsvc.SystemInput{}, errors.New("input_json must contain a single JSON object")
		}
		return designsvc.SystemInput{}, err
	}

	if strings.TrimSpace(input.SystemName) == "" {
		input.SystemName = fallback.SystemName
	}
	if strings.TrimSpace(input.BusinessCapability) == "" {
		input.BusinessCapability = fallback.BusinessCapability
	}
	if strings.TrimSpace(input.DeploymentTarget) == "" {
		input.DeploymentTarget = fallback.DeploymentTarget
	}
	if len(input.Services) == 0 {
		input.Services = fallback.Services
	}
	return input, nil
}

// parseServicesCSV splits a comma-separated services string into a slice of
// minimal ServiceDescriptor values (name only). Empty trimmed tokens are
// dropped. An empty input returns nil so the service-layer normaliser sees no
// services at all (which is its expected "sparse input" path).
func parseServicesCSV(csv string) []designsvc.ServiceDescriptor {
	csv = strings.TrimSpace(csv)
	if csv == "" {
		return nil
	}
	parts := strings.Split(csv, ",")
	out := make([]designsvc.ServiceDescriptor, 0, len(parts))
	for _, p := range parts {
		name := strings.TrimSpace(p)
		if name == "" {
			continue
		}
		out = append(out, designsvc.ServiceDescriptor{Name: name})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

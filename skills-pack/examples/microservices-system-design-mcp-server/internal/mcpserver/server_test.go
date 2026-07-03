package mcpserver

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserverlib "github.com/mark3labs/mcp-go/server"
)

func TestNewServerRegistersExpectedTools(t *testing.T) {
	s := NewServer(testLogger())
	if got := len(s.ListTools()); got != 13 {
		t.Fatalf("expected 13 tools, got %d", got)
	}
	for _, name := range []string{
		"review_microservice_design",
		"recommend_microservice_patterns",
		"score_well_architected_readiness",
		"generate_service_boundary_canvas",
		"generate_api_contract",
		"detect_architecture_risks",
		"map_patterns_to_azure_services",
		"generate_observability_plan",
		"generate_architecture_decision_record",
		"generate_deployment_topology",
		"generate_event_contract",
		"generate_resilience_plan",
		"generate_diagram_assets",
	} {
		if s.GetTool(name) == nil {
			t.Fatalf("expected tool %q to be registered", name)
		}
	}
}

func TestDesignScoreToolAcceptsRichInputJSON(t *testing.T) {
	s := NewServer(testLogger())

	sparse := callToolText(t, s, "score_well_architected_readiness", map[string]any{
		"system_name": "orders",
	})
	rich := callToolText(t, s, "score_well_architected_readiness", map[string]any{
		"input_json": `{
			"system_name": "orders",
			"deployment_target": "container_apps",
			"services": [
				{
					"name": "billing",
					"criticality": "high",
					"replicated": true,
					"resilience": ["timeout", "retry", "circuit_breaker"],
					"team": "billing"
				}
			],
			"observability": ["otel", "appinsights", "grafana"],
			"security_controls": ["entra_id", "managed_identity", "key_vault", "mtls"],
			"api_contracts": ["openapi", "versioning"],
			"messaging": ["service_bus"],
			"patterns": ["saga", "transactional_outbox", "dlq", "cache_aside"],
			"non_functional_requirements": {
				"availability_target": "99.95",
				"latency_p99_ms": 200,
				"rto_minutes": 30,
				"rpo_minutes": 5
			},
			"autoscale_declared": true,
			"scale_to_zero": true
		}`,
	})

	var sparseCard, richCard struct {
		OverallScore     int `json:"overall_score"`
		CostOptimization struct {
			Evidence []string `json:"evidence"`
		} `json:"cost_optimization"`
	}
	if err := json.Unmarshal([]byte(sparse), &sparseCard); err != nil {
		t.Fatalf("parse sparse scorecard: %v", err)
	}
	if err := json.Unmarshal([]byte(rich), &richCard); err != nil {
		t.Fatalf("parse rich scorecard: %v", err)
	}
	if richCard.OverallScore <= sparseCard.OverallScore {
		t.Fatalf("expected rich input score to exceed sparse score: rich=%d sparse=%d", richCard.OverallScore, sparseCard.OverallScore)
	}
	if !containsText(richCard.CostOptimization.Evidence, "autoscaling declared") {
		t.Fatalf("expected rich input to reach cost evidence, got %+v", richCard.CostOptimization.Evidence)
	}
}

func TestDesignReviewToolTypedFallbackStillWorks(t *testing.T) {
	s := NewServer(testLogger())
	raw := callToolText(t, s, "review_microservice_design", map[string]any{
		"system_name":       "orders",
		"deployment_target": "container_apps",
		"services":          "orders, payments",
	})

	var report struct {
		SystemName string `json:"system_name"`
		Score      int    `json:"score"`
	}
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatalf("parse review report: %v", err)
	}
	if report.SystemName != "orders" {
		t.Fatalf("unexpected system name: %+v", report)
	}
	if report.Score < 0 || report.Score > 100 {
		t.Fatalf("score out of range: %+v", report)
	}
}

func callToolText(t *testing.T, s *mcpserverlib.MCPServer, name string, args map[string]any) string {
	t.Helper()

	tool := s.GetTool(name)
	if tool == nil {
		t.Fatalf("tool %q is not registered", name)
	}
	res, err := tool.Handler(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	})
	if err != nil {
		t.Fatalf("tool %q returned protocol error: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("tool %q returned tool error: %+v", name, res.Content)
	}
	if len(res.Content) != 1 {
		t.Fatalf("tool %q returned %d content blocks", name, len(res.Content))
	}
	text, ok := mcp.AsTextContent(res.Content[0])
	if !ok {
		t.Fatalf("tool %q returned non-text content: %+v", name, res.Content[0])
	}
	return text.Text
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func containsText(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

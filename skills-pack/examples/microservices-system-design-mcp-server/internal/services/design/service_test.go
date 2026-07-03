// Tests for the design service: review, recommend, and Well-Architected
// scoring. The pivotal property under test is input-awareness — given two
// different inputs, the output must differ. This is the regression guard
// against the prior stub implementation which returned hardcoded responses.
package design

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Review
// ---------------------------------------------------------------------------

func TestReviewRequiresSystemName(t *testing.T) {
	s := NewService()
	if _, err := s.Review(SystemInput{Services: []ServiceDescriptor{{Name: "x"}}}); err == nil {
		t.Fatal("expected error when system_name is missing")
	}
}

func TestReviewRequiresAtLeastOneService(t *testing.T) {
	s := NewService()
	if _, err := s.Review(SystemInput{SystemName: "orders"}); err == nil {
		t.Fatal("expected error when no services are provided")
	}
}

func TestReviewWeakInputProducesLowScoreAndDefects(t *testing.T) {
	s := NewService()
	res, err := s.Review(SystemInput{
		SystemName: "orders",
		Services: []ServiceDescriptor{
			{Name: "UserService"},
			{Name: "OrderService", DependsOn: []string{"UserService", "PaymentService", "InventoryService", "ShippingService"}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Score >= 75 {
		t.Fatalf("expected lower score for sparse input; got %d", res.Score)
	}
	if len(res.HardFails)+len(res.SoftFails) == 0 {
		t.Fatal("expected at least one hard or soft fail on a sparse input")
	}
	if len(res.NextSteps) == 0 {
		t.Fatal("expected non-empty next_steps when defects exist")
	}
}

func TestReviewStrongInputScoresHigher(t *testing.T) {
	s := NewService()
	weak, err := s.Review(SystemInput{
		SystemName: "orders",
		Services:   []ServiceDescriptor{{Name: "UserService"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	strong, err := s.Review(SystemInput{
		SystemName:       "orders",
		DeploymentTarget: "container_apps",
		Services: []ServiceDescriptor{
			{
				Name: "billing", Capability: "manages invoices and dunning",
				Replicated: true, Criticality: "high",
				DependsOn:  []string{"identity"},
				Resilience: []string{"timeout", "retry", "circuit_breaker"},
				OwnsData:   []string{"billing_db"}, Team: "billing",
			},
		},
		Observability:     []string{"otel", "appinsights"},
		SecurityControls:  []string{"entra_id", "managed_identity", "key_vault"},
		APIContracts:      []string{"openapi", "versioning"},
		Messaging:         []string{"service_bus"},
		Patterns:          []string{"saga", "transactional_outbox", "idempotent_consumer", "dlq"},
		NFR:               NFR{AvailabilityTarget: "99.95", RTOMinutes: 30, RPOMinutes: 5, LatencyP99Ms: 200},
		AutoscaleDeclared: true, ScaleToZero: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strong.Score <= weak.Score {
		t.Fatalf("expected strong input to outscore weak: strong=%d weak=%d", strong.Score, weak.Score)
	}
}

func TestReviewFlagsSharedDataStores(t *testing.T) {
	s := NewService()
	res, err := s.Review(SystemInput{
		SystemName: "orders",
		Services: []ServiceDescriptor{
			{Name: "a", OwnsData: []string{"shared_db"}, Capability: "x"},
			{Name: "b", OwnsData: []string{"shared_db"}, Capability: "y"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, d := range res.Dimensions {
		if d.Number == 1 {
			for _, defect := range d.Defects {
				if strings.Contains(defect, "shared data store") {
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("expected D1 to flag shared data stores")
	}
}

func TestReviewIsInputAware(t *testing.T) {
	// Two different inputs must produce different outputs. This is the
	// stub-detection regression guard.
	s := NewService()
	a, err := s.Review(SystemInput{
		SystemName: "alpha",
		Services:   []ServiceDescriptor{{Name: "a"}, {Name: "b"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := s.Review(SystemInput{
		SystemName:       "beta",
		DeploymentTarget: "container_apps",
		Services: []ServiceDescriptor{
			{Name: "c", Capability: "x", Team: "t", Resilience: []string{"timeout", "retry"}},
		},
		Observability:     []string{"otel"},
		SecurityControls:  []string{"entra_id", "managed_identity", "key_vault"},
		APIContracts:      []string{"openapi", "versioning"},
		Patterns:          []string{"saga", "outbox", "idempotent"},
		NFR:               NFR{AvailabilityTarget: "99.9", RTOMinutes: 60, RPOMinutes: 5},
		AutoscaleDeclared: true, ScaleToZero: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Score == b.Score && a.Verdict == b.Verdict && len(a.HardFails) == len(b.HardFails) {
		t.Fatalf("review is not input-aware: a=%+v b=%+v", a, b)
	}
}

// ---------------------------------------------------------------------------
// RecommendPatterns
// ---------------------------------------------------------------------------

func TestRecommendRequiresProblem(t *testing.T) {
	s := NewService()
	if _, err := s.RecommendPatterns(PatternRecommendInput{}); err == nil {
		t.Fatal("expected error when problem is empty")
	}
}

func TestRecommendSagaForCrossServiceTransaction(t *testing.T) {
	s := NewService()
	res, err := s.RecommendPatterns(PatternRecommendInput{
		Problem: "We need to coordinate a long-running transaction across services with compensation steps.",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasPattern(res.Recommendations, "saga") {
		t.Fatalf("expected saga recommendation; got %+v", res.Recommendations)
	}
}

func TestRecommendBFFForMultipleFrontends(t *testing.T) {
	s := NewService()
	res, err := s.RecommendPatterns(PatternRecommendInput{
		Problem: "We have a mobile app and a web frontend that each need different views of the same backend.",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasPattern(res.Recommendations, "backend_for_frontend") {
		t.Fatalf("expected backend_for_frontend recommendation; got %+v", res.Recommendations)
	}
}

func TestRecommendStranglerForLegacyMigration(t *testing.T) {
	s := NewService()
	res, err := s.RecommendPatterns(PatternRecommendInput{
		Problem: "We need to migrate a legacy monolith and extract bounded contexts incrementally.",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasPattern(res.Recommendations, "strangler_fig") {
		t.Fatalf("expected strangler_fig recommendation; got %+v", res.Recommendations)
	}
}

func TestRecommendFlagsTwoPhaseCommitAntiPattern(t *testing.T) {
	s := NewService()
	res, err := s.RecommendPatterns(PatternRecommendInput{
		Problem: "We want to use two-phase commit across services for the order workflow.",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.NotRecommended) == 0 {
		t.Fatal("expected 2PC anti-pattern to be flagged")
	}
}

func TestRecommendIsInputAware(t *testing.T) {
	// Two different problems must yield different recommendation sets.
	s := NewService()
	a, err := s.RecommendPatterns(PatternRecommendInput{
		Problem: "Cache hot reads and reduce database pressure on the catalog browse endpoint.",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := s.RecommendPatterns(PatternRecommendInput{
		Problem: "Coordinate a long-running transaction across services with compensation steps and DLQ for failed events.",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if patternSet(a.Recommendations) == patternSet(b.Recommendations) {
		t.Fatalf("recommend is not input-aware: both inputs returned the same set %v", patternSet(a.Recommendations))
	}
}

// ---------------------------------------------------------------------------
// ScoreWellArchitected
// ---------------------------------------------------------------------------

func TestScoreRequiresSystemName(t *testing.T) {
	s := NewService()
	if _, err := s.ScoreWellArchitected(SystemInput{}); err == nil {
		t.Fatal("expected error when system_name is missing")
	}
}

func TestScoreReturnsAllFivePillars(t *testing.T) {
	s := NewService()
	res, err := s.ScoreWellArchitected(SystemInput{SystemName: "orders"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.OverallScore < 0 || res.OverallScore > 100 {
		t.Fatalf("overall score out of range: %d", res.OverallScore)
	}
	if res.Rating == "" {
		t.Fatal("expected non-empty rating")
	}
	// Each pillar must have a score in [0, 100].
	for label, ps := range map[string]PillarScore{
		"reliability": res.Reliability,
		"security":    res.Security,
		"ops":         res.OperationalExcellence,
		"perf":        res.PerformanceEfficiency,
		"cost":        res.CostOptimization,
	} {
		if ps.Score < 0 || ps.Score > 100 {
			t.Fatalf("%s score out of range: %d", label, ps.Score)
		}
	}
}

func TestScoreIsInputAware(t *testing.T) {
	// Sparse input vs. signal-rich input must produce materially different
	// scorecards. This is the regression guard against the prior stub.
	s := NewService()
	sparse, err := s.ScoreWellArchitected(SystemInput{
		SystemName: "alpha",
		Services:   []ServiceDescriptor{{Name: "a"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rich, err := s.ScoreWellArchitected(SystemInput{
		SystemName:       "alpha",
		DeploymentTarget: "container_apps",
		Services: []ServiceDescriptor{
			{Name: "a", Criticality: "high", Replicated: true, Resilience: []string{"timeout", "retry", "circuit_breaker"}, Team: "t1"},
		},
		Observability:     []string{"otel", "appinsights", "grafana"},
		SecurityControls:  []string{"entra_id", "managed_identity", "key_vault", "mtls"},
		APIContracts:      []string{"openapi", "versioning"},
		Messaging:         []string{"service_bus"},
		Patterns:          []string{"saga", "transactional_outbox", "dlq", "cache_aside"},
		NFR:               NFR{AvailabilityTarget: "99.95", LatencyP99Ms: 200, RTOMinutes: 30, RPOMinutes: 5},
		AutoscaleDeclared: true, ScaleToZero: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sparse.OverallScore >= rich.OverallScore {
		t.Fatalf("rich input should score higher than sparse: sparse=%d rich=%d", sparse.OverallScore, rich.OverallScore)
	}
	if sparse.Reliability.Score >= rich.Reliability.Score {
		t.Fatalf("rich reliability should exceed sparse: sparse=%d rich=%d", sparse.Reliability.Score, rich.Reliability.Score)
	}
	if sparse.Security.Score >= rich.Security.Score {
		t.Fatalf("rich security should exceed sparse: sparse=%d rich=%d", sparse.Security.Score, rich.Security.Score)
	}
}

func TestScoreSecurityRewardsKeyVaultAndPenalizesUnencryptedSensitive(t *testing.T) {
	s := NewService()
	bad, err := s.ScoreWellArchitected(SystemInput{
		SystemName: "x",
		Services:   []ServiceDescriptor{{Name: "a"}},
		DataStores: []DataStoreDescriptor{{Name: "users", Kind: "postgres", Classification: "pii", Encrypted: false}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	good, err := s.ScoreWellArchitected(SystemInput{
		SystemName:       "x",
		Services:         []ServiceDescriptor{{Name: "a"}},
		SecurityControls: []string{"entra_id", "managed_identity", "key_vault"},
		DataStores:       []DataStoreDescriptor{{Name: "users", Kind: "postgres", Classification: "pii", Encrypted: true}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if good.Security.Score <= bad.Security.Score {
		t.Fatalf("Key Vault + encrypted PII should score higher than missing controls + unencrypted PII: bad=%d good=%d",
			bad.Security.Score, good.Security.Score)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func hasPattern(recs []PatternRecommendation, name string) bool {
	for _, r := range recs {
		if r.Pattern == name {
			return true
		}
	}
	return false
}

func patternSet(recs []PatternRecommendation) string {
	names := make([]string, 0, len(recs))
	for _, r := range recs {
		names = append(names, r.Pattern)
	}
	// rely on stable ordering (the service sorts by category then pattern)
	return strings.Join(names, ",")
}

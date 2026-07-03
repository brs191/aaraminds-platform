package boundary

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		input     Input
		wantError string
	}{
		{
			name:      "empty system name",
			input:     Input{Services: []ProposedService{{Name: "svc1", BusinessCapability: "order management"}}},
			wantError: "system_name is required",
		},
		{
			name:      "no services",
			input:     Input{SystemName: "test"},
			wantError: "at least one proposed service is required",
		},
		{
			name: "service with empty name",
			input: Input{
				SystemName: "test",
				Services:   []ProposedService{{Name: "", BusinessCapability: "x"}},
			},
			wantError: "every service must have a non-empty name",
		},
		{
			name: "duplicate service names",
			input: Input{
				SystemName: "test",
				Services: []ProposedService{
					{Name: "svc1", BusinessCapability: "a"},
					{Name: "svc1", BusinessCapability: "b"},
				},
			},
			wantError: "duplicate service name",
		},
	}

	svc := NewService()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.GenerateCanvas(tc.input)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantError)
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("expected error containing %q, got %q", tc.wantError, err.Error())
			}
		})
	}
}

func TestGenerateCanvas_CleanBoundaries(t *testing.T) {
	in := Input{
		SystemName: "clean-system",
		Services: []ProposedService{
			{
				Name:               "order-service",
				BusinessCapability: "order management",
				OwnsData:           []string{"orders"},
				DependsOn:          []string{},
				Team:               "platform",
			},
			{
				Name:               "inventory-service",
				BusinessCapability: "inventory management",
				OwnsData:           []string{"inventory"},
				DependsOn:          []string{},
				Team:               "platform",
			},
		},
	}

	canvas, err := NewService().GenerateCanvas(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if canvas.OverallScore != 100 {
		t.Errorf("expected score 100 for clean boundaries, got %d", canvas.OverallScore)
	}
	if len(canvas.BoundaryRisks) != 0 {
		t.Errorf("expected no risks for clean boundaries, got %d: %+v", len(canvas.BoundaryRisks), canvas.BoundaryRisks)
	}
	if len(canvas.ServiceAssessments) != 2 {
		t.Errorf("expected 2 service assessments, got %d", len(canvas.ServiceAssessments))
	}
}

func TestGenerateCanvas_DataCoOwnership(t *testing.T) {
	in := Input{
		SystemName: "co-owned-system",
		Services: []ProposedService{
			{Name: "a", BusinessCapability: "order management", OwnsData: []string{"orders"}, Team: "team-a"},
			{Name: "b", BusinessCapability: "fulfillment workflow", OwnsData: []string{"orders"}, Team: "team-b"},
		},
	}
	canvas, err := NewService().GenerateCanvas(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundCoOwnership := false
	for _, r := range canvas.BoundaryRisks {
		if r.Category == "data_co_ownership" && r.Severity == "high" {
			foundCoOwnership = true
			if len(r.ServicesAffected) != 2 {
				t.Errorf("expected 2 services affected by co-ownership risk, got %d", len(r.ServicesAffected))
			}
		}
	}
	if !foundCoOwnership {
		t.Errorf("expected high-severity data_co_ownership risk, none found: %+v", canvas.BoundaryRisks)
	}
}

func TestGenerateCanvas_NoOwnerTeam(t *testing.T) {
	in := Input{
		SystemName: "no-owner-system",
		Services: []ProposedService{
			{Name: "orphan", BusinessCapability: "data processing", OwnsData: []string{"events"}},
		},
	}
	canvas, _ := NewService().GenerateCanvas(in)

	found := false
	for _, r := range canvas.BoundaryRisks {
		if r.Category == "no_owner" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected no_owner risk, none found")
	}
}

func TestGenerateCanvas_ChattyDependency(t *testing.T) {
	in := Input{
		SystemName: "chatty-system",
		Services: []ProposedService{
			{Name: "client", BusinessCapability: "order coordination", OwnsData: []string{"orders"},
				DependsOn: []string{"a", "b", "c", "d", "e"}, Team: "team-1"},
			{Name: "a", BusinessCapability: "inventory", OwnsData: []string{"inventory"}, Team: "team-2"},
			{Name: "b", BusinessCapability: "pricing", OwnsData: []string{"prices"}, Team: "team-2"},
			{Name: "c", BusinessCapability: "promotions", OwnsData: []string{"promotions"}, Team: "team-3"},
			{Name: "d", BusinessCapability: "tax calculation", OwnsData: []string{"tax_rates"}, Team: "team-3"},
			{Name: "e", BusinessCapability: "fulfillment", OwnsData: []string{"shipments"}, Team: "team-3"},
		},
	}
	canvas, _ := NewService().GenerateCanvas(in)

	found := false
	for _, r := range canvas.BoundaryRisks {
		if r.Category == "chatty_dependency" && contains(r.ServicesAffected, "client") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected chatty_dependency risk for client, none found: %+v", canvas.BoundaryRisks)
	}
}

func TestGenerateCanvas_FanOut(t *testing.T) {
	// 6 services depending on a central service triggers the fan-out check (>5).
	in := Input{
		SystemName: "fan-out-system",
		Services: []ProposedService{
			{Name: "central", BusinessCapability: "user identity", OwnsData: []string{"users"}, Team: "platform"},
			{Name: "a", BusinessCapability: "order management", DependsOn: []string{"central"}, Team: "team-1"},
			{Name: "b", BusinessCapability: "payment processing", DependsOn: []string{"central"}, Team: "team-1"},
			{Name: "c", BusinessCapability: "inventory management", DependsOn: []string{"central"}, Team: "team-1"},
			{Name: "d", BusinessCapability: "fulfillment workflow", DependsOn: []string{"central"}, Team: "team-1"},
			{Name: "e", BusinessCapability: "notification delivery", DependsOn: []string{"central"}, Team: "team-1"},
			{Name: "f", BusinessCapability: "audit reporting", DependsOn: []string{"central"}, Team: "team-1"},
		},
	}
	canvas, _ := NewService().GenerateCanvas(in)

	foundFanOut := false
	for _, r := range canvas.BoundaryRisks {
		if r.Category == "fan_out" && contains(r.ServicesAffected, "central") && r.Severity == "high" {
			foundFanOut = true
		}
	}
	if !foundFanOut {
		t.Errorf("expected high-severity fan_out risk for central, none found: %+v", canvas.BoundaryRisks)
	}
}

func TestGenerateCanvas_ScoringRationality(t *testing.T) {
	// More risks should produce lower score.
	clean := Input{
		SystemName: "x",
		Services: []ProposedService{
			{Name: "a", BusinessCapability: "order management", OwnsData: []string{"d1"}, Team: "t1"},
		},
	}
	messy := Input{
		SystemName: "y",
		Services: []ProposedService{
			{Name: "a", OwnsData: []string{"shared"}}, // no capability, no team, co-owned data
			{Name: "b", OwnsData: []string{"shared"}}, // same
		},
	}

	cleanCanvas, _ := NewService().GenerateCanvas(clean)
	messyCanvas, _ := NewService().GenerateCanvas(messy)

	if cleanCanvas.OverallScore <= messyCanvas.OverallScore {
		t.Errorf("expected clean (%d) > messy (%d) score", cleanCanvas.OverallScore, messyCanvas.OverallScore)
	}
}

func TestGenerateCanvas_StableOrdering(t *testing.T) {
	// Repeated runs of the same input must produce identical risk ordering.
	in := Input{
		SystemName: "repeat",
		Services: []ProposedService{
			{Name: "a", OwnsData: []string{"x"}, Team: "t1"},
			{Name: "b", OwnsData: []string{"x"}, Team: "t2"},
			{Name: "c", BusinessCapability: "order management", Team: "t3"},
		},
	}
	svc := NewService()
	first, _ := svc.GenerateCanvas(in)
	second, _ := svc.GenerateCanvas(in)

	if len(first.BoundaryRisks) != len(second.BoundaryRisks) {
		t.Fatalf("risk counts differ: %d vs %d", len(first.BoundaryRisks), len(second.BoundaryRisks))
	}
	for i := range first.BoundaryRisks {
		if first.BoundaryRisks[i].Category != second.BoundaryRisks[i].Category {
			t.Errorf("risk ordering not stable at index %d: %q vs %q",
				i, first.BoundaryRisks[i].Category, second.BoundaryRisks[i].Category)
		}
	}
}

func contains(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}

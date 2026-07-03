package azuremap

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
		{"empty system name", Input{Patterns: []string{"cache"}}, "system_name is required"},
		{"no patterns", Input{SystemName: "t"}, "at least one pattern is required"},
	}
	m := NewService()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := m.Map(tc.input)
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("expected error containing %q, got %v", tc.wantError, err)
			}
		})
	}
}

func TestMap_KnownPatternsMapped(t *testing.T) {
	in := Input{
		SystemName:       "sys",
		DeploymentTarget: "aks",
		Patterns:         []string{"api_gateway", "secrets_management", "observability", "container_orchestration"},
	}
	out, err := NewService().Map(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Mappings) != 4 {
		t.Fatalf("expected 4 mappings, got %d", len(out.Mappings))
	}
	if out.Coverage.MappedCount != 4 || out.Coverage.TotalCount != 4 {
		t.Errorf("expected coverage 4/4, got %d/%d", out.Coverage.MappedCount, out.Coverage.TotalCount)
	}
	for _, m := range out.Mappings {
		if m.Pattern == "api_gateway" {
			if len(m.AzureServices) == 0 || !strings.Contains(m.AzureServices[0].Name, "API Management") {
				t.Errorf("expected API Management for api_gateway, got %+v", m.AzureServices)
			}
		}
	}
}

func TestMap_NormalizationAndDedup(t *testing.T) {
	in := Input{
		SystemName:       "sys",
		DeploymentTarget: "aks",
		Patterns:         []string{"API-Gateway", "api gateway", "secrets_management", "observability"},
	}
	out, _ := NewService().Map(in)
	count := 0
	for _, m := range out.Mappings {
		if m.Pattern == "api_gateway" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected api_gateway mapped once after normalization+dedup, got %d", count)
	}
}

func TestMap_UnknownPatternFinding(t *testing.T) {
	in := Input{
		SystemName:       "sys",
		DeploymentTarget: "aks",
		Patterns:         []string{"frobnicator", "secrets_management", "observability"},
	}
	out, _ := NewService().Map(in)
	if !contains(out.UnmappedPatterns, "frobnicator") {
		t.Errorf("expected frobnicator in unmapped, got %+v", out.UnmappedPatterns)
	}
	if !hasFinding(out.MappingFindings, "unknown_pattern") {
		t.Errorf("expected unknown_pattern finding, got %+v", out.MappingFindings)
	}
}

func TestMap_DeploymentMismatchFinding(t *testing.T) {
	in := Input{
		SystemName:       "sys",
		DeploymentTarget: "functions",
		Patterns:         []string{"container_orchestration", "secrets_management", "observability"},
	}
	out, _ := NewService().Map(in)
	if !hasFinding(out.MappingFindings, "deployment_mismatch") {
		t.Errorf("expected deployment_mismatch finding, got %+v", out.MappingFindings)
	}
}

func TestMap_MissingCrossCuttingFindings(t *testing.T) {
	in := Input{
		SystemName:       "sys",
		DeploymentTarget: "aks",
		Patterns:         []string{"cache"},
	}
	out, _ := NewService().Map(in)
	if !hasFinding(out.MappingFindings, "missing_secrets_management") {
		t.Errorf("expected missing_secrets_management finding")
	}
	if !hasFinding(out.MappingFindings, "missing_observability") {
		t.Errorf("expected missing_observability finding")
	}
}

func TestMap_ScoringRationality(t *testing.T) {
	clean := Input{
		SystemName: "x", DeploymentTarget: "aks",
		Patterns: []string{"api_gateway", "secrets_management", "observability", "relational_data"},
	}
	messy := Input{
		SystemName: "y", DeploymentTarget: "functions",
		Patterns: []string{"container_orchestration", "frob1", "frob2"},
	}
	cc, _ := NewService().Map(clean)
	mc, _ := NewService().Map(messy)
	if cc.MappingScore <= mc.MappingScore {
		t.Errorf("expected clean (%d) > messy (%d)", cc.MappingScore, mc.MappingScore)
	}
}

func TestMap_StableOrdering(t *testing.T) {
	in := Input{
		SystemName:       "repeat",
		DeploymentTarget: "aks",
		Patterns:         []string{"observability", "api_gateway", "cache", "secrets_management"},
	}
	m := NewService()
	first, _ := m.Map(in)
	second, _ := m.Map(in)
	if len(first.Mappings) != len(second.Mappings) {
		t.Fatalf("mapping counts differ")
	}
	for i := range first.Mappings {
		if first.Mappings[i].Pattern != second.Mappings[i].Pattern {
			t.Errorf("mapping ordering not stable at %d: %q vs %q", i, first.Mappings[i].Pattern, second.Mappings[i].Pattern)
		}
	}
}

func hasFinding(fs []Finding, category string) bool {
	for _, f := range fs {
		if f.Category == category {
			return true
		}
	}
	return false
}

func contains(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}

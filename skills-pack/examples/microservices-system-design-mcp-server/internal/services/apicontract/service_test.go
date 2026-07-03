package apicontract

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
		{"empty system name", Input{Services: []Service{{Name: "svc"}}}, "system_name is required"},
		{"no services", Input{SystemName: "test"}, "at least one service is required"},
		{"empty service name", Input{SystemName: "t", Services: []Service{{Name: ""}}}, "every service must have a non-empty name"},
		{"duplicate service", Input{SystemName: "t", Services: []Service{{Name: "a"}, {Name: "a"}}}, "duplicate service name"},
	}
	g := NewService()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := g.Generate(tc.input)
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("expected error containing %q, got %v", tc.wantError, err)
			}
		})
	}
}

func TestGenerate_EndpointGeneration(t *testing.T) {
	in := Input{
		SystemName:         "shop",
		VersioningStrategy: "uri",
		Services: []Service{
			{
				Name: "order-service", BusinessCapability: "order management", BasePath: "/orders", Auth: "oauth2",
				Resources: []Resource{
					{Name: "orders", Operations: []string{"list", "get", "create", "update", "delete"}, Paginated: true, Versioned: true},
				},
			},
		},
	}
	c, err := NewService().Generate(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.APIContracts) != 1 {
		t.Fatalf("expected 1 service contract, got %d", len(c.APIContracts))
	}
	sc := c.APIContracts[0]
	if len(sc.Endpoints) != 5 {
		t.Errorf("expected 5 endpoints for 5 operations, got %d", len(sc.Endpoints))
	}
	if sc.Security != "oauth2" {
		t.Errorf("expected security oauth2, got %q", sc.Security)
	}
	if c.OpenAPISummary.TotalOperations != 5 {
		t.Errorf("expected 5 total operations, got %d", c.OpenAPISummary.TotalOperations)
	}
	// POST collection should be 201.
	for _, e := range sc.Endpoints {
		if e.Method == "POST" && e.SuccessStatus != 201 {
			t.Errorf("expected POST success 201, got %d", e.SuccessStatus)
		}
		if e.Method == "DELETE" && e.SuccessStatus != 204 {
			t.Errorf("expected DELETE success 204, got %d", e.SuccessStatus)
		}
	}
}

func TestGenerate_UnsecuredEndpointFinding(t *testing.T) {
	in := Input{
		SystemName:         "shop",
		VersioningStrategy: "uri",
		Services: []Service{
			{Name: "public-service", BusinessCapability: "catalog browsing", Auth: "",
				Resources: []Resource{{Name: "products", Operations: []string{"list"}, Paginated: true}}},
		},
	}
	c, _ := NewService().Generate(in)
	if !hasFinding(c.ContractFindings, "unsecured_endpoint", "high") {
		t.Errorf("expected high unsecured_endpoint finding, got %+v", c.ContractFindings)
	}
}

func TestGenerate_MissingVersioningFinding(t *testing.T) {
	in := Input{
		SystemName: "shop",
		Services: []Service{
			{Name: "s", BusinessCapability: "x", Auth: "oauth2",
				Resources: []Resource{{Name: "r", Operations: []string{"get"}}}},
		},
	}
	c, _ := NewService().Generate(in)
	if !hasFinding(c.ContractFindings, "missing_versioning", "medium") {
		t.Errorf("expected missing_versioning finding, got %+v", c.ContractFindings)
	}
}

func TestGenerate_NoPaginationFinding(t *testing.T) {
	in := Input{
		SystemName:         "shop",
		VersioningStrategy: "uri",
		Services: []Service{
			{Name: "s", BusinessCapability: "x", Auth: "oauth2",
				Resources: []Resource{{Name: "items", Operations: []string{"list"}, Paginated: false}}},
		},
	}
	c, _ := NewService().Generate(in)
	if !hasFinding(c.ContractFindings, "no_pagination", "medium") {
		t.Errorf("expected no_pagination finding, got %+v", c.ContractFindings)
	}
}

func TestGenerate_ScoringRationality(t *testing.T) {
	clean := Input{
		SystemName: "x", VersioningStrategy: "uri",
		Services: []Service{{Name: "a", BusinessCapability: "order management", Auth: "oauth2",
			Resources: []Resource{{Name: "o", Operations: []string{"get"}, Paginated: true}}}},
	}
	messy := Input{
		SystemName: "y",
		Services: []Service{{Name: "a", Auth: "",
			Resources: []Resource{{Name: "o", Operations: []string{"list"}, Paginated: false}}}},
	}
	cc, _ := NewService().Generate(clean)
	mc, _ := NewService().Generate(messy)
	if cc.ContractScore <= mc.ContractScore {
		t.Errorf("expected clean (%d) > messy (%d)", cc.ContractScore, mc.ContractScore)
	}
}

func TestGenerate_StableOrdering(t *testing.T) {
	in := Input{
		SystemName: "repeat",
		Services: []Service{
			{Name: "b", Resources: []Resource{{Name: "x", Operations: []string{"list"}}}},
			{Name: "a", Resources: []Resource{{Name: "y", Operations: []string{"get"}}}},
		},
	}
	g := NewService()
	first, _ := g.Generate(in)
	second, _ := g.Generate(in)
	if len(first.ContractFindings) != len(second.ContractFindings) {
		t.Fatalf("finding counts differ")
	}
	for i := range first.ContractFindings {
		if first.ContractFindings[i].Category != second.ContractFindings[i].Category {
			t.Errorf("finding ordering not stable at %d", i)
		}
	}
}

func hasFinding(fs []Finding, category, severity string) bool {
	for _, f := range fs {
		if f.Category == category && f.Severity == severity {
			return true
		}
	}
	return false
}

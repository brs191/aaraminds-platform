package archrisks

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
			input:     Input{Services: []Component{{Name: "svc1"}}},
			wantError: "system_name is required",
		},
		{
			name:      "no services",
			input:     Input{SystemName: "test"},
			wantError: "at least one service is required",
		},
		{
			name: "service with empty name",
			input: Input{
				SystemName: "test",
				Services:   []Component{{Name: ""}},
			},
			wantError: "every service must have a non-empty name",
		},
		{
			name: "duplicate service names",
			input: Input{
				SystemName: "test",
				Services: []Component{
					{Name: "svc1"},
					{Name: "svc1"},
				},
			},
			wantError: "duplicate service name",
		},
	}

	svc := NewService()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Detect(tc.input)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantError)
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("expected error containing %q, got %q", tc.wantError, err.Error())
			}
		})
	}
}

func TestDetect_CleanArchitecture(t *testing.T) {
	in := Input{
		SystemName:       "clean-system",
		DeploymentTarget: "container_apps",
		NonFunctionalRequirements: NFR{
			AvailabilityTarget: "99.9", RTOMinutes: 15, RPOMinutes: 5,
		},
		Services: []Component{
			{
				Name: "order-service", Criticality: "high", Replicated: true,
				DataStores: []string{"orders-db"}, Resilience: []string{"retry", "circuit_breaker", "timeout"},
			},
			{
				Name: "inventory-service", Criticality: "medium", Replicated: true,
				DataStores: []string{"inventory-db"}, Resilience: []string{"timeout"},
			},
		},
		DataStores: []DataStore{
			{Name: "orders-db", Kind: "postgres", Encrypted: true},
			{Name: "inventory-db", Kind: "postgres", Encrypted: true},
		},
	}
	rep, err := NewService().Detect(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rep.Risks) != 0 {
		t.Errorf("expected no risks for a clean architecture, got %d: %+v", len(rep.Risks), rep.Risks)
	}
	if rep.RiskPostureScore != 100 {
		t.Errorf("expected score 100 for a clean architecture, got %d", rep.RiskPostureScore)
	}
}

func TestDetect_SinglePointOfFailure(t *testing.T) {
	in := Input{
		SystemName:       "spof-system",
		DeploymentTarget: "aks",
		Services: []Component{
			{Name: "auth", Criticality: "high", Replicated: false, Resilience: []string{"timeout"}},
			{Name: "a", DependsOn: []string{"auth"}},
			{Name: "b", DependsOn: []string{"auth"}},
			{Name: "c", DependsOn: []string{"auth"}},
			{Name: "d", DependsOn: []string{"auth"}},
			{Name: "e", DependsOn: []string{"auth"}},
			{Name: "f", DependsOn: []string{"auth"}},
		},
	}
	rep, _ := NewService().Detect(in)
	found := false
	for _, r := range rep.Risks {
		if r.Category == "single_point_of_failure" && contains(r.ComponentsAffected, "auth") && r.Severity == "high" {
			found = true
			if r.Likelihood != "high" {
				t.Errorf("expected likelihood high for fan-in 6, got %q", r.Likelihood)
			}
		}
	}
	if !found {
		t.Errorf("expected high-severity single_point_of_failure for auth, none found: %+v", rep.Risks)
	}
}

func TestDetect_StatefulWithoutDatastore(t *testing.T) {
	in := Input{
		SystemName:       "stateful-system",
		DeploymentTarget: "aks",
		Services: []Component{
			{Name: "session-service", Stateful: true},
			{Name: "api", Replicated: true},
		},
	}
	rep, _ := NewService().Detect(in)
	found := false
	for _, r := range rep.Risks {
		if r.Category == "stateful_without_datastore" && contains(r.ComponentsAffected, "session-service") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected stateful_without_datastore risk, none found: %+v", rep.Risks)
	}
}

func TestDetect_SharedDataStore(t *testing.T) {
	in := Input{
		SystemName:       "shared-db-system",
		DeploymentTarget: "aks",
		Services: []Component{
			{Name: "a", DataStores: []string{"shared"}, Replicated: true},
			{Name: "b", DataStores: []string{"shared"}, Replicated: true},
			{Name: "c", DataStores: []string{"shared"}, Replicated: true},
		},
		DataStores: []DataStore{{Name: "shared", Kind: "postgres", Encrypted: true}},
	}
	rep, _ := NewService().Detect(in)
	found := false
	for _, r := range rep.Risks {
		if r.Category == "shared_data_store" && contains(r.ComponentsAffected, "a") {
			found = true
			if r.Likelihood != "high" {
				t.Errorf("expected likelihood high for 3 users, got %q", r.Likelihood)
			}
		}
	}
	if !found {
		t.Errorf("expected shared_data_store risk, none found: %+v", rep.Risks)
	}
}

func TestDetect_UnencryptedSensitiveData(t *testing.T) {
	in := Input{
		SystemName:       "pii-system",
		DeploymentTarget: "aks",
		Services:         []Component{{Name: "user-service", DataStores: []string{"users"}, Replicated: true}},
		DataStores:       []DataStore{{Name: "users", Kind: "postgres", Classification: "pii", Encrypted: false}},
	}
	rep, _ := NewService().Detect(in)
	found := false
	for _, r := range rep.Risks {
		if r.Category == "unencrypted_sensitive_data" && r.Severity == "high" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected unencrypted_sensitive_data risk, none found: %+v", rep.Risks)
	}
}

func TestDetect_ComplianceConstraintUnaddressed(t *testing.T) {
	in := Input{
		SystemName:       "compliance-system",
		DeploymentTarget: "aks",
		Constraints:      []string{"must satisfy GDPR data residency in EU"},
		Services:         []Component{{Name: "svc", DataStores: []string{"db"}, Replicated: true}},
		DataStores:       []DataStore{{Name: "db", Kind: "postgres", Encrypted: true}}, // no classification
	}
	rep, _ := NewService().Detect(in)
	found := false
	for _, r := range rep.Risks {
		if r.Category == "compliance_constraint_unaddressed" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected compliance_constraint_unaddressed risk, none found: %+v", rep.Risks)
	}
}

func TestDetect_MissingDecisions(t *testing.T) {
	in := Input{
		SystemName: "ambiguous-system",
		Services:   []Component{{Name: "a"}, {Name: "b"}},
	}
	rep, _ := NewService().Detect(in)
	joined := strings.Join(rep.MissingDecisions, " | ")
	for _, want := range []string{"deployment target", "availability SLO", "criticality not classified", "RTO/RPO"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected missing decision mentioning %q, got: %s", want, joined)
		}
	}
}

func TestDetect_ScoringRationality(t *testing.T) {
	clean := Input{
		SystemName:                "x",
		DeploymentTarget:          "aks",
		NonFunctionalRequirements: NFR{AvailabilityTarget: "99.9", RTOMinutes: 10, RPOMinutes: 5},
		Services: []Component{
			{Name: "a", Criticality: "medium", Replicated: true, DataStores: []string{"d1"}},
		},
		DataStores: []DataStore{{Name: "d1", Encrypted: true}},
	}
	messy := Input{
		SystemName: "y",
		Services: []Component{
			{Name: "a", Criticality: "high", Stateful: true},
			{Name: "b", DataStores: []string{"shared"}},
			{Name: "c", DataStores: []string{"shared"}},
		},
		Constraints: []string{"HIPAA"},
		DataStores:  []DataStore{{Name: "shared"}},
	}
	cleanRep, _ := NewService().Detect(clean)
	messyRep, _ := NewService().Detect(messy)
	if cleanRep.RiskPostureScore <= messyRep.RiskPostureScore {
		t.Errorf("expected clean (%d) > messy (%d) score", cleanRep.RiskPostureScore, messyRep.RiskPostureScore)
	}
}

func TestDetect_StableOrdering(t *testing.T) {
	in := Input{
		SystemName: "repeat",
		Services: []Component{
			{Name: "a", Criticality: "high", Stateful: true},
			{Name: "b", DataStores: []string{"x"}},
			{Name: "c", DataStores: []string{"x"}},
		},
		DataStores: []DataStore{{Name: "x"}},
	}
	svc := NewService()
	first, _ := svc.Detect(in)
	second, _ := svc.Detect(in)
	if len(first.Risks) != len(second.Risks) {
		t.Fatalf("risk counts differ: %d vs %d", len(first.Risks), len(second.Risks))
	}
	for i := range first.Risks {
		if first.Risks[i].Category != second.Risks[i].Category {
			t.Errorf("risk ordering not stable at index %d: %q vs %q",
				i, first.Risks[i].Category, second.Risks[i].Category)
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

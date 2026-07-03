package obsplan

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
		{"empty system name", Input{Services: []Service{{Name: "s"}}}, "system_name is required"},
		{"no services", Input{SystemName: "t"}, "at least one service is required"},
		{"empty service name", Input{SystemName: "t", Services: []Service{{Name: ""}}}, "every service must have a non-empty name"},
		{"duplicate service", Input{SystemName: "t", Services: []Service{{Name: "a"}, {Name: "a"}}}, "duplicate service name"},
	}
	p := NewService()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := p.Generate(tc.input)
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("expected error containing %q, got %v", tc.wantError, err)
			}
		})
	}
}

func TestGenerate_APIServiceGetsLatencySLI(t *testing.T) {
	in := Input{
		SystemName:                "sys",
		NonFunctionalRequirements: NFR{AvailabilityTarget: "99.9", LatencyP99Ms: 250},
		Services: []Service{
			{Name: "api-svc", Criticality: "high", Type: "api", HasDashboards: true, HasAlerts: true},
		},
	}
	plan, err := NewService().Generate(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	so := plan.ServiceObservability[0]
	if !hasSLI(so.SLIs, "latency_p99") {
		t.Errorf("expected latency_p99 SLI for api service, got %+v", so.SLIs)
	}
	foundLatencySLO := false
	for _, s := range so.SLOs {
		if s.SLI == "latency_p99" && strings.Contains(s.Objective, "250ms") {
			foundLatencySLO = true
		}
	}
	if !foundLatencySLO {
		t.Errorf("expected latency SLO using NFR 250ms, got %+v", so.SLOs)
	}
}

func TestGenerate_WorkerGetsThroughputSLI(t *testing.T) {
	in := Input{
		SystemName:                "sys",
		NonFunctionalRequirements: NFR{AvailabilityTarget: "99.9"},
		Services:                  []Service{{Name: "worker", Type: "worker", HasDashboards: true, HasAlerts: true}},
	}
	plan, _ := NewService().Generate(in)
	if !hasSLI(plan.ServiceObservability[0].SLIs, "throughput") {
		t.Errorf("expected throughput SLI for worker, got %+v", plan.ServiceObservability[0].SLIs)
	}
}

func TestGenerate_NoAlertsGapHighForCritical(t *testing.T) {
	in := Input{
		SystemName:                "sys",
		NonFunctionalRequirements: NFR{AvailabilityTarget: "99.9", LatencyP99Ms: 200},
		Services:                  []Service{{Name: "critical-svc", Criticality: "high", Type: "api", HasDashboards: true, HasAlerts: false}},
	}
	plan, _ := NewService().Generate(in)
	found := false
	for _, g := range plan.CoverageGaps {
		if g.Category == "no_alerts" && g.Severity == "high" && contains(g.ServicesAffected, "critical-svc") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected high no_alerts gap for critical service, got %+v", plan.CoverageGaps)
	}
}

func TestGenerate_MissingAvailabilityTargetGap(t *testing.T) {
	in := Input{
		SystemName: "sys",
		Services:   []Service{{Name: "s", Type: "worker", HasDashboards: true, HasAlerts: true}},
	}
	plan, _ := NewService().Generate(in)
	found := false
	for _, g := range plan.CoverageGaps {
		if g.Category == "missing_availability_target" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing_availability_target gap, got %+v", plan.CoverageGaps)
	}
}

func TestGenerate_ScoringRationality(t *testing.T) {
	clean := Input{
		SystemName:                "x",
		NonFunctionalRequirements: NFR{AvailabilityTarget: "99.9", LatencyP99Ms: 200},
		Services:                  []Service{{Name: "a", Criticality: "medium", Type: "api", HasDashboards: true, HasAlerts: true}},
	}
	messy := Input{
		SystemName: "y",
		Services:   []Service{{Name: "a", Criticality: "high", Type: "api", HasDashboards: false, HasAlerts: false}},
	}
	cc, _ := NewService().Generate(clean)
	mc, _ := NewService().Generate(messy)
	if cc.ObservabilityScore <= mc.ObservabilityScore {
		t.Errorf("expected clean (%d) > messy (%d)", cc.ObservabilityScore, mc.ObservabilityScore)
	}
}

func TestGenerate_StableOrdering(t *testing.T) {
	in := Input{
		SystemName: "repeat",
		Services: []Service{
			{Name: "b", Type: "api", HasDashboards: false, HasAlerts: false},
			{Name: "a", Type: "worker", HasDashboards: false, HasAlerts: false},
		},
	}
	p := NewService()
	first, _ := p.Generate(in)
	second, _ := p.Generate(in)
	if len(first.CoverageGaps) != len(second.CoverageGaps) {
		t.Fatalf("gap counts differ")
	}
	for i := range first.CoverageGaps {
		if first.CoverageGaps[i].Category != second.CoverageGaps[i].Category {
			t.Errorf("gap ordering not stable at %d", i)
		}
	}
}

func hasSLI(slis []SLI, name string) bool {
	for _, s := range slis {
		if s.Name == name {
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

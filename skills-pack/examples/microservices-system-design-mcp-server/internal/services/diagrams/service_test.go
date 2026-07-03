package diagrams

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      Input
		wantErr bool
	}{
		{name: "ok context", in: Input{SystemName: "sys", DiagramType: "context"}},
		{name: "missing system", in: Input{DiagramType: "context"}, wantErr: true},
		{name: "missing type", in: Input{SystemName: "sys"}, wantErr: true},
		{name: "unsupported type", in: Input{SystemName: "sys", DiagramType: "doodle"}, wantErr: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewService().Generate(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerate_ContextProducesAllThreeAssets(t *testing.T) {
	t.Parallel()
	out, err := NewService().Generate(Input{
		SystemName:      "ecommerce",
		Audience:        "technical",
		DiagramType:     "context",
		ExternalSystems: []ExternalSystem{{Name: "stripe", Direction: "outbound"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.Mermaid, "flowchart LR") {
		t.Error("expected Mermaid flowchart")
	}
	if !strings.Contains(out.PlantUML, "@startuml") {
		t.Error("expected PlantUML markers")
	}
	if !strings.Contains(out.DrawIOPrompt, "context") {
		t.Error("expected draw.io prompt to reference diagram type")
	}
}

func TestGenerate_DeploymentMentionsContainerApps(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName:  "ecommerce",
		DiagramType: "deployment",
		Services:    []Service{{Name: "order", Type: "api", OwnsData: []string{"orders"}}},
	})
	if !strings.Contains(out.Mermaid, "Container Apps") {
		t.Errorf("deployment Mermaid should reference Container Apps, got %q", out.Mermaid)
	}
	if !strings.Contains(out.Mermaid, "orders") {
		t.Error("deployment Mermaid missing data tier reference")
	}
}

func TestGenerate_EventFlowConnectsProducersToConsumers(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName:  "sys",
		DiagramType: "event_flow",
		Events: []Event{
			{Name: "OrderCreated", Producer: "order-service", Consumers: []string{"notification-service", "inventory-service"}},
		},
	})
	if !strings.Contains(out.Mermaid, "OrderCreated") {
		t.Error("event flow should include event name")
	}
	if !strings.Contains(out.Mermaid, "notification-service") {
		t.Error("event flow should include consumer name")
	}
}

func TestGenerate_AudienceNotes(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName:  "sys",
		DiagramType: "context",
		Audience:    "executive",
	})
	hasAudienceNote := false
	for _, n := range out.Notes {
		if strings.Contains(strings.ToLower(n), "executive") || strings.Contains(strings.ToLower(n), "jargon") {
			hasAudienceNote = true
		}
	}
	if !hasAudienceNote {
		t.Error("expected executive audience note")
	}
}

func TestGenerate_ServiceBoundaryShowsDataPerBoundary(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName:  "sys",
		DiagramType: "service_boundary",
		Services: []Service{
			{Name: "order", Type: "api", OwnsData: []string{"orders"}, DependsOn: []string{"payment"}},
			{Name: "payment", Type: "api", OwnsData: []string{"payments"}},
		},
	})
	if !strings.Contains(out.Mermaid, "subgraph") {
		t.Error("service boundary should use subgraph per service")
	}
	if !strings.Contains(out.Mermaid, "orders") || !strings.Contains(out.Mermaid, "payments") {
		t.Error("service boundary should display owned data")
	}
}

func TestGenerate_Stable(t *testing.T) {
	t.Parallel()
	in := Input{
		SystemName:  "sys",
		DiagramType: "context",
		ExternalSystems: []ExternalSystem{
			{Name: "z", Direction: "outbound"},
			{Name: "a", Direction: "outbound"},
		},
	}
	a, _ := NewService().Generate(in)
	b, _ := NewService().Generate(in)
	if a.Mermaid != b.Mermaid || a.PlantUML != b.PlantUML || a.DrawIOPrompt != b.DrawIOPrompt {
		t.Error("output not byte-stable across calls")
	}
}

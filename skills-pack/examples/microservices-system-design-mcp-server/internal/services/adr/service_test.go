package adr

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
		{name: "ok minimum", in: Input{SystemName: "sys", Title: "Use X", Decision: "Adopt X"}},
		{name: "missing system", in: Input{Title: "Use X", Decision: "Adopt X"}, wantErr: true},
		{name: "missing title", in: Input{SystemName: "sys", Decision: "Adopt X"}, wantErr: true},
		{name: "missing decision", in: Input{SystemName: "sys", Title: "Use X"}, wantErr: true},
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

func TestGenerate_FullInput_ProducesCompleteADR(t *testing.T) {
	t.Parallel()
	in := Input{
		SystemName: "order-platform",
		Title:      "Use Saga with Transactional Outbox for Order Workflow",
		Status:     "accepted",
		Date:       "2026-05-18",
		DecidedBy:  "platform-team",
		Drivers: []string{
			"cloud-native fit",
			"reliability across service boundaries",
			"avoid distributed two-phase commit",
		},
		Context: "The order workflow spans order capture, inventory reservation, payment authorization, " +
			"and fulfillment. Each service owns its own data, so a distributed database transaction is not " +
			"appropriate. We need reliable event publication and compensation on failure.",
		Decision: "Use the Saga pattern coordinated by Durable Functions, with each service publishing " +
			"events via a transactional outbox and consumers implementing idempotency.",
		Options: []Option{
			{
				Name:            "Distributed two-phase commit",
				Pros:            []string{"Strong consistency"},
				Cons:            []string{"Poor cloud fit", "Operational complexity"},
				Rejected:        true,
				RejectedBecause: "Cloud-native services cannot economically participate in 2PC across boundaries.",
			},
			{
				Name:            "Single checkout service (monolith)",
				Pros:            []string{"Simpler initial implementation"},
				Cons:            []string{"Weak boundaries", "Limits independent scale"},
				Rejected:        true,
				RejectedBecause: "Conflicts with the architecture's bounded-context discipline and team ownership.",
			},
			{
				Name:     "Saga + Outbox + Idempotent Consumer",
				Pros:     []string{"Cloud-native", "Recoverable", "Auditable"},
				Cons:     []string{"Requires compensation logic", "Eventual consistency"},
				Rejected: false,
			},
		},
		Consequences: Consequences{
			Positive: []string{"Services remain independently deployable", "Workflow state is recoverable from event history"},
			Negative: []string{"Compensation must be implemented and tested per step", "Eventual consistency must surface in product UX"},
			Neutral:  []string{"Observability must include correlation IDs across all steps"},
		},
		References: []string{
			"skills/microservices/05-data-architecture.md",
			"patterns/microservices/saga.md",
			"patterns/microservices/transactional-outbox.md",
		},
	}

	got, err := NewService().Generate(in)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if got.Status != "Accepted" {
		t.Errorf("Status = %q, want Accepted", got.Status)
	}
	if got.Date != "2026-05-18" {
		t.Errorf("Date = %q, want 2026-05-18", got.Date)
	}
	if got.QualityScore < 90 {
		t.Errorf("QualityScore = %d, want >=90 (full input)", got.QualityScore)
	}
	if len(got.Warnings) != 0 {
		t.Errorf("Warnings = %v, want none for full input", got.Warnings)
	}
	if !strings.Contains(got.Markdown, "# ADR: Use Saga with Transactional Outbox") {
		t.Errorf("Markdown missing ADR title header")
	}
	if !strings.Contains(got.Markdown, "**Decided by:** platform-team") {
		t.Errorf("Markdown missing decided-by line")
	}
	if !strings.Contains(got.Markdown, "Rejected because:") {
		t.Errorf("Markdown missing rejection rationale section")
	}
}

func TestGenerate_MinimalInput_HasWarningsAndLowScore(t *testing.T) {
	t.Parallel()
	in := Input{
		SystemName: "sys",
		Title:      "Adopt Y",
		Context:    "Short.",
		Decision:   "Use Y.",
	}
	got, err := NewService().Generate(in)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if got.QualityScore >= 80 {
		t.Errorf("QualityScore = %d, want <80 for minimal input", got.QualityScore)
	}
	if len(got.Warnings) < 4 {
		t.Errorf("Warnings = %d, want >=4 (short context, no drivers, no options, no refs)", len(got.Warnings))
	}
	if got.Status != "Proposed" {
		t.Errorf("Status = %q, want Proposed (default)", got.Status)
	}
}

func TestGenerate_DerivesConsequencesWhenEmpty(t *testing.T) {
	t.Parallel()
	in := Input{
		SystemName: "sys",
		Title:      "Adopt Y",
		Decision:   "Use Y",
		Drivers:    []string{"speed", "cost"},
		Options:    []Option{{Name: "X", Rejected: true, RejectedBecause: "slow"}},
	}
	got, err := NewService().Generate(in)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if isEmptyConsequences(got.Consequences) {
		t.Error("expected derived consequences when input had none")
	}
	hasDriverPositive := false
	for _, p := range got.Consequences.Positive {
		if strings.Contains(p, "speed") {
			hasDriverPositive = true
		}
	}
	if !hasDriverPositive {
		t.Error("expected derived positive consequence to mention drivers")
	}
}

func TestGenerate_MarkdownStable(t *testing.T) {
	t.Parallel()
	in := Input{
		SystemName: "sys",
		Title:      "Adopt Y",
		Decision:   "Use Y",
		Date:       "2026-01-01",
		Drivers:    []string{"a", "b"},
	}
	a, _ := NewService().Generate(in)
	b, _ := NewService().Generate(in)
	if a.Markdown != b.Markdown {
		t.Error("Markdown not byte-stable across calls with the same input")
	}
}

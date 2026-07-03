package resilience

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
		{name: "ok", in: Input{SystemName: "sys", Services: []Service{{Name: "a"}}}},
		{name: "missing system", in: Input{Services: []Service{{Name: "a"}}}, wantErr: true},
		{name: "no services", in: Input{SystemName: "sys"}, wantErr: true},
		{name: "service missing name", in: Input{SystemName: "sys", Services: []Service{{}}}, wantErr: true},
		{name: "dep missing from", in: Input{SystemName: "sys", Services: []Service{{Name: "a"}}, Dependencies: []Dependency{{To: "a"}}}, wantErr: true},
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

func TestGenerate_HighCriticalityHasTighterRetry(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName: "sys",
		Services: []Service{
			{Name: "order", Criticality: "high"},
			{Name: "payment", Criticality: "high"},
		},
		Dependencies: []Dependency{{From: "order", To: "payment"}},
	})
	if len(out.DependencyControls) != 1 {
		t.Fatalf("want 1 control, got %d", len(out.DependencyControls))
	}
	c := out.DependencyControls[0]
	if c.RetryAttempts != 2 {
		t.Errorf("high-crit retry attempts = %d, want 2 (tighter)", c.RetryAttempts)
	}
	if !strings.Contains(c.Timeout, "1.5") {
		t.Errorf("high-crit timeout = %q, want 1.5s-shape", c.Timeout)
	}
}

func TestGenerate_ExternalAPIGetsStrictPosture(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName:   "sys",
		Services:     []Service{{Name: "payment"}},
		ExternalAPIs: []ExternalAPI{{Name: "stripe", UsedBy: []string{"payment"}}},
	})
	if len(out.DependencyControls) != 1 {
		t.Fatalf("want 1 control, got %d", len(out.DependencyControls))
	}
	c := out.DependencyControls[0]
	if !c.IdempotencyKey {
		t.Error("external API mutation should require idempotency key")
	}
	if !strings.Contains(c.Notes, "External") {
		t.Errorf("expected note flagging external dependency, got %q", c.Notes)
	}
	if !strings.Contains(c.CircuitBreaker, "error rate") {
		t.Errorf("expected error-rate based breaker, got %q", c.CircuitBreaker)
	}
}

func TestGenerate_IdempotentSkipKeyRequirement(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName:   "sys",
		Services:     []Service{{Name: "a"}, {Name: "b"}},
		Dependencies: []Dependency{{From: "a", To: "b", Idempotent: true}},
	})
	if out.DependencyControls[0].IdempotencyKey {
		t.Error("idempotent operation should not require an explicit idempotency key")
	}
	if !strings.Contains(out.DependencyControls[0].Notes, "Idempotent") {
		t.Errorf("expected idempotent note, got %q", out.DependencyControls[0].Notes)
	}
}

func TestGenerate_SingleReplicaHighCritFlagged(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName: "sys",
		Services:   []Service{{Name: "auth", Criticality: "high", Replicated: false}},
	})
	if len(out.BulkheadNotes) == 0 {
		t.Fatal("expected a bulkhead note for high-crit single-replica service")
	}
	if !strings.Contains(out.BulkheadNotes[0].Notes, "Single-replica") {
		t.Errorf("expected single-replica note, got %q", out.BulkheadNotes[0].Notes)
	}
}

func TestGenerate_WorkerGetsQueueGuidance(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName: "sys",
		Services:   []Service{{Name: "notif", Type: "worker"}},
	})
	if len(out.LoadLevelingNotes) == 0 {
		t.Fatal("expected load-leveling note for worker")
	}
	if !strings.Contains(out.LoadLevelingNotes[0].Notes, "queue") {
		t.Errorf("expected queue-based note, got %q", out.LoadLevelingNotes[0].Notes)
	}
}

func TestGenerate_FallbackPerCriticality(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName: "sys",
		Services: []Service{
			{Name: "a", Criticality: "high"},
			{Name: "b", Criticality: "high"},
			{Name: "c", Criticality: "medium"},
			{Name: "d", Criticality: "medium"},
		},
		Dependencies: []Dependency{
			{From: "a", To: "b"},
			{From: "c", To: "d"},
		},
	})
	var high, med string
	for _, f := range out.Fallbacks {
		switch f.Dependency {
		case "b":
			high = f.Strategy
		case "d":
			med = f.Strategy
		}
	}
	if high != "fail_fast" {
		t.Errorf("high-crit fallback = %q, want fail_fast", high)
	}
	if med != "degrade" {
		t.Errorf("medium fallback = %q, want degrade", med)
	}
}

func TestGenerate_CoverageScoreReasonable(t *testing.T) {
	t.Parallel()
	out, _ := NewService().Generate(Input{
		SystemName: "sys",
		Services: []Service{
			{Name: "a", Criticality: "high", Replicated: false},
			{Name: "w", Type: "worker"},
		},
		Dependencies: []Dependency{{From: "a", To: "w"}},
	})
	if out.CoverageScore < 60 {
		t.Errorf("coverage score = %d, want >=60 for non-trivial plan", out.CoverageScore)
	}
}

func TestGenerate_StableOrdering(t *testing.T) {
	t.Parallel()
	in := Input{
		SystemName: "sys",
		Services:   []Service{{Name: "z"}, {Name: "a"}},
		Dependencies: []Dependency{
			{From: "z", To: "a"},
			{From: "a", To: "z"},
		},
	}
	a, _ := NewService().Generate(in)
	b, _ := NewService().Generate(in)
	if a.Summary != b.Summary {
		t.Error("summary not byte-stable")
	}
	if a.DependencyControls[0].From != "a" {
		t.Errorf("expected alphabetical from-ordering, got %v", a.DependencyControls)
	}
}

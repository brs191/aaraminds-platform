// Package resilience implements the service-layer logic for the
// generate_resilience_plan MCP tool.
//
// The service takes a structured architecture description and produces a
// resilience plan: per-dependency timeout / retry / circuit-breaker
// configuration, queue-based load-leveling recommendations, bulkhead
// suggestions, fallback strategies, and detection signals for each control.
//
// The logic is deterministic and rule-based:
//
//   - High-criticality cross-service calls need timeout + retry + breaker
//   - External dependencies (third parties) need stricter retry budget and
//     circuit breaker thresholds
//   - Workers with queue inputs benefit from queue-based load leveling
//   - Stateful or single-replica components need bulkhead notes
//   - Idempotency keys required on POST-shaped retryable mutations
package resilience

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Input is the structured request for a resilience plan.
type Input struct {
	SystemName   string        `json:"system_name"`
	Description  string        `json:"description,omitempty"`
	Services     []Service     `json:"services"`
	Dependencies []Dependency  `json:"dependencies,omitempty"`
	ExternalAPIs []ExternalAPI `json:"external_apis,omitempty"`
	NFR          NFR           `json:"non_functional_requirements,omitempty"`
}

// Service describes a service whose resilience posture we're planning.
type Service struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`        // api | gateway | worker | function
	Criticality string `json:"criticality,omitempty"` // high | medium | low
	Stateful    bool   `json:"stateful,omitempty"`
	Replicated  bool   `json:"replicated,omitempty"`
}

// Dependency is one synchronous call from a service to another internal target.
type Dependency struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Idempotent bool   `json:"idempotent,omitempty"`
}

// ExternalAPI is a third-party HTTP dependency.
type ExternalAPI struct {
	Name      string   `json:"name"`
	UsedBy    []string `json:"used_by,omitempty"`
	StatedSLA string   `json:"stated_sla,omitempty"` // e.g. "99.5" or "varies"
}

// NFR captures targets.
type NFR struct {
	AvailabilityTarget string `json:"availability_target,omitempty"`
	LatencyP99Ms       int    `json:"latency_p99_ms,omitempty"`
}

// Plan is the structured output.
type Plan struct {
	SystemName         string              `json:"system_name"`
	DependencyControls []DependencyControl `json:"dependency_controls"`
	BulkheadNotes      []BulkheadNote      `json:"bulkhead_notes"`
	LoadLevelingNotes  []LoadLevelingNote  `json:"load_leveling_notes"`
	Fallbacks          []FallbackStrategy  `json:"fallbacks"`
	DetectionSignals   []DetectionSignal   `json:"detection_signals"`
	NextSteps          []string            `json:"next_steps"`
	CoverageScore      int                 `json:"coverage_score"` // 0-100
	Summary            string              `json:"summary"`
}

// DependencyControl is the timeout / retry / breaker config for a dependency.
type DependencyControl struct {
	From           string `json:"from"`
	To             string `json:"to"`
	Timeout        string `json:"timeout"` // e.g., "2s"
	RetryAttempts  int    `json:"retry_attempts"`
	RetryBackoff   string `json:"retry_backoff"`   // e.g., "exponential with jitter"
	CircuitBreaker string `json:"circuit_breaker"` // e.g., "5 consecutive failures → open for 30s"
	IdempotencyKey bool   `json:"idempotency_key"`
	Notes          string `json:"notes,omitempty"`
}

// BulkheadNote captures isolation guidance for a service.
type BulkheadNote struct {
	Service string `json:"service"`
	Notes   string `json:"notes"`
}

// LoadLevelingNote captures queue-based smoothing guidance.
type LoadLevelingNote struct {
	Service string `json:"service"`
	Notes   string `json:"notes"`
}

// FallbackStrategy captures the degradation path for a dependency.
type FallbackStrategy struct {
	Dependency string `json:"dependency"`
	Strategy   string `json:"strategy"` // cache | default | degrade | fail_fast | escalate
	Detail     string `json:"detail"`
}

// DetectionSignal names what to watch and how.
type DetectionSignal struct {
	Signal    string `json:"signal"`
	Source    string `json:"source"`
	AlertWhen string `json:"alert_when"`
}

// GeneratorService is the resilience plan service.
type GeneratorService struct{}

// NewService constructs a Service.
func NewService() *GeneratorService { return &GeneratorService{} }

// Generate validates input and produces the plan.
func (s *GeneratorService) Generate(in Input) (Plan, error) {
	if err := validate(in); err != nil {
		return Plan{}, err
	}
	servicesByName := index(in.Services)

	out := Plan{SystemName: in.SystemName}
	out.DependencyControls = controlsFor(in, servicesByName)
	out.BulkheadNotes = bulkheadNotesFor(in)
	out.LoadLevelingNotes = loadLevelingNotesFor(in)
	out.Fallbacks = fallbacksFor(in, servicesByName)
	out.DetectionSignals = detectionSignals()
	out.NextSteps = nextSteps(in)
	out.CoverageScore = computeCoverage(in, out)
	out.Summary = fmt.Sprintf(
		"Resilience plan for %s: %d dependency control(s), %d bulkhead note(s), %d load-leveling note(s), %d fallback(s); coverage %d.",
		in.SystemName, len(out.DependencyControls), len(out.BulkheadNotes), len(out.LoadLevelingNotes), len(out.Fallbacks), out.CoverageScore,
	)
	return out, nil
}

func validate(in Input) error {
	if strings.TrimSpace(in.SystemName) == "" {
		return errors.New("system_name is required")
	}
	if len(in.Services) == 0 {
		return errors.New("at least one service is required")
	}
	for i, s := range in.Services {
		if strings.TrimSpace(s.Name) == "" {
			return fmt.Errorf("services[%d].name is required", i)
		}
	}
	for i, d := range in.Dependencies {
		if strings.TrimSpace(d.From) == "" || strings.TrimSpace(d.To) == "" {
			return fmt.Errorf("dependencies[%d] requires both from and to", i)
		}
	}
	return nil
}

func index(svcs []Service) map[string]Service {
	out := make(map[string]Service, len(svcs))
	for _, s := range svcs {
		out[s.Name] = s
	}
	return out
}

func controlsFor(in Input, byName map[string]Service) []DependencyControl {
	out := []DependencyControl{}

	// Internal dependencies.
	for _, d := range in.Dependencies {
		from := byName[d.From]
		to := byName[d.To]
		c := DependencyControl{From: d.From, To: d.To}
		c.IdempotencyKey = !d.Idempotent // if NOT idempotent, an idempotency key is required for safe retry
		// Defaults.
		c.Timeout = "2s"
		c.RetryAttempts = 3
		c.RetryBackoff = "exponential with jitter (200ms base, max 2s)"
		c.CircuitBreaker = "5 consecutive failures → open for 30s, half-open with one probe"

		// High-criticality caller or callee → tighter posture.
		if isHigh(from) || isHigh(to) {
			c.Timeout = "1.5s"
			c.RetryAttempts = 2
			c.CircuitBreaker = "3 consecutive failures → open for 60s; half-open with one probe"
			c.Notes = "High-criticality boundary; budget retries tightly to preserve the caller's SLA."
		}
		if d.Idempotent {
			c.Notes = strings.TrimSpace(c.Notes + " Idempotent operation: retry is safe without explicit idempotency key.")
		}
		out = append(out, c)
	}

	// External APIs — always strict.
	for _, ext := range in.ExternalAPIs {
		for _, by := range ext.UsedBy {
			c := DependencyControl{
				From:           by,
				To:             ext.Name,
				Timeout:        "5s",
				RetryAttempts:  2,
				RetryBackoff:   "exponential with jitter (500ms base, max 4s)",
				CircuitBreaker: "10% error rate over 1 min → open for 60s",
				IdempotencyKey: true,
				Notes:          "External dependency; classify errors and skip retry on 4xx; idempotency key mandatory for any mutation.",
			}
			if ext.StatedSLA == "varies" || ext.StatedSLA == "" {
				c.Notes = strings.TrimSpace(c.Notes + " Stated SLA unknown; treat as unreliable.")
			}
			out = append(out, c)
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].From == out[j].From {
			return out[i].To < out[j].To
		}
		return out[i].From < out[j].From
	})
	return out
}

func bulkheadNotesFor(in Input) []BulkheadNote {
	out := []BulkheadNote{}
	for _, s := range in.Services {
		notes := []string{}
		if isHigh(s) && !s.Replicated {
			notes = append(notes, "Single-replica high-criticality service — replicate to at least 2 instances.")
		}
		if s.Stateful {
			notes = append(notes, "Stateful service — bulkhead per partition or per tenant to prevent noisy-neighbor saturation.")
		}
		if strings.EqualFold(s.Type, "worker") {
			notes = append(notes, "Worker — set a maximum concurrent message count per replica; reject overflow into DLQ.")
		}
		if len(notes) > 0 {
			out = append(out, BulkheadNote{Service: s.Name, Notes: strings.Join(notes, " ")})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Service < out[j].Service })
	return out
}

func loadLevelingNotesFor(in Input) []LoadLevelingNote {
	out := []LoadLevelingNote{}
	for _, s := range in.Services {
		if strings.EqualFold(s.Type, "worker") {
			out = append(out, LoadLevelingNote{
				Service: s.Name,
				Notes:   "Front the worker with a queue (Service Bus or Storage Queue). Set max-delivery-count and DLQ; scale on queue depth.",
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Service < out[j].Service })
	return out
}

func fallbacksFor(in Input, byName map[string]Service) []FallbackStrategy {
	out := []FallbackStrategy{}
	for _, d := range in.Dependencies {
		to := byName[d.To]
		strategy := FallbackStrategy{Dependency: d.To}
		switch {
		case isHigh(to):
			strategy.Strategy = "fail_fast"
			strategy.Detail = "On open breaker, return 503 to caller; surface in dashboard; do not synthesise data."
		default:
			strategy.Strategy = "degrade"
			strategy.Detail = "On open breaker, serve cached/default; surface degraded status in response payload."
		}
		out = append(out, strategy)
	}
	for _, ext := range in.ExternalAPIs {
		out = append(out, FallbackStrategy{
			Dependency: ext.Name,
			Strategy:   "degrade",
			Detail:     "Cache the last successful response; serve cached on breaker open; queue mutations for replay.",
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Dependency < out[j].Dependency })
	return out
}

func detectionSignals() []DetectionSignal {
	return []DetectionSignal{
		{Signal: "circuit_breaker_state", Source: "Application Insights custom metric", AlertWhen: "state = open for > 60s"},
		{Signal: "retry_count_per_request", Source: "Application Insights logs", AlertWhen: "avg retries/req > 0.5 over 5 min"},
		{Signal: "dependency_latency_p99_ms", Source: "Application Insights dependency telemetry", AlertWhen: "p99 > target × 2 over 5 min"},
		{Signal: "dlq_depth", Source: "Service Bus metric", AlertWhen: "DLQ depth > 0 (any message warrants triage)"},
		{Signal: "request_rejection_rate", Source: "ingress/app logs", AlertWhen: "rejected (503/429) > 1% over 5 min"},
	}
}

func nextSteps(in Input) []string {
	return []string{
		"Implement the dependency controls above using a resilience library (Polly, gobreaker, or platform mesh).",
		"Verify idempotency keys are propagated on every retryable mutation in code review.",
		"Wire the detection signals into Application Insights with dashboards and alerts.",
		"Run a chaos test that opens a circuit breaker in staging and confirms the fallback behaves as designed.",
		"Document the runbook for the most likely incident shape (one downstream becomes slow).",
	}
}

func computeCoverage(in Input, out Plan) int {
	score := 0
	if len(in.Dependencies) > 0 && len(out.DependencyControls) > 0 {
		score += 40
	}
	if len(out.BulkheadNotes) > 0 {
		score += 15
	}
	if len(out.LoadLevelingNotes) > 0 {
		score += 15
	}
	if len(out.Fallbacks) > 0 {
		score += 15
	}
	if len(out.DetectionSignals) > 0 {
		score += 15
	}
	if score > 100 {
		score = 100
	}
	return score
}

func isHigh(s Service) bool { return strings.EqualFold(s.Criticality, "high") }
